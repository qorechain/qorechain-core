package cmd

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"

	cmtcfg "github.com/cometbft/cometbft/config"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/debug"
	"github.com/cosmos/cosmos-sdk/client/keys"
	"github.com/cosmos/cosmos-sdk/client/pruning"
	"github.com/cosmos/cosmos-sdk/client/rpc"
	"github.com/cosmos/cosmos-sdk/client/snapshot"
	"github.com/cosmos/cosmos-sdk/server"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"

	cosmosevmserver "github.com/cosmos/evm/server"
	cosmosevmserverconfig "github.com/cosmos/evm/server/config"

	"github.com/qorechain/qorechain-core/app"
)

func initConsensusConfig() *cmtcfg.Config {
	cfg := cmtcfg.DefaultConfig()

	// QoreChain: increase max block size to accommodate PQC signatures
	// Dilithium-5 sigs are ~4.6KB vs ECDSA's 64B
	cfg.Consensus.TimeoutCommit = 5_000_000_000 // 5s

	return cfg
}

// QoreChainConfig extends the standard server config with EVM and JSON-RPC sections.
type QoreChainConfig struct {
	serverconfig.Config `mapstructure:",squash"`

	EVM     cosmosevmserverconfig.EVMConfig     `mapstructure:"evm"`
	JSONRPC cosmosevmserverconfig.JSONRPCConfig `mapstructure:"json-rpc"`
	TLS     cosmosevmserverconfig.TLSConfig     `mapstructure:"tls"`
}

func initAppConfig() (string, interface{}) {
	srvCfg := serverconfig.DefaultConfig()
	srvCfg.MinGasPrices = "0uqor" // base denom: uqor (10^6 = 1 QOR)

	evmCfg := cosmosevmserverconfig.DefaultEVMConfig()
	// Align the JSON-RPC backend's EIP-155 chain ID with the value the EVM keeper
	// resolves at runtime (app.resolveEVMChainID). The cosmos/evm default (262144)
	// otherwise leaks into the RPC backend (rpc/backend EvmChainID) and the
	// tx-conversion layer, so every eth_sendRawTransaction is rejected with
	// "incorrect chain-id; expected 262144". Default to the testnet EVM chain ID;
	// QORE_EVM_CHAIN_ID overrides it (e.g. 9801 for mainnet) — the same env var the
	// keeper honors, keeping both layers in agreement on a single binary.
	evmCfg.EVMChainID = app.EVMChainIDTestnet
	if v := os.Getenv(app.EnvEVMChainID); v != "" {
		if n, err := strconv.ParseUint(v, 10, 64); err == nil && n != 0 {
			evmCfg.EVMChainID = n
		}
	}
	jsonrpcCfg := cosmosevmserverconfig.DefaultJSONRPCConfig()
	// Enable JSON-RPC by default for QoreChain testnet
	jsonrpcCfg.Enable = true
	jsonrpcCfg.API = []string{"eth", "net", "web3", "txpool"}
	tlsCfg := cosmosevmserverconfig.DefaultTLSConfig()

	qoreConfig := QoreChainConfig{
		Config:  *srvCfg,
		EVM:     *evmCfg,
		JSONRPC: *jsonrpcCfg,
		TLS:     *tlsCfg,
	}

	// Combine the standard SDK template with the EVM config template
	customAppTemplate := serverconfig.DefaultConfigTemplate + cosmosevmserverconfig.DefaultEVMConfigTemplate

	return customAppTemplate, qoreConfig
}

// initCmdWithQoreChainDenoms wraps the standard `init` command so that the
// freshly written genesis is configured for the native uqor token: bank denom
// metadata for uqor and the EVM coin pointed at uqor/aqor. The stock InitCmd
// emits upstream default denoms, which leave x/vm InitGenesis unable to find
// denom metadata for the EVM coin, so the node cannot start.
func initCmdWithQoreChainDenoms(basicManager module.BasicManager) *cobra.Command {
	cmd := genutilcli.InitCmd(basicManager, app.DefaultNodeHome)
	stockRunE := cmd.RunE
	cmd.RunE = func(c *cobra.Command, args []string) error {
		if err := stockRunE(c, args); err != nil {
			return err
		}
		clientCtx := client.GetClientContextFromCmd(c)
		genFile := filepath.Join(clientCtx.HomeDir, "config", "genesis.json")
		return app.PatchGenesisFileDenoms(clientCtx.Codec, genFile)
	}
	return cmd
}

