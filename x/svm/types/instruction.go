package types

import errorsmod "cosmossdk.io/errors"

// AccountMeta describes the role of an account in an SVM instruction.
type AccountMeta struct {
	Address    [32]byte `json:"address"`
	IsSigner   bool     `json:"is_signer"`
	IsWritable bool     `json:"is_writable"`
}

// Instruction represents a single SVM instruction to be executed by a program.
type Instruction struct {
	ProgramID [32]byte      `json:"program_id"`
	Accounts  []AccountMeta `json:"accounts"`
	Data      []byte        `json:"data"`
}

// Validate checks the instruction for basic correctness.
func (ix *Instruction) Validate() error {
	var zeroAddr [32]byte
	if ix.ProgramID == zeroAddr {
		return errorsmod.Wrap(ErrInvalidInstruction, "program ID cannot be zero")
	}
	return nil
}
