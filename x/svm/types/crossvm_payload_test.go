package types_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/qorechain/qorechain-core/x/svm/types"
)

func meta(b byte, signer, writable bool) types.AccountMeta {
	var a [32]byte
	for i := range a {
		a[i] = b
	}
	return types.AccountMeta{Address: a, IsSigner: signer, IsWritable: writable}
}

func TestCrossVMInstruction_RoundTrip(t *testing.T) {
	accounts := []types.AccountMeta{
		meta(0x01, true, true),
		meta(0x02, false, true),
		meta(0x03, false, false),
	}
	data := []byte{0xDE, 0xAD, 0xBE, 0xEF}

	gotAccounts, gotData, err := types.DecodeCrossVMInstruction(
		types.EncodeCrossVMInstruction(accounts, data))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(gotAccounts) != len(accounts) {
		t.Fatalf("got %d accounts, want %d", len(gotAccounts), len(accounts))
	}
	for i := range accounts {
		if gotAccounts[i] != accounts[i] {
			t.Fatalf("account %d: got %+v, want %+v", i, gotAccounts[i], accounts[i])
		}
	}
	if !bytes.Equal(gotData, data) {
		t.Fatalf("data: got %x, want %x", gotData, data)
	}
}

// An instruction with no accounts is legitimate, and an empty data field is too, so
// neither may be mistaken for a malformed payload.
func TestCrossVMInstruction_EmptyIsValid(t *testing.T) {
	accounts, data, err := types.DecodeCrossVMInstruction(
		types.EncodeCrossVMInstruction(nil, nil))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(accounts) != 0 || len(data) != 0 {
		t.Fatalf("got %d accounts and %d data bytes, want none of either",
			len(accounts), len(data))
	}
}

// A declared count must be paid for by bytes actually present, or a payload of a few
// bytes could ask the node to allocate for billions of accounts.
func TestCrossVMInstruction_DeclaredCountMustBeBackedByBytes(t *testing.T) {
	var payload []byte
	payload = binary.BigEndian.AppendUint64(payload, 1<<40)

	if _, _, err := types.DecodeCrossVMInstruction(payload); err == nil {
		t.Fatal("a count with no accounts behind it was accepted")
	}
}

// Exactly one interpretation per payload: trailing bytes, or a data length that does
// not match what follows, mean the caller and the callee could disagree about what was
// authorised while each believes it read the same instruction.
func TestCrossVMInstruction_AmbiguousFramingIsRefused(t *testing.T) {
	good := types.EncodeCrossVMInstruction([]types.AccountMeta{meta(0x01, false, true)},
		[]byte{0xAA, 0xBB})

	if _, _, err := types.DecodeCrossVMInstruction(append(good, 0x00)); err == nil {
		t.Fatal("trailing bytes were accepted")
	}
	if _, _, err := types.DecodeCrossVMInstruction(good[:len(good)-1]); err == nil {
		t.Fatal("a short data field was accepted")
	}

	// A data length that overstates what follows must not be read past the end.
	overstated := types.EncodeCrossVMInstruction(nil, []byte{0x01, 0x02})
	binary.BigEndian.PutUint64(overstated[8:16], 99)
	if _, _, err := types.DecodeCrossVMInstruction(overstated); err == nil {
		t.Fatal("an overstated data length was accepted")
	}
}

func TestCrossVMInstruction_ShortPayloadsAreRefused(t *testing.T) {
	for _, p := range [][]byte{nil, {0x00}, make([]byte, 7)} {
		if _, _, err := types.DecodeCrossVMInstruction(p); err == nil {
			t.Fatalf("payload of %d bytes was accepted", len(p))
		}
	}
}

// Unknown flag bits are refused rather than ignored. A caller setting a bit this
// version does not define is asking for something the runtime will not do, and
// silently dropping it would execute a different instruction than the one intended.
func TestCrossVMInstruction_UnknownFlagBitsAreRefused(t *testing.T) {
	payload := types.EncodeCrossVMInstruction([]types.AccountMeta{meta(0x01, true, true)}, nil)
	payload[8+32] = 0xFF

	if _, _, err := types.DecodeCrossVMInstruction(payload); err == nil {
		t.Fatal("an account with undefined flag bits was accepted")
	}
}
