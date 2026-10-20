package cmd

import (
	"strings"
	"testing"

	"cosmossdk.io/math"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	antetypes "github.com/cosmos/evm/ante/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

// QSR-2026-0577: the EVM dynamic-fee extension option is not registered in
// the app, so a Cosmos tx carrying it never decodes. The ante route and the
// extension checker entry for it were dead code and are gone; this pins the
// premise that made removing them a no-op.
func TestDynamicFeeExtensionOptionDoesNotDecode(t *testing.T) {
	a := sharedTestApp(t)
	cfg := a.TxConfig()

	encode := func(opt *codectypes.Any) []byte {
		b := cfg.NewTxBuilder()
		ext, ok := b.(tx.ExtensionOptionsTxBuilder)
		if !ok {
			t.Fatalf("tx builder %T cannot carry extension options", b)
		}
		ext.SetExtensionOptions(opt)
		bz, err := cfg.TxEncoder()(b.GetTx())
		if err != nil {
			t.Fatal(err)
		}
		return bz
	}

	dyn, err := codectypes.NewAnyWithValue(&antetypes.ExtensionOptionDynamicFeeTx{MaxPriorityPrice: math.LegacyNewDec(1)})
	if err != nil {
		t.Fatal(err)
	}
	_, err = cfg.TxDecoder()(encode(dyn))
	if err == nil {
		t.Fatal("a tx carrying ExtensionOptionDynamicFeeTx must not decode")
	}
	if !strings.Contains(err.Error(), "ExtensionOptionDynamicFeeTx") {
		t.Fatalf("unexpected decode error: %v", err)
	}

	// Control: a registered extension option does decode, so the refusal
	// above is about the type, not the encoding.
	eth, err := codectypes.NewAnyWithValue(&evmtypes.ExtensionOptionsEthereumTx{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cfg.TxDecoder()(encode(eth)); err != nil {
		t.Fatalf("a tx carrying a registered extension option must decode: %v", err)
	}
}
