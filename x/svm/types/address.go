package types

import (
	"fmt"
	"math/big"

	"github.com/qorechain/qorechain-core/qorehash"
)

// base58Alphabet is the Bitcoin Base58 alphabet used by SVM-compatible chains.
const base58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

// base58LookupTable maps ASCII byte values to Base58 digit values (-1 = invalid).
// Initialized via function call so it is ready before other var initializers
// (e.g. mustBase58Decode calls in the system address block below).
var base58LookupTable = buildBase58Lookup()

func buildBase58Lookup() [256]int8 {
	var table [256]int8
	for i := range table {
		table[i] = -1
	}
	for i := 0; i < len(base58Alphabet); i++ {
		table[base58Alphabet[i]] = int8(i)
	}
	return table
}

// System program address constants (well-known SVM addresses)
var (
	SystemProgramAddress = mustBase58Decode("11111111111111111111111111111111")
	SPLTokenAddress      = mustBase58Decode("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")
	ATAAddress           = mustBase58Decode("ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL")
	MemoAddress          = mustBase58Decode("MemoSq4gqABAXKb96qnH8TysNcWxMyWCqXgDLGmfcHr")
	QorPQCAddress        = mustBase58Decode("QorPQC1111111111111111111111111111111111111")
	QorAIAddress         = mustBase58Decode("QorAi11111111111111111111111111111111111111")
	// BPFLoaderAddress owns the bytecode account of every deployed program
	// (v3.2.8, svm-bytecode-loader-owned), as Solana's upgradeable loader owns
	// program data. No deployed program can have this id, so no program owns
	// its own bytecode.
	BPFLoaderAddress = mustBase58Decode("BPFLoaderUpgradeab1e11111111111111111111111")
)

func mustBase58Decode(s string) [32]byte {
	addr, err := Base58Decode(s)
	if err != nil {
		panic("invalid base58 address: " + s)
	}
	return addr
}

// SVMToCosmosAddress returns the first 20 bytes of a 32-byte SVM address.
//
// LOSSY. It is the inverse of CosmosToSVMAddress only for a canonical address
// (bytes 20..32 zero). For any other address, such as a real Solana pubkey,
// 2^96 distinct SVM addresses map to the same 20 bytes, so the result names an
// unrelated native account. It is kept, unchanged, because blocks below the
// v3.2.5 and v3.2.7 plan heights were made with it. New code that needs the
// native account of an SVM address must call CanonicalSVMToCosmos, which
// refuses a non-canonical address instead of truncating it.
//
// UNIFIED IDENTITY: for a canonical address this makes the SVM address the SAME
// account as the Cosmos/EVM address for a given key, so a wallet holds ONE
// x/bank balance visible across all three VMs.
func SVMToCosmosAddress(svmAddr [32]byte) []byte {
	out := make([]byte, 20)
	copy(out, svmAddr[:20])
	return out
}

// CanonicalSVMToCosmos returns the 20-byte native account of a canonical SVM
// address, and an error for any other address. It never truncates: an address
// with a non-zero tail is not the image of a native account, and treating its
// prefix as one would let it read, receive into or spend that account.
func CanonicalSVMToCosmos(svmAddr [32]byte) ([]byte, error) {
	if !IsCanonicalWalletAddress(svmAddr) {
		return nil, fmt.Errorf("SVM address %s is not the canonical image of a native account", Base58Encode(svmAddr))
	}
	return SVMToCosmosAddress(svmAddr), nil
}

// CanonicalCosmosToSVM returns the canonical SVM address of a native account,
// and an error when the account is not 20 bytes. A 32-byte native address (a
// CosmWasm contract or an ICA host account) has no SVM image: copying its first
// 20 bytes would name a different native account.
func CanonicalCosmosToSVM(nativeAddr []byte) ([32]byte, error) {
	if len(nativeAddr) != 20 {
		return [32]byte{}, fmt.Errorf("native address of %d bytes has no SVM address; only 20-byte accounts do", len(nativeAddr))
	}
	return CosmosToSVMAddress(nativeAddr), nil
}

// CosmosToSVMAddress derives the canonical 32-byte SVM address for a 20-byte
// native account by right-padding with zeros. It is the inverse of
// SVMToCosmosAddress, so a wallet's SVM address maps back to the same account
// and shares its single x/bank balance. Programs and PDAs use their own
// 32-byte addresses and are not wallet accounts.
//
// LOSSY for an input longer than 20 bytes: only the first 20 are kept. New
// code must call CanonicalCosmosToSVM, which refuses such an input.
func CosmosToSVMAddress(cosmosAddr []byte) [32]byte {
	var out [32]byte
	copy(out[:20], cosmosAddr)
	return out
}

// IsCanonicalWalletAddress reports whether a 32-byte SVM address is the image of
// a 20-byte native address under CosmosToSVMAddress, i.e. whether bytes 20..32
// are all zero.
//
// WHY THIS MATTERS: SVMToCosmosAddress maps an SVM address to a native account by
// TRUNCATING to the first 20 bytes. For a canonical address that is lossless and
// the two are the same account. For any other 32-byte address - a real Solana
// pubkey, for instance, which carries full entropy in all 32 bytes - the
// truncation is lossy: 2^96 distinct SVM addresses collapse onto one native
// account. Treating such an address as a native-backed wallet would mean two
// unrelated holders sharing a single x/bank balance, and value credited to an
// address whose holder cannot spend it while whoever controls that 20-byte native
// account can. Only canonical addresses may be backed by x/bank; everything else
// is a program, a PDA or a store-only account.
func IsCanonicalWalletAddress(svmAddr [32]byte) bool {
	for _, b := range svmAddr[20:] {
		if b != 0 {
			return false
		}
	}
	return true
}

