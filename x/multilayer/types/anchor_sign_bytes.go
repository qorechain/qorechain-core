package types

import "encoding/binary"

// AnchorSignDomainV2 tags the v2 anchor sign bytes.
const AnchorSignDomainV2 = "qorechain-multilayer-anchor-v2"

// AnchorSignBytesFor returns the bytes a state anchor's PQC signature covers.
// v2 is on from v3.2.2 (FeatureRDKIntegrity); below that height v1 applies.
func AnchorSignBytesFor(chainID string, msg *MsgAnchorState, v2 bool) []byte {
	if v2 {
		return AnchorSignBytesV2(chainID, msg.LayerID, msg.LayerHeight, msg.StateRoot,
			msg.ValidatorSetHash, msg.TransactionCount, msg.CompressedStateProof)
	}
	return AnchorSignBytesV1(msg.LayerID, msg.LayerHeight, msg.StateRoot, msg.ValidatorSetHash)
}

// AnchorSignBytesV1 is the pre-v3.2.2 anchor message:
// layer_id | layer_height (8-byte BE) | state_root | validator_set_hash,
// with no length prefixes and no chain id, so field boundaries are ambiguous
// and a signature is valid on every chain.
func AnchorSignBytesV1(layerID string, layerHeight uint64, stateRoot, validatorSetHash []byte) []byte {
	b := make([]byte, 0, len(layerID)+8+len(stateRoot)+len(validatorSetHash))
	b = append(b, []byte(layerID)...)
	h := make([]byte, 8)
	binary.BigEndian.PutUint64(h, layerHeight)
	b = append(b, h...)
	b = append(b, stateRoot...)
	b = append(b, validatorSetHash...)
	return b
}

// AnchorSignBytesV2 is the v3.2.2 anchor message. Every variable-length field
// carries an 8-byte big-endian length, the chain id binds the signature to one
// network, and the transaction count and compressed state proof are covered
// too, because the anchor fraud check (canonical transition root) is computed
// from them:
//
//	"qorechain-multilayer-anchor-v2"
//	| len(chain_id) chain_id
//	| len(layer_id) layer_id
//	| layer_height (8-byte BE)
//	| len(state_root) state_root
//	| len(validator_set_hash) validator_set_hash
//	| transaction_count (8-byte BE)
//	| len(compressed_state_proof) compressed_state_proof
//
// An RDK sequencer signs this with the rollup creator's registered Dilithium
// key, over layer_id = the rollup's layer id, layer_height = the batch index,
// state_root = the batch state root, transaction_count = the batch tx count and
// an empty compressed_state_proof.
func AnchorSignBytesV2(chainID, layerID string, layerHeight uint64, stateRoot, validatorSetHash []byte,
	txCount uint64, compressedStateProof []byte) []byte {
	out := make([]byte, 0, len(AnchorSignDomainV2)+6*8+len(chainID)+len(layerID)+
		len(stateRoot)+len(validatorSetHash)+len(compressedStateProof))
	out = append(out, AnchorSignDomainV2...)
	putBytes := func(b []byte) {
		out = binary.BigEndian.AppendUint64(out, uint64(len(b)))
		out = append(out, b...)
	}
	putBytes([]byte(chainID))
	putBytes([]byte(layerID))
	out = binary.BigEndian.AppendUint64(out, layerHeight)
	putBytes(stateRoot)
	putBytes(validatorSetHash)
	out = binary.BigEndian.AppendUint64(out, txCount)
	putBytes(compressedStateProof)
	return out
}
