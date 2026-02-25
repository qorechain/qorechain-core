//go:build proprietary

package keeper

import (
	"fmt"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	rlconsensusmod "github.com/qorechain/qorechain-core/x/rlconsensus"
	"github.com/qorechain/qorechain-core/x/rlconsensus/mathutil"
)

// QDRWTallyHandler computes QDRW voting power for governance proposals.
type QDRWTallyHandler struct {
	keeper           *Keeper
	tokenomicsKeeper rlconsensusmod.TokenomicsKeeper
}

// NewQDRWTallyHandler creates a new QDRW tally handler.
func NewQDRWTallyHandler(keeper *Keeper, tokenomicsKeeper rlconsensusmod.TokenomicsKeeper) *QDRWTallyHandler {
	return &QDRWTallyHandler{
		keeper:           keeper,
		tokenomicsKeeper: tokenomicsKeeper,
	}
}

// CalculateVotingPower computes the QDRW voting power for a voter.
// VP(v) = sqrt(staked + xqore_multiplier * xQORE) * ReputationMultiplier(r)
func (h *QDRWTallyHandler) CalculateVotingPower(
	ctx sdk.Context,
	voterAddr sdk.AccAddress,
	stakedAmount uint64,
	reputationScore float64,
) (math.LegacyDec, error) {
	config := h.keeper.GetConfig(ctx)
	qdrwConfig := config.QDRWConfig

	if !qdrwConfig.Enabled {
		// QDRW disabled -- return raw stake as voting power
		return math.LegacyNewDec(int64(stakedAmount)), nil
	}

	xqoreMultiplier, err := math.LegacyNewDecFromStr(qdrwConfig.XQOREMultiplier)
	if err != nil {
		return math.LegacyZeroDec(), fmt.Errorf("invalid xqore_multiplier: %w", err)
	}

	// Get xQORE balance (stubbed to 0 until tokenomics module is implemented)
	xqoreBalance := h.tokenomicsKeeper.GetXQOREBalance(ctx, voterAddr)
	xqoreDec := math.LegacyNewDecFromInt(xqoreBalance)

	// staked + xqore_multiplier * xQORE
	stakedDec := math.LegacyNewDec(int64(stakedAmount))
	totalStake := stakedDec.Add(xqoreMultiplier.Mul(xqoreDec))

	// sqrt(totalStake)
	sqrtStake := mathutil.IntegerSqrt(totalStake)

	// ReputationMultiplier(r) using float64 to LegacyDec conversion
	repDec, err := math.LegacyNewDecFromStr(fmt.Sprintf("%.18f", reputationScore))
	if err != nil {
		repDec = math.LegacyMustNewDecFromStr("0.5")
	}
	repMultiplier := mathutil.ReputationMultiplier(repDec)

	// VP = sqrt(totalStake) * repMultiplier
	vp := sqrtStake.Mul(repMultiplier)

	return vp, nil
}

// IsEnabled returns whether QDRW governance is enabled.
func (h *QDRWTallyHandler) IsEnabled(ctx sdk.Context) bool {
	config := h.keeper.GetConfig(ctx)
	return config.QDRWConfig.Enabled
}
