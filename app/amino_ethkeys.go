package app

import (
	"sync"

	"github.com/cosmos/evm/crypto/ethsecp256k1"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
)

var (
	ethKeysAminoMu   sync.Mutex
	ethKeysAminoDone = map[*codec.LegacyAmino]bool{}
)

// RegisterEthKeysOnLegacyAmino teaches the SDK's global legacy amino codec
// (legacy.Cdc), and any codec passed in, the eth_secp256k1 key types
// (QSR-2026-0491).
//
// The app registered those types on the proto InterfaceRegistry only. The
// SDK's ConsumeTxSizeGasDecorator, in simulate, amino-encodes a StdSignature
// carrying the signer's stored pubkey with legacy.Cdc, so gas simulation for
// any eth_secp256k1 account that had already signed panicked with "Cannot
// encode unregistered concrete type", and keys export of such a key did too.
//
// This only adds concrete types to the codec; no existing encoding changes.
// In block execution legacy.Cdc is reached on no path (the decorator uses it
// only when simulate is set), so it is node-local. It is idempotent and safe
// to call from both the app constructor and the CLI root. It must run before
// the codec first meets one of these types: go-amino caches an unregistered
// type on first use and then refuses to register it.
func RegisterEthKeysOnLegacyAmino(cdcs ...*codec.LegacyAmino) {
	ethKeysAminoMu.Lock()
	defer ethKeysAminoMu.Unlock()
	for _, cdc := range append([]*codec.LegacyAmino{legacy.Cdc}, cdcs...) {
		if cdc == nil || ethKeysAminoDone[cdc] {
			continue
		}
		registerConcreteOnce(cdc, &ethsecp256k1.PubKey{}, ethsecp256k1.PubKeyName)
		registerConcreteOnce(cdc, &ethsecp256k1.PrivKey{}, ethsecp256k1.PrivKeyName)
		ethKeysAminoDone[cdc] = true
	}
}

// registerConcreteOnce registers o under name unless cdc already has it (for
// instance from cosmos/evm's RegisterCrypto); go-amino panics on a second
// registration, before changing anything.
func registerConcreteOnce(cdc *codec.LegacyAmino, o any, name string) {
	defer func() { _ = recover() }()
	cdc.RegisterConcrete(o, name, nil)
}
