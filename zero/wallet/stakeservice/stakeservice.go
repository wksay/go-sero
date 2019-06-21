package stakeservice

import (
	"github.com/robfig/cron"
	"github.com/sero-cash/go-sero/common"
	"github.com/sero-cash/go-sero/common/hexutil"
	"github.com/sero-cash/go-sero/zero/utils"
	"sync"
	"sync/atomic"

	"github.com/sero-cash/go-czero-import/keys"
	"github.com/sero-cash/go-czero-import/seroparam"
	"github.com/sero-cash/go-sero/accounts"
	"github.com/sero-cash/go-sero/core"
	"github.com/sero-cash/go-sero/event"
	"github.com/sero-cash/go-sero/log"
	"github.com/sero-cash/go-sero/serodb"
	"github.com/sero-cash/go-sero/zero/stake"
)

type Account struct {
	pk *keys.Uint512
	tk *keys.Uint512
}

type StakeService struct {
	bc             *core.BlockChain
	accountManager *accounts.Manager
	db             *serodb.LDBDatabase

	nextBlockNumber uint64

	accounts sync.Map

	feed    event.Feed
	updater event.Subscription        // Wallet update subscriptions for all backends
	update  chan accounts.WalletEvent // Subscription sink for backend wallet changes
	quit    chan chan error
	lock    sync.RWMutex
}

var current_StakeService *StakeService

func CurrentStakeService() *StakeService {
	return current_StakeService
}

func NewStakeService(dbpath string, bc *core.BlockChain, accountManager *accounts.Manager) *StakeService {
	update := make(chan accounts.WalletEvent, 1)
	updater := accountManager.Subscribe(update)

	stakeService := &StakeService{
		bc:             bc,
		accountManager: accountManager,
		update:         update,
		updater:        updater,
	}
	current_StakeService = stakeService

	db, err := serodb.NewLDBDatabase(dbpath, 1024, 1024)
	if err != nil {
		panic(err)
	}
	stakeService.db = db

	stakeService.accounts = sync.Map{}
	for _, w := range accountManager.Wallets() {
		stakeService.initWallet(w)
	}

	value, err := db.Get(nextKey)
	if err != nil {
		stakeService.nextBlockNumber = uint64(1)
	} else {
		stakeService.nextBlockNumber = utils.DecodeNumber(value)
	}

	AddJob("0/10 * * * * ?", stakeService.stakeIndex)
	return stakeService
}

func (self *StakeService) StakePools() (pools []*stake.StakePool) {
	iterator := self.db.NewIteratorWithPrefix(poolPrefix)
	for iterator.Next() {

		value := iterator.Value()
		pool := stake.StakePoolDB.GetObject(self.bc.GetDB(), value, &stake.StakePool{})
		pools = append(pools, pool.(*stake.StakePool))
	}
	return
}

func (self *StakeService) Shares() (shares []*stake.Share) {
	iterator := self.db.NewIteratorWithPrefix(sharePrefix)
	for iterator.Next() {
		value := iterator.Value()
		share := stake.ShareDB.GetObject(self.bc.GetDB(), value, &stake.Share{})
		shares = append(shares, share.(*stake.Share))
	}
	return
}

func (self *StakeService) SharesById(id common.Hash) *stake.Share {
	hash, err := self.db.Get(sharekey(id[:]))
	if err != nil {
		return nil
	}
	return self.getShareByHash(hash)
}

func (self *StakeService) getShareByHash(hash []byte) *stake.Share {
	ret := stake.ShareDB.GetObject(self.bc.GetDB(), hash, &stake.Share{})
	if ret == nil {
		return nil
	}
	return ret.(*stake.Share)
}

func (self *StakeService) SharesByPk(pk keys.Uint512) (shares []*stake.Share) {
	iterator := self.db.NewIteratorWithPrefix(pk[:])
	for iterator.Next() {
		value := iterator.Value()
		share := stake.ShareDB.GetObject(self.bc.GetDB(), value, &stake.Share{})
		shares = append(shares, share.(*stake.Share))
	}
	return
}

func (self *StakeService) GetBlockRecords(blockNumber uint64) (shares []*stake.Share, pools []*stake.StakePool) {
	header := self.bc.GetHeaderByNumber(blockNumber)
	return stake.GetBlockRecords(self.bc.GetDB(), header.Hash(), blockNumber)
}

func (self *StakeService) stakeIndex() {
	header := self.bc.CurrentHeader()

	blockNumber := self.nextBlockNumber

	sharesCount := 0
	poolsCount := 0
	batch := self.db.NewBatch()
	for blockNumber+seroparam.DefaultConfirmedBlock() < header.Number.Uint64() {
		shares, pools := self.GetBlockRecords(blockNumber + 1)
		for _, share := range shares {
			batch.Put(sharekey(share.Id()), share.State())
			if pk, ok := self.ownPkr(share.PKr); ok {
				batch.Put(pkShareKey(pk, share.Id()), share.State())
			}
		}

		for _, pool := range pools {
			log.Info("indexpool","id",hexutil.Encode(pool.Id()), "hash",hexutil.Encode(pool.State()))
			batch.Put(poolKey(pool.Id()), pool.State())
		}
		sharesCount += len(shares)
		poolsCount += len(pools)
		blockNumber += 1
	}

	if batch.ValueSize() > 0 {
		batch.Put(nextKey, utils.EncodeNumber(blockNumber+1))
		err := batch.Write()
		if err == nil {
			self.nextBlockNumber = blockNumber
			log.Info("StakeIndex", "blockNumber", blockNumber, "sharesCount", sharesCount, "poolsCount", poolsCount)
		}
	} else {
		self.nextBlockNumber = blockNumber
	}
}

func (self *StakeService) ownPkr(pkr keys.PKr) (pk *keys.Uint512, ok bool) {
	var account *Account
	self.accounts.Range(func(key, value interface{}) bool {
		a := value.(*Account)
		if keys.IsMyPKr(a.tk, &pkr) {
			account = a
			return false
		}
		return true
	})
	if account != nil {
		return account.pk, true
	}
	return
}

var (
	sharePrefix = []byte("SHARE")
	poolPrefix  = []byte("POOL")
	nextKey     = []byte("NEXT")
)

func pkShareKey(pk *keys.Uint512, key []byte) []byte {
	return append(pk[:], key[:]...)
}

func sharekey(key []byte) []byte {
	return append(sharePrefix, key[:]...)
}

func poolKey(key []byte) []byte {
	return append(poolPrefix, key[:]...)
}

func (self *StakeService) initWallet(w accounts.Wallet) {

	if _, ok := self.accounts.Load(*w.Accounts()[0].Address.ToUint512()); !ok {
		account := Account{}
		account.pk = w.Accounts()[0].Address.ToUint512()
		account.tk = w.Accounts()[0].Tk.ToUint512()
		self.accounts.Store(*account.pk, &account)
	}
}

func AddJob(spec string, run RunFunc) *cron.Cron {
	c := cron.New()
	c.AddJob(spec, &RunJob{run: run})
	c.Start()
	return c
}

type (
	RunFunc func()
)

type RunJob struct {
	runing int32
	run    RunFunc
}

func (r *RunJob) Run() {
	x := atomic.LoadInt32(&r.runing)
	if x == 1 {
		return
	}

	atomic.StoreInt32(&r.runing, 1)
	defer func() {
		atomic.StoreInt32(&r.runing, 0)
	}()

	r.run()
}