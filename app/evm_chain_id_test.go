package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/client/flags"
)

type mockAppOpts map[string]interface{}

func (m mockAppOpts) Get(k string) interface{} { return m[k] }

// homeWithGenesis writes a minimal genesis naming chainID and returns the home.
func homeWithGenesis(t *testing.T, chainID string) string {
	t.Helper()
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	g := `{"chain_id":"` + chainID + `"}`
	if err := os.WriteFile(filepath.Join(home, "config", "genesis.json"), []byte(g), 0o600); err != nil {
		t.Fatal(err)
	}
	return home
}

// TestResolveEVMChainID pins QSR-2026-0151 over chain-id source (flag, genesis,
// none) x override (unset, matching, mismatching, garbage) x network (known,
// unknown).
func TestResolveEVMChainID(t *testing.T) {
	sources := map[string]func(t *testing.T, chainID string) mockAppOpts{
		"flag": func(_ *testing.T, id string) mockAppOpts { return mockAppOpts{flags.FlagChainID: id} },
		"genesis": func(t *testing.T, id string) mockAppOpts {
			return mockAppOpts{flags.FlagHome: homeWithGenesis(t, id)}
		},
	}
	cases := []struct {
		name    string
		chainID string
		env     string
		want    uint64
		wantErr bool
	}{
		{"mainnet", CosmosChainIDMainnet, "", EVMChainIDMainnet, false},
		{"testnet", CosmosChainIDTestnet, "", EVMChainIDTestnet, false},
		{"mainnet matching override", CosmosChainIDMainnet, "9801", EVMChainIDMainnet, false},
		{"testnet matching override", CosmosChainIDTestnet, "9800", EVMChainIDTestnet, false},
		{"mainnet with testnet id", CosmosChainIDMainnet, "9800", 0, true},
		{"testnet with mainnet id", CosmosChainIDTestnet, "9801", 0, true},
		{"mainnet arbitrary override", CosmosChainIDMainnet, "12345", 0, true},
		{"mainnet garbage override", CosmosChainIDMainnet, "not-a-number", 0, true},
		{"mainnet zero override", CosmosChainIDMainnet, "0", 0, true},
		{"devnet default", "some-devnet", "", EVMChainIDTestnet, false},
		{"devnet override", "some-devnet", "12345", 12345, false},
		{"devnet garbage override", "some-devnet", "x", 0, true},
	}
	for srcName, src := range sources {
		for _, c := range cases {
			t.Run(srcName+"/"+c.name, func(t *testing.T) {
				t.Setenv(EnvEVMChainID, c.env)
				got, cosmosID, err := resolveEVMChainID(src(t, c.chainID))
				if cosmosID != c.chainID {
					t.Fatalf("resolved chain-id %q, want %q", cosmosID, c.chainID)
				}
				if c.wantErr {
					if err == nil {
						t.Fatalf("got %d, want an error", got)
					}
					return
				}
				if err != nil || got != c.want {
					t.Fatalf("got %d, %v; want %d", got, err, c.want)
				}
			})
		}
	}

	t.Run("no chain-id never refuses", func(t *testing.T) {
		for env, want := range map[string]uint64{"": EVMChainIDTestnet, "9801": 9801, "junk": EVMChainIDTestnet} {
			t.Setenv(EnvEVMChainID, env)
			got, _, err := resolveEVMChainID(mockAppOpts{flags.FlagHome: t.TempDir()})
			if err != nil || got != want {
				t.Fatalf("env %q: got %d, %v; want %d", env, got, err, want)
			}
		}
	})

	t.Run("mainnet and testnet EVM ids are distinct", func(t *testing.T) {
		if EVMChainIDMainnet == EVMChainIDTestnet {
			t.Fatal("EVM chain IDs must differ between networks")
		}
	})
}

func TestCheckRPCEVMChainID(t *testing.T) {
	for _, c := range []struct {
		val     interface{}
		wantErr bool
	}{
		{nil, false},
		{uint64(9801), false},
		{"9801", false},
		{0, false},
		{uint64(9800), true},
		{int64(262144), true},
		{"junk", true},
	} {
		err := checkRPCEVMChainID(mockAppOpts{appOptEVMChainID: c.val}, EVMChainIDMainnet)
		if (err != nil) != c.wantErr {
			t.Errorf("app.toml %v: err = %v, wantErr %v", c.val, err, c.wantErr)
		}
	}
}

func TestMustResolveEVMChainIDRefuses(t *testing.T) {
	t.Setenv(EnvEVMChainID, "")
	home := homeWithGenesis(t, CosmosChainIDMainnet)
	if got := mustResolveEVMChainID(log.NewNopLogger(), mockAppOpts{flags.FlagHome: home, appOptEVMChainID: uint64(9801)}); got != EVMChainIDMainnet {
		t.Fatalf("got %d", got)
	}

	refuses := func(name string, opts mockAppOpts, env string) {
		t.Run(name, func(t *testing.T) {
			t.Setenv(EnvEVMChainID, env)
			defer func() {
				r := recover()
				if r == nil {
					t.Fatal("started with a mismatched EVM chain ID")
				}
				if msg, _ := r.(string); !strings.Contains(msg, "refusing to start") {
					t.Fatalf("unexpected panic %v", r)
				}
			}()
			mustResolveEVMChainID(log.NewNopLogger(), opts)
		})
	}
	refuses("env on mainnet", mockAppOpts{flags.FlagHome: home}, "9800")
	refuses("app.toml on mainnet", mockAppOpts{flags.FlagHome: home, appOptEVMChainID: uint64(9800)}, "")
}
