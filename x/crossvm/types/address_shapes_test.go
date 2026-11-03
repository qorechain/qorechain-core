package types

import (
	"bytes"
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	svmtypes "github.com/qorechain/qorechain-core/x/svm/types"
)

func TestValidateCallAddresses(t *testing.T) {
	narrow := sdk.AccAddress(bytes.Repeat([]byte{0x11}, 20))
	wide := sdk.AccAddress(bytes.Repeat([]byte{0x22}, 32))
	contract := sdk.AccAddress(bytes.Repeat([]byte{0x33}, 32)).String()
	program := svmtypes.Base58Encode(svmtypes.SystemProgramAddress)
	evmTarget := "0x1234567890abcdef1234567890ABCDEF12345678"

	cases := []struct {
		name     string
		sender   sdk.AccAddress
		target   VMType
		contract string
		want     string // substring of the error; empty = accepted
	}{
		{"evm from 20-byte", narrow, VMTypeEVM, evmTarget, ""},
		{"evm from 32-byte contract", wide, VMTypeEVM, evmTarget, ""},
		{"cosmwasm from 20-byte", narrow, VMTypeCosmWasm, contract, ""},
		{"cosmwasm from 32-byte", wide, VMTypeCosmWasm, contract, ""},
		{"svm from 20-byte", narrow, VMTypeSVM, program, ""},

		{"svm from 32-byte (F1)", wide, VMTypeSVM, program, "no SVM identity"},
		{"sender 0 bytes", sdk.AccAddress{}, VMTypeEVM, evmTarget, "sender is 0 bytes"},
		{"sender 19 bytes", sdk.AccAddress(bytes.Repeat([]byte{1}, 19)), VMTypeCosmWasm, contract, "sender is 19 bytes"},
		{"sender 21 bytes", sdk.AccAddress(bytes.Repeat([]byte{1}, 21)), VMTypeEVM, evmTarget, "sender is 21 bytes"},
		{"sender 33 bytes", sdk.AccAddress(bytes.Repeat([]byte{1}, 33)), VMTypeSVM, program, "sender is 33 bytes"},
		{"sender 64 bytes", sdk.AccAddress(bytes.Repeat([]byte{1}, 64)), VMTypeCosmWasm, contract, "sender is 64 bytes"},

		{"evm target without 0x", narrow, VMTypeEVM, evmTarget[2:], "must be 0x"},
		{"evm target 19 bytes", narrow, VMTypeEVM, evmTarget[:40], "must be 0x"},
		{"evm target 21 bytes", narrow, VMTypeEVM, evmTarget + "00", "must be 0x"},
		{"evm target 0X", narrow, VMTypeEVM, "0X" + evmTarget[2:], "must be 0x"},
		{"evm target not hex", narrow, VMTypeEVM, "0x" + strings.Repeat("zz", 20), "not hex"},
		{"evm target bech32", narrow, VMTypeEVM, contract, "must be 0x"},
		{"evm target zero address", narrow, VMTypeEVM, "0x" + strings.Repeat("0", 40), "precompile"},
		{"evm target bridge 0x0901", narrow, VMTypeEVM, "0x0000000000000000000000000000000000000901", "precompile"},
		{"evm target staking 0x0800", narrow, VMTypeEVM, "0x0000000000000000000000000000000000000800", "precompile"},
		{"evm target 0xffff", narrow, VMTypeEVM, "0x000000000000000000000000000000000000ffff", "precompile"},
		{"evm target 0x010000 is not a precompile", narrow, VMTypeEVM, "0x0000000000000000000000000000000000010000", ""},

		{"cosmwasm target 20 bytes", narrow, VMTypeCosmWasm, narrow.String(), "32 bytes"},
		{"cosmwasm target 0x", narrow, VMTypeCosmWasm, evmTarget, "bech32"},
		{"cosmwasm target empty", narrow, VMTypeCosmWasm, "", "bech32"},

		{"svm target non-canonical", narrow, VMTypeSVM, "2", "canonical"},
		{"svm target hex", narrow, VMTypeSVM, evmTarget, "Base58"},
		{"svm target empty", narrow, VMTypeSVM, "", "Base58"},

		{"unknown target vm", narrow, VMType("move"), evmTarget, "unknown target VM"},
		{"native is not a target", narrow, VMTypeNative, evmTarget, "unknown target VM"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCallAddresses(tc.sender, tc.target, tc.contract)
			switch {
			case tc.want == "" && err != nil:
				t.Fatalf("refused: %v", err)
			case tc.want != "" && err == nil:
				t.Fatalf("accepted, want an error containing %q", tc.want)
			case tc.want != "" && !strings.Contains(err.Error(), tc.want):
				t.Fatalf("error %q does not contain %q", err, tc.want)
			}
		})
	}
}

// Never panics, whatever it is given.
func FuzzValidateCallAddresses(f *testing.F) {
	f.Add([]byte{1, 2, 3}, "evm", "0x0000000000000000000000000000000000000901")
	f.Add(bytes.Repeat([]byte{7}, 32), "svm", "11111111111111111111111111111111")
	f.Add(bytes.Repeat([]byte{7}, 20), "cosmwasm", "")
	f.Fuzz(func(t *testing.T, sender []byte, target, contract string) {
		_ = ValidateCallAddresses(sender, VMType(target), contract)
	})
}
