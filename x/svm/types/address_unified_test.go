package types

import (
	"bytes"
	"testing"
)

// TestUnifiedIdentityRoundTrip proves the SVM address is the SAME account as the
// Cosmos/EVM address for a given secp256k1 key: CosmosToSVMAddress right-pads to
// 32 bytes, SVMToCosmosAddress truncates back, and SVMToEVMAddress yields the
// identical 20 bytes. This is what lets one x/bank balance be shared across all
// three VMs.
func TestUnifiedIdentityRoundTrip(t *testing.T) {
	// A representative 20-byte native account (e.g. keccak(pubkey)[12:]).
	cosmos := []byte{
		0xde, 0xad, 0xbe, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab,
		0xcd, 0xef, 0xfe, 0xed, 0xfa, 0xce, 0x00, 0x11, 0x22, 0x33,
	}

	svm := CosmosToSVMAddress(cosmos)

	if got := SVMToCosmosAddress(svm); !bytes.Equal(got, cosmos) {
		t.Fatalf("SVMToCosmosAddress not the inverse: got %x want %x", got, cosmos)
	}

	if evm := SVMToEVMAddress(svm); !bytes.Equal(evm[:], cosmos) {
		t.Fatalf("EVM view differs from Cosmos account: got %x want %x", evm[:], cosmos)
	}

	// Canonical wallet form: high 20 bytes = account, low 12 bytes = zero.
	for i := 20; i < 32; i++ {
		if svm[i] != 0 {
			t.Fatalf("svm[%d] = %d, want 0 (canonical wallet padding)", i, svm[i])
		}
	}
}

// TestLamportUqorRatio pins the fixed conversion so SVM balances and x/bank
// balances denote the same value (1 uqor = 1000 lamports; 1 QOR = 1e9 lamports).
func TestLamportUqorRatio(t *testing.T) {
	if LamportsPerUqor != 1000 {
		t.Fatalf("LamportsPerUqor = %d, want 1000", LamportsPerUqor)
	}
	const oneQORinUqor = 1_000_000
	if got := oneQORinUqor * LamportsPerUqor; got != 1_000_000_000 {
		t.Fatalf("1 QOR = %d lamports, want 1_000_000_000", got)
	}
	if NativeDenom != "uqor" {
		t.Fatalf("NativeDenom = %q, want uqor", NativeDenom)
	}
}
