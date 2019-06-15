package lstate

import (
	"github.com/sero-cash/go-czero-import/keys"
	"github.com/sero-cash/go-sero/zero/light/light_ref"
	"github.com/sero-cash/go-sero/zero/lstate/balance"
	"github.com/sero-cash/go-sero/zero/lstate/lstate_types"
	"github.com/sero-cash/go-sero/zero/txs/zstate"
	"github.com/sero-cash/go-sero/zero/utils"
)

var current_lstate LState

type LState struct {
	b *balance.Balance
}

func (self *LState) ZState() *zstate.ZState {
	return light_ref.Ref_inst.GetState()
}

func (self *LState) GetOut(root *keys.Uint256) (src *lstate_types.OutState, e error) {
	return self.b.GetOut(root)
}

func (self *LState) GetPkgs(tk *keys.Uint512, is_from bool) (ret []*lstate_types.Pkg) {
	return self.b.GetPkgs(tk, is_from)
}

func (self *LState) GetOuts(tk *keys.Uint512) (outs []*lstate_types.OutState, e error) {
	return self.b.GetOuts(tk)
}

func (self *LState) AddAccount(tk *keys.Uint512) (ret bool) {
	return self.b.AddAccount(tk)
}

func (self *LState) GetAccount(tk *keys.Uint512) (tkn map[keys.Uint256]*utils.U256, tkt map[keys.Uint256][]keys.Uint256) {
	a := self.b.GetAccount(tk)
	tkn = a.Token
	tkt = a.Ticket
	return
}

func CurrentLState() *LState {
	return &current_lstate
}

func InitLState() {
	current_lstate.b = balance.NewBalance()
	return
}
