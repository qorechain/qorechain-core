package app

import (
	"io"

	dbm "github.com/cosmos/cosmos-db"

	clienthelpers "cosmossdk.io/client/v2/helpers"
	"cosmossdk.io/depinject"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	circuitkeeper "cosmossdk.io/x/circuit/keeper"
	evidencekeeper "cosmossdk.io/x/evidence/keeper"
	feegrantkeeper "cosmossdk.io/x/feegrant/keeper"
	nftkeeper "cosmossdk.io/x/nft/keeper"
	upgradekeeper "cosmossdk.io/x/upgrade/keeper"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authsims "github.com/cosmos/cosmos-sdk/x/auth/simulation"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	consensuskeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	epochskeeper "github.com/cosmos/cosmos-sdk/x/epochs/keeper"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	groupkeeper "github.com/cosmos/cosmos-sdk/x/group/keeper"
	mintkeeper "github.com/cosmos/cosmos-sdk/x/mint/keeper"
	protocolpoolkeeper "github.com/cosmos/cosmos-sdk/x/protocolpool/keeper"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

	pqcmod "github.com/qorechain/qorechain-core/x/pqc"
	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"

	aimod "github.com/qorechain/qorechain-core/x/ai"
	aitypes "github.com/qorechain/qorechain-core/x/ai/types"

	bridgemod "github.com/qorechain/qorechain-core/x/bridge"
	bridgetypes "github.com/qorechain/qorechain-core/x/bridge/types"

	multilayermod "github.com/qorechain/qorechain-core/x/multilayer"
	multilayertypes "github.com/qorechain/qorechain-core/x/multilayer/types"

	reputationmodule "github.com/qorechain/qorechain-core/x/reputation"
	reputationkeeper "github.com/qorechain/qorechain-core/x/reputation/keeper"
	reputationtypes "github.com/qorechain/qorechain-core/x/reputation/types"

	qcamodule "github.com/qorechain/qorechain-core/x/qca"
	qcakeeper "github.com/qorechain/qorechain-core/x/qca/keeper"
	qcatypes "github.com/qorechain/qorechain-core/x/qca/types"
)

const AppName = "QoreChain"

// DefaultNodeHome is the default home directory for the QoreChain daemon.
var DefaultNodeHome string

var (
	_ runtime.AppI            = (*QoreChainApp)(nil)
	_ servertypes.Application = (*QoreChainApp)(nil)
)

// QoreChainApp extends an ABCI application with all standard modules
// plus custom QoreChain modules (pqc, ai, reputation, qca, bridge).
type QoreChainApp struct {
	*runtime.App
	legacyAmino       *codec.LegacyAmino
	appCodec          codec.Codec
	txConfig          client.TxConfig
	interfaceRegistry codectypes.InterfaceRegistry

	// Standard keepers
	AccountKeeper         authkeeper.AccountKeeper
	BankKeeper            bankkeeper.BaseKeeper
	StakingKeeper         *stakingkeeper.Keeper
	SlashingKeeper        slashingkeeper.Keeper
	MintKeeper            mintkeeper.Keeper
	DistrKeeper           distrkeeper.Keeper
	GovKeeper             *govkeeper.Keeper
	UpgradeKeeper         *upgradekeeper.Keeper
	EvidenceKeeper        evidencekeeper.Keeper
	ConsensusParamsKeeper consensuskeeper.Keeper
	CircuitKeeper         circuitkeeper.Keeper
	FeeGrantKeeper        feegrantkeeper.Keeper
	GroupKeeper           groupkeeper.Keeper
	AuthzKeeper           authzkeeper.Keeper
	NFTKeeper             nftkeeper.Keeper
	EpochsKeeper          epochskeeper.Keeper
	ProtocolPoolKeeper    protocolpoolkeeper.Keeper

	// Custom QoreChain keepers (interface types for open-core architecture)
	PQCKeeper          pqcmod.PQCKeeper
	AIKeeper           aimod.AIKeeper
	ReputationKeeper   reputationkeeper.Keeper
	QCAKeeper          qcakeeper.Keeper
	BridgeKeeper       bridgemod.BridgeKeeper
	MultilayerKeeper   multilayermod.MultilayerKeeper

	// PQC client (interface type)
	pqcClient pqcmod.PQCClient

	// simulation manager
	sm *module.SimulationManager
}

