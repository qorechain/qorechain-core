package app

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/client/flags"
)

type mockAppOpts map[string]interface{}

func (m mockAppOpts) Get(k string) interface{} { return m[k] }

func TestResolveEVMChainID(t *testing.T) {
	t.Run("env override wins", func(t *testing.T) {
		t.Setenv(EnvEVMChainID, "12345")
		opts := mockAppOpts{flags.FlagChainID: CosmosChainIDMainnet}
		if got := resolveEVMChainID(opts); got != 12345 {
			t.Fatalf("env override: got %d, want 12345", got)
		}
	})

	t.Run("mainnet chain-id maps to 9801", func(t *testing.T) {
		t.Setenv(EnvEVMChainID, "")
		opts := mockAppOpts{flags.FlagChainID: CosmosChainIDMainnet}
		if got := resolveEVMChainID(opts); got != EVMChainIDMainnet {
			t.Fatalf("mainnet: got %d, want %d", got, EVMChainIDMainnet)
		}
	})

	t.Run("testnet chain-id maps to 9800", func(t *testing.T) {
		t.Setenv(EnvEVMChainID, "")
		opts := mockAppOpts{flags.FlagChainID: CosmosChainIDTestnet}
		if got := resolveEVMChainID(opts); got != EVMChainIDTestnet {
			t.Fatalf("testnet: got %d, want %d", got, EVMChainIDTestnet)
		}
	})

	t.Run("unknown chain-id falls back to testnet default", func(t *testing.T) {
		t.Setenv(EnvEVMChainID, "")
		opts := mockAppOpts{flags.FlagChainID: "some-other-chain"}
		if got := resolveEVMChainID(opts); got != EVMChainIDTestnet {
			t.Fatalf("default: got %d, want %d", got, EVMChainIDTestnet)
		}
	})

	t.Run("invalid env value ignored", func(t *testing.T) {
		t.Setenv(EnvEVMChainID, "not-a-number")
		opts := mockAppOpts{flags.FlagChainID: CosmosChainIDMainnet}
		if got := resolveEVMChainID(opts); got != EVMChainIDMainnet {
			t.Fatalf("invalid env: got %d, want %d", got, EVMChainIDMainnet)
		}
	})

	t.Run("mainnet and testnet EVM ids are distinct", func(t *testing.T) {
		if EVMChainIDMainnet == EVMChainIDTestnet {
			t.Fatal("EVM chain IDs must differ between networks")
		}
	})
}
