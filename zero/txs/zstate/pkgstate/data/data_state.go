package data

import (
	"errors"

	"github.com/sero-cash/go-czero-import/keys"
	"github.com/sero-cash/go-sero/serodb"
	"github.com/sero-cash/go-sero/zero/localdb"
	"github.com/sero-cash/go-sero/zero/txs/zstate/tri"
)

func (self *Data) SaveState(tr tri.Tri) {
	G2pkgs_dirty := self.IdDirtys.List()
	for _, k := range G2pkgs_dirty {
		self.Id2Hash.Save(tr, &k)
	}
}

func (self *Data) RecordState(putter serodb.Putter, hash *keys.Uint256) {
	if pkg, ok := self.Hash2Pkg[*hash]; ok {
		localdb.PutPkg(putter, hash, pkg)
	} else {
		panic(errors.New("PKG record index error: hash2pkg"))
	}
}

func (self *Data) GetPkgById(tr tri.Tri, id *keys.Uint256) (pg *localdb.ZPkg) {
	if hash := self.Id2Hash.GetByTri(tr, id); hash != keys.Empty_Uint256 {
		pg = localdb.GetPkg(tr.GlobalGetter(), &hash)
		return
	} else {
		return
	}
}

func (self *Data) GetPkgByHash(tr tri.Tri, hash *keys.Uint256) (pg *localdb.ZPkg) {
	pg = localdb.GetPkg(tr.GlobalGetter(), hash)
	return
}