package types

const (
	// QCB-bridge umbrella license: holders may operate the bridge protocol
	// at all (independent of any specific chain license).
	FeatureQCBBridge = "qcb_bridge"

	// Light-node operator license: required (in addition to a min QOR stake) to
	// register a light node. Granted by the governance authority after the
	// off-chain (dashboard) registration fee is paid.
	FeatureLightNodeOperator = "lightnode_operator"

	// Base validator license: required to create a validator at all (in addition
	// to the min QOR self-bond). Granted by the governance authority. (Distinct
	// from FeatureValidatorBase below, which is the Base-chain validator license.)
	FeatureValidatorOperator = "validator_operator"

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

	// ----- Validator licenses (v2.27.0): 19 new non-IBC chains -----
	//
	// Note: Injective (added as a chain in v2.25.0) is IBC-typed; its
	// validator license is registered below in the IBC group, not here.

	FeatureValidatorZKSyncEra   = "validator_zksync_era"
	FeatureValidatorLinea       = "validator_linea"
	FeatureValidatorScroll      = "validator_scroll"
	FeatureValidatorStarknet    = "validator_starknet"
	FeatureValidatorBlast       = "validator_blast"
	FeatureValidatorMantle      = "validator_mantle"
	FeatureValidatorHyperliquid = "validator_hyperliquid"
	FeatureValidatorBerachain   = "validator_berachain"
	FeatureValidatorSonic       = "validator_sonic"
	FeatureValidatorSei         = "validator_sei"
	FeatureValidatorMonad       = "validator_monad"
	FeatureValidatorPlasma      = "validator_plasma"
	FeatureValidatorXRPL        = "validator_xrpl"
	FeatureValidatorStellar     = "validator_stellar"
	FeatureValidatorHedera      = "validator_hedera"
	FeatureValidatorAlgorand    = "validator_algorand"
	FeatureValidatorFilecoin    = "validator_filecoin"
	FeatureValidatorCronos      = "validator_cronos"
	FeatureValidatorKaia        = "validator_kaia"

	// ----- IBC validator licenses (v2.27.0): 7 chains -----
	//
	// These chains already have IBC connectivity; their bridge_* license is
	// implicit through the IBC handler. The validator role (running a
	// remote validator on these chains as part of cross-network validation)
	// is the new feature granted here.

	FeatureValidatorCosmosHub = "validator_cosmoshub"
	FeatureValidatorOsmosis   = "validator_osmosis"
	FeatureValidatorNoble     = "validator_noble"
	FeatureValidatorCelestia  = "validator_celestia"
	FeatureValidatorStride    = "validator_stride"
	FeatureValidatorAkash     = "validator_akash"
	FeatureValidatorBabylon   = "validator_babylon"

	// IBC chain validator (added as a v2.25.0 chain — Injective)
	FeatureValidatorInjective = "validator_injective"
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
// Stable order grouped by addition wave:
//   - v1.4.0 baseline: 10 chains
//   - v2.27.0 non-IBC additions: 19 chains
//   - v2.27.0 IBC additions: 8 chains (7 pre-existing IBC + Injective)
//
// Total in v2.27.0 = 37 entries.
func AllValidatorFeatureIDs() []string {
	return []string{
		// v1.4.0 baseline
		FeatureValidatorEthereum, FeatureValidatorSolana, FeatureValidatorBSC,
		FeatureValidatorPolygon, FeatureValidatorArbitrum, FeatureValidatorOptimism,
		FeatureValidatorBase, FeatureValidatorAvalanche, FeatureValidatorTON,
		FeatureValidatorSui,
		// v2.27.0 non-IBC
		FeatureValidatorZKSyncEra, FeatureValidatorLinea, FeatureValidatorScroll,
		FeatureValidatorStarknet, FeatureValidatorBlast, FeatureValidatorMantle,
		FeatureValidatorHyperliquid, FeatureValidatorBerachain, FeatureValidatorSonic,
		FeatureValidatorSei, FeatureValidatorMonad, FeatureValidatorPlasma,
		FeatureValidatorXRPL, FeatureValidatorStellar, FeatureValidatorHedera,
		FeatureValidatorAlgorand, FeatureValidatorFilecoin, FeatureValidatorCronos,
		FeatureValidatorKaia,
		// v2.27.0 IBC validator licenses
		FeatureValidatorCosmosHub, FeatureValidatorOsmosis, FeatureValidatorNoble,
		FeatureValidatorCelestia, FeatureValidatorStride, FeatureValidatorAkash,
		FeatureValidatorBabylon, FeatureValidatorInjective,
	}
}

func AllFeatureIDs() []string {
	out := []string{FeatureQCBBridge, FeatureLightNodeOperator, FeatureValidatorOperator}
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
