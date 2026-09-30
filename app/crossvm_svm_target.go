package app

import (
	"fmt"

	svmtypes "github.com/qorechain/qorechain-core/x/svm/types"
)

// decodeCrossVMProgramAddress decodes the SVM program a cross-VM call targets.
//
// The error is stored with the cross-VM message, so its text is consensus
// state: the wrapping below is the v3.2.0 text, and Base58Decode keeps its own
// messages byte for byte. From v3.2.2 (FeatureSVMBase58Canonical) the target
// must also be the canonical spelling of the address, so one program is named
// by one string in stored messages and events (QSR-2026-0262). Below the plan
// height the lenient decode runs, as it did when those blocks were made.
func decodeCrossVMProgramAddress(target string, canonical bool) ([32]byte, error) {
	decode := svmtypes.Base58Decode
	if canonical {
		decode = svmtypes.Base58DecodeCanonical
	}
	addr, err := decode(target)
	if err != nil {
		return addr, fmt.Errorf("invalid SVM program address: %w", err)
	}
	return addr, nil
}
