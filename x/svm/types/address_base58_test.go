package types

import (
	"fmt"
	"math/big"
	"math/rand"
	"strings"
	"testing"
	"time"
)

// base58DecodeV320 is a frozen copy of Base58Decode as it shipped in v3.2.0.
// The cross-VM SVM call handler stores this function's error text in consensus
// state, so the bounded decoder must match it exactly: same bytes, same error
// string, for every input.
func base58DecodeV320(s string) ([32]byte, error) {
	var result [32]byte

	if len(s) == 0 {
		return result, fmt.Errorf("empty base58 string")
	}

	n := new(big.Int)
	base := big.NewInt(58)

	for i := 0; i < len(s); i++ {
		val := base58LookupTable[s[i]]
		if val < 0 {
			return result, fmt.Errorf("invalid Base58 character: %c", s[i])
		}
		n.Mul(n, base)
		n.Add(n, big.NewInt(int64(val)))
	}

	leadingOnes := 0
	for i := 0; i < len(s); i++ {
		if s[i] != base58Alphabet[0] {
			break
		}
		leadingOnes++
	}

	numBytes := n.Bytes()
	totalLen := leadingOnes + len(numBytes)
	if totalLen > 32 {
		return result, fmt.Errorf("decoded address exceeds 32 bytes")
	}
	offset := 32 - len(numBytes)
	if offset < leadingOnes {
		return result, fmt.Errorf("decoded address exceeds 32 bytes")
	}
	copy(result[offset:], numBytes)
	return result, nil
}

func sameDecode(t *testing.T, s string) {
	t.Helper()
	gotAddr, gotErr := Base58Decode(s)
	wantAddr, wantErr := base58DecodeV320(s)
	if gotAddr != wantAddr {
		t.Fatalf("bytes differ for %q: got %x want %x", s, gotAddr, wantAddr)
	}
	switch {
	case gotErr == nil && wantErr == nil:
	case gotErr == nil || wantErr == nil:
		t.Fatalf("ok/err differ for %q: got %v want %v", s, gotErr, wantErr)
	case gotErr.Error() != wantErr.Error():
		t.Fatalf("error text differs for %q: got %q want %q", s, gotErr.Error(), wantErr.Error())
	}
}

// TestBase58DecodeMatchesV320 is the differential test the node-local rollout
// rests on: random and adversarial strings of length 0 to 300 decode to the
// same bytes and the same error text as v3.2.0.
func TestBase58DecodeMatchesV320(t *testing.T) {
	rng := rand.New(rand.NewSource(20260929))
	const invalid = "0OIl+/ \x00\xff-_."

	// Every length with every leading-'1' run, all valid characters.
	for length := 0; length <= 300; length++ {
		for ones := 0; ones <= length && ones <= 60; ones++ {
			var b strings.Builder
			b.WriteString(strings.Repeat("1", ones))
			for b.Len() < length {
				b.WriteByte(base58Alphabet[rng.Intn(len(base58Alphabet))])
			}
			sameDecode(t, b.String())
		}
	}

	// Invalid characters at varied positions, on both sides of the bound.
	for _, length := range []int{1, 2, 31, 32, 43, 44, 45, 46, 64, 100, 300} {
		for pos := 0; pos < length; pos += 1 + length/12 {
			for _, bad := range []byte(invalid) {
				buf := make([]byte, length)
				for i := range buf {
					buf[i] = base58Alphabet[rng.Intn(len(base58Alphabet))]
				}
				buf[pos] = bad
				sameDecode(t, string(buf))
				// A second bad character later: the first one must be named.
				if pos+1 < length {
					buf[length-1] = '0'
					sameDecode(t, string(buf))
				}
			}
		}
	}

	// Boundary shapes: exactly 44 and 45 characters of the largest digit,
	// all '1's, and the canonical encodings of edge values.
	fixed := []string{
		strings.Repeat("z", 43), strings.Repeat("z", 44), strings.Repeat("z", 45),
		strings.Repeat("1", 31), strings.Repeat("1", 32), strings.Repeat("1", 33),
		strings.Repeat("1", 44), strings.Repeat("1", 45),
		strings.Repeat("1", 32) + "2", strings.Repeat("1", 31) + "2", "2",
		"1" + strings.Repeat("z", 43), "11" + strings.Repeat("z", 42),
	}
	var max [32]byte
	for i := range max {
		max[i] = 0xff
	}
	fixed = append(fixed, Base58Encode(max), "1"+Base58Encode(max), Base58Encode(max)+"1")
	for _, s := range fixed {
		sameDecode(t, s)
	}

	// Round-trips of random 32-byte values, with and without extra leading '1's.
	for i := 0; i < 2000; i++ {
		var a [32]byte
		zeros := rng.Intn(33)
		for j := zeros; j < 32; j++ {
			a[j] = byte(rng.Intn(256))
		}
		enc := Base58Encode(a)
		sameDecode(t, enc)
		sameDecode(t, "1"+enc)
		sameDecode(t, strings.TrimLeft(enc, "1"))
	}
}

