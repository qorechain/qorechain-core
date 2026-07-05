package types

import (
	"bytes"
	"testing"
)

func baseArgs() (string, string, []byte, string, string, uint64) {
	return "qorechain-diana",
		"qor1account00000000000000000000000000000000",
		[]byte{0x01, 0x02, 0x03, 0x04},
		"qor1recipient000000000000000000000000000000",
		"100uqor",
		7
}

// TestCosmosAuthSignBytes_deterministic: identical inputs -> identical output
// (the relayer/QoreX must be able to reproduce exactly what the chain hashes).
func TestCosmosAuthSignBytes_deterministic(t *testing.T) {
	c, a, p, to, amt, n := baseArgs()
	if !bytes.Equal(CosmosAuthSignBytes(c, a, p, to, amt, n), CosmosAuthSignBytes(c, a, p, to, amt, n)) {
		t.Fatal("CosmosAuthSignBytes is not deterministic")
	}
	if len(CosmosAuthSignBytes(c, a, p, to, amt, n)) != 32 {
		t.Fatal("expected a 32-byte sha256 digest")
	}
}

// TestCosmosAuthSignBytes_domainSeparation: an EVM-lane signature must never be
// valid on the Native lane. The two helpers take different arg shapes, but with
// value-strings that could coincide the digests must still differ because of the
// distinct domain prefixes.
func TestCosmosAuthSignBytes_domainSeparation(t *testing.T) {
	c, a, p, to, amt, n := baseArgs()
	cosmos := CosmosAuthSignBytes(c, a, p, to, amt, n)
	evm := EVMAuthSignBytes(c, a, p, to, amt, nil, n)
	if bytes.Equal(cosmos, evm) {
		t.Fatal("Cosmos and EVM auth sign-bytes collide — domain separation broken")
	}
}

// TestCosmosAuthSignBytes_everyFieldBinds: changing ANY bound field must change
// the digest, so a signature cannot be replayed across chains, accounts, keys,
// recipients, amounts or sequences.
func TestCosmosAuthSignBytes_everyFieldBinds(t *testing.T) {
	c, a, p, to, amt, n := baseArgs()
	ref := CosmosAuthSignBytes(c, a, p, to, amt, n)

	mutations := map[string][]byte{
		"chainID":   CosmosAuthSignBytes("qorechain-vladi", a, p, to, amt, n),
		"account":   CosmosAuthSignBytes(c, "qor1other0000000000000000000000000000000000", p, to, amt, n),
		"pubkey":    CosmosAuthSignBytes(c, a, []byte{0x01, 0x02, 0x03, 0x05}, to, amt, n),
		"recipient": CosmosAuthSignBytes(c, a, p, "qor1elsewhere00000000000000000000000000000", amt, n),
		"amount":    CosmosAuthSignBytes(c, a, p, to, "101uqor", n),
		"nonce":     CosmosAuthSignBytes(c, a, p, to, amt, n+1),
	}
	for field, got := range mutations {
		if bytes.Equal(ref, got) {
			t.Fatalf("changing %q did not change the sign-bytes — replay binding broken", field)
		}
	}
}

// TestCosmosAuthSignBytes_lengthPrefixNoAmbiguity: length-prefixing must prevent
// two different field splits from producing the same concatenation (e.g. moving
// a character between account and recipient).
func TestCosmosAuthSignBytes_lengthPrefixNoAmbiguity(t *testing.T) {
	p := []byte{0xaa}
	x := CosmosAuthSignBytes("c", "ab", p, "cd", "1uqor", 0)
	y := CosmosAuthSignBytes("c", "a", p, "bcd", "1uqor", 0)
	if bytes.Equal(x, y) {
		t.Fatal("length-prefix framing is ambiguous across field boundaries")
	}
}

// TestAuthNonceKey_distinctPerAccountAndKey: the sequence store must not alias
// across accounts or authenticator keys.
func TestAuthNonceKey_distinctPerAccountAndKey(t *testing.T) {
	k1 := AuthNonceKey("qor1a", []byte{0x01})
	k2 := AuthNonceKey("qor1b", []byte{0x01})
	k3 := AuthNonceKey("qor1a", []byte{0x02})
	if bytes.Equal(k1, k2) || bytes.Equal(k1, k3) || bytes.Equal(k2, k3) {
		t.Fatal("AuthNonceKey collides across (account, pubkey)")
	}
	if !bytes.HasPrefix(k1, AuthNoncePrefix) {
		t.Fatal("AuthNonceKey missing the aa/authnonce/ prefix")
	}
}