func init() {
	var err error
	DefaultNodeHome, err = clienthelpers.GetNodeHomeDirectory(".qorechaind")
	if err != nil {
		panic(err)
	}
}

// NewQoreChainApp returns a reference to an initialized QoreChainApp.
func NewQoreChainApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	appOpts servertypes.AppOptions,
	baseAppOptions ...func(*baseapp.BaseApp),
) *QoreChainApp {
	var (
		app        = &QoreChainApp{}
		appBuilder *runtime.AppBuilder

		appConfig = depinject.Configs(
			AppConfig,
			depinject.Supply(
				appOpts,
				logger,
			),
		)
	)

	if err := depinject.Inject(appConfig,
		&appBuilder,
		&app.appCodec,
		&app.legacyAmino,
		&app.txConfig,
		&app.interfaceRegistry,
		&app.AccountKeeper,
		&app.BankKeeper,
		&app.StakingKeeper,
		&app.SlashingKeeper,
		&app.MintKeeper,
		&app.DistrKeeper,
		&app.GovKeeper,
		&app.UpgradeKeeper,
		&app.AuthzKeeper,
		&app.EvidenceKeeper,
		&app.FeeGrantKeeper,
		&app.GroupKeeper,
		&app.NFTKeeper,
		&app.ConsensusParamsKeeper,
		&app.CircuitKeeper,
		&app.EpochsKeeper,
		&app.ProtocolPoolKeeper,
	); err != nil {
		panic(err)
	}

	app.App = appBuilder.Build(db, traceStore, baseAppOptions...)

	// --- Initialize PQC module (via factory) ---
	pqcStoreKey := storetypes.NewKVStoreKey(pqctypes.StoreKey)
	app.MountStores(pqcStoreKey)

	app.pqcClient = NewPQCClient()
	app.PQCKeeper = NewPQCKeeper(
		app.appCodec,
		pqcStoreKey,
		app.pqcClient,
		logger,
	)

	// --- Initialize AI module (via factory) ---
	aiStoreKey := storetypes.NewKVStoreKey(aitypes.StoreKey)
	app.MountStores(aiStoreKey)

	app.AIKeeper = NewAIKeeper(
		app.appCodec,
		aiStoreKey,
		logger,
	)

	// --- Initialize Reputation module (manual, not depinject) ---
	reputationStoreKey := storetypes.NewKVStoreKey(reputationtypes.StoreKey)
	app.MountStores(reputationStoreKey)

	app.ReputationKeeper = reputationkeeper.NewKeeper(
		app.appCodec,
		reputationStoreKey,
		app.StakingKeeper,
		logger,
	)

	// --- Initialize QCA module (manual, not depinject) ---
	qcaStoreKey := storetypes.NewKVStoreKey(qcatypes.StoreKey)
	app.MountStores(qcaStoreKey)

	qcaSelector := qcakeeper.NewHeuristicSelector()
	app.QCAKeeper = qcakeeper.NewKeeper(
		app.appCodec,
		qcaStoreKey,
		app.ReputationKeeper,
		qcaSelector,
		logger,
	)

	// --- Initialize Bridge module (via factory) ---
	bridgeStoreKey := storetypes.NewKVStoreKey(bridgetypes.StoreKey)
	app.MountStores(bridgeStoreKey)

	app.BridgeKeeper = NewBridgeKeeper(
		app.appCodec,
		bridgeStoreKey,
		app.PQCKeeper,
		logger,
	)

	// --- Initialize Multilayer module (via factory) ---
	multilayerStoreKey := storetypes.NewKVStoreKey(multilayertypes.StoreKey)
	app.MountStores(multilayerStoreKey)

	app.MultilayerKeeper = NewMultilayerKeeper(
		app.appCodec,
		multilayerStoreKey,
		logger,
	)

	// Register custom modules with both ModuleManager AND basicManager
	// so they participate in genesis init/export (not just ModuleManager.Modules[])
	if err := app.RegisterModules(
		NewPQCAppModule(app.PQCKeeper),
		NewAIAppModule(app.AIKeeper),
		reputationmodule.NewAppModule(app.ReputationKeeper),
		qcamodule.NewAppModule(app.QCAKeeper),
		NewBridgeAppModule(app.BridgeKeeper),
		NewMultilayerAppModule(app.MultilayerKeeper),
	); err != nil {
		panic(err)
	}

	if err := app.RegisterStreamingServices(appOpts, app.kvStoreKeys()); err != nil {
		panic(err)
	}

	app.RegisterUpgradeHandlers()

	overrideModules := map[string]module.AppModuleSimulation{
		authtypes.ModuleName: auth.NewAppModule(app.appCodec, app.AccountKeeper, authsims.RandomGenesisAccounts, nil),
	}
	app.sm = module.NewSimulationManagerFromAppModules(app.ModuleManager.Modules, overrideModules)
	app.sm.RegisterStoreDecoders()

	app.setAnteHandler(app.txConfig)

	if err := app.Load(loadLatest); err != nil {
		panic(err)
	}

	return app
}

