package types

import (
	"encoding/binary"
)

const (
	// ModuleName defines the module name for the QoreChain multi-layer architecture
	ModuleName = "multilayer"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName

	// QuerierRoute defines the module's query routing key
	QuerierRoute = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_multilayer"
)

// KVStore key prefixes for the multi-layer architecture module
var (
	// LayerKeyPrefix stores layer configurations: 0x01 | layer_id -> LayerConfig
	LayerKeyPrefix = []byte{0x01}

	// AnchorKeyPrefix stores state anchors: 0x02 | layer_id | "/" | height(8 bytes) -> StateAnchor
	AnchorKeyPrefix = []byte{0x02}

	// LatestAnchorKeyPrefix stores latest anchor per layer: 0x03 | layer_id -> StateAnchor
	LatestAnchorKeyPrefix = []byte{0x03}

	// RoutingStatsKeyPrefix stores QCAI routing statistics: 0x04 -> RoutingStats
	RoutingStatsKeyPrefix = []byte{0x04}

	// CrossLayerMessageKeyPrefix stores pending cross-layer messages: 0x05 | msg_id -> CrossLayerMessage
	CrossLayerMessageKeyPrefix = []byte{0x05}

	// ParamsKey stores module parameters: 0x06 -> Params
	ParamsKey = []byte{0x06}
)

// LayerKey returns the store key for a layer configuration
func LayerKey(layerID string) []byte {
	return append(LayerKeyPrefix, []byte(layerID)...)
}

// AnchorKey returns the store key for a specific state anchor
func AnchorKey(layerID string, height uint64) []byte {
	key := append(AnchorKeyPrefix, []byte(layerID)...)
	key = append(key, '/')
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, height)
	key = append(key, heightBytes...)
	return key
}

// AnchorPrefixForLayer returns the key prefix for all anchors of a specific layer
func AnchorPrefixForLayer(layerID string) []byte {
	key := append(AnchorKeyPrefix, []byte(layerID)...)
	key = append(key, '/')
	return key
}

// LatestAnchorKey returns the store key for the latest anchor of a layer
func LatestAnchorKey(layerID string) []byte {
	return append(LatestAnchorKeyPrefix, []byte(layerID)...)
}

// CrossLayerMessageKey returns the store key for a cross-layer message
func CrossLayerMessageKey(messageID string) []byte {
	return append(CrossLayerMessageKeyPrefix, []byte(messageID)...)
}
