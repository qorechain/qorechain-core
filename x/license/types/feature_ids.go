package types

const (
	FeatureQCBBridge       = "qcb_bridge"
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

func AllFeatureIDs() []string {
	return []string{
		FeatureQCBBridge,
		FeatureBridgeEthereum, FeatureBridgeSolana, FeatureBridgeBSC,
		FeatureBridgePolygon, FeatureBridgeArbitrum, FeatureBridgeOptimism,
		FeatureBridgeBase, FeatureBridgeAvalanche, FeatureBridgeTON,
		FeatureBridgeSui, FeatureBridgeBitcoin, FeatureBridgeNEAR,
		FeatureBridgeCardano, FeatureBridgePolkadot, FeatureBridgeTezos,
		FeatureBridgeTRON,
		FeatureValidatorEthereum, FeatureValidatorSolana, FeatureValidatorBSC,
		FeatureValidatorPolygon, FeatureValidatorArbitrum, FeatureValidatorOptimism,
		FeatureValidatorBase, FeatureValidatorAvalanche, FeatureValidatorTON,
		FeatureValidatorSui,
	}
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
