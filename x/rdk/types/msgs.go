package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// The rdk Msg types are generated from proto/qorechain/rdk/v1/tx.proto
// (see tx.pb.go). The ValidateBasic methods below are attached to them.

func (m *MsgCreateRollup) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}
	if m.RollupID == "" {
		return fmt.Errorf("rollup_id cannot be empty")
	}
	if m.StakeAmount <= 0 {
		return fmt.Errorf("stake_amount must be positive")
	}
	return nil
}

func (m *MsgSubmitBatch) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sequencer); err != nil {
		return fmt.Errorf("invalid sequencer address: %w", err)
	}
	if m.RollupID == "" {
		return fmt.Errorf("rollup_id cannot be empty")
	}
	if len(m.StateRoot) == 0 {
		return fmt.Errorf("state_root cannot be empty")
	}
	return nil
}

func (m *MsgChallengeBatch) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Challenger); err != nil {
		return fmt.Errorf("invalid challenger address: %w", err)
	}
	if m.RollupID == "" {
		return fmt.Errorf("rollup_id cannot be empty")
	}
	return nil
}

func (m *MsgResolveChallenge) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Resolver); err != nil {
		return fmt.Errorf("invalid resolver address: %w", err)
	}
	if m.RollupID == "" {
		return fmt.Errorf("rollup_id cannot be empty")
	}
	return nil
}

func (m *MsgPauseRollup) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}
	if m.RollupID == "" {
		return fmt.Errorf("rollup_id cannot be empty")
	}
	return nil
}

func (m *MsgResumeRollup) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}
	if m.RollupID == "" {
		return fmt.Errorf("rollup_id cannot be empty")
	}
	return nil
}

func (m *MsgStopRollup) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}
	if m.RollupID == "" {
		return fmt.Errorf("rollup_id cannot be empty")
	}
	return nil
}

// RollupEscrowDenom is the denomination a rollup's stake is escrowed in, and so
// the only one its escrow ledger can pay a withdrawal in.
const RollupEscrowDenom = "uqor"

// MaxWithdrawalProofDepth bounds the Merkle path of a withdrawal: a batch
// commits at most 2^64 withdrawals, so a longer path can never verify.
const MaxWithdrawalProofDepth = 64

// ValidateWithdrawal is the stateless check of an L2->L1 withdrawal message.
//
// It is deliberately not ValidateBasic. The SDK runs ValidateBasic before the
// ante handler, in block execution as well as in CheckTx, so adding one would
// change, without any switch, how a historical block treats a malformed
// withdrawal (refused before fees instead of failing in the handler). The
// keeper calls this from v3.2.3 on (FeatureRDKEscrowLedger), at the start of
// ExecuteWithdrawal, which keeps replays below the plan height identical.
func (m *MsgExecuteWithdrawal) ValidateWithdrawal() error {
	if m == nil {
		return ErrInvalidWithdrawal.Wrap("empty message")
	}
	if _, err := sdk.AccAddressFromBech32(m.Submitter); err != nil {
		return ErrInvalidWithdrawal.Wrapf("invalid submitter address: %v", err)
	}
	// Only emptiness is checked: ids created before v3.2.2 may sit outside the
	// id grammar and stay valid, and an unknown id fails on the lookup.
	if m.RollupID == "" {
		return ErrInvalidWithdrawal.Wrap("rollup_id cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(m.Recipient); err != nil {
		return ErrInvalidWithdrawal.Wrapf("invalid recipient address: %v", err)
	}
	if m.Denom != RollupEscrowDenom {
		return ErrInvalidWithdrawal.Wrapf("denom %q: rollup escrow is held in %s only", m.Denom, RollupEscrowDenom)
	}
	if m.Amount <= 0 {
		return ErrInvalidWithdrawal.Wrap("amount must be positive")
	}
	if len(m.Proof) > MaxWithdrawalProofDepth {
		return ErrInvalidWithdrawal.Wrapf("proof has %d levels, at most %d", len(m.Proof), MaxWithdrawalProofDepth)
	}
	for i, sib := range m.Proof {
		if len(sib) != 32 {
			return ErrInvalidWithdrawal.Wrapf("proof level %d is %d bytes, want 32", i, len(sib))
		}
	}
	return nil
}

// ValidateBasic checks the authority address; the parameter values are checked
// by Params.ValidateUpdate in the message server.
func (m *MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	return nil
}

// MaxVerificationKeySize bounds the verification key a message may carry. A
// Groth16 BLS12-381 key for four public inputs in gnark's encoding is a few
// hundred bytes and a QSTK key is 16; the bound only keeps a key from being
// used to store arbitrary data.
const MaxVerificationKeySize = 16 * 1024

// validateVerificationKeyFields is the stateless part of a verification key
// message. Whether the key decodes for the rollup's proof system is checked by
// the keeper, which knows the rollup.
func validateVerificationKeyFields(rollupID string, vk []byte, system string) error {
	if rollupID == "" {
		return fmt.Errorf("rollup_id cannot be empty")
	}
	if len(vk) == 0 {
		return ErrInvalidVerificationKey.Wrap("verification_key cannot be empty")
	}
	if len(vk) > MaxVerificationKeySize {
		return ErrInvalidVerificationKey.Wrapf("verification_key is %d bytes, at most %d", len(vk), MaxVerificationKeySize)
	}
	switch ProofSystem(system) {
	case "", ProofSystemSNARK, ProofSystemSTARK:
	default:
		return ErrInvalidProofSystem.Wrapf("proof_system %q: want snark, stark or empty", system)
	}
	return nil
}

// ValidateBasic checks the creator and the shape of the key. These messages
// are new in v3.2.7, so no historical block holds one that decoded.
func (m *MsgSetVerificationKey) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}
	return validateVerificationKeyFields(m.RollupID, m.VerificationKey, m.ProofSystem)
}

// ValidateBasic checks the authority address and the shape of the key; the
// handler checks that the authority is governance.
func (m *MsgUpdateVerificationKey) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	return validateVerificationKeyFields(m.RollupID, m.VerificationKey, m.ProofSystem)
}

// ToParams returns the parameter set the message carries.
func (m *MsgUpdateParams) ToParams() Params {
	return Params{
		MaxRollups:             m.MaxRollups,
		MinStakeForRollup:      m.MinStakeForRollup,
		RollupCreationBurnRate: m.RollupCreationBurnRate,
		DefaultChallengeWindow: m.DefaultChallengeWindow,
		MaxDABlobSize:          m.MaxDaBlobSize,
		BlobRetentionBlocks:    m.BlobRetentionBlocks,
		MaxBatchesPerBlock:     m.MaxBatchesPerBlock,
		// Zero (the default) for every message built before v3.2.8, so the
		// stored parameter record keeps its bytes.
		FinalizedBatchRetentionBlocks: m.FinalizedBatchRetentionBlocks,
	}
}
