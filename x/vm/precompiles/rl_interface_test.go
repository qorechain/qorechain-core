package precompiles

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestStaticRLProviderDefaults(t *testing.T) {
	p := DefaultStaticRLProvider()

	if p.BlockTimeMs != 5000 {
		t.Errorf("expected 5000, got %d", p.BlockTimeMs)
	}
	if p.BaseGasPrice != 100 {
		t.Errorf("expected 100, got %d", p.BaseGasPrice)
	}
	if p.ValidatorSetSize != 100 {
		t.Errorf("expected 100, got %d", p.ValidatorSetSize)
	}
}

func TestStaticRLProviderMethods(t *testing.T) {
	p := DefaultStaticRLProvider()
	ctx := sdk.Context{} // StaticRLProvider ignores context

	if p.GetCurrentBlockTime(ctx) != 5*time.Second {
		t.Error("block time mismatch")
	}

	gasPrice := p.GetCurrentBaseGasPrice(ctx)
	if !gasPrice.Equal(math.LegacyNewDec(100)) {
		t.Errorf("gas price mismatch: %s", gasPrice.String())
	}

	if p.GetValidatorSetSize(ctx) != 100 {
		t.Error("validator set size mismatch")
	}

	if p.GetCurrentEpoch(ctx) != 0 {
		t.Error("epoch should be 0")
	}

	if p.IsRLActive(ctx) {
		t.Error("RL should not be active")
	}
}

func TestStaticRLProviderCustomValues(t *testing.T) {
	p := &StaticRLProvider{
		BlockTimeMs:      3000,
		BaseGasPrice:     200,
		ValidatorSetSize: 50,
	}
	ctx := sdk.Context{}

	if p.GetCurrentBlockTime(ctx) != 3*time.Second {
		t.Error("custom block time mismatch")
	}
	if p.GetValidatorSetSize(ctx) != 50 {
		t.Errorf("expected 50, got %d", p.GetValidatorSetSize(ctx))
	}
}
