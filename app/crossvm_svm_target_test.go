package app

import (
	"strings"
	"testing"

	svmtypes "github.com/qorechain/qorechain-core/x/svm/types"
)

func TestDecodeCrossVMProgramAddress(t *testing.T) {
	var one [32]byte
	one[31] = 1
	canonical := svmtypes.Base58Encode(one)

	// Feature off: the v3.2.0 behaviour. A short form is accepted and names
	// the same program as the canonical string.
	got, err := decodeCrossVMProgramAddress("2", false)
	if err != nil || got != one {
		t.Fatalf("feature off: short form refused or wrong: %v %x", err, got)
	}

	// Feature on: only the canonical spelling is accepted.
	if _, err := decodeCrossVMProgramAddress("2", true); err == nil ||
		!strings.HasPrefix(err.Error(), "invalid SVM program address: non-canonical Base58 address") {
		t.Fatalf("feature on: short form not refused as non-canonical: %v", err)
	}
	if got, err := decodeCrossVMProgramAddress(canonical, true); err != nil || got != one {
		t.Fatalf("feature on: canonical form refused: %v", err)
	}

	// A 100 KB target costs one linear pass and fails with the exact text
	// v3.2.0 stored for it, with the feature off and on.
	huge := strings.Repeat("z", 100*1024)
	for _, on := range []bool{false, true} {
		_, err := decodeCrossVMProgramAddress(huge, on)
		if err == nil || err.Error() != "invalid SVM program address: decoded address exceeds 32 bytes" {
			t.Fatalf("canonical=%v: 100 KB target: got %v", on, err)
		}
		bad := huge + "0"
		_, err = decodeCrossVMProgramAddress(bad, on)
		if err == nil || err.Error() != "invalid SVM program address: invalid Base58 character: 0" {
			t.Fatalf("canonical=%v: bad character text: got %v", on, err)
		}
	}
}
