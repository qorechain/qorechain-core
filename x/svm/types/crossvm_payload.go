package types

import (
	"encoding/binary"
	"fmt"
)

// The payload a contract on another VM sends to call an SVM program.
//
// WHY A FORMAT IS NEEDED AT ALL. An SVM instruction is not just bytes: it is bytes
// PLUS the list of accounts it may touch, each with its signer and writable flags.
// The runtime's whole safety model rests on that list, because a program may only
// read or modify accounts that were passed to it. The cross-VM handler had no way to
// express one, so it called programs with no accounts and no signers at all, which
// left the SVM lane reachable in form and useless in substance: a program handed zero
// accounts can do nothing.
//
//	BE64(num_accounts) ‖ num_accounts × [ address(32) ‖ flags(1) ] ‖
//	BE64(len(data)) ‖ data
//
// where flags = (isSigner?1) | (isWritable?2), the same flag byte SVMAuthSignBytes
// uses, so one encoding of an account meta exists in this codebase rather than two.
//
// Fixed-width big-endian framing, rather than JSON, because both kinds of caller have
// to be able to build it: this is abi.encodePacked in Solidity and a few pushes in
// Rust, while JSON is painful from Solidity and would make the EVM lane second-class.
// The BE64 widths match the other length-prefixed payloads here
// (x/abstractaccount/types/cosmos_sign.go, evm_sign.go, SVMAuthSignBytes); one
// convention across every framed payload is worth more than the bytes a narrower
// count would save, because the next person writing one copies what they find.
//
// WHAT THIS FORMAT DOES NOT DECIDE. Whether the caller may claim a signer flag is not
// a question about encoding, and so is not answered here. The decoder returns the
// flags as written; the handler that executes the call is what establishes that only
// the caller's own address can be a signer. Keeping that separate means the rule
// lives with the identity it protects rather than in a parser.

// EncodeCrossVMInstruction builds the payload for a cross-VM call into an SVM program.
func EncodeCrossVMInstruction(accounts []AccountMeta, data []byte) []byte {
	out := make([]byte, 0, 8+len(accounts)*33+8+len(data))
	out = binary.BigEndian.AppendUint64(out, uint64(len(accounts)))
	for _, m := range accounts {
		out = append(out, m.Address[:]...)
		var flags byte
		if m.IsSigner {
			flags |= 1
		}
		if m.IsWritable {
			flags |= 2
		}
		out = append(out, flags)
	}
	out = binary.BigEndian.AppendUint64(out, uint64(len(data)))
	return append(out, data...)
}

// DecodeCrossVMInstruction parses a cross-VM SVM payload.
//
// Every length is checked against what remains, and trailing bytes are refused: a
// payload that decodes successfully has exactly one interpretation, so a caller and a
// callee cannot disagree about what was authorised while both believe they parsed the
// same thing.
func DecodeCrossVMInstruction(payload []byte) ([]AccountMeta, []byte, error) {
	if len(payload) < 8 {
		return nil, nil, fmt.Errorf("payload too short for the account count")
	}
	n := binary.BigEndian.Uint64(payload[:8])
	rest := payload[8:]

	// Bound the count against the bytes actually present before allocating, so a
	// declared length cannot ask for memory the payload does not pay for.
	if n > uint64(len(rest)/33) {
		return nil, nil, fmt.Errorf(
			"payload declares %d accounts but carries at most %d", n, len(rest)/33)
	}

	accounts := make([]AccountMeta, n)
	for i := uint64(0); i < n; i++ {
		entry := rest[i*33 : i*33+33]
		copy(accounts[i].Address[:], entry[:32])
		flags := entry[32]
		accounts[i].IsSigner = flags&1 != 0
		accounts[i].IsWritable = flags&2 != 0
		if flags&^byte(3) != 0 {
			return nil, nil, fmt.Errorf("account %d has unknown flag bits set: %08b", i, flags)
		}
	}
	rest = rest[n*33:]

	if len(rest) < 8 {
		return nil, nil, fmt.Errorf("payload too short for the data length")
	}
	dataLen := binary.BigEndian.Uint64(rest[:8])
	rest = rest[8:]
	if dataLen != uint64(len(rest)) {
		return nil, nil, fmt.Errorf(
			"payload declares %d data bytes but carries %d", dataLen, len(rest))
	}

	return accounts, rest, nil
}
