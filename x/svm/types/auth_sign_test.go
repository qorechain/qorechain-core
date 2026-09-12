package types

import (
	"bytes"
	"crypto/sha256"
	"testing"
)

func mkAddr(b byte) [32]byte {
	var a [32]byte
	for i := range a {
		a[i] = b
	}
	return a
}

// The v1 encoding placed fixed-size account entries directly before `data`, with
// no count. So N accounts plus data D produced the same byte stream as N-1
// accounts plus data (addr‖flags‖D): one approval authorised a different account
// set, including different writable and signer flags. The count prefix must make
// these two distinct.
func TestSVMAuthSignBytes_AccountBoundaryNoLongerAmbiguous(t *testing.T) {
	prog := mkAddr(0xAA)
	a1, a2 := mkAddr(0x01), mkAddr(0x02)

	// Variant A: two accounts, short data.
	accountsA := []AccountMeta{
		{Address: a1, IsSigner: true, IsWritable: true},
		{Address: a2, IsSigner: false, IsWritable: true},
	}
	dataA := []byte{0xDE, 0xAD}

	// Variant B: one account, and the second account's bytes folded into data.
	// flags for a2 above = IsWritable only = 2.
	accountsB := []AccountMeta{
		{Address: a1, IsSigner: true, IsWritable: true},
	}
	dataB := append(append([]byte{}, a2[:]...), 0x02)
	dataB = append(dataB, dataA...)

	bh := bytes.Repeat([]byte{0x11}, 32)

	dA := SVMAuthSignBytes(prog, accountsA, dataA, bh)
	dB := SVMAuthSignBytes(prog, accountsB, dataB, bh)

	if bytes.Equal(dA, dB) {
		t.Fatal("account-list boundary is still ambiguous: one signature authorises " +
			"two different account sets")
	}
}

// `data` and `recentBlockhash` were adjacent and both variable length, so bytes
// could be shifted from the end of one into the front of the other.
func TestSVMAuthSignBytes_DataBlockhashBoundaryNoLongerAmbiguous(t *testing.T) {
	prog := mkAddr(0xAA)
	accounts := []AccountMeta{{Address: mkAddr(0x01), IsSigner: true, IsWritable: true}}

	// (data = A‖B, blockhash = C) versus (data = A, blockhash = B‖C)
	A := []byte{0x01, 0x02}
	B := []byte{0x03, 0x04}
	C := []byte{0x05, 0x06}

	d1 := SVMAuthSignBytes(prog, accounts, append(append([]byte{}, A...), B...), C)
	d2 := SVMAuthSignBytes(prog, accounts, A, append(append([]byte{}, B...), C...))

	if bytes.Equal(d1, d2) {
		t.Fatal("data/blockhash boundary is still ambiguous")
	}
}

// Flags must be covered: the same accounts with different writable/signer flags
// must not share a digest.
func TestSVMAuthSignBytes_FlagsAreCovered(t *testing.T) {
	prog := mkAddr(0xAA)
	a := mkAddr(0x01)
	bh := bytes.Repeat([]byte{0x11}, 32)

	ro := []AccountMeta{{Address: a, IsSigner: true, IsWritable: false}}
	rw := []AccountMeta{{Address: a, IsSigner: true, IsWritable: true}}

	if bytes.Equal(SVMAuthSignBytes(prog, ro, nil, bh), SVMAuthSignBytes(prog, rw, nil, bh)) {
		t.Fatal("writable flag does not change the digest")
	}
}

func TestSVMAuthSignBytes_Deterministic(t *testing.T) {
	prog := mkAddr(0xAA)
	accounts := []AccountMeta{{Address: mkAddr(0x01), IsSigner: true, IsWritable: true}}
	data := []byte{0x09, 0x08, 0x07}
	bh := bytes.Repeat([]byte{0x11}, 32)

	first := SVMAuthSignBytes(prog, accounts, data, bh)
	second := SVMAuthSignBytes(prog, accounts, data, bh)
	if !bytes.Equal(first, second) {
		t.Fatal("digest is not deterministic")
	}
	if len(first) != 32 {
		t.Fatalf("digest must be 32 bytes, got %d", len(first))
	}
}

// The domain tag must be part of the digest, so a v1 client cannot accidentally
// produce a digest a v2 node accepts.
func TestSVMAuthSignBytes_DomainTagIsV2(t *testing.T) {
	prog := mkAddr(0xAA)
	accounts := []AccountMeta{{Address: mkAddr(0x01), IsSigner: true, IsWritable: true}}
	bh := bytes.Repeat([]byte{0x11}, 32)

	got := SVMAuthSignBytes(prog, accounts, []byte{0x01}, bh)

	// Rebuild the v1 stream by hand and confirm it does NOT match.
	var v1 bytes.Buffer
	v1.WriteString("qorechain-svm-auth-v1")
	v1.Write(prog[:])
	v1.Write(accounts[0].Address[:])
	v1.WriteByte(3) // signer|writable
	v1.Write([]byte{0x01})
	v1.Write(bh)
	if bytes.Equal(got, sha256SumForTest(v1.Bytes())) {
		t.Fatal("v2 digest equals the v1 stream: the tag change did not take effect")
	}
}

func sha256SumForTest(p []byte) []byte {
	s := sha256.Sum256(p)
	return s[:]
}
