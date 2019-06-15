package balance

import (
	"runtime/debug"
	"time"

	"github.com/sero-cash/go-czero-import/keys"
	"github.com/sero-cash/go-sero/log"
	"github.com/sero-cash/go-sero/zero/light/light_ref"
	"github.com/sero-cash/go-sero/zero/lstate/balance/accounts"
	lstate "github.com/sero-cash/go-sero/zero/lstate/lstate_types"
)

type Balance struct {
	db *accounts.DB
}

func NewBalance() (ret *Balance) {
	ret = &Balance{}
	go ret.run()
	return ret
}

func (self *Balance) parseEntry() uint64 {
	defer func() {
		if r := recover(); r != nil {
			log.Error("parse block chain error : ", "number", light_ref.Ref_inst.Bc.GetCurrenHeader().Number, "recover", r)
			debug.PrintStack()
		}
	}()
	return self.Parse()
}

func (self *Balance) run() {
	for {
		num := self.parseEntry()

		if num <= 1 {
			time.Sleep(1000 * 1000 * 1000 * 8)
		} else {
			time.Sleep(1000 * 1000 * 10)
		}
	}
}

func (self *Balance) GetOut(root *keys.Uint256) (src *lstate.OutState, e error) {
	s, err := self.db.GetOut(root)
	return &s, err
}

func (self *Balance) GetPkgs(tk *keys.Uint512, is_from bool) (ret []*lstate.Pkg) {
	return
}
func (self *Balance) GetOuts(tk *keys.Uint512) (outs []*lstate.OutState, e error) {
	outs, e = self.db.GetOuts(tk)
	lstate.SortOutStats(light_ref.Ref_inst.Bc.GetDB(), outs)
	return
}

func (self *Balance) AddAccount(tk *keys.Uint512) (ret bool) {
	top_num := light_ref.Ref_inst.Bc.GetCurrenHeader().Number.Uint64()
	return self.db.AddAccount(tk, top_num)
}

func (self *Balance) GetAccount(tk *keys.Uint512) (ret accounts.Account) {
	return self.db.GetAccount(tk)
}
