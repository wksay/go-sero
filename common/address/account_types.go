// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package address

import (
	"errors"

	"github.com/sero-cash/go-sero/zero/utils"

	"github.com/btcsuite/btcutil/base58"

	"github.com/sero-cash/go-czero-import/c_superzk"
	"github.com/sero-cash/go-czero-import/c_type"
	"github.com/sero-cash/go-czero-import/superzk"
)

// Lengths of hashes and Accountes in bytes.
const (
	// AccountAddressLength is the expected length of the adddress
	AccountAddressLength = 64
	SeedLength           = 32
)

type Seed [SeedLength]byte

func (priv *Seed) SeedToUint256() *c_type.Uint256 {
	seed := c_type.Uint256{}
	copy(seed[:], priv[:])
	return &seed

}

type MixBase58Adrress []byte

func (b MixBase58Adrress) MarshalText() ([]byte, error) {
	return []byte(base58.Encode(b)), nil
}

func (b MixBase58Adrress) IsPkr() bool {
	return len(b) == 96
}

func (b MixBase58Adrress) ToPkr() c_type.PKr {
	var pkr c_type.PKr
	if b.IsPkr() {
		copy(pkr[:], b[:])
	} else {
		var pk c_type.Uint512
		copy(pk[:], b[:])
		pkr = superzk.Pk2PKr(&pk, nil)
	}
	return pkr
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (b *MixBase58Adrress) UnmarshalText(input []byte) error {

	if len(input) == 0 {
		return errors.New("empty string")
	}

	if addr, e := utils.NewAddressByString(string(input)); e != nil {
		return e
	} else {
		if addr.IsHex {
			return errors.New("is not base58 address")
		}
		out := addr.Bytes
		if len(out) == 96 {
			err := ValidPkr(addr)
			if err != nil {
				return err
			}
			*b = out[:]
			return nil
		} else if len(out) == 64 {
			err := ValidPk(addr)
			if err != nil {
				return err
			}
			*b = out[:]
			return nil
		} else {
			return errors.New("invalid mix address")
		}
	}
}

func ValidPkr(addr utils.Address) error {
	if len(addr.Bytes) == 96 {
		if !addr.MatchProtocol("SC") {
			return errors.New("address protocol is not pkr")
		}
		var pkr c_type.PKr
		copy(pkr[:], addr.Bytes)
		if !addr.IsHex {
			if c_superzk.IsSzkPKr(&pkr) {
				if addr.Protocol == "" {
					return errors.New("pkr address is new version  must have prefix SC")
				}
			}
		}

		if !superzk.IsPKrValid(&pkr) {
			return errors.New("invalid pkr")
		}
	} else {
		return errors.New("pkr address must be 96 bytes")
	}
	return nil
}

func ValidPk(addr utils.Address) error {
	if len(addr.Bytes) == 64 {
		if !addr.MatchProtocol("SP") {
			return errors.New("address protocol is not pk")
		}
		pk := c_type.Uint512{}
		copy(pk[:], addr.Bytes)
		if c_superzk.IsSzkPK(&pk) {
			if addr.Protocol == "" {
				return errors.New("pk is new version mush have prefix scp1")
			}
		}
		if !superzk.IsPKValid(&pk) {
			return errors.New("invalid PK")
		}
	} else {
		return errors.New("pk address must be 64 bytes")
	}
	return nil
}

type TKAddress [64]byte

func Base58ToTk(str string) (ret TKAddress) {
	b := base58.Decode(str)
	copy(ret[:], b)
	return
}

func (b TKAddress) ToTk() c_type.Tk {
	result := c_type.Tk{}
	copy(result[:], b[:])

	return result
}

func (c TKAddress) String() string {
	return base58.Encode(c[:])
}

func (b TKAddress) MarshalText() ([]byte, error) {
	return []byte(base58.Encode(b[:])), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (b *TKAddress) UnmarshalText(input []byte) error {

	if len(input) == 0 {
		return nil
	}
	if addr, e := utils.NewAddressByString(string(input)); e != nil {
		return e
	} else {
		if !addr.MatchProtocol("ST") {
			return errors.New("address protocol is not tk")
		}
		if len(addr.Bytes) == 64 {
			copy(b[:], addr.Bytes)
		} else {
			return errors.New("ivalid TK")
		}
		return nil
	}
}

type PKAddress [64]byte

func Base58ToPk(str string) (ret PKAddress) {
	b := base58.Decode(str)
	copy(ret[:], b)
	return
}

func (b PKAddress) String() string {
	if c_superzk.IsFlagSet(b[:]) {
		a := utils.NewAddressByBytes(b[:])
		a.SetProtocol("SP")
		return a.ToCode()
	} else {
		return base58.Encode(b[:])
	}
}

func (b PKAddress) ToUint512() c_type.Uint512 {
	result := c_type.Uint512{}
	copy(result[:], b[:])

	return result
}

func NewPKAddres(b []byte) (ret PKAddress) {
	copy(ret[:], b)
	return
}

func (b PKAddress) MarshalText() ([]byte, error) {
	return []byte(b.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (b *PKAddress) UnmarshalText(input []byte) error {
	if len(input) == 0 {
		return nil
	}
	if addr, e := utils.NewAddressByString(string(input)); e != nil {
		return e
	} else {
		if !addr.MatchProtocol("SP") {
			return errors.New("address protocol is not pk")
		}
		err := ValidPk(addr)
		if err != nil {
			return err
		}
		copy(b[:], addr.Bytes)
		return nil
	}
}
