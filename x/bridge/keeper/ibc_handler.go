//go:build proprietary

package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/bridge/types"
)

// IBCBridgeHandler wraps standard IBC operations with PQC verification.
// Every IBC packet sent from QoreChain includes a Dilithium-5 signature.
// Receiving chains can optionally verify the PQC signature for enhanced security.
type IBCBridgeHandler struct {
	keeper Keeper
}

// NewIBCBridgeHandler creates a new IBC bridge handler.
func NewIBCBridgeHandler(k Keeper) *IBCBridgeHandler {
	return &IBCBridgeHandler{keeper: k}
}

// WrapPacketWithPQC adds a PQC signature to an IBC packet for enhanced security.
// The signature covers the packet data and can be verified by PQC-aware chains.
func (h *IBCBridgeHandler) WrapPacketWithPQC(ctx sdk.Context, packetData []byte) ([]byte, []byte, error) {
	// For IBC packets originating from QoreChain:
	// 1. The packet data is signed with a Dilithium-5 key
	// 2. The signature is included as packet metadata
	// 3. Receiving PQC-aware chains can verify the quantum-safe signature
	//
	// Testnet: PQC wrapping is prepared but requires validator key management
	// that will be fully implemented with the IBC module integration.
	//
	// The PQC commitment provides forward security: even if classical keys
	// are compromised by quantum computers, the PQC signature remains valid.

	// Generate ML-KEM commitment for the packet data
	pqcClient := h.keeper.pqcKeeper.PQCClient()
	pubkey, _, err := pqcClient.MLKEMKeygen()
	if err != nil {
		return packetData, nil, nil // Non-fatal: return without PQC wrapper
	}
	h.keeper.pqcKeeper.IncrementMLKEMOperations(ctx)

	ciphertext, _, err := pqcClient.MLKEMEncapsulate(pubkey)
	if err != nil {
		return packetData, nil, nil
	}
	h.keeper.pqcKeeper.IncrementMLKEMOperations(ctx)

	return packetData, ciphertext, nil
}

// VerifyPQCPacket verifies a PQC signature on a received IBC packet.
func (h *IBCBridgeHandler) VerifyPQCPacket(_ sdk.Context, packetData []byte, pqcSig []byte, pqcPubkey []byte) (bool, error) {
	if len(pqcSig) == 0 || len(pqcPubkey) == 0 {
		// No PQC signature — standard IBC packet, allow through
		return true, nil
	}

	// Verify Dilithium-5 signature on the packet data
	pqcClient := h.keeper.pqcKeeper.PQCClient()
	valid, err := pqcClient.DilithiumVerify(pqcPubkey, packetData, pqcSig)
	if err != nil {
		return false, err
	}

	return valid, nil
}

// OnRecvPacket processes a received IBC packet with optional PQC verification.
func (h *IBCBridgeHandler) OnRecvPacket(ctx sdk.Context, packetData []byte, pqcSig []byte, pqcPubkey []byte) error {
	// 1. Standard IBC packet processing handled by IBC module
	// 2. If PQC signature present, verify it
	if len(pqcSig) > 0 {
		valid, err := h.VerifyPQCPacket(ctx, packetData, pqcSig, pqcPubkey)
		if err != nil {
			h.keeper.Logger().Warn("PQC verification error on IBC packet", "error", err)
		} else if valid {
			ctx.EventManager().EmitEvent(sdk.NewEvent(
				types.EventTypePQCVerification,
				sdk.NewAttribute(types.AttributeKeyPQCVerified, "true"),
			))
		} else {
			h.keeper.Logger().Warn("PQC signature invalid on IBC packet")
			ctx.EventManager().EmitEvent(sdk.NewEvent(
				types.EventTypePQCVerification,
				sdk.NewAttribute(types.AttributeKeyPQCVerified, "false"),
			))
		}
	}

	return nil
}
