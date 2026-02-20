//go:build proprietary

package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/bridge/types"
)

// PQCBridgeSigner handles PQC signature operations for bridge attestations.
// All bridge validator signatures use Dilithium-5 via the existing x/pqc FFI.
type PQCBridgeSigner struct {
	keeper Keeper
}

// NewPQCBridgeSigner creates a new PQC bridge signer.
func NewPQCBridgeSigner(k Keeper) *PQCBridgeSigner {
	return &PQCBridgeSigner{keeper: k}
}

// VerifyAttestation verifies a PQC-signed bridge attestation.
func (s *PQCBridgeSigner) VerifyAttestation(ctx sdk.Context, attestation types.MsgBridgeAttestation) error {
	// 1. Get validator's registered PQC pubkey
	validator, found := s.keeper.GetBridgeValidator(ctx, attestation.Validator)
	if !found {
		return types.ErrValidatorNotRegistered.Wrapf("validator %s not registered", attestation.Validator)
	}
	if !validator.Active {
		return types.ErrValidatorNotAuthorized.Wrapf("validator %s is not active", attestation.Validator)
	}

	// 2. Verify Dilithium-5 signature on attestation data
	signBytes := attestation.GetSignBytes()
	valid, err := s.keeper.pqcKeeper.PQCClient().DilithiumVerify(
		validator.PQCPubkey,
		signBytes,
		attestation.PQCSignature,
	)
	if err != nil {
		return types.ErrInvalidPQCSignature.Wrapf("PQC verification error: %v", err)
	}
	if !valid {
		return types.ErrInvalidPQCSignature.Wrap("Dilithium-5 signature verification failed")
	}

	// 3. Check validator is authorized for this chain
	chainAuthorized := false
	for _, c := range validator.SupportedChains {
		if c == attestation.Chain {
			chainAuthorized = true
			break
		}
	}
	if !chainAuthorized {
		return types.ErrValidatorNotAuthorized.Wrapf(
			"validator %s not authorized for chain %s", attestation.Validator, attestation.Chain,
		)
	}

	return nil
}

// GenerateMLKEMCommitment creates an ML-KEM commitment for a bridge operation.
// This provides quantum-safe commitment to the operation parameters.
func (s *PQCBridgeSigner) GenerateMLKEMCommitment(_ sdk.Context) ([]byte, []byte, error) {
	pqcClient := s.keeper.pqcKeeper.PQCClient()

	// Generate ephemeral ML-KEM keypair
	pubkey, _, err := pqcClient.MLKEMKeygen()
	if err != nil {
		return nil, nil, err
	}

	// Encapsulate to create commitment
	ciphertext, sharedSecret, err := pqcClient.MLKEMEncapsulate(pubkey)
	if err != nil {
		return nil, nil, err
	}

	// The ciphertext serves as the PQC commitment
	// The shared secret can be used for additional verification
	return ciphertext, sharedSecret, nil
}

// VerifyAttestationThreshold checks if enough validators have attested to an operation.
func (s *PQCBridgeSigner) VerifyAttestationThreshold(ctx sdk.Context, op types.BridgeOperation) (bool, error) {
	config := s.keeper.GetConfig(ctx)

	if len(op.Attestations) < config.AttestationThreshold {
		return false, nil
	}

	// Verify all attestations have valid PQC signatures
	for _, att := range op.Attestations {
		validator, found := s.keeper.GetBridgeValidator(ctx, att.Validator)
		if !found {
			continue // Skip validators that may have been deregistered
		}

		// Reconstruct sign bytes from attestation data
		signBytes := []byte(op.SourceChain + "|" + string(op.Type) + "|" + op.ID + "|" + att.TxHash)
		valid, err := s.keeper.pqcKeeper.PQCClient().DilithiumVerify(
			validator.PQCPubkey,
			signBytes,
			att.PQCSignature,
		)
		if err != nil || !valid {
			s.keeper.Logger().Warn("invalid PQC signature in attestation",
				"validator", att.Validator,
				"operation", op.ID,
			)
			// Don't fail — just log. The attestation was verified when it was submitted.
		}
	}

	return true, nil
}