func (app *QoreChainApp) setAnteHandler(txConfig client.TxConfig) {
	anteHandler, err := NewAnteHandler(
		HandlerOptions{
			HandlerOptions: ante.HandlerOptions{
				AccountKeeper:   app.AccountKeeper,
				BankKeeper:      app.BankKeeper,
				SignModeHandler: txConfig.SignModeHandler(),
				FeegrantKeeper:  app.FeeGrantKeeper,
				SigGasConsumer:  ante.DefaultSigVerificationGasConsumer,
			},
			CircuitKeeper: &app.CircuitKeeper,
			PQCKeeper:     app.PQCKeeper,
			PQCClient:     app.pqcClient,
			AIKeeper:      app.AIKeeper,
		},
	)
	if err != nil {
		panic(err)
	}
	app.SetAnteHandler(anteHandler)
}

// RegisterUpgradeHandlers registers any on-chain upgrade handlers.
func (app *QoreChainApp) RegisterUpgradeHandlers() {
	// Upgrade handlers will be registered here as needed.
}

// LegacyAmino returns the app's legacy amino codec.
func (app *QoreChainApp) LegacyAmino() *codec.LegacyAmino {
	return app.legacyAmino
}

// AppCodec returns the app's codec.
func (app *QoreChainApp) AppCodec() codec.Codec {
	return app.appCodec
}

// InterfaceRegistry returns the app's InterfaceRegistry.
func (app *QoreChainApp) InterfaceRegistry() codectypes.InterfaceRegistry {
	return app.interfaceRegistry
}

// TxConfig returns the app's TxConfig.
func (app *QoreChainApp) TxConfig() client.TxConfig {
	return app.txConfig
}

// GetKey returns the KVStoreKey for the provided store key.
func (app *QoreChainApp) GetKey(storeKey string) *storetypes.KVStoreKey {
	sk := app.UnsafeFindStoreKey(storeKey)
	kvStoreKey, ok := sk.(*storetypes.KVStoreKey)
	if !ok {
		return nil
	}
	return kvStoreKey
}

func (app *QoreChainApp) kvStoreKeys() map[string]*storetypes.KVStoreKey {
	keys := make(map[string]*storetypes.KVStoreKey)
	for _, k := range app.GetStoreKeys() {
		if kv, ok := k.(*storetypes.KVStoreKey); ok {
			keys[kv.Name()] = kv
		}
	}
	return keys
}

// SimulationManager implements the SimulationApp interface.
func (app *QoreChainApp) SimulationManager() *module.SimulationManager {
	return app.sm
}

// RegisterAPIRoutes registers all application module routes with the provided API server.
func (app *QoreChainApp) RegisterAPIRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
	app.App.RegisterAPIRoutes(apiSvr, apiConfig)
	if err := server.RegisterSwaggerAPI(apiSvr.ClientCtx, apiSvr.Router, apiConfig.Swagger); err != nil {
		panic(err)
	}
}

// GetMaccPerms returns a copy of the module account permissions.
func GetMaccPerms() map[string][]string {
	dup := make(map[string][]string)
	for _, perms := range moduleAccPerms {
		dup[perms.Account] = perms.Permissions
	}
	return dup
}

// BlockedAddresses returns all the app's blocked account addresses.
func BlockedAddresses() map[string]bool {
	result := make(map[string]bool)
	if len(blockAccAddrs) > 0 {
		for _, addr := range blockAccAddrs {
			result[addr] = true
		}
	} else {
		for addr := range GetMaccPerms() {
			result[addr] = true
		}
	}
	return result
}