// TestBase58DecodeBoundedTime: a 1 MB input is refused after one linear pass.
// The v3.2.0 decoder spent quadratic big-integer work on it.
func TestBase58DecodeBoundedTime(t *testing.T) {
	huge := strings.Repeat("z", 1<<20)
	start := time.Now()
	_, err := Base58Decode(huge)
	elapsed := time.Since(start)
	if err == nil || err.Error() != "decoded address exceeds 32 bytes" {
		t.Fatalf("1 MB input: got %v", err)
	}
	if elapsed > 250*time.Millisecond {
		t.Fatalf("1 MB input took %s; the decoder is not bounded", elapsed)
	}
}

func BenchmarkBase58Decode1MB(b *testing.B) {
	huge := strings.Repeat("z", 1<<20)
	for i := 0; i < b.N; i++ {
		_, _ = Base58Decode(huge)
	}
}

func TestBase58DecodeCanonical(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	for i := 0; i < 2000; i++ {
		var a [32]byte
		for j := rng.Intn(33); j < 32; j++ {
			a[j] = byte(rng.Intn(256))
		}
		enc := Base58Encode(a)
		got, err := Base58DecodeCanonical(enc)
		if err != nil || got != a {
			t.Fatalf("canonical %q refused or wrong: %v %x", enc, err, got)
		}
	}

	var one [32]byte
	one[31] = 1
	canonicalOne := Base58Encode(one)
	if canonicalOne != strings.Repeat("1", 31)+"2" {
		t.Fatalf("unexpected canonical form of 1: %q", canonicalOne)
	}

	for _, s := range []string{
		"2",                           // short form of the same address
		strings.Repeat("1", 30) + "2", // too few leading '1's
		strings.Repeat("1", 33),       // one '1' too many for the zero address
		"1" + canonicalOne,            // overflow by a leading '1'
		strings.Repeat("z", 45),       // over the bound
		"",                            // empty
		"0" + canonicalOne[1:],        // bad character
	} {
		if _, err := Base58DecodeCanonical(s); err == nil {
			t.Fatalf("non-canonical %q accepted", s)
		}
	}

	// The error names the canonical form, so a caller can fix the input.
	_, err := Base58DecodeCanonical("2")
	if err == nil || !strings.Contains(err.Error(), canonicalOne) {
		t.Fatalf("error does not name the canonical form: %v", err)
	}
}

// The well-known program addresses are written as literals; every one of them
// must be the canonical spelling, or the canonical decoder would refuse the
// documented address.
func TestBuiltInProgramAddressesCanonical(t *testing.T) {
	for name, addr := range map[string][32]byte{
		"system": SystemProgramAddress, "spl-token": SPLTokenAddress, "ata": ATAAddress,
		"memo": MemoAddress, "qor-pqc": QorPQCAddress, "qor-ai": QorAIAddress,
	} {
		enc := Base58Encode(addr)
		if got, err := Base58DecodeCanonical(enc); err != nil || got != addr {
			t.Fatalf("%s: %q does not round-trip canonically: %v", name, enc, err)
		}
	}
	for _, lit := range []string{
		"11111111111111111111111111111111",
		"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA",
		"ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL",
		"MemoSq4gqABAXKb96qnH8TysNcWxMyWCqXgDLGmfcHr",
		"QorPQC1111111111111111111111111111111111111",
		"QorAi11111111111111111111111111111111111111",
	} {
		if _, err := Base58DecodeCanonical(lit); err != nil {
			t.Errorf("built-in literal %q is not canonical: %v", lit, err)
		}
	}
}
