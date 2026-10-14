package types

import (
	"bytes"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// QSR-2026-0474: the v2 digest binds the gas limit; v1 is unchanged.
func TestEVMAuthSignBytesV2BindsGasLimit(t *testing.T) {
	pub := bytes.Repeat([]byte{7}, 32)
	v1 := EVMAuthSignBytes("c", "qor1acc", pub, "0x01", "0", []byte{1, 2}, 3)
	a := EVMAuthSignBytesV2("c", "qor1acc", pub, "0x01", "0", []byte{1, 2}, 3, 100000)
	b := EVMAuthSignBytesV2("c", "qor1acc", pub, "0x01", "0", []byte{1, 2}, 3, 100001)
	if bytes.Equal(a, b) {
		t.Fatal("a different gas limit must give a different v2 digest")
	}
	if bytes.Equal(a, v1) {
		t.Fatal("v2 must not equal v1")
	}
	// v1 is pinned: a wallet that signed it before must still match.
	again := EVMAuthSignBytes("c", "qor1acc", pub, "0x01", "0", []byte{1, 2}, 3)
	if !bytes.Equal(v1, again) || len(v1) != 32 {
		t.Fatal("v1 digest changed")
	}
}

// QSR-2026-0433: the reported collision exists in the legacy layout and not in
// the framed one.
func TestSpendAccumKeyFramedNoCollision(t *testing.T) {
	acct := sdk.AccAddress(bytes.Repeat([]byte{1}, 20)).String()
	legacyA := SpendAccumKey(acct, []byte("/abc"), "uqor", 5)
	legacyB := SpendAccumKey(acct, nil, "abc/uqor", 5)
	if !bytes.Equal(legacyA, legacyB) {
		t.Fatal("expected the legacy layout to collide (the reported case)")
	}
	framedA := SpendAccumKeyFramed(acct, []byte("/abc"), "uqor", 5)
	framedB := SpendAccumKeyFramed(acct, nil, "abc/uqor", 5)
	if bytes.Equal(framedA, framedB) {
		t.Fatal("framed keys must not collide")
	}
	if bytes.Equal(SpendAccumIntKeyFramed(acct, []byte("/abc"), "uqor", 5), SpendAccumIntKeyFramed(acct, nil, "abc/uqor", 5)) {
		t.Fatal("framed int keys must not collide")
	}
	// Same prefix and trailing day as the legacy layout, so export and import
	// read both alike.
	if !bytes.HasPrefix(framedA, SpendAccumPrefix) {
		t.Fatal("framed key must sit under SpendAccumPrefix")
	}
	if d := SpendAccumKeyDay(framedA[len(SpendAccumPrefix):]); d != 5 {
		t.Fatalf("day %d, want 5", d)
	}
	// Genesis accepts both layouts and refuses a truncated framed key.
	for _, k := range [][]byte{legacyA, framedA, framedB} {
		if err := validateSpendAccumSuffix(k[len(SpendAccumPrefix):]); err != nil {
			t.Fatalf("suffix %x refused: %v", k, err)
		}
	}
	if err := validateSpendAccumSuffix(framedA[len(SpendAccumPrefix) : len(framedA)-1]); err == nil {
		t.Fatal("a truncated framed key must be refused")
	}
}

// QSR-2026-0403 and QSR-2026-0436 (a): strict registration checks.
func TestValidateAuthenticatorStrict(t *testing.T) {
	base := Authenticator{
		Scheme:      SchemeSecp256k1,
		PubKey:      append([]byte{0x04}, bytes.Repeat([]byte{1}, 64)...),
		Permissions: []string{PermSend},
		Expiry:      time.Unix(1_900_000_000, 0),
	}
	if err := ValidateAuthenticator(base); err != nil {
		t.Fatalf("the historical check accepts a 65-byte key (replay): %v", err)
	}
	if err := ValidateAuthenticatorStrict(base); err == nil {
		t.Fatal("a 65-byte key must be refused by the strict check")
	}

	c := base
	c.PubKey = append([]byte{0x02}, bytes.Repeat([]byte{1}, 32)...)
	if err := ValidateAuthenticatorStrict(c); err != nil {
		t.Fatalf("a compressed key with known permissions must pass: %v", err)
	}
	c.Permissions = []string{PermSend, "sned"}
	if err := ValidateAuthenticator(c); err != nil {
		t.Fatalf("the historical check accepts unknown permissions: %v", err)
	}
	if err := ValidateAuthenticatorStrict(c); err == nil {
		t.Fatal("an unknown permission must be refused by the strict check")
	}
}

// QSR-2026-0436 (b).
func TestValidateAccountAddress(t *testing.T) {
	if err := ValidateAccountAddress(sdk.AccAddress(bytes.Repeat([]byte{2}, 20)).String()); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"", "not-an-address", "qor1"} {
		if err := ValidateAccountAddress(bad); err == nil {
			t.Fatalf("%q must be refused", bad)
		}
	}
}
