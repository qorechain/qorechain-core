package govauth

import (
	"strings"
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func TestAssertAcceptsOnlyTheGovAccount(t *testing.T) {
	gov := Address()
	if gov == "" {
		t.Fatal("gov module address resolved empty")
	}
	if err := Assert(gov); err != nil {
		t.Fatalf("gov account must be accepted, got %v", err)
	}

	// The attack this exists to stop: the caller writes their own address into
	// the authority field and signs the transaction themselves.
	for _, bad := range []string{
		"",
		"qor1py5v23qgzl2v2tgx5jt3mrxm3f28cmv986cw6e",
		gov + "x",
		strings.ToUpper(gov),
		"not-an-address",
	} {
		err := Assert(bad)
		if err == nil {
			t.Fatalf("authority %q must be rejected", bad)
		}
		if !strings.Contains(err.Error(), sdkerrors.ErrUnauthorized.Error()) {
			t.Fatalf("authority %q: want an unauthorized error, got %v", bad, err)
		}
	}
}

func TestAddressIsStable(t *testing.T) {
	// Derived, not injected — so it must not depend on call order or state.
	if a, b := Address(), Address(); a != b {
		t.Fatalf("address not stable: %s vs %s", a, b)
	}
}