// EVMToSVMAddress derives a deterministic 32-byte SVM address from a 20-byte EVM address.
// Uses SHAKE-256(evmAddr || "qorechain-svm") for domain separation. NOTE: this is a one-way
// derivation; SVMToEVMAddress is NOT its inverse.
func EVMToSVMAddress(evmAddr [20]byte) [32]byte {
	data := make([]byte, 0, 20+len("qorechain-svm"))
	data = append(data, evmAddr[:]...)
	data = append(data, []byte("qorechain-svm")...)
	return qorehash.Sum256(data)
}

// SVMToEVMAddress returns the first 20 bytes of a 32-byte SVM address.
//
// LOSSY, and unused by the chain: like SVMToCosmosAddress it names an unrelated
// account for a non-canonical address. It is NOT the inverse of EVMToSVMAddress,
// which hashes. New code must call CanonicalSVMToCosmos and convert the 20
// bytes, so a non-canonical address is refused rather than truncated.
//
// Deprecated: use CanonicalSVMToCosmos.
func SVMToEVMAddress(svmAddr [32]byte) [20]byte {
	var evmAddr [20]byte
	copy(evmAddr[:], svmAddr[:20])
	return evmAddr
}

// Base58Encode encodes a [32]byte address to a Base58 string using the Bitcoin alphabet.
func Base58Encode(addr [32]byte) string {
	// Count leading zero bytes
	leadingZeros := 0
	for _, b := range addr {
		if b != 0 {
			break
		}
		leadingZeros++
	}

	// Convert to big integer
	n := new(big.Int).SetBytes(addr[:])
	zero := big.NewInt(0)
	base := big.NewInt(58)
	mod := new(big.Int)

	var encoded []byte
	for n.Cmp(zero) > 0 {
		n.DivMod(n, base, mod)
		encoded = append(encoded, base58Alphabet[mod.Int64()])
	}

	// Add leading '1' characters for each leading zero byte
	for i := 0; i < leadingZeros; i++ {
		encoded = append(encoded, base58Alphabet[0])
	}

	// Reverse the result
	for i, j := 0, len(encoded)-1; i < j; i, j = i+1, j-1 {
		encoded[i], encoded[j] = encoded[j], encoded[i]
	}

	return string(encoded)
}

// base58MaxAddressLen is the longest Base58 string that can decode to 32
// bytes. 58^44 > 2^256, and each extra leading '1' adds a whole zero byte while
// the remaining number loses fewer than 8 bits, so every valid-alphabet string
// longer than this already failed the 32-byte bound below. Checking the length
// first only skips the work (QSR-2026-0053 finding 5).
const base58MaxAddressLen = 44

// Base58Decode decodes a Base58 string to a [32]byte address using the Bitcoin alphabet.
//
// It is reachable from consensus (the cross-VM SVM call target), where its
// error text is stored, so the order of checks and every message are exactly
// those of v3.2.0: an empty string first, then the first character outside the
// alphabet in scan order, then the 32-byte bound. Only the cost changed: the
// alphabet check and the length bound run before any big-integer work, so an
// attacker-sized input costs one linear pass instead of quadratic arithmetic.
// TestBase58DecodeMatchesV320 holds it to the old function byte for byte.
func Base58Decode(s string) ([32]byte, error) {
	var result [32]byte

	if len(s) == 0 {
		return result, fmt.Errorf("empty base58 string")
	}

	for i := 0; i < len(s); i++ {
		if base58LookupTable[s[i]] < 0 {
			return result, fmt.Errorf("invalid Base58 character: %c", s[i])
		}
	}
	if len(s) > base58MaxAddressLen {
		return result, fmt.Errorf("decoded address exceeds 32 bytes")
	}

	n := new(big.Int)
	base := big.NewInt(58)

	for i := 0; i < len(s); i++ {
		n.Mul(n, base)
		n.Add(n, big.NewInt(int64(base58LookupTable[s[i]])))
	}

	// Count leading '1' characters (represent leading zero bytes)
	leadingOnes := 0
	for i := 0; i < len(s); i++ {
		if s[i] != base58Alphabet[0] {
			break
		}
		leadingOnes++
	}

	// Convert big.Int to bytes
	numBytes := n.Bytes()

	// Total length = leading zeros + numeric bytes
	totalLen := leadingOnes + len(numBytes)
	if totalLen > 32 {
		return result, fmt.Errorf("decoded address exceeds 32 bytes")
	}

	// Fill result: leading zeros are already zero in the array
	// Place the numeric bytes right-aligned in the 32-byte array
	offset := 32 - len(numBytes)
	if offset < leadingOnes {
		return result, fmt.Errorf("decoded address exceeds 32 bytes")
	}
	copy(result[offset:], numBytes)

	return result, nil
}

// Base58DecodeCanonical decodes s like Base58Decode and then requires s to be
// the one encoding of the result, the string Base58Encode (and every standard
// wallet) produces for those 32 bytes. Base58Decode right-aligns whatever
// number a string encodes, so "2" and thirty-one '1's followed by "2" name the
// same address; a string-keyed allowlist, an indexer or an event filter would
// treat them as two (QSR-2026-0262). The error names the canonical form.
func Base58DecodeCanonical(s string) ([32]byte, error) {
	addr, err := Base58Decode(s)
	if err != nil {
		return addr, err
	}
	if canonical := Base58Encode(addr); canonical != s {
		return [32]byte{}, fmt.Errorf("non-canonical Base58 address %q: the canonical form is %q", s, canonical)
	}
	return addr, nil
}
