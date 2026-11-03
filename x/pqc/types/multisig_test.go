package types

import (
	"bytes"
	errorsmod "cosmossdk.io/errors"
	"os"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func msKey(b byte) []byte {
	k := bytes.Repeat([]byte{b}, mldsa87PublicKeySize)
	return k
}

func msSig(b byte) []byte { return bytes.Repeat([]byte{b}, mldsa87SignatureSize) }

func msKeys(n int) [][]byte {
	out := make([][]byte, n)
	for i := range out {
		out[i] = msKey(byte(i + 1))
	}
	return out
}

func TestValidateMultisigSet(t *testing.T) {
	degenerate := make([]byte, mldsa87PublicKeySize)
	copy(degenerate, bytes.Repeat([]byte{7}, mldsa87RhoSize))
	cases := []struct {
		name string
		algo AlgorithmID
		k    uint32
		keys [][]byte
		want *errorsmod.Error
	}{
		{"2-of-3", AlgorithmDilithium5, 2, msKeys(3), nil},
		{"1-of-1", AlgorithmDilithium5, 1, msKeys(1), nil},
		{"16-of-16", AlgorithmDilithium5, 16, msKeys(16), nil},
		{"17 members", AlgorithmDilithium5, 2, msKeys(17), ErrInvalidMultisig},
		{"no members", AlgorithmDilithium5, 1, nil, ErrInvalidMultisig},
		{"threshold 0", AlgorithmDilithium5, 0, msKeys(3), ErrInvalidMultisig},
		{"threshold above n", AlgorithmDilithium5, 4, msKeys(3), ErrInvalidMultisig},
		{"duplicate key", AlgorithmDilithium5, 2, [][]byte{msKey(1), msKey(2), msKey(1)}, ErrInvalidMultisig},
		{"short key", AlgorithmDilithium5, 1, [][]byte{msKey(1)[:100]}, ErrInvalidMultisig},
		{"degenerate key", AlgorithmDilithium5, 1, [][]byte{degenerate}, ErrWeakPQCKey},
		{"KEM algorithm", AlgorithmMLKEM1024, 1, msKeys(1), ErrInvalidMultisig},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateMultisigSet(c.algo, c.k, c.keys)
			if c.want == nil {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if !c.want.Is(err) {
				t.Fatalf("got %v, want %v", err, c.want)
			}
		})
	}
}

func TestMsgRegisterPQCMultisigValidateBasic(t *testing.T) {
	addr := sdk.AccAddress(bytes.Repeat([]byte{1}, 20)).String()
	ok := &MsgRegisterPQCMultisig{Sender: addr, AlgorithmID: AlgorithmDilithium5, Threshold: 2,
		PublicKeys: msKeys(3), ProofsOfPossession: [][]byte{msSig(1), msSig(2), msSig(3)}}
	if err := ok.ValidateBasic(); err != nil {
		t.Fatal(err)
	}
	missing := *ok
	missing.ProofsOfPossession = missing.ProofsOfPossession[:2]
	if err := missing.ValidateBasic(); !ErrInvalidMultisig.Is(err) {
		t.Fatalf("a missing proof must be refused, got %v", err)
	}
	short := *ok
	short.ProofsOfPossession = [][]byte{msSig(1), msSig(2), msSig(3)[:10]}
	if err := short.ValidateBasic(); !ErrInvalidProofOfPossession.Is(err) {
		t.Fatalf("a short proof must be refused, got %v", err)
	}
	bad := *ok
	bad.Sender = "nope"
	if err := bad.ValidateBasic(); err == nil {
		t.Fatal("a bad sender must be refused")
	}
}

func TestPQCMultisigSignatureValidateStateless(t *testing.T) {
	good := PQCMultisigSignature{AlgorithmID: AlgorithmDilithium5, Bitmap: []byte{0b101},
		Signatures: [][]byte{msSig(1), msSig(2)}}
	if err := good.ValidateStateless(); err != nil {
		t.Fatal(err)
	}
	many := make([][]byte, MaxPQCMultisigMembers+1)
	for i := range many {
		many[i] = msSig(byte(i))
	}
	cases := map[string]PQCMultisigSignature{
		"wrong algorithm":    {AlgorithmID: AlgorithmMLKEM1024, Bitmap: []byte{1}, Signatures: [][]byte{msSig(1)}},
		"empty bitmap":       {AlgorithmID: AlgorithmDilithium5, Signatures: [][]byte{msSig(1)}},
		"bitmap too long":    {AlgorithmID: AlgorithmDilithium5, Bitmap: []byte{1, 0, 0}, Signatures: [][]byte{msSig(1)}},
		"no signatures":      {AlgorithmID: AlgorithmDilithium5, Bitmap: []byte{0}},
		"too many":           {AlgorithmID: AlgorithmDilithium5, Bitmap: []byte{0xff, 0xff}, Signatures: many},
		"count mismatch":     {AlgorithmID: AlgorithmDilithium5, Bitmap: []byte{0b111}, Signatures: [][]byte{msSig(1), msSig(2)}},
		"short signature":    {AlgorithmID: AlgorithmDilithium5, Bitmap: []byte{1}, Signatures: [][]byte{msSig(1)[:4626]}},
		"long signature":     {AlgorithmID: AlgorithmDilithium5, Bitmap: []byte{1}, Signatures: [][]byte{append(msSig(1), 0)}},
		"repeated signature": {AlgorithmID: AlgorithmDilithium5, Bitmap: []byte{0b11}, Signatures: [][]byte{msSig(1), msSig(1)}},
	}
	for name, s := range cases {
		if err := s.ValidateStateless(); !ErrMalformedMultisigSig.Is(err) {
			t.Errorf("%s: got %v, want ErrMalformedMultisigSig", name, err)
		}
	}
}

