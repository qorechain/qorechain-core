package cmd

import (
	"os"

	"github.com/spf13/cobra"

	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	"cosmossdk.io/client/v2/autocli"
	"cosmossdk.io/depinject"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/config"
	nodeservice "github.com/cosmos/cosmos-sdk/client/grpc/node"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtxconfig "github.com/cosmos/cosmos-sdk/x/auth/tx/config"
	"github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/qorechain/qorechain-core/app"
	aimodule "github.com/qorechain/qorechain-core/x/ai"
	bridgemodule "github.com/qorechain/qorechain-core/x/bridge"
	pqcmodule "github.com/qorechain/qorechain-core/x/pqc"
	qcamodule "github.com/qorechain/qorechain-core/x/qca"
	reputationmodule "github.com/qorechain/qorechain-core/x/reputation"
)

func init() {
	// Set bech32 prefixes before any depinject usage so that
	// module authority addresses are derived with the correct prefix.
	cfg := sdk.GetConfig()
	cfg.SetBech32PrefixForAccount("qor", "qorpub")
	cfg.SetBech32PrefixForValidator("qorvaloper", "qorvaloperpub")
	cfg.SetBech32PrefixForConsensusNode("qorvalcons", "qorvalconspub")
	cfg.Seal()
}

// NewRootCmd creates a new root command for qorechaind.
func NewRootCmd() *cobra.Command {
	var (
		autoCliOpts        autocli.AppOptions
		moduleBasicManager module.BasicManager
		clientCtx          client.Context
	)

	if err := depinject.Inject(
		depinject.Configs(app.AppConfig,
			depinject.Supply(
				log.NewNopLogger(),
			),
			depinject.Provide(
				ProvideClientContext,
			),
		),
		&autoCliOpts,
		&moduleBasicManager,
		&clientCtx,
	); err != nil {
		panic(err)
	}

	// Register custom module basics so they participate in genesis init/export.
	// These modules are not registered via depinject so we add them here.
	moduleBasicManager[pqcmodule.AppModuleBasic{}.Name()] = pqcmodule.AppModuleBasic{}
	moduleBasicManager[aimodule.AppModuleBasic{}.Name()] = aimodule.AppModuleBasic{}
	moduleBasicManager[reputationmodule.AppModuleBasic{}.Name()] = reputationmodule.AppModuleBasic{}
	moduleBasicManager[qcamodule.AppModuleBasic{}.Name()] = qcamodule.AppModuleBasic{}
	moduleBasicManager[bridgemodule.AppModuleBasic{}.Name()] = bridgemodule.AppModuleBasic{}

	rootCmd := &cobra.Command{
		Use:           "qorechaind",
		Short:         "QoreChain — Quantum-Safe AI-Native Blockchain",
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			cmd.SetOut(cmd.OutOrStdout())
			cmd.SetErr(cmd.ErrOrStderr())

			clientCtx = clientCtx.WithCmdContext(cmd.Context()).WithViper("")
			clientCtx, err := client.ReadPersistentCommandFlags(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}

			clientCtx, err = config.ReadFromClientConfig(clientCtx)
			if err != nil {
				return err
			}

			if err := client.SetCmdClientContextHandler(clientCtx, cmd); err != nil {
				return err
			}

			customAppTemplate, customAppConfig := initAppConfig()
			customCMTConfig := initCometBFTConfig()

			return server.InterceptConfigsPreRunHandler(cmd, customAppTemplate, customAppConfig, customCMTConfig)
		},
	}

	initRootCmd(rootCmd, clientCtx.TxConfig, moduleBasicManager)

	nodeCmds := nodeservice.NewNodeCommands()
	autoCliOpts.ModuleOptions = make(map[string]*autocliv1.ModuleOptions)
	autoCliOpts.ModuleOptions[nodeCmds.Name()] = nodeCmds.AutoCLIOptions()

	if err := autoCliOpts.EnhanceRootCommand(rootCmd); err != nil {
		panic(err)
	}

	return rootCmd
}

// ProvideClientContext provides a client.Context for dependency injection.
func ProvideClientContext(
	appCodec codec.Codec,
	interfaceRegistry codectypes.InterfaceRegistry,
	txConfigOpts tx.ConfigOptions,
	legacyAmino *codec.LegacyAmino,
) client.Context {
	clientCtx := client.Context{}.
		WithCodec(appCodec).
		WithInterfaceRegistry(interfaceRegistry).
		WithLegacyAmino(legacyAmino).
		WithInput(os.Stdin).
		WithAccountRetriever(types.AccountRetriever{}).
		WithHomeDir(app.DefaultNodeHome).
		WithViper("")

	clientCtx, _ = config.ReadFromClientConfig(clientCtx)

	txConfigOpts.TextualCoinMetadataQueryFn = authtxconfig.NewGRPCCoinMetadataQueryFn(clientCtx)
	txConfig, err := tx.NewTxConfigWithOptions(clientCtx.Codec, txConfigOpts)
	if err != nil {
		panic(err)
	}
	clientCtx = clientCtx.WithTxConfig(txConfig)

	return clientCtx
}
