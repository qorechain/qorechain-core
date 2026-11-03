package types

import (
	"encoding/hex"
	"strings"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	svmtypes "github.com/qorechain/qorechain-core/x/svm/types"
)

// Address shapes a cross-VM call may carry (FeatureCrossVMAddressShapes).
//
// Every lane names its accounts differently: EVM accounts are 20 bytes, SVM
// accounts are 32 bytes and a native account has an SVM identity only when it
// is 20 bytes (its canonical image, the account followed by 12 zero bytes),
// CosmWasm contracts are 32-byte bech32 accounts. A call that names a sender
// or a target its lane cannot represent used to be accepted and to fail, or
// panic, only when it ran, which for a queued call is the end of the block.
// ValidateCallAddresses refuses it when it is submitted instead.

const (
	// NativeAccountLen is the length of a key-backed account (and of an EVM address).
	NativeAccountLen = 20
	// WideAccountLen is the length of a derived account: a CosmWasm contract,
	// a group policy, an ICA or module-derived account.
	WideAccountLen = 32
	// MaxEVMPrecompileAddress is the highest address treated as a precompile:
	// the Ethereum ones (0x01..0x11), the cosmos/evm standard ones (0x0800..)
	// and QoreChain's (0x0901, 0x0A01.., 0x0B01.., 0x0C01..).
	MaxEVMPrecompileAddress = 0xffff
)

// ValidateCallAddresses checks that the sender and the target of a cross-VM
// call into target can be represented on that lane. sender is the account the
// call acts for (already decoded from bech32).
func ValidateCallAddresses(sender sdk.AccAddress, target VMType, targetContract string) error {
	if n := len(sender); n != NativeAccountLen && n != WideAccountLen {
		return errorsmod.Wrapf(ErrInvalidMessage, "sender is %d bytes; a cross-VM caller must be a 20-byte account or a 32-byte derived account", n)
	}
	switch target {
	case VMTypeSVM:
		if len(sender) != NativeAccountLen {
			return errorsmod.Wrapf(ErrUnauthorized, "a %d-byte account has no SVM identity; only a 20-byte account can call into SVM", len(sender))
		}
		if _, err := svmtypes.Base58DecodeCanonical(targetContract); err != nil {
			return errorsmod.Wrapf(ErrInvalidTarget, "SVM target must be a canonical Base58 32-byte address: %s", err)
		}
	case VMTypeEVM:
		addr, err := parseStrictEVMAddress(targetContract)
		if err != nil {
			return err
		}
		if isEVMPrecompileAddress(addr) {
			return errorsmod.Wrapf(ErrInvalidTarget, "EVM target %s is a precompile address; precompiles are not cross-VM targets", targetContract)
		}
	case VMTypeCosmWasm:
		contract, err := sdk.AccAddressFromBech32(targetContract)
		if err != nil {
			return errorsmod.Wrapf(ErrInvalidTarget, "CosmWasm target must be a bech32 contract address: %s", err)
		}
		if len(contract) != WideAccountLen {
			return errorsmod.Wrapf(ErrInvalidTarget, "CosmWasm target is %d bytes; a contract address is 32 bytes", len(contract))
		}
	default:
		return errorsmod.Wrapf(ErrUnsupportedVM, "unknown target VM: %s", target)
	}
	return nil
}

// parseStrictEVMAddress accepts exactly 0x followed by 40 hex digits. The
// lenient parser the EVM call used (common.IsHexAddress) also takes the bare
// 40 digits, so one contract had two spellings in stored messages and events.
func parseStrictEVMAddress(s string) ([20]byte, error) {
	var out [20]byte
	if len(s) != 42 || !strings.HasPrefix(s, "0x") {
		return out, errorsmod.Wrapf(ErrInvalidTarget, "EVM target %q must be 0x followed by 40 hex digits", s)
	}
	bz, err := hex.DecodeString(s[2:])
	if err != nil {
		return out, errorsmod.Wrapf(ErrInvalidTarget, "EVM target %q is not hex", s)
	}
	copy(out[:], bz)
	return out, nil
}

func isEVMPrecompileAddress(a [20]byte) bool {
	for _, b := range a[:18] {
		if b != 0 {
			return false
		}
	}
	return uint16(a[18])<<8|uint16(a[19]) <= MaxEVMPrecompileAddress
}
