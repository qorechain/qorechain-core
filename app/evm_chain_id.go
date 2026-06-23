package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/cast"

	"github.com/cosmos/cosmos-sdk/client/flags"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
)

// EVM (EIP-155) chain IDs per QoreChain network. Both were verified unregistered
// in the canonical ethereum-lists/chains registry. They MUST stay distinct so
// EIP-155 replay protection isolates the two networks.
const (
	EVMChainIDTestnet uint64 = 9800 // qorechain-diana (testnet)
	EVMChainIDMainnet uint64 = 9801 // qorechain-vladi (mainnet)

	// CosmosChainIDTestnet and CosmosChainIDMainnet are the Cosmos chain-ids the
	// EVM chain IDs map from.
	CosmosChainIDTestnet = "qorechain-diana"
	CosmosChainIDMainnet = "qorechain-vladi"

	// EnvEVMChainID overrides the resolved EVM chain ID for a deployment.
	EnvEVMChainID = "QORE_EVM_CHAIN_ID"
)

// resolveEVMChainID determines the EVM (EIP-155) chain ID so a single binary can
// serve either network. Resolution order:
//
//  1. the QORE_EVM_CHAIN_ID environment variable (explicit deploy override);
//  2. the Cosmos chain-id (from the --chain-id flag, else the node's genesis),
//     mapped to the per-network EVM chain ID;
//  3. the testnet default.
func resolveEVMChainID(appOpts servertypes.AppOptions) uint64 {
	if v := os.Getenv(EnvEVMChainID); v != "" {
		if n, err := strconv.ParseUint(v, 10, 64); err == nil && n != 0 {
			return n
		}
	}

	cosmosID := cast.ToString(appOpts.Get(flags.FlagChainID))
	if cosmosID == "" {
		cosmosID = genesisChainID(cast.ToString(appOpts.Get(flags.FlagHome)))
	}

	switch cosmosID {
	case CosmosChainIDMainnet:
		return EVMChainIDMainnet
	case CosmosChainIDTestnet:
		return EVMChainIDTestnet
	default:
		return EVMChainIDTestnet
	}
}

// genesisChainID best-effort reads the chain_id from <home>/config/genesis.json.
// It returns "" when the home dir is unknown or the genesis is not yet written
// (e.g. during `init`), in which case the caller falls back to the default.
func genesisChainID(home string) string {
	if home == "" {
		return ""
	}
	b, err := os.ReadFile(filepath.Join(home, "config", "genesis.json"))
	if err != nil {
		return ""
	}
	var g struct {
		ChainID string `json:"chain_id"`
	}
	if err := json.Unmarshal(b, &g); err != nil {
		return ""
	}
	return g.ChainID
}
