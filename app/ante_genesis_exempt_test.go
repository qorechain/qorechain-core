package app

import (
	"errors"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var errFloor = errors.New("insufficient fee")

type refusingDecorator struct{}

func (refusingDecorator) AnteHandle(ctx sdk.Context, _ sdk.Tx, _ bool, _ sdk.AnteHandler) (sdk.Context, error) {
	return ctx, errFloor
}

// Fee-free gentxs are exempt from the min-gas-price floor during InitChain.
// The exemption keyed on height 0, which is InitChain only for
// initial_height 1: a chain started at initial_height 1000 runs its gentxs
// at height 1000 and refused them ("insufficient fee"). It now follows the
// InitChain flag, and a block at the same height is still checked.
func TestGenesisExemptFollowsInitChainNotHeightZero(t *testing.T) {
	inInitChain := false
	executing := func(ctx sdk.Context) bool { return ctx.BlockHeight() == 0 || inInitChain }
	d := genesisExemptDecorator{inner: refusingDecorator{}, executingGenesis: executing}
	tx := &memTx{}

	run := func(height int64) error {
		n := 0
		_, err := d.AnteHandle(sdk.Context{}.WithBlockHeight(height), tx, false, reached(&n))
		if err == nil && n != 1 {
			t.Fatalf("height %d: admitted without reaching next", height)
		}
		return err
	}

	if err := run(0); err != nil {
		t.Fatalf("InitChain at initial_height 1 (height 0): %v", err)
	}
	inInitChain = true
	if err := run(1000); err != nil {
		t.Fatalf("InitChain at initial_height 1000: %v", err)
	}
	inInitChain = false
	if err := run(1000); !errors.Is(err, errFloor) {
		t.Fatalf("a block at height 1000: %v, want the floor applied", err)
	}

	// No InitChain flag wired: the height-0 rule, as before.
	plain := genesisExemptDecorator{inner: refusingDecorator{}}
	if _, err := plain.AnteHandle(sdk.Context{}.WithBlockHeight(0), tx, false, reached(new(int))); err != nil {
		t.Fatalf("nil flag at height 0: %v", err)
	}
	if _, err := plain.AnteHandle(sdk.Context{}.WithBlockHeight(1000), tx, false, reached(new(int))); !errors.Is(err, errFloor) {
		t.Fatalf("nil flag at height 1000: %v", err)
	}
}
