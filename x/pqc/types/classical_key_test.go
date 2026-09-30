package types

import (
	"errors"
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/evm/crypto/ethsecp256k1"
)

func TestValidateClassicalKey(t *testing.T) {
	eth, err := ethsecp256k1.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	ethPub := eth.PubKey().Bytes() // 33-byte compressed
	ethSender := sdk.AccAddress(eth.PubKey().Address()).String()

	cos := secp256k1.GenPrivKey()
	cosPub := cos.PubKey().Bytes()
	cosSender := sdk.AccAddress(cos.PubKey().Address()).String()

	other := secp256k1.GenPrivKey().PubKey().Bytes()

	uncompressed := make([]byte, 65)
	uncompressed[0] = 0x04

	offCurve := append([]byte{0x02}, make([]byte, 32)...)
	for i := 1; i < len(offCurve); i++ {
		offCurve[i] = 0xff
	}

	cases := []struct {
		name   string
		sender string
		pub    []byte
		ok     bool
	}{
		{"empty stays optional", ethSender, nil, true},
		{"unified eth_secp256k1 key of the signer", ethSender, ethPub, true},
		{"cosmos secp256k1 key of the signer", cosSender, cosPub, true},
		{"another account's key", ethSender, other, false},
		{"uncompressed", ethSender, uncompressed, false},
		{"32 bytes of junk", ethSender, make([]byte, 32), false},
		{"not a curve point", ethSender, offCurve, false},
		{"bad sender", "nonsense", ethPub, false},
	}
	for _, c := range cases {
		err := ValidateClassicalKey(c.sender, c.pub)
		if (err == nil) != c.ok {
			t.Errorf("%s: err=%v, want ok=%v", c.name, err, c.ok)
		}
		if err != nil && !errors.Is(err, ErrInvalidClassicalKey) {
			t.Errorf("%s: error %v is not ErrInvalidClassicalKey", c.name, err)
		}
	}
}