func initRootCmd(
	rootCmd *cobra.Command,
	txConfig client.TxConfig,
	basicManager module.BasicManager,
) {
	// Wrap newApp for SDK commands that expect servertypes.AppCreator
	sdkNewApp := func(l log.Logger, d dbm.DB, w io.Writer, o servertypes.AppOptions) servertypes.Application {
		return newApp(l, d, w, o)
	}

	rootCmd.AddCommand(
		initCmdWithQoreChainDenoms(basicManager),
		debug.Cmd(),
		pruning.Cmd(sdkNewApp, app.DefaultNodeHome),
		snapshot.Cmd(sdkNewApp),
	)

	// Use QoreChain EVM's custom server commands which include JSON-RPC support.
	// This replaces server.AddCommandsWithStartCmdOptions with an EVM-aware
	// start command that runs the JSON-RPC, WebSocket, and EVM indexer servers.
	//
	// The customizer wires the sidecar orchestrator into the start command's
	// PreRunE — no-op in public builds, real orchestrator + Docker client in
	// extended builds.
	cosmosevmserver.AddCommands(
		rootCmd,
		cosmosevmserver.NewDefaultStartOptions(newApp, app.DefaultNodeHome),
		appExport,
		WireSidecarHooks,
	)

	rootCmd.AddCommand(
		server.StatusCommand(),
		genesisCommand(txConfig, basicManager),
		queryCommand(basicManager),
		txCommand(basicManager),
		keys.Commands(),
	)
}

func genesisCommand(txConfig client.TxConfig, basicManager module.BasicManager, cmds ...*cobra.Command) *cobra.Command {
	cmd := genutilcli.Commands(txConfig, basicManager, app.DefaultNodeHome)
	for _, subCmd := range cmds {
		cmd.AddCommand(subCmd)
	}
	return cmd
}

func queryCommand(basicManager module.BasicManager) *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "query",
		Aliases:                    []string{"q"},
		Short:                      "Querying subcommands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		rpc.WaitTxCmd(),
		server.QueryBlockCmd(),
		authcmd.QueryTxsByEventsCmd(),
		server.QueryBlocksCmd(),
		authcmd.QueryTxCmd(),
		server.QueryBlockResultsCmd(),
	)

	// Wire the per-module query commands (GetQueryCmd). The custom QoreChain
	// modules are registered manually rather than via depinject, so autocli's
	// EnhanceRootCommand does not see them; AddQueryCommands picks up every
	// module in the basic manager that implements GetQueryCmd.
	basicManager.AddQueryCommands(cmd)

	return cmd
}

func txCommand(basicManager module.BasicManager) *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "tx",
		Short:                      "Transactions subcommands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		authcmd.GetSignCommand(),
		authcmd.GetSignBatchCommand(),
		authcmd.GetMultiSignCommand(),
		authcmd.GetMultiSignBatchCmd(),
		authcmd.GetValidateSignaturesCommand(),
		authcmd.GetBroadcastCommand(),
		authcmd.GetEncodeCommand(),
		authcmd.GetDecodeCommand(),
		authcmd.GetSimulateCmd(),
	)

	// Wire the per-module tx commands (GetTxCmd) for the manually-registered
	// custom modules, which autocli does not see.
	basicManager.AddTxCommands(cmd)

	return cmd
}

func newApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	appOpts servertypes.AppOptions,
) cosmosevmserver.Application {
	baseappOptions := server.DefaultBaseappOptions(appOpts)
	return app.NewQoreChainApp(
		logger, db, traceStore, true,
		appOpts,
		baseappOptions...,
	)
}

func appExport(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	height int64,
	forZeroHeight bool,
	jailAllowedAddrs []string,
	appOpts servertypes.AppOptions,
	modulesToExport []string,
) (servertypes.ExportedApp, error) {
	viperAppOpts, ok := appOpts.(*viper.Viper)
	if !ok {
		return servertypes.ExportedApp{}, errors.New("appOpts is not viper.Viper")
	}

	viperAppOpts.Set(server.FlagInvCheckPeriod, 1)
	appOpts = viperAppOpts

	var qoreApp *app.QoreChainApp
	if height != -1 {
		qoreApp = app.NewQoreChainApp(logger, db, traceStore, false, appOpts)

		if err := qoreApp.LoadHeight(height); err != nil {
			return servertypes.ExportedApp{}, err
		}
	} else {
		qoreApp = app.NewQoreChainApp(logger, db, traceStore, true, appOpts)
	}

	return qoreApp.ExportAppStateAndValidators(forZeroHeight, jailAllowedAddrs, modulesToExport)
}

// NewTestnetCmd creates a root testnet command.
func NewTestnetCmd(mbm module.BasicManager, genBalIterator banktypes.GenesisBalancesIterator) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "testnet",
		Short: "subcommands for starting or configuring local testnets",
	}
	return cmd
}
