package reputation

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// QSR-2026-0511: a genesis with proto-JSON quoted 64-bit integers is refused
// by ValidateGenesis with an error, and InitGenesis stops with the cause
// wrapped and a hint, rather than a bare formatted string.
func TestReputationGenesisQuotedIntegers(t *testing.T) {
	bz := json.RawMessage(`{"params":{},"validators":[{"address":"x","uptime_blocks":"42"}]}`)

	if err := (AppModuleBasic{}).ValidateGenesis(nil, nil, bz); err == nil {
		t.Fatal("ValidateGenesis accepted quoted integers")
	}

	defer func() {
		r := recover()
		err, ok := r.(error)
		if !ok {
			t.Fatalf("InitGenesis panic value is %T (%v), want an error", r, r)
		}
		var typeErr *json.UnmarshalTypeError
		if !errors.As(err, &typeErr) {
			t.Fatalf("cause not wrapped: %v", err)
		}
		if !strings.Contains(err.Error(), "genesis validate") {
			t.Fatalf("no hint in %q", err)
		}
	}()
	AppModule{}.InitGenesis(sdk.Context{}, nil, bz)
}
