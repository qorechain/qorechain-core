package types

const (
	// QCB-bridge umbrella license: holders may operate the bridge protocol
	// at all (independent of any specific chain license).
	FeatureQCBBridge = "qcb_bridge"

	// ----- Bridge licenses (pre-v2.26.0): 16 chains -----

	FeatureBridgeEthereum  = "bridge_ethereum"
	FeatureBridgeSolana    = "bridge_solana"
	FeatureBridgeBSC       = "bridge_bsc"
	FeatureBridgePolygon   = "bridge_polygon"
	FeatureBridgeArbitrum  = "bridge_arbitrum"
	FeatureBridgeOptimism  = "bridge_optimism"
	FeatureBridgeBase      = "bridge_base"
	FeatureBridgeAvalanche = "bridge_avalanche"
	FeatureBridgeTON       = "bridge_ton"
	FeatureBridgeSui       = "bridge_sui"
	FeatureBridgeBitcoin   = "bridge_bitcoin"
	FeatureBridgeNEAR      = "bridge_near"
	FeatureBridgeCardano   = "bridge_cardano"
	FeatureBridgePolkadot  = "bridge_polkadot"
	FeatureBridgeTezos     = "bridge_tezos"
	FeatureBridgeTRON      = "bridge_tron"

	// ----- Bridge licenses (v2.26.0): 20 new chains for cross-network expansion -----

	// EVM L2 ZK rollups
	FeatureBridgeZKSyncEra = "bridge_zksync_era"
	FeatureBridgeLinea     = "bridge_linea"
	FeatureBridgeScroll    = "bridge_scroll"

	// Cairo L2
	FeatureBridgeStarknet = "bridge_starknet"

	// EVM L2 Optimistic (yield-bearing)
	FeatureBridgeBlast = "bridge_blast"

	// EVM L2
	FeatureBridgeMantle = "bridge_mantle"

	// HyperEVM L1
	FeatureBridgeHyperliquid = "bridge_hyperliquid"

	// EVM L1
	FeatureBridgeBerachain = "bridge_berachain"
	FeatureBridgeSonic     = "bridge_sonic"
	FeatureBridgeMonad     = "bridge_monad"
	FeatureBridgePlasma    = "bridge_plasma"
	FeatureBridgeFilecoin  = "bridge_filecoin"
	FeatureBridgeCronos    = "bridge_cronos"
	FeatureBridgeKaia      = "bridge_kaia"

	// Parallel EVM L1 (Cosmos-based; dual EVM+IBC)
	FeatureBridgeSei = "bridge_sei"

	// Non-EVM L1
	FeatureBridgeXRPL     = "bridge_xrpl"
	FeatureBridgeStellar  = "bridge_stellar"
	FeatureBridgeHedera   = "bridge_hedera"
	FeatureBridgeAlgorand = "bridge_algorand"

	// IBC
	FeatureBridgeInjective = "bridge_injective"

	// ----- Validator licenses (pre-v2.27.0): 10 chains -----

	FeatureValidatorEthereum  = "validator_ethereum"
	FeatureValidatorSolana    = "validator_solana"
	FeatureValidatorBSC       = "validator_bsc"
	FeatureValidatorPolygon   = "validator_polygon"
	FeatureValidatorArbitrum  = "validator_arbitrum"
	FeatureValidatorOptimism  = "validator_optimism"
	FeatureValidatorBase      = "validator_base"
	FeatureValidatorAvalanche = "validator_avalanche"
	FeatureValidatorTON       = "validator_ton"
	FeatureValidatorSui       = "validator_sui"
)

// AllBridgeFeatureIDs returns every bridge_* license feature ID.
// Stable order — the keeper's iteration over licenses uses this list.
func AllBridgeFeatureIDs() []string {
	return []string{
		// Pre-v2.26.0
		FeatureBridgeEthereum, FeatureBridgeSolana, FeatureBridgeBSC,
		FeatureBridgePolygon, FeatureBridgeArbitrum, FeatureBridgeOptimism,
		FeatureBridgeBase, FeatureBridgeAvalanche, FeatureBridgeTON,
		FeatureBridgeSui, FeatureBridgeBitcoin, FeatureBridgeNEAR,
		FeatureBridgeCardano, FeatureBridgePolkadot, FeatureBridgeTezos,
		FeatureBridgeTRON,
		// v2.26.0
		FeatureBridgeZKSyncEra, FeatureBridgeLinea, FeatureBridgeScroll,
		FeatureBridgeStarknet, FeatureBridgeBlast, FeatureBridgeMantle,
		FeatureBridgeHyperliquid, FeatureBridgeBerachain, FeatureBridgeSonic,
		FeatureBridgeMonad, FeatureBridgePlasma, FeatureBridgeFilecoin,
		FeatureBridgeCronos, FeatureBridgeKaia, FeatureBridgeSei,
		FeatureBridgeXRPL, FeatureBridgeStellar, FeatureBridgeHedera,
		FeatureBridgeAlgorand, FeatureBridgeInjective,
	}
}

// AllValidatorFeatureIDs returns every validator_* license feature ID.
// Stable order. Will grow in v2.27.0 with the new chains' validator
// licenses + the 7 IBC chains.
func AllValidatorFeatureIDs() []string {
	return []string{
		FeatureValidatorEthereum, FeatureValidatorSolana, FeatureValidatorBSC,
		FeatureValidatorPolygon, FeatureValidatorArbitrum, FeatureValidatorOptimism,
		FeatureValidatorBase, FeatureValidatorAvalanche, FeatureValidatorTON,
		FeatureValidatorSui,
	}
}

func AllFeatureIDs() []string {
	out := []string{FeatureQCBBridge}
	out = append(out, AllBridgeFeatureIDs()...)
	out = append(out, AllValidatorFeatureIDs()...)
	return out
}

func IsValidFeatureID(id string) bool {
	for _, f := range AllFeatureIDs() {
		if f == id {
			return true
		}
	}
	return false
}

func ChainFromFeature(featureID string) string {
	for _, prefix := range []string{"bridge_", "validator_"} {
		if len(featureID) > len(prefix) && featureID[:len(prefix)] == prefix {
			return featureID[len(prefix):]
		}
	}
	return ""
}
