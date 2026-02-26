//go:build proprietary

package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/rdk/types"
)

// GetPresetProfile returns a hardcoded preset rollup configuration for the given profile.
func GetPresetProfile(profile types.RollupProfile) types.RollupConfig {
	switch profile {
	case types.ProfileDeFi:
		return types.RollupConfig{
			Profile:        types.ProfileDeFi,
			SettlementMode: types.SettlementZK,
			SequencerConfig: types.SequencerConfig{
				Mode:             types.SequencerDedicated,
				SharedSetMinSize: 1,
				InclusionDelay:   10,
				PriorityFeeShare: "0.0",
			},
			DABackend:     types.DANative,
			BlockTimeMs:   500,
			MaxTxPerBlock: 10000,
			GasConfig: types.RollupGasConfig{
				GasModel:     "eip1559",
				BaseGasPrice: "0.001",
				MaxGasLimit:  30000000,
			},
			VMType: "evm",
			ProofConfig: types.ProofConfig{
				System:             types.ProofSystemSNARK,
				ChallengeWindowSec: 0, // Instant finality on proof
				ChallengeBond:      0,
				MaxProofSize:       1048576,
				RecursionDepth:     2,
			},
		}

	case types.ProfileGaming:
		return types.RollupConfig{
			Profile:        types.ProfileGaming,
			SettlementMode: types.SettlementBased,
			SequencerConfig: types.SequencerConfig{
				Mode:             types.SequencerBased,
				SharedSetMinSize: 1,
				InclusionDelay:   2,
				PriorityFeeShare: "0.5",
			},
			DABackend:     types.DANative,
			BlockTimeMs:   200,
			MaxTxPerBlock: 50000,
			GasConfig: types.RollupGasConfig{
				GasModel:     "flat",
				BaseGasPrice: "0.0001",
				MaxGasLimit:  50000000,
			},
			VMType: "custom",
			ProofConfig: types.ProofConfig{
				System:             types.ProofSystemNone,
				ChallengeWindowSec: 0,
				ChallengeBond:      0,
				MaxProofSize:       0,
				RecursionDepth:     0,
			},
		}

	case types.ProfileNFT:
		return types.RollupConfig{
			Profile:        types.ProfileNFT,
			SettlementMode: types.SettlementOptimistic,
			SequencerConfig: types.SequencerConfig{
				Mode:             types.SequencerDedicated,
				SharedSetMinSize: 1,
				InclusionDelay:   10,
				PriorityFeeShare: "0.0",
			},
			DABackend:     types.DACelestia,
			BlockTimeMs:   2000,
			MaxTxPerBlock: 5000,
			GasConfig: types.RollupGasConfig{
				GasModel:     "standard",
				BaseGasPrice: "0.01",
				MaxGasLimit:  10000000,
			},
			VMType: "cosmwasm",
			ProofConfig: types.ProofConfig{
				System:             types.ProofSystemFraud,
				ChallengeWindowSec: 604800, // 7 days
				ChallengeBond:      1000000000,
				MaxProofSize:       2097152,
				RecursionDepth:     0,
			},
		}

	case types.ProfileEnterprise:
		return types.RollupConfig{
			Profile:        types.ProfileEnterprise,
			SettlementMode: types.SettlementBased,
			SequencerConfig: types.SequencerConfig{
				Mode:             types.SequencerBased,
				SharedSetMinSize: 1,
				InclusionDelay:   5,
				PriorityFeeShare: "0.3",
			},
			DABackend:     types.DANative,
			BlockTimeMs:   1000,
			MaxTxPerBlock: 20000,
			GasConfig: types.RollupGasConfig{
				GasModel:     "subsidized",
				BaseGasPrice: "0.0",
				MaxGasLimit:  20000000,
			},
			VMType: "evm",
			ProofConfig: types.ProofConfig{
				System:             types.ProofSystemNone,
				ChallengeWindowSec: 0,
				ChallengeBond:      0,
				MaxProofSize:       0,
				RecursionDepth:     0,
			},
		}

	default:
		// Custom profile returns minimal defaults
		return types.RollupConfig{
			Profile:         types.ProfileCustom,
			SettlementMode:  types.SettlementOptimistic,
			SequencerConfig: types.DefaultSequencerConfig(),
			DABackend:       types.DANative,
			BlockTimeMs:     1000,
			MaxTxPerBlock:   10000,
			GasConfig:       types.DefaultRollupGasConfig(),
			VMType:          "evm",
			ProofConfig:     types.DefaultProofConfig(),
		}
	}
}

// SuggestProfile delegates to the RL consensus module for AI-assisted profile selection.
func (k Keeper) SuggestProfile(ctx sdk.Context, useCase string) (*types.RollupProfile, error) {
	suggested, err := k.rlKeeper.SuggestRollupProfile(ctx, useCase)
	if err != nil {
		// Fallback to DeFi profile
		p := types.ProfileDeFi
		return &p, nil
	}
	profile := types.RollupProfile(suggested)
	return &profile, nil
}

// OptimizeGasConfig delegates to the RL consensus module for AI-optimized gas parameters.
func (k Keeper) OptimizeGasConfig(ctx sdk.Context, rollupID string) (*types.RollupGasConfig, error) {
	// Build metrics from current rollup state
	metrics := map[string]uint64{
		"block_time_ms": 0,
		"max_tx":        0,
	}
	if rollup, err := k.GetRollup(ctx, rollupID); err == nil {
		metrics["block_time_ms"] = rollup.BlockTimeMs
		metrics["max_tx"] = rollup.MaxTxPerBlock
	}

	optimizedGas, err := k.rlKeeper.OptimizeRollupGas(ctx, metrics)
	if err != nil || optimizedGas == 0 {
		gc := types.DefaultRollupGasConfig()
		return &gc, nil
	}

	gc := types.RollupGasConfig{
		GasModel:     "standard",
		BaseGasPrice: "0.001",
		MaxGasLimit:  optimizedGas,
	}
	return &gc, nil
}