func TestPQCMultisigSignatureSigners(t *testing.T) {
	s := PQCMultisigSignature{Bitmap: []byte{0b101}}
	got, err := s.Signers(3)
	if err != nil || len(got) != 2 || got[0] != 0 || got[1] != 2 {
		t.Fatalf("got %v %v", got, err)
	}
	// A bit at or beyond n has no member.
	if _, err := (PQCMultisigSignature{Bitmap: []byte{0b1001}}).Signers(3); !ErrMalformedMultisigSig.Is(err) {
		t.Fatalf("bit 3 of a 3-member set must be refused, got %v", err)
	}
	// One set of signers has one encoding: a trailing zero byte is refused.
	if _, err := (PQCMultisigSignature{Bitmap: []byte{0b101, 0}}).Signers(3); !ErrMalformedMultisigSig.Is(err) {
		t.Fatalf("a padded bitmap must be refused, got %v", err)
	}
	// Members 8..15 live in the second byte.
	got, err = (PQCMultisigSignature{Bitmap: []byte{0, 0b10}}).Signers(10)
	if err != nil || len(got) != 1 || got[0] != 9 {
		t.Fatalf("got %v %v", got, err)
	}
	bm, err := MultisigBitmap(10, []int{9, 0})
	if err != nil || !bytes.Equal(bm, []byte{1, 0b10}) {
		t.Fatalf("got %x %v", bm, err)
	}
	if _, err := MultisigBitmap(3, []int{0, 0}); err == nil {
		t.Fatal("a member given twice must be refused")
	}
	if _, err := MultisigBitmap(3, []int{3}); err == nil {
		t.Fatal("a member outside the set must be refused")
	}
}

// Every field the sign-bytes bind changes them, and length prefixes keep
// field boundaries from sliding.
func TestMultisigSignBytesBinding(t *testing.T) {
	addr := bytes.Repeat([]byte{9}, 20)
	base := MultisigSignBytes("qorechain-diana", 7, 3, addr, []byte("body"), []byte("auth"))
	variants := map[string][]byte{
		"chain":    MultisigSignBytes("qorechain-vladi", 7, 3, addr, []byte("body"), []byte("auth")),
		"account":  MultisigSignBytes("qorechain-diana", 8, 3, addr, []byte("body"), []byte("auth")),
		"sequence": MultisigSignBytes("qorechain-diana", 7, 4, addr, []byte("body"), []byte("auth")),
		"address":  MultisigSignBytes("qorechain-diana", 7, 3, bytes.Repeat([]byte{8}, 20), []byte("body"), []byte("auth")),
		"body":     MultisigSignBytes("qorechain-diana", 7, 3, addr, []byte("bodx"), []byte("auth")),
		"auth":     MultisigSignBytes("qorechain-diana", 7, 3, addr, []byte("body"), []byte("autx")),
		"boundary": MultisigSignBytes("qorechain-diana", 7, 3, addr, []byte("bodya"), []byte("uth")),
	}
	for name, v := range variants {
		if bytes.Equal(base, v) {
			t.Errorf("changing the %s does not change the sign-bytes", name)
		}
	}
	if !bytes.HasPrefix(base, []byte(MultisigSignBytesDomain)) {
		t.Fatal("sign-bytes must start with the domain tag")
	}
	// Not the single-signer form for the same body and auth info.
	if bytes.Equal(base, HybridSignBytes("qorechain-diana", []byte("body"), []byte("auth"))) {
		t.Fatal("multisig and single-signer sign-bytes must differ")
	}
}

func TestMultisigMemberPossessionDomain(t *testing.T) {
	pub := msKey(1)
	acct := sdk.AccAddress(bytes.Repeat([]byte{1}, 20)).String()
	a := MultisigMemberPossessionSignBytes("c", 1, acct, pub)
	if bytes.Equal(a, ProofOfPossessionSignBytes("c", 1, acct, pub)) {
		t.Fatal("a member proof must not be a single-key proof")
	}
	if bytes.Equal(a, MultisigMemberPossessionSignBytes("d", 1, acct, pub)) {
		t.Fatal("the proof must name the chain")
	}
	other := sdk.AccAddress(bytes.Repeat([]byte{2}, 20)).String()
	if bytes.Equal(a, MultisigMemberPossessionSignBytes("c", 1, other, pub)) {
		t.Fatal("the proof must name the account")
	}
}

func TestPQCMultisigFeatureReleased(t *testing.T) {
	found := 0
	for _, f := range ReleaseV3_2_7ProtocolFeatures {
		if f == FeaturePQCMultisig {
			found++
		}
	}
	if found != 1 {
		t.Fatalf("%s listed %d times in the v3.2.7 release set", FeaturePQCMultisig, found)
	}
}

// The multisig query's REST route is in its generated gateway (the same check
// TestEveryProtoRouteIsInTheGateway makes for query.proto).
func TestMultisigRouteIsInTheGateway(t *testing.T) {
	gw, err := os.ReadFile("multisig.pb.gw.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(gw, []byte(`"qorechain", "pqc", "v1", "multisig", "address"`)) {
		t.Fatal("the multisig route is not in multisig.pb.gw.go")
	}
	if !bytes.Contains(gw, []byte("RegisterMultisigQueryHandlerClient")) {
		t.Fatal("no gateway registration for the multisig query service")
	}
}
