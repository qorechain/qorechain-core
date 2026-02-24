package types

import (
	"crypto/sha256"
	"fmt"
	"math/big"
)

// base58Alphabet is the Bitcoin Base58 alphabet used by SVM-compatible chains.
const base58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

// System program address constants (well-known SVM addresses)
var (
	SystemProgramAddress = mustBase58Decode("11111111111111111111111111111111")
	SPLTokenAddress      = mustBase58Decode("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")
	ATAAddress           = mustBase58Decode("ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL")
	MemoAddress          = mustBase58Decode("MemoSq4gqABAXKb96qnH8TysNcWxMyWCqXgDLGmfcHr")
	QorPQCAddress        = mustBase58Decode("QorPQC1111111111111111111111111111111111111")
	QorAIAddress         = mustBase58Decode("QorAi11111111111111111111111111111111111111")
)

func mustBase58Decode(s string) [32]byte {
	addr, err := Base58Decode(s)
	if err != nil {
		panic("invalid base58 address: " + s)
	}
	return addr
}

// SVMToCosmosAddress derives a 20-byte native address from a 32-byte SVM address.
func SVMToCosmosAddress(svmAddr [32]byte) []byte {
	hash := sha256.Sum256(svmAddr[:])
	return hash[:20]
}

// EVMToSVMAddress derives a 32-byte SVM address from a 20-byte EVM address.
func EVMToSVMAddress(evmAddr [20]byte) [32]byte {
	data := make([]byte, 0, 20+len("qorechain-svm"))
	data = append(data, evmAddr[:]...)
	data = append(data, []byte("qorechain-svm")...)
	return sha256.Sum256(data)
}

// SVMToEVMAddress truncates a 32-byte SVM address to its first 20 bytes.
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

// Base58Decode decodes a Base58 string to a [32]byte address using the Bitcoin alphabet.
func Base58Decode(s string) ([32]byte, error) {
	var result [32]byte

	if len(s) == 0 {
		return result, fmt.Errorf("empty base58 string")
	}

	// Build reverse lookup table
	alphabetMap := make(map[byte]int64)
	for i := 0; i < len(base58Alphabet); i++ {
		alphabetMap[base58Alphabet[i]] = int64(i)
	}

	n := new(big.Int)
	base := big.NewInt(58)

	for i := 0; i < len(s); i++ {
		val, ok := alphabetMap[s[i]]
		if !ok {
			return result, fmt.Errorf("invalid base58 character: %c", s[i])
		}
		n.Mul(n, base)
		n.Add(n, big.NewInt(val))
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
