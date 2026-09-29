package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/cast"

	"cosmossdk.io/log"

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

// appOptEVMChainID is the app.toml key the JSON-RPC backend reads its EIP-155
// chain ID from. It must agree with the keeper's value.
const appOptEVMChainID = "evm.evm-chain-id"

// knownEVMChainIDs pins the EVM chain ID of each public network. A node of one
// of these networks may not run with any other value: signatures made for the
// other network, or for an arbitrary chain ID, would then verify on it.
var knownEVMChainIDs = map[string]uint64{
	CosmosChainIDMainnet: EVMChainIDMainnet,
	CosmosChainIDTestnet: EVMChainIDTestnet,
}

// PinnedEVMChainID returns the EVM chain ID pinned for a public network's
// Cosmos chain-id, and whether the chain-id is one of them.
func PinnedEVMChainID(cosmosID string) (uint64, bool) {
	id, ok := knownEVMChainIDs[cosmosID]
	return id, ok
}

// resolveEVMChainID determines the EVM (EIP-155) chain ID so a single binary can
// serve either network, and returns the Cosmos chain-id it resolved it for.
//
// The Cosmos chain-id comes from the --chain-id flag, else the node's genesis.
// For a known network the EVM chain ID is the pinned value, and a
// QORE_EVM_CHAIN_ID that disagrees with it (or is not a number) is an error.
// For any other chain-id (a devnet) QORE_EVM_CHAIN_ID is used when set, else
// the testnet value. With no chain-id at all (a command that builds the app
// without a node home) the override is used when it parses, else the testnet
// value, and nothing is refused.
func resolveEVMChainID(appOpts servertypes.AppOptions) (uint64, string, error) {
	cosmosID := cast.ToString(appOpts.Get(flags.FlagChainID))
	if cosmosID == "" {
		cosmosID = genesisChainID(cast.ToString(appOpts.Get(flags.FlagHome)))
	}
	pinned, known := knownEVMChainIDs[cosmosID]

	if v := os.Getenv(EnvEVMChainID); v != "" {
		n, err := strconv.ParseUint(v, 10, 64)
		switch {
		case err != nil || n == 0:
			if cosmosID != "" {
				return 0, cosmosID, fmt.Errorf("%s=%q is not a valid EVM chain ID", EnvEVMChainID, v)
			}
		case known && n != pinned:
			return 0, cosmosID, fmt.Errorf("%s=%d does not match chain %s, whose EVM chain ID is %d; unset it or set it to %d",
				EnvEVMChainID, n, cosmosID, pinned, pinned)
		default:
			return n, cosmosID, nil
		}
	}
	if known {
		return pinned, cosmosID, nil
	}
	return EVMChainIDTestnet, cosmosID, nil
}

// checkRPCEVMChainID refuses an app.toml [evm] evm-chain-id that differs from
// the keeper's EVM chain ID. The JSON-RPC backend reads that value on its own,
// so a mismatch lets the RPC layer and the keeper disagree about which
// signatures are valid. A missing or zero entry is left to the EVM server's
// own default handling.
func checkRPCEVMChainID(appOpts servertypes.AppOptions, keeperID uint64) error {
	v := appOpts.Get(appOptEVMChainID)
	if v == nil {
		return nil
	}
	n, err := cast.ToUint64E(v)
	if err != nil {
		return fmt.Errorf("app.toml %s = %v is not a valid chain ID", appOptEVMChainID, v)
	}
	if n != 0 && n != keeperID {
		return fmt.Errorf("app.toml %s = %d but this node's EVM chain ID is %d; set it to %d", appOptEVMChainID, n, keeperID, keeperID)
	}
	return nil
}

// mustResolveEVMChainID resolves and cross-checks the EVM chain ID at app
// construction and stops the process on a configuration that would make this
// node accept signatures meant for another chain. It returns the EVM chain ID
// and the Cosmos chain-id it was resolved for. It does not log: commands such
// as export build the app too and write their result to stdout, where the log
// also goes; LogStartupIdentity reports the mapping when the node starts.
func mustResolveEVMChainID(appOpts servertypes.AppOptions) (uint64, string) {
	id, cosmosID, err := resolveEVMChainID(appOpts)
	if err == nil && cosmosID != "" {
		err = checkRPCEVMChainID(appOpts, id)
	}
	if err != nil {
		panic(fmt.Sprintf("refusing to start: %v", err))
	}
	return id, cosmosID
}

// LogStartupIdentity logs, once at node start, the Cosmos chain-id to EVM
// chain ID mapping this node runs with and the PQC library it loaded.
func (app *QoreChainApp) LogStartupIdentity(logger log.Logger) {
	_, known := knownEVMChainIDs[app.evmCosmosChainID]
	switch {
	case app.evmCosmosChainID == "":
	case known:
		logger.Info("EVM chain ID", "chain_id", app.evmCosmosChainID, "evm_chain_id", app.evmChainID)
	case os.Getenv(EnvEVMChainID) == "":
		logger.Warn("unknown chain-id, using the testnet EVM chain ID; set "+EnvEVMChainID+" to choose another",
			"chain_id", app.evmCosmosChainID, "evm_chain_id", app.evmChainID)
	default:
		logger.Info("EVM chain ID from "+EnvEVMChainID, "chain_id", app.evmCosmosChainID, "evm_chain_id", app.evmChainID)
	}
	if PQCLibraryInfo != nil {
		path, version := PQCLibraryInfo()
		logger.Info("libqorepqc self-test passed", "library", path, "version", version)
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
