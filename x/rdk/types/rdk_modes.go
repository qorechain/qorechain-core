package types

import (
	"bytes"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// v3.2.8 RDK settlement modes: qledger-v1 optimistic rollups
// (FeatureRDKOptimisticReexec), the STARK statement gate
// (FeatureRDKStarkStatement) and private rollups (FeatureRDKPrivateRollups).
// Design: handoff-v328/RDK-DATA-BASED-DESIGN.md, sections 12-14.

const (
	// ExecutionVMQLedgerV1 is the qledger-v1 execution model (x/rdk/qledger).
	ExecutionVMQLedgerV1 = "qledger-v1"
	// PrivacyEncapsulated: every post is an ML-KEM-1024 envelope to the
	// rollup's members (x/rdk/privpost).
	PrivacyEncapsulated = "encapsulated"
	// MaxRollupMembers bounds a private rollup's member list (an envelope
	// entry is about 1.7 KB).
	MaxRollupMembers = 32

	// DefaultMinSubmitterBond is 1,000 QOR.
	DefaultMinSubmitterBond uint64 = 1_000_000_000
	// MaxMinSubmitterBond bounds the parameter (1,000,000 QOR).
	MaxMinSubmitterBond uint64 = 1_000_000_000_000
	// MinDepositAmount is the smallest deposit (1 QOR): each deposit is an
	// inbox entry, so dust deposits cannot fill a batch cheaply.
	MinDepositAmount uint64 = 1_000_000
	// QLedgerAccountLen is the length of a qledger-v1 account id.
	QLedgerAccountLen = 32
)

// Errors of the v3.2.8 settlement modes.
var (
	// ErrRDKModeNotEnabled is returned below v3.2.8 for the new messages and
	// for a MsgCreateRollup or MsgUpdateParams that sets one of the new
	// fields.
	ErrRDKModeNotEnabled = errorsmod.Register(ModuleName, 47, "this rollup mode is not enabled at this height")
	// ErrInvalidRollupMode is returned for an execution VM or privacy mode
	// the rollup's settlement cannot use.
	ErrInvalidRollupMode = errorsmod.Register(ModuleName, 48, "invalid rollup execution or privacy mode")
	// ErrSubmitterBondRequired is returned for a qledger-v1 batch whose
	// submitter does not hold the minimum bond.
	ErrSubmitterBondRequired = errorsmod.Register(ModuleName, 49, "submitter bond required")
	// ErrSubmitterBondLocked is returned for an unbond before a full
	// challenge window has passed since the submitter's last batch.
	ErrSubmitterBondLocked = errorsmod.Register(ModuleName, 50, "submitter bond is still locked")
	// ErrFraudNotProven is returned for a qledger-v1 fraud proof that does
	// not show the batch wrong; nothing changes.
	ErrFraudNotProven = errorsmod.Register(ModuleName, 51, "fraud proof does not show the batch invalid")
	// ErrStarkStatementUnsupported is returned from FeatureRDKStarkStatement
	// for the demonstration STARK AIR.
	ErrStarkStatementUnsupported = errorsmod.Register(ModuleName, 52, "this STARK verification key proves no rollup statement")
	// ErrInvalidEnvelope is returned for a post on a private rollup that is
	// not an envelope to its current members.
	ErrInvalidEnvelope = errorsmod.Register(ModuleName, 53, "post is not a valid envelope to the rollup's members")
	// ErrDepositPoolNotEmpty is returned when the creator stops a qledger-v1
	// rollup that still holds user deposits.
	ErrDepositPoolNotEmpty = errorsmod.Register(ModuleName, 54, "rollup still holds user deposits")
)

// Events of the v3.2.8 settlement modes.
const (
	EventRollupDeposit      = "rdk_rollup_deposit"
	EventSubmitterBonded    = "rdk_submitter_bonded"
	EventSubmitterUnbonded  = "rdk_submitter_unbonded"
	EventFraudUpheld        = "rdk_fraud_upheld"
	EventRollupMembersSet   = "rdk_members_set"
	EventInboxRolledBack    = "rdk_inbox_rolled_back"
	EventGovernancePauseSet = "rdk_governance_pause"
)

// Store prefixes of the v3.2.8 settlement modes.
var (
	// SubmitterBondPrefix: <rollup id len-prefixed>/<submitter> -> SubmitterBond.
	SubmitterBondPrefix = []byte("rdk/sbond/")
	// DepositPoolPrefix: <rollup id> -> uqor (be64).
	DepositPoolPrefix = []byte("rdk/dpool/")
)

// SubmitterBondKey is the key of a submitter's bond on a rollup.
func SubmitterBondKey(rollupID, submitter string) []byte {
	k := append([]byte{}, SubmitterBondPrefix...)
	k = append(k, byte(len(rollupID)))
	k = append(k, rollupID...)
	return append(k, submitter...)
}

// DepositPoolKey is the key of a rollup's deposit pool.
func DepositPoolKey(rollupID string) []byte {
	return append(append([]byte{}, DepositPoolPrefix...), rollupID...)
}

// SubmitterBond is a submitter's bond on a qledger-v1 rollup.
type SubmitterBond struct {
	RollupID  string `json:"rollup_id"`
	Submitter string `json:"submitter"`
	Amount    uint64 `json:"amount,string"`
	// LastSubmitTime is the block time (unix seconds) of the submitter's
	// last batch; the bond is locked for a challenge window after it.
	LastSubmitTime int64 `json:"last_submit_time,omitempty"`
}

// IsQLedger reports whether the rollup runs the qledger-v1 execution VM.
func (c RollupConfig) IsQLedger() bool { return c.ExecutionVM == ExecutionVMQLedgerV1 }

// IsPrivate reports whether the rollup is private (encapsulated posts).
func (c RollupConfig) IsPrivate() bool { return c.Privacy == PrivacyEncapsulated }

// validateModes checks the execution VM and privacy against the settlement:
// qledger-v1 is optimistic and based; a private rollup is zk with Groth16
// (outsiders cannot re-execute ciphertext, and the STARK statement is gated).
func (c RollupConfig) validateModes() error {
	switch c.ExecutionVM {
	case "":
	case ExecutionVMQLedgerV1:
		if c.SettlementMode != SettlementOptimistic {
			return ErrInvalidRollupMode.Wrapf("execution_vm %s needs optimistic settlement, not %s", c.ExecutionVM, c.SettlementMode)
		}
		if !c.IsBased() {
			return ErrInvalidRollupMode.Wrapf("execution_vm %s needs based sequencing: its batches consume the inbox", c.ExecutionVM)
		}
		if c.IsPrivate() {
			return ErrInvalidRollupMode.Wrap("a qledger-v1 rollup cannot be private: a fraud proof re-executes plaintext")
		}
	default:
		return ErrInvalidRollupMode.Wrapf("execution_vm %q: want %q or empty", c.ExecutionVM, ExecutionVMQLedgerV1)
	}
	switch c.Privacy {
	case "":
		if len(c.Members) != 0 || c.MembersEpoch != 0 {
			return ErrInvalidRollupMode.Wrap("only a private rollup has members")
		}
	case PrivacyEncapsulated:
		if c.SettlementMode != SettlementZK || c.ProofConfig.System != ProofSystemSNARK {
			return ErrInvalidRollupMode.Wrapf("a private rollup must be zk with a Groth16 (snark) key, not %s/%s",
				c.SettlementMode, c.ProofConfig.System)
		}
		if err := ValidateMembers(c.Members, true); err != nil {
			return err
		}
	default:
		return ErrInvalidRollupMode.Wrapf("privacy %q: want %q or empty", c.Privacy, PrivacyEncapsulated)
	}
	return nil
}

// ValidateMembers checks a member list: valid addresses, no duplicate, at
// most MaxRollupMembers, and at least one unless allowEmpty.
func ValidateMembers(members []string, allowEmpty bool) error {
	if len(members) == 0 && !allowEmpty {
		return ErrInvalidRollupMode.Wrap("a private rollup needs at least one member")
	}
	if len(members) > MaxRollupMembers {
		return ErrInvalidRollupMode.Wrapf("%d members, at most %d", len(members), MaxRollupMembers)
	}
	seen := map[string]bool{}
	for _, m := range members {
		a, err := sdk.AccAddressFromBech32(m)
		if err != nil || a.String() != m {
			return ErrInvalidRollupMode.Wrapf("member %q is not a canonical account address", m)
		}
		if seen[m] {
			return ErrInvalidRollupMode.Wrapf("member %s is listed twice", m)
		}
		seen[m] = true
	}
	return nil
}

// SubmitterBondMin returns the minimum submitter bond, resolving zero to
// the default.
func (p Params) SubmitterBondMin() uint64 {
	if p.MinSubmitterBond == 0 {
		return DefaultMinSubmitterBond
	}
	return p.MinSubmitterBond
}

func (p Params) validateModes() error {
	if p.MinSubmitterBond > MaxMinSubmitterBond {
		return ErrInvalidParams.Wrapf("min_submitter_bond must be 0 (default) or at most %d, got %d",
			MaxMinSubmitterBond, p.MinSubmitterBond)
	}
	return nil
}

// validateGenesisModes checks the qledger-v1 genesis sections.
func validateGenesisModes(rollups map[string]RollupConfig, bonds []SubmitterBond, pools []RollupEscrowEntry) error {
	seen := map[string]bool{}
	for i, b := range bonds {
		r, ok := rollups[b.RollupID]
		if !ok || !r.IsQLedger() {
			return fmt.Errorf("submitter bond %d: rollup %q is not a qledger-v1 rollup in genesis", i, b.RollupID)
		}
		if _, err := sdk.AccAddressFromBech32(b.Submitter); err != nil {
			return fmt.Errorf("submitter bond %d: %w", i, err)
		}
		k := string(SubmitterBondKey(b.RollupID, b.Submitter))
		if seen[k] {
			return fmt.Errorf("submitter bond %d: duplicate", i)
		}
		seen[k] = true
	}
	seenPool := map[string]bool{}
	for i, p := range pools {
		r, ok := rollups[p.RollupID]
		if !ok || !r.IsQLedger() {
			return fmt.Errorf("deposit pool %d: rollup %q is not a qledger-v1 rollup in genesis", i, p.RollupID)
		}
		if seenPool[p.RollupID] {
			return fmt.Errorf("deposit pool %d: duplicate rollup %q", i, p.RollupID)
		}
		seenPool[p.RollupID] = true
	}
	return nil
}

// ValidateBasic checks the depositor, the account id and the amount.
func (m *MsgDepositToRollup) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Depositor); err != nil {
		return fmt.Errorf("invalid depositor address: %w", err)
	}
	if m.RollupID == "" {
		return fmt.Errorf("rollup_id cannot be empty")
	}
	if len(m.Account) != QLedgerAccountLen || bytes.Equal(m.Account, make([]byte, QLedgerAccountLen)) {
		return ErrInvalidRollupMode.Wrapf("account must be a 32-byte qledger-v1 account id")
	}
	if m.Amount < MinDepositAmount {
		return ErrInvalidRollupMode.Wrapf("deposit %d uqor is below the minimum %d", m.Amount, MinDepositAmount)
	}
	return nil
}

// ValidateBasic checks the submitter and the amount.
func (m *MsgBondRollupSubmitter) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Submitter); err != nil {
		return fmt.Errorf("invalid submitter address: %w", err)
	}
	if m.RollupID == "" {
		return fmt.Errorf("rollup_id cannot be empty")
	}
	if m.Amount == 0 {
		return fmt.Errorf("amount must be positive")
	}
	return nil
}

// ValidateBasic checks the submitter.
func (m *MsgUnbondRollupSubmitter) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Submitter); err != nil {
		return fmt.Errorf("invalid submitter address: %w", err)
	}
	if m.RollupID == "" {
		return fmt.Errorf("rollup_id cannot be empty")
	}
	return nil
}

// ValidateBasic checks the creator and the member list.
func (m *MsgSetRollupMembers) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}
	if m.RollupID == "" {
		return fmt.Errorf("rollup_id cannot be empty")
	}
	return ValidateMembers(m.Members, false)
}
