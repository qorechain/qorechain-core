package types

import (
	"bytes"
	"encoding/hex"
	"testing"
)

// Golden vector for the v2 anchor sign bytes. SDK signer helpers must produce
// exactly these bytes for the same inputs.
func TestAnchorSignBytesV2Golden(t *testing.T) {
	got := AnchorSignBytesV2("qorechain-diana", "my-rollup", 7,
		[]byte{0xaa, 0xbb}, nil, 3, nil)
	want := "" +
		hex.EncodeToString([]byte("qorechain-multilayer-anchor-v2")) +
		"000000000000000f" + hex.EncodeToString([]byte("qorechain-diana")) +
		"0000000000000009" + hex.EncodeToString([]byte("my-rollup")) +
		"0000000000000007" +
		"0000000000000002" + "aabb" +
		"0000000000000000" +
		"0000000000000003" +
		"0000000000000000"
	if hex.EncodeToString(got) != want {
		t.Fatalf("v2 sign bytes changed:\n got %x\nwant %s", got, want)
	}
}

func TestAnchorSignBytesVersions(t *testing.T) {
	msg := &MsgAnchorState{LayerID: "layer", LayerHeight: 1, StateRoot: []byte("root"),
		ValidatorSetHash: []byte("vsh"), TransactionCount: 9, CompressedStateProof: []byte("p")}

	// v1 is byte for byte the pre-v3.2.2 layout.
	v1 := append(append(append([]byte("layer"), 0, 0, 0, 0, 0, 0, 0, 1), "root"...), "vsh"...)
	if !bytes.Equal(AnchorSignBytesFor("c", msg, false), v1) {
		t.Fatal("v1 layout changed")
	}

	// v1 is ambiguous across field boundaries; v2 is not.
	a := AnchorSignBytesV1("ab", 1, []byte("cd"), []byte("ef"))
	b := AnchorSignBytesV1("ab", 1, []byte("cde"), []byte("f"))
	if !bytes.Equal(a, b) {
		t.Fatal("expected the v1 ambiguity this fixes")
	}
	if bytes.Equal(AnchorSignBytesV2("c", "ab", 1, []byte("cd"), []byte("ef"), 0, nil),
		AnchorSignBytesV2("c", "ab", 1, []byte("cde"), []byte("f"), 0, nil)) {
		t.Fatal("v2 is ambiguous across field boundaries")
	}

	// v2 binds the chain, the tx count and the compressed proof.
	base := AnchorSignBytesFor("chain-a", msg, true)
	if bytes.Equal(base, AnchorSignBytesFor("chain-b", msg, true)) {
		t.Fatal("v2 does not bind the chain id")
	}
	m2 := *msg
	m2.TransactionCount = 10
	if bytes.Equal(base, AnchorSignBytesFor("chain-a", &m2, true)) {
		t.Fatal("v2 does not bind the transaction count")
	}
	m3 := *msg
	m3.CompressedStateProof = []byte("q")
	if bytes.Equal(base, AnchorSignBytesFor("chain-a", &m3, true)) {
		t.Fatal("v2 does not bind the compressed state proof")
	}
}
