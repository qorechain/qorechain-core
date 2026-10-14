package types

import (
	"errors"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestValidateWithdrawal(t *testing.T) {
	addr := sdk.AccAddress([]byte("rdk-withdrawal-valid")).String()
	sib := make([]byte, 32)
	good := func() *MsgExecuteWithdrawal {
		return &MsgExecuteWithdrawal{
			Submitter: addr, RollupID: "roll-1", Recipient: addr,
			Denom: RollupEscrowDenom, Amount: 1, Proof: [][]byte{sib, sib},
		}
	}
	if err := good().ValidateWithdrawal(); err != nil {
		t.Fatalf("well-formed withdrawal refused: %v", err)
	}
	deep := make([][]byte, MaxWithdrawalProofDepth+1)
	for i := range deep {
		deep[i] = sib
	}
	cases := map[string]func(m *MsgExecuteWithdrawal){
		"bad submitter":  func(m *MsgExecuteWithdrawal) { m.Submitter = "nope" },
		"empty rollup":   func(m *MsgExecuteWithdrawal) { m.RollupID = "" },
		"bad recipient":  func(m *MsgExecuteWithdrawal) { m.Recipient = "nope" },
		"other denom":    func(m *MsgExecuteWithdrawal) { m.Denom = "uatom" },
		"zero amount":    func(m *MsgExecuteWithdrawal) { m.Amount = 0 },
		"negative":       func(m *MsgExecuteWithdrawal) { m.Amount = -5 },
		"short sibling":  func(m *MsgExecuteWithdrawal) { m.Proof = [][]byte{{1, 2, 3}} },
		"too deep proof": func(m *MsgExecuteWithdrawal) { m.Proof = deep },
	}
	for name, mut := range cases {
		m := good()
		mut(m)
		if err := m.ValidateWithdrawal(); !errors.Is(err, ErrInvalidWithdrawal) {
			t.Errorf("%s: want ErrInvalidWithdrawal, got %v", name, err)
		}
	}
	var nilMsg *MsgExecuteWithdrawal
	if err := nilMsg.ValidateWithdrawal(); !errors.Is(err, ErrInvalidWithdrawal) {
		t.Errorf("nil message: got %v", err)
	}
}

// The escrow-shortfall error is distinct from the proof error, so a client can
// tell a bad proof from an exhausted rollup.
func TestInsufficientRollupEscrowIsDistinct(t *testing.T) {
	if errors.Is(ErrInsufficientRollupEscrow, ErrInvalidWithdrawalProof) {
		t.Fatal("ErrInsufficientRollupEscrow must not match ErrInvalidWithdrawalProof")
	}
	if ErrInsufficientRollupEscrow.ABCICode() != 32 || ErrInvalidWithdrawal.ABCICode() != 33 {
		t.Fatal("rdk error codes moved")
	}
}
