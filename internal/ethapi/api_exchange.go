package ethapi

import (
	"context"
	"fmt"
	"math/big"

	"github.com/sero-cash/go-sero/log"

	"github.com/pkg/errors"

	"github.com/sero-cash/go-czero-import/seroparam"

	"github.com/sero-cash/go-sero/zero/exchange"

	"github.com/sero-cash/go-sero/common"

	"github.com/sero-cash/go-czero-import/keys"
	"github.com/sero-cash/go-sero/common/hexutil"
	"github.com/sero-cash/go-sero/zero/light/light_types"
)

type PublicExchangeAPI struct {
	b Backend
}

func (s *PublicExchangeAPI) GetPkSynced(ctx context.Context, pk *keys.Uint512) (map[string]interface{}, error) {
	currentPKBlock, err := s.b.GetPkNumber(*pk)
	if err != nil {
		return nil, err
	}
	progress := s.b.Downloader().Progress()
	if progress.CurrentBlock >= progress.HighestBlock {
		progress.HighestBlock = progress.CurrentBlock
	}
	exchangeInstance := exchange.CurrentExchange()
	if exchangeInstance == nil {
		return nil, errors.New("exchange mode no start")
	}
	numbers := exchangeInstance.GetUtxoNum(*pk)

	// Otherwise gather the block sync stats
	return map[string]interface{}{
		"currentPKBlock": currentPKBlock,
		"confirmedBlock": seroparam.DefaultConfirmedBlock(),
		"currentBlock":   progress.CurrentBlock,
		"highestBlock":   progress.HighestBlock,
		"utxoCount":      numbers,
	}, nil

}

func (s *PublicExchangeAPI) GetPkr(ctx context.Context, address *keys.Uint512, index *keys.Uint256) (pkr keys.PKr, e error) {
	return s.b.GetPkr(address, index)
}

func (s *PublicExchangeAPI) GetLockedBalances(address keys.Uint512) map[string]*Big {
	result := map[string]*Big{}

	balances := s.b.GetLockedBalances(address)
	for k, v := range balances {
		result[k] = (*Big)(v)
	}
	return result
}

func (s *PublicExchangeAPI) GetMaxAvailable(address keys.Uint512, currency Smbol) (amount *Big) {
	return (*Big)(s.b.GetMaxAvailable(address, string(currency)))
}

func (s *PublicExchangeAPI) GetBalances(ctx context.Context, address keys.Uint512) map[string]*Big {
	result := map[string]*Big{}
	balances := s.b.GetBalances(address)
	for k, v := range balances {
		result[k] = (*Big)(v)
	}
	return result
}

type Big big.Int

func (b Big) MarshalJSON() ([]byte, error) {
	i := big.Int(b)
	by, err := i.MarshalJSON()
	if err != nil {
		return nil, err
	}
	if seroparam.IsExchangeVlueStr() {
		bytes := make([]byte, len(by)+2)
		copy(bytes[1:len(bytes)-1], by[:])
		bytes[0] = '"'
		bytes[len(bytes)-1] = '"'
		return bytes, nil
	} else {
		return by, err
	}
}

// UnmarshalJSON implements json.Unmarshaler.
func (b *Big) UnmarshalJSON(input []byte) error {
	if isString(input) {
		input = input[1 : len(input)-1]
	}
	i := big.Int{}
	if e := i.UnmarshalText(input); e != nil {
		return e
	} else {
		*b = Big(i)
		return nil
	}
}

func (b *Big) ToInt() *big.Int {
	return (*big.Int)(b)
}

func isString(input []byte) bool {
	return len(input) >= 2 && input[0] == '"' && input[len(input)-1] == '"'
}

type ReceptionArgs struct {
	Addr     hexutil.Bytes
	Currency Smbol
	Value    *Big
}

type GenTxArgs struct {
	From       keys.Uint512
	RefoundTo  *keys.PKr
	Receptions []ReceptionArgs
	Gas        uint64
	GasPrice   *Big
	Roots      []keys.Uint256
}

func (args GenTxArgs) check() error {
	if len(args.Receptions) == 0 {
		return errors.New("have no receptions")
	}
	if args.GasPrice == nil {
		return fmt.Errorf("gasPrice not specified")
	}

	if args.RefoundTo != nil {
		if !keys.PKrValid(args.RefoundTo) {
			return errors.New("RefoundTo is not a valid pkr")
		}
	}

	for _, rec := range args.Receptions {
		_, err := validAddress(rec.Addr)
		if err != nil {
			return err
		}
		if rec.Currency.IsEmpty() {
			return errors.Errorf("%v reception currency is nil", hexutil.Encode(rec.Addr[:]))
		}
		if rec.Value == nil {
			return errors.Errorf("%v reception value is nil", hexutil.Encode(rec.Addr[:]))
		}
	}
	return nil

}

