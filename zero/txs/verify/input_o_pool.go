package verify

import (
	"github.com/sero-cash/go-sero/zero/zconfig"

	"github.com/sero-cash/go-czero-import/c_czero"
	"github.com/sero-cash/go-czero-import/c_type"
	"github.com/sero-cash/go-czero-import/cpt"
	"github.com/sero-cash/go-sero/zero/localdb"
	"github.com/sero-cash/go-sero/zero/txs/stx"
	"github.com/sero-cash/go-sero/zero/utils"
)

var verify_input_o_procs_pool = utils.NewProcsPool(func() int { return zconfig.G_v_thread_num })

type verify_input_o_desc struct {
	hash_z   c_type.Uint256
	src      localdb.OutState
	in       stx.In_S
	asset_cc c_type.Uint256
	e        error
}

func (self *verify_input_o_desc) Run() error {
	g := c_czero.VerifyInputSDesc{}
	g.Ehash = self.hash_z
	g.Nil = self.in.Nil
	g.RootCM = *self.src.ToRootCM()
	g.Sign = self.in.Sign
	g.Pkr = *self.src.ToPKr()
	if err := c_czero.VerifyInputS(&g); err != nil {
		self.e = err
		return err
	} else {
		asset := self.src.Out_O.Asset.ToFlatAsset()
		asset_desc := c_czero.AssetDesc{
			Tkn_currency: asset.Tkn.Currency,
			Tkn_value:    asset.Tkn.Value.ToUint256(),
			Tkt_category: asset.Tkt.Category,
			Tkt_value:    asset.Tkt.Value,
		}
		cpt.GenAssetCC(&asset_desc)
		self.asset_cc = asset_desc.Asset_cc
		return nil
	}
}
