package types

import (
	"crypto/sha256"
	"errors"
)

// Cluster identity for Solana Wallet Standard wallets.
//
// A Solana cluster is identified by its genesis hash: getGenesisHash returns
// it as base58 of 32 bytes, and CAIP-2 / Wallet Standard name the chain
// "solana:<first 32 characters of that base58 string>". QoreChain has no
// Solana genesis block, so the SVM genesis hash is derived from the chain id:
//
//	genesis_hash = SHA-256(ClusterGenesisHashDomain || chain_id)
//
// The chain id is fixed for the life of a network and identical on every node,
// so the value is the same on every node, never changes, and needs no stored
// state: it is pure query code and not a consensus value. The domain prefix
// keeps it from colliding with any other hash of the chain id.
//
// The derivation and the domain string are frozen. Changing either changes the
// identity every wallet saved for the network (see the pinned values in
// cluster_test.go).

// ClusterGenesisHashDomain is the domain-separation prefix of the derivation.
const ClusterGenesisHashDomain = "qorechain/svm/genesis-hash/v1\x00"

// ClusterGenesisHashDerivation describes the derivation for query answers.
const ClusterGenesisHashDerivation = "base58(SHA-256(\"qorechain/svm/genesis-hash/v1\" || 0x00 || chain_id))"

// CAIP2Namespace is the CAIP-2 namespace Solana wallets use.
const CAIP2Namespace = "solana"

// caip2ReferenceLen is the length of a Solana CAIP-2 reference: the first 32
// characters of the base58 genesis hash.
const caip2ReferenceLen = 32

var errEmptyChainID = errors.New("svm cluster: chain id is empty")

// ClusterGenesisHash returns the 32-byte SVM genesis hash for a chain id.
func ClusterGenesisHash(chainID string) ([32]byte, error) {
	if chainID == "" {
		return [32]byte{}, errEmptyChainID
	}
	return sha256.Sum256([]byte(ClusterGenesisHashDomain + chainID)), nil
}

// ClusterGenesisHashBase58 returns the genesis hash as getGenesisHash reports it.
func ClusterGenesisHashBase58(chainID string) (string, error) {
	h, err := ClusterGenesisHash(chainID)
	if err != nil {
		return "", err
	}
	return Base58Encode(h), nil
}

// ClusterCAIP2ChainID returns the Wallet Standard / CAIP-2 chain string,
// "solana:<first 32 chars of the base58 genesis hash>".
func ClusterCAIP2ChainID(chainID string) (string, error) {
	s, err := ClusterGenesisHashBase58(chainID)
	if err != nil {
		return "", err
	}
	if len(s) < caip2ReferenceLen {
		// Base58 of 32 bytes is at least 32 characters unless the hash starts
		// with many zero bytes; SHA-256 output for a real chain id never does,
		// but refuse rather than emit a malformed reference.
		return "", errors.New("svm cluster: genesis hash too short for a CAIP-2 reference")
	}
	return CAIP2Namespace + ":" + s[:caip2ReferenceLen], nil
}

// SlotsPerEpoch is the SVM epoch length: Solana's default. The chain has no
// epoch of its own for SVM, so the Clock sysvar and the RPC getEpochInfo
// derive the epoch from the slot with it.
const SlotsPerEpoch uint64 = 432_000
