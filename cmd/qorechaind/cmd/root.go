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

	// IBC
	ibc "github.com/cosmos/ibc-go/v10/modules/core"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
	ibctransfer "github.com/cosmos/ibc-go/v10/modules/apps/transfer"

	// QoreChain EVM
	evmvm "github.com/cosmos/evm/x/vm"
	evmfeemarket "github.com/cosmos/evm/x/feemarket"
	evmerc20 "github.com/cosmos/evm/x/erc20"
	evmprecisebank "github.com/cosmos/evm/x/precisebank"

	// CosmWasm
	wasm "github.com/CosmWasm/wasmd/x/wasm"

	"github.com/qorechain/qorechain-core/app"
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

	// Register non-depinject module basics so they participate in genesis init/export.

	// IBC modules
	moduleBasicManager[ibc.AppModule{}.Name()] = ibc.AppModule{}
	moduleBasicManager[ibctransfer.AppModule{}.Name()] = ibctransfer.AppModule{}
	moduleBasicManager[ibctm.AppModuleBasic{}.Name()] = ibctm.AppModuleBasic{}

	// EVM modules
	moduleBasicManager[evmvm.AppModuleBasic{}.Name()] = evmvm.AppModuleBasic{}
	moduleBasicManager[evmfeemarket.AppModuleBasic{}.Name()] = evmfeemarket.AppModuleBasic{}
	moduleBasicManager[evmerc20.AppModuleBasic{}.Name()] = evmerc20.AppModuleBasic{}
	moduleBasicManager[evmprecisebank.AppModuleBasic{}.Name()] = evmprecisebank.AppModuleBasic{}

	// CosmWasm module
	moduleBasicManager[wasm.AppModuleBasic{}.Name()] = wasm.AppModuleBasic{}

	// QoreChain custom modules (proprietary use factory pattern)
	pqcBasic := app.NewPQCModuleBasic()
	aiBasic := app.NewAIModuleBasic()
	bridgeBasic := app.NewBridgeModuleBasic()
	moduleBasicManager[pqcBasic.Name()] = pqcBasic
	moduleBasicManager[aiBasic.Name()] = aiBasic
	moduleBasicManager[reputationmodule.AppModuleBasic{}.Name()] = reputationmodule.AppModuleBasic{}
	moduleBasicManager[qcamodule.AppModuleBasic{}.Name()] = qcamodule.AppModuleBasic{}
	moduleBasicManager[bridgeBasic.Name()] = bridgeBasic
	crossvmBasic := app.NewCrossVMModuleBasic()
	moduleBasicManager[crossvmBasic.Name()] = crossvmBasic
	multilayerBasic := app.NewMultilayerModuleBasic()
	moduleBasicManager[multilayerBasic.Name()] = multilayerBasic
	svmBasic := app.NewSVMModuleBasic()
	moduleBasicManager[svmBasic.Name()] = svmBasic
	rlBasic := app.NewRLConsensusModuleBasic()
	moduleBasicManager[rlBasic.Name()] = rlBasic
	burnBasic := app.NewBurnModuleBasic()
	moduleBasicManager[burnBasic.Name()] = burnBasic
	xqoreBasic := app.NewXQOREModuleBasic()
	moduleBasicManager[xqoreBasic.Name()] = xqoreBasic
	inflationBasic := app.NewInflationModuleBasic()
	moduleBasicManager[inflationBasic.Name()] = inflationBasic

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
