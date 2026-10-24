package cli

import (
	"math/big"
	"strconv"
	"testing"

	sdkmath "cosmossdk.io/math"

	"github.com/qorechain/qorechain-core/x/pqc/types"
)

// The arguments of open-evm-window are optional, and the defaults are a valid
// window that covers a contract deployment session (plan item 11).
func TestOpenEVMWindowDefaultsCoverADeploy(t *testing.T) {
	cmd := GetCmdOpenEVMWindow()
	for n := 0; n <= 3; n++ {
		if err := cmd.Args(cmd, make([]string, n)); err != nil {
			t.Fatalf("%d arguments refused: %v", n, err)
		}
	}
	if err := cmd.Args(cmd, make([]string, 4)); err == nil {
		t.Fatal("four arguments accepted")
	}

	blocks, err := strconv.ParseUint(defaultEVMWindowArgs[0], 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	maxTxs, err := strconv.ParseUint(defaultEVMWindowArgs[1], 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	maxValue, err := parseValueUqor(defaultEVMWindowArgs[2])
	if err != nil {
		t.Fatal(err)
	}
	msg := types.MsgOpenEVMWindow{Sender: "qor1x", Blocks: blocks, MaxTxs: maxTxs, MaxValue: maxValue}
	if err := msg.Validate(); err != nil {
		t.Fatalf("the default window is invalid: %v", err)
	}

	// What the window counts for one deploy is its gas limit times its fee cap
	// (x/pqc evmOutflow). Diana and mainnet today: base fee 100 gwei, tip 12.5
	// gwei, and wallets cap at twice the base fee plus the tip. A large deploy
	// with Foundry's 1.3x gas margin is about 8,000,000 gas.
	feeCap := big.NewInt(212_500_000_000)
	deploy := types.WeiToUqorRoundUp(sdkmath.NewIntFromBigInt(new(big.Int).Mul(feeCap, big.NewInt(8_000_000))))
	// Three such deploys at a base fee four times today's must still fit.
	need := deploy.MulRaw(3 * 4)
	if maxValue.LT(need) {
		t.Fatalf("default max-value %s uqor does not cover %s uqor", maxValue, need)
	}
	if maxTxs < 5 {
		t.Fatalf("default max-txs %d is too few for a deploy and its set-up calls", maxTxs)
	}
}