func byteToPkr(addr hexutil.Bytes) keys.PKr {
	pkr := keys.PKr{}
	if len(addr) == 64 {
		pk := keys.Uint512{}
		copy(pk[:], addr[:])
		pkr = keys.Addr2PKr(&pk, nil)
	} else {
		copy(pkr[:], addr[:])
	}
	return pkr
}

func (args GenTxArgs) toTxParam() exchange.TxParam {
	gasPrice := args.GasPrice.ToInt()

	if gasPrice.Sign() == 0 {
		gasPrice = new(big.Int).SetUint64(defaultGasPrice)
	}
	receptions := []exchange.Reception{}
	for _, rec := range args.Receptions {
		pkr := byteToPkr(rec.Addr)
		receptions = append(receptions, exchange.Reception{
			pkr,
			string(rec.Currency),
			(*big.Int)(rec.Value),
		})
	}
	return exchange.TxParam{args.From, args.RefoundTo, receptions, args.Gas, gasPrice, args.Roots}
}

func (s *PublicExchangeAPI) GenTx(ctx context.Context, param GenTxArgs) (*light_types.GenTxParam, error) {
	if err := param.check(); err != nil {
		return nil, err
	}

	return s.b.GenTx(param.toTxParam())
}

func (s *PublicExchangeAPI) GenTxWithSign(ctx context.Context, param GenTxArgs) (*light_types.GTx, error) {
	if err := param.check(); err != nil {
		return nil, err
	}
	tx, e := s.b.GenTxWithSign(param.toTxParam())
	return tx, e
}

type Record struct {
	Pkr      keys.PKr
	Root     keys.Uint256
	TxHash   keys.Uint256
	Nil      keys.Uint256
	Num      uint64
	Currency string
	Value    *Big
}

func (s *PublicExchangeAPI) GetRecords(ctx context.Context, address hexutil.Bytes, begin, end uint64) (records []Record, err error) {

	utxos, err := s.b.GetRecords(address, begin, end)
	if err != nil {
		return
	}
	for _, utxo := range utxos {
		if utxo.Asset.Tkn != nil {
			records = append(records, Record{Pkr: utxo.Pkr, Root: utxo.Root, TxHash: utxo.TxHash, Nil: utxo.Nil, Num: utxo.Num, Currency: common.BytesToString(utxo.Asset.Tkn.Currency[:]), Value: (*Big)(utxo.Asset.Tkn.Value.ToIntRef())})
		}
	}
	return
}

func (s *PublicExchangeAPI) Merge(ctx context.Context, address *keys.Uint512, cy Smbol) (map[string]interface{}, error) {
	if address == nil {
		return nil, errors.New("pk can not bi nil")
	}
	if cy == "" {
		return nil, errors.New("cy can not bi nil")

	}
	exchangeInstance := exchange.CurrentExchange()
	if exchangeInstance == nil {
		return nil, errors.New("exchange mode no start")
	}
	count, hash, err := exchangeInstance.Merge(address, string(cy), true)
	log.Info("merge query utxo", "cy=", cy, "count=", count)
	if err != nil {
		return nil, err
	}
	log.Info("merge query utxo", "count=", count)
	txhash := common.Hash{}
	copy(txhash[:], hash[:])
	return map[string]interface{}{
		"utxoCount": count,
		"txhash":    txhash,
	}, nil
}

func validAddress(addr hexutil.Bytes) (bool, error) {
	if len(addr) != 64 && len(addr) != 96 {
		return false, errors.Errorf("invalid addr %v", hexutil.Encode(addr[:]))
	}

	if len(addr) == 64 {
		pk := keys.Uint512{}
		copy(pk[:], addr[:])
		if !keys.IsPKValid(&pk) {
			return false, errors.Errorf("invalid pk %v", hexutil.Encode(addr[:]))
		}
	}
	if len(addr) == 96 {
		pkr := keys.PKr{}
		copy(pkr[:], addr[:])
		if !keys.PKrValid(&pkr) {
			return false, errors.Errorf("invalid  pkr %v", hexutil.Encode(addr[:]))
		}
	}
	return true, nil
}

func (s *PublicExchangeAPI) ValidAddress(ctx context.Context, addr hexutil.Bytes) (bool, error) {
	return validAddress(addr)

}

func (s *PublicExchangeAPI) CommitTx(ctx context.Context, args *light_types.GTx) error {
	return s.b.CommitTx(args)
}

func (s *PublicExchangeAPI) ClearUsedFlag(ctx context.Context, address keys.Uint512) (count int, e error) {
	count = exchange.CurrentExchange().ClearUsedFlagForPK(&address)
	return
}

func (s *PublicExchangeAPI) ClearUsedFlagForRoot(ctx context.Context, roots []keys.Uint256) (count int, e error) {
	if len(roots) > 0 {
		for _, root := range roots {
			count += exchange.CurrentExchange().ClearUsedFlagForRoot(&root)
		}
	}
	return
}
