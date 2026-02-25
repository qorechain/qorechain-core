package app

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/spf13/cast"

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
	sdk "github.com/cosmos/cosmos-sdk/types"
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
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	groupkeeper "github.com/cosmos/cosmos-sdk/x/group/keeper"
	mintkeeper "github.com/cosmos/cosmos-sdk/x/mint/keeper"
	protocolpoolkeeper "github.com/cosmos/cosmos-sdk/x/protocolpool/keeper"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

	// IBC
	ibc "github.com/cosmos/ibc-go/v10/modules/core"
	ibckeeper "github.com/cosmos/ibc-go/v10/modules/core/keeper"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	ibcapi "github.com/cosmos/ibc-go/v10/modules/core/api"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"

	// QoreChain EVM
	srvflags "github.com/cosmos/evm/server/flags"
	cosmosevmtypes "github.com/cosmos/evm/types"
	evmante "github.com/cosmos/evm/ante/evm"
	evmkeeper "github.com/cosmos/evm/x/vm/keeper"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	erc20keeper "github.com/cosmos/evm/x/erc20/keeper"
	erc20types "github.com/cosmos/evm/x/erc20/types"
	erc20v2 "github.com/cosmos/evm/x/erc20/v2"
	feemarketkeeper "github.com/cosmos/evm/x/feemarket/keeper"
	feemarkettypes "github.com/cosmos/evm/x/feemarket/types"
	precisebankkeeper "github.com/cosmos/evm/x/precisebank/keeper"
	precisebanktypes "github.com/cosmos/evm/x/precisebank/types"
	evmvm "github.com/cosmos/evm/x/vm"
	evmfeemarket "github.com/cosmos/evm/x/feemarket"
	evmerc20 "github.com/cosmos/evm/x/erc20"
	evmprecisebank "github.com/cosmos/evm/x/precisebank"
	evmtransfer "github.com/cosmos/evm/x/ibc/transfer"
	evmtransferkeeper "github.com/cosmos/evm/x/ibc/transfer/keeper"
	evmtransferv2 "github.com/cosmos/evm/x/ibc/transfer/v2"

	// CosmWasm
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	wasm "github.com/CosmWasm/wasmd/x/wasm"

	// Params (for legacy IBC subspace compatibility)
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"

	// geth tracers (required side-effect imports for EVM)
	_ "github.com/ethereum/go-ethereum/eth/tracers/js"
	_ "github.com/ethereum/go-ethereum/eth/tracers/native"

	// QoreChain custom modules
	abstractaccountmod "github.com/qorechain/qorechain-core/x/abstractaccount"
	abstractaccounttypes "github.com/qorechain/qorechain-core/x/abstractaccount/types"

	aimod "github.com/qorechain/qorechain-core/x/ai"
	aitypes "github.com/qorechain/qorechain-core/x/ai/types"

	babylonmod "github.com/qorechain/qorechain-core/x/babylon"
	babylontypes "github.com/qorechain/qorechain-core/x/babylon/types"

	bridgemod "github.com/qorechain/qorechain-core/x/bridge"
	bridgetypes "github.com/qorechain/qorechain-core/x/bridge/types"

	burnmod "github.com/qorechain/qorechain-core/x/burn"
	burntypes "github.com/qorechain/qorechain-core/x/burn/types"

	crossvmmod "github.com/qorechain/qorechain-core/x/crossvm"
	crossvmtypes "github.com/qorechain/qorechain-core/x/crossvm/types"

	fairblockmod "github.com/qorechain/qorechain-core/x/fairblock"
	fairblocktypes "github.com/qorechain/qorechain-core/x/fairblock/types"

	gasabstractionmod "github.com/qorechain/qorechain-core/x/gasabstraction"
	gasabstractiontypes "github.com/qorechain/qorechain-core/x/gasabstraction/types"

	inflationmod "github.com/qorechain/qorechain-core/x/inflation"
	inflationtypes "github.com/qorechain/qorechain-core/x/inflation/types"

	multilayermod "github.com/qorechain/qorechain-core/x/multilayer"
	multilayertypes "github.com/qorechain/qorechain-core/x/multilayer/types"

	pqcmod "github.com/qorechain/qorechain-core/x/pqc"
	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"

	rlconsensusmod "github.com/qorechain/qorechain-core/x/rlconsensus"
	rlconsensustypes "github.com/qorechain/qorechain-core/x/rlconsensus/types"

	svmmod "github.com/qorechain/qorechain-core/x/svm"
	svmtypes "github.com/qorechain/qorechain-core/x/svm/types"

	xqoremod "github.com/qorechain/qorechain-core/x/xqore"
	xqoretypes "github.com/qorechain/qorechain-core/x/xqore/types"

	reputationmodule "github.com/qorechain/qorechain-core/x/reputation"
	reputationkeeper "github.com/qorechain/qorechain-core/x/reputation/keeper"
	reputationtypes "github.com/qorechain/qorechain-core/x/reputation/types"

	qcamodule "github.com/qorechain/qorechain-core/x/qca"
	qcakeeper "github.com/qorechain/qorechain-core/x/qca/keeper"
	qcatypes "github.com/qorechain/qorechain-core/x/qca/types"
)

const AppName = "QoreChain"

// emptySubspace implements ibc-go ParamSubspace interface as a no-op.
// IBC v10 only uses this for legacy migration reads; fresh chains don't need it.
type emptySubspace struct{}

func (emptySubspace) GetParamSet(_ sdk.Context, _ paramstypes.ParamSet) {}

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

	// IBC keepers
	IBCKeeper      *ibckeeper.Keeper
	TransferKeeper evmtransferkeeper.Keeper

	// EVM keepers (QoreChain EVM)
	FeeMarketKeeper   feemarketkeeper.Keeper
	EVMKeeper         *evmkeeper.Keeper
	Erc20Keeper       erc20keeper.Keeper
	PreciseBankKeeper precisebankkeeper.Keeper

	// CosmWasm keeper
	WasmKeeper wasmkeeper.Keeper

	// Custom QoreChain keepers (interface types for open-core architecture)
	PQCKeeper        pqcmod.PQCKeeper
	AIKeeper         aimod.AIKeeper
	ReputationKeeper reputationkeeper.Keeper
	QCAKeeper        qcakeeper.Keeper
	BridgeKeeper     bridgemod.BridgeKeeper
	CrossVMKeeper    crossvmmod.CrossVMKeeper
	MultilayerKeeper  multilayermod.MultilayerKeeper
	SVMKeeper         svmmod.SVMKeeper
	RLConsensusKeeper rlconsensusmod.RLConsensusKeeper
	BurnKeeper             burnmod.BurnKeeper
	XQOREKeeper            xqoremod.XQOREKeeper
	InflationKeeper        inflationmod.InflationKeeper
	BabylonKeeper          babylonmod.BabylonKeeper
	AbstractAccountKeeper  abstractaccountmod.AbstractAccountKeeper
	FairBlockKeeper        fairblockmod.FairBlockKeeper
	GasAbstractionKeeper   gasabstractionmod.GasAbstractionKeeper

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

	// Governance authority address (used as keeper authority for IBC/EVM/Wasm modules)
	authAddr := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	// ==========================================================================
	// IBC Core + Transfer
	// ==========================================================================
	ibcStoreKey := storetypes.NewKVStoreKey(ibcexported.StoreKey)
	transferStoreKey := storetypes.NewKVStoreKey(ibctransfertypes.StoreKey)
	app.MountStores(ibcStoreKey, transferStoreKey)

	app.IBCKeeper = ibckeeper.NewKeeper(
		app.appCodec,
		runtime.NewKVStoreService(ibcStoreKey),
		emptySubspace{}, // legacy param subspace (only used for migration, not needed for fresh chains)
		app.UpgradeKeeper,
		authAddr,
	)

	// ==========================================================================
	// QoreChain EVM: FeeMarket → PreciseBank → EVM → ERC20 (init order matters)
	// ==========================================================================
	feeMarketStoreKey := storetypes.NewKVStoreKey(feemarkettypes.StoreKey)
	feeMarketTransientKey := storetypes.NewTransientStoreKey(feemarkettypes.TransientKey)
	preciseBankStoreKey := storetypes.NewKVStoreKey(precisebanktypes.StoreKey)
	evmStoreKey := storetypes.NewKVStoreKey(evmtypes.StoreKey)
	evmTransientKey := storetypes.NewTransientStoreKey(evmtypes.TransientKey)
	erc20StoreKey := storetypes.NewKVStoreKey(erc20types.StoreKey)
	app.MountStores(
		feeMarketStoreKey, feeMarketTransientKey,
		preciseBankStoreKey,
		evmStoreKey, evmTransientKey,
		erc20StoreKey,
	)

	// Step 1: FeeMarketKeeper (no EVM deps)
	app.FeeMarketKeeper = feemarketkeeper.NewKeeper(
		app.appCodec,
		authtypes.NewModuleAddress(govtypes.ModuleName),
		feeMarketStoreKey,
		feeMarketTransientKey,
	)

	// Step 2: PreciseBankKeeper (wraps BankKeeper for 18-decimal EVM operations)
	app.PreciseBankKeeper = precisebankkeeper.NewKeeper(
		app.appCodec,
		preciseBankStoreKey,
		app.BankKeeper,
		app.AccountKeeper,
	)

	// Step 3: EVMKeeper (depends on FeeMarket, PreciseBank; forward-ref to Erc20Keeper)
	allKeys := app.kvStoreKeys()
	app.EVMKeeper = evmkeeper.NewKeeper(
		app.appCodec,
		evmStoreKey,
		evmTransientKey,
		allKeys,
		authtypes.NewModuleAddress(govtypes.ModuleName),
		app.AccountKeeper,
		app.PreciseBankKeeper,
		app.StakingKeeper,
		app.FeeMarketKeeper,
		&app.ConsensusParamsKeeper,
		&app.Erc20Keeper, // forward reference — assigned below
		"",               // tracer (empty = default)
	)

	// Step 4: Erc20Keeper (depends on EVMKeeper; forward-ref to TransferKeeper)
	app.Erc20Keeper = erc20keeper.NewKeeper(
		erc20StoreKey,
		app.appCodec,
		authtypes.NewModuleAddress(govtypes.ModuleName),
		app.AccountKeeper,
		app.PreciseBankKeeper,
		app.EVMKeeper,
		app.StakingKeeper,
		&app.TransferKeeper, // forward reference — assigned below
	)

	// Step 5: EVM-wrapped IBC TransferKeeper (adds ERC-20 auto-registration)
	app.TransferKeeper = evmtransferkeeper.NewKeeper(
		app.appCodec,
		runtime.NewKVStoreService(transferStoreKey),
		paramstypes.Subspace{}, // legacy param subspace (not needed for fresh chains)
		app.IBCKeeper.ChannelKeeper, // ics4Wrapper
		app.IBCKeeper.ChannelKeeper, // channelKeeper
		app.MsgServiceRouter(),
		app.AccountKeeper,
		app.BankKeeper,
		app.Erc20Keeper,
		authAddr,
	)

	// ==========================================================================
	// CosmWasm (x/wasm)
	// ==========================================================================
	wasmStoreKey := storetypes.NewKVStoreKey(wasmtypes.StoreKey)
	app.MountStores(wasmStoreKey)

	wasmDir := filepath.Join(DefaultNodeHome, "wasm")
	wasmNodeConfig := wasmtypes.DefaultNodeConfig()

	app.WasmKeeper = wasmkeeper.NewKeeper(
		app.appCodec,
		runtime.NewKVStoreService(wasmStoreKey),
		app.AccountKeeper,
		app.BankKeeper,
		app.StakingKeeper,
		distrkeeper.NewQuerier(app.DistrKeeper),
		app.IBCKeeper.ChannelKeeper, // ics4Wrapper
		app.IBCKeeper.ChannelKeeper, // channelKeeper
		app.TransferKeeper.Keeper,   // ICS20TransferPortSource
		app.MsgServiceRouter(),
		app.GRPCQueryRouter(),
		wasmDir,
		wasmNodeConfig,
		wasmtypes.VMConfig{},
		wasmkeeper.BuiltInCapabilities(),
		authAddr,
	)

	// ==========================================================================
	// QoreChain custom modules
	// ==========================================================================

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

	// --- Initialize CrossVM module (via factory) ---
	crossvmStoreKey := storetypes.NewKVStoreKey(crossvmtypes.StoreKey)
	app.MountStores(crossvmStoreKey)

	app.CrossVMKeeper = NewCrossVMKeeper(
		app.appCodec,
		crossvmStoreKey,
		app.EVMKeeper,
		&app.WasmKeeper,
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

	// --- Initialize SVM module (via factory) ---
	svmStoreKey := storetypes.NewKVStoreKey(svmtypes.StoreKey)
	app.MountStores(svmStoreKey)

	app.SVMKeeper = NewSVMKeeper(
		app.appCodec,
		svmStoreKey,
		app.PQCKeeper,
		app.AIKeeper,
		app.CrossVMKeeper,
		logger,
	)

	// Wire SVM into CrossVM routing so cross-VM messages can target SVM programs.
	crossvmmod.SetSVMCallHandler(func(ctx sdk.Context, targetContract string, payload []byte, _ string) ([]byte, error) {
		programAddr, err := svmtypes.Base58Decode(targetContract)
		if err != nil {
			return nil, fmt.Errorf("invalid SVM program address: %w", err)
		}
		result, err := app.SVMKeeper.ExecuteProgram(ctx, programAddr, payload, nil, nil)
		if err != nil {
			return nil, err
		}
		if !result.Success {
			return nil, fmt.Errorf("SVM execution failed: %s", result.Error)
		}
		return result.ReturnData, nil
	})

	// --- Initialize RL Consensus module (via factory) ---
	rlconsensusStoreKey := storetypes.NewKVStoreKey(rlconsensustypes.StoreKey)
	app.MountStores(rlconsensusStoreKey)

	app.RLConsensusKeeper = NewRLConsensusKeeper(
		app.appCodec,
		rlconsensusStoreKey,
		logger,
	)

	// --- Initialize Burn module (via factory) ---
	burnStoreKey := storetypes.NewKVStoreKey(burntypes.StoreKey)
	app.MountStores(burnStoreKey)

	app.BurnKeeper = NewBurnKeeper(
		app.appCodec,
		burnStoreKey,
		app.BankKeeper,
		logger,
	)

	// --- Initialize xQORE module (via factory) ---
	xqoreStoreKey := storetypes.NewKVStoreKey(xqoretypes.StoreKey)
	app.MountStores(xqoreStoreKey)

	app.XQOREKeeper = NewXQOREKeeper(
		app.appCodec,
		xqoreStoreKey,
		app.BankKeeper,
		logger,
	)

	// --- Initialize Inflation module (via factory) ---
	inflationStoreKey := storetypes.NewKVStoreKey(inflationtypes.StoreKey)
	app.MountStores(inflationStoreKey)

	app.InflationKeeper = NewInflationKeeper(
		app.appCodec,
		inflationStoreKey,
		app.BankKeeper,
		logger,
	)

	// --- Initialize Babylon module (via factory, v1.2.0 — BTC restaking) ---
	babylonStoreKey := storetypes.NewKVStoreKey(babylontypes.StoreKey)
	app.MountStores(babylonStoreKey)

	app.BabylonKeeper = NewBabylonKeeper(
		app.appCodec,
		babylonStoreKey,
		logger,
	)

	// --- Initialize AbstractAccount module (via factory, v1.2.0 — account abstraction) ---
	abstractaccountStoreKey := storetypes.NewKVStoreKey(abstractaccounttypes.StoreKey)
	app.MountStores(abstractaccountStoreKey)

	app.AbstractAccountKeeper = NewAbstractAccountKeeper(
		app.appCodec,
		abstractaccountStoreKey,
		logger,
	)

	// --- Initialize FairBlock module (via factory, v1.2.0 — threshold IBE) ---
	fairblockStoreKey := storetypes.NewKVStoreKey(fairblocktypes.StoreKey)
	app.MountStores(fairblockStoreKey)

	app.FairBlockKeeper = NewFairBlockKeeper(
		app.appCodec,
		fairblockStoreKey,
		logger,
	)

	// --- Initialize GasAbstraction module (via factory, v1.2.0 — IBC token fees) ---
	gasabstractionStoreKey := storetypes.NewKVStoreKey(gasabstractiontypes.StoreKey)
	app.MountStores(gasabstractionStoreKey)

	app.GasAbstractionKeeper = NewGasAbstractionKeeper(
		app.appCodec,
		gasabstractionStoreKey,
		logger,
	)

	// ==========================================================================
	// IBC Router Setup (transfer stack with ERC-20 middleware)
	// ==========================================================================

	// IBC v1 transfer stack: channel → erc20 middleware → transfer
	var transferStack porttypes.IBCModule
	transferStack = evmtransfer.NewIBCModule(app.TransferKeeper)
	transferStack = evmerc20.NewIBCMiddleware(app.Erc20Keeper, transferStack)

	// IBC v2 transfer stack
	var transferStackV2 ibcapi.IBCModule
	transferStackV2 = evmtransferv2.NewIBCModule(app.TransferKeeper)
	transferStackV2 = erc20v2.NewIBCMiddleware(transferStackV2, app.Erc20Keeper)

	// Create IBC routers, add transfer route, set and seal
	ibcRouter := porttypes.NewRouter()
	ibcRouter.AddRoute(ibctransfertypes.ModuleName, transferStack)
	ibcRouterV2 := ibcapi.NewRouter()
	ibcRouterV2.AddRoute(ibctransfertypes.ModuleName, transferStackV2)

	app.IBCKeeper.SetRouter(ibcRouter)
	app.IBCKeeper.SetRouterV2(ibcRouterV2)

	// Register IBC light client module
	storeProvider := app.IBCKeeper.ClientKeeper.GetStoreProvider()
	tmLightClientModule := ibctm.NewLightClientModule(app.appCodec, storeProvider)
	app.IBCKeeper.ClientKeeper.AddRoute(ibctm.ModuleName, &tmLightClientModule)

	// ==========================================================================
	// Module Registration
	// ==========================================================================

	// Register all non-depinject modules with the ModuleManager.
	if err := app.RegisterModules(
		// IBC modules
		ibc.NewAppModule(app.IBCKeeper),
		evmtransfer.NewAppModule(app.TransferKeeper),
		ibctm.NewAppModule(tmLightClientModule),
		// EVM modules
		evmvm.NewAppModule(app.EVMKeeper, app.AccountKeeper, app.AccountKeeper.AddressCodec()),
		evmfeemarket.NewAppModule(app.FeeMarketKeeper),
		evmerc20.NewAppModule(app.Erc20Keeper, app.AccountKeeper),
		evmprecisebank.NewAppModule(app.PreciseBankKeeper, app.BankKeeper, app.AccountKeeper),
		// CosmWasm module
		wasm.NewAppModule(app.appCodec, &app.WasmKeeper, app.StakingKeeper, app.AccountKeeper, app.BankKeeper, app.MsgServiceRouter(), nil),
		// QoreChain custom modules
		NewPQCAppModule(app.PQCKeeper),
		NewAIAppModule(app.AIKeeper),
		reputationmodule.NewAppModule(app.ReputationKeeper),
		qcamodule.NewAppModule(app.QCAKeeper),
		NewBridgeAppModule(app.BridgeKeeper),
		NewCrossVMAppModule(app.CrossVMKeeper),
		NewMultilayerAppModule(app.MultilayerKeeper),
		NewSVMAppModule(app.SVMKeeper),
		NewRLConsensusAppModule(app.RLConsensusKeeper),
		NewBurnAppModule(app.BurnKeeper),
		NewXQOREAppModule(app.XQOREKeeper),
		NewInflationAppModule(app.InflationKeeper),
		NewBabylonAppModule(app.BabylonKeeper),
		NewAbstractAccountAppModule(app.AbstractAccountKeeper),
		NewFairBlockAppModule(app.FairBlockKeeper),
		NewGasAbstractionAppModule(app.GasAbstractionKeeper),
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

	app.registerEVMPrecompiles()

	// Read max gas wanted from server flags (evm.max-tx-gas-wanted).
	// Default is 0 (unlimited) if not configured.
	maxGasWanted := cast.ToUint64(appOpts.Get(srvflags.EVMMaxTxGasWanted))

	// Read CosmWasm node config from app options.
	wasmNodeConfig, err := wasm.ReadNodeConfig(appOpts)
	if err != nil {
		panic("failed to read wasm node config: " + err.Error())
	}
	app.setAnteHandler(app.txConfig, maxGasWanted, wasmNodeConfig, wasmStoreKey)

	if err := app.Load(loadLatest); err != nil {
		panic(err)
	}

	return app
}

func (app *QoreChainApp) setAnteHandler(
	txConfig client.TxConfig,
	maxTxGasWanted uint64,
	wasmNodeConfig wasmtypes.NodeConfig,
	wasmStoreKey *storetypes.KVStoreKey,
) {
	anteHandler, err := NewAnteHandler(
		HandlerOptions{
			HandlerOptions: ante.HandlerOptions{
				AccountKeeper:          app.AccountKeeper,
				BankKeeper:             app.BankKeeper,
				SignModeHandler:        txConfig.SignModeHandler(),
				FeegrantKeeper:         app.FeeGrantKeeper,
				SigGasConsumer:         sigVerificationGasConsumerWithPQC,
				ExtensionOptionChecker: cosmosevmtypes.HasDynamicFeeExtensionOption,
				TxFeeChecker:          evmante.NewDynamicFeeChecker(app.FeeMarketKeeper),
			},
			CircuitKeeper:         &app.CircuitKeeper,
			PQCKeeper:             app.PQCKeeper,
			PQCClient:             app.pqcClient,
			AIKeeper:              app.AIKeeper,
			SVMKeeper:             app.SVMKeeper,
			EVMAccountKeeper:      app.AccountKeeper,
			FeeMarketKeeper:       app.FeeMarketKeeper,
			EvmKeeper:             app.EVMKeeper,
			IBCKeeper:             app.IBCKeeper,
			MaxTxGasWanted:        maxTxGasWanted,
			WasmKeeper:            &app.WasmKeeper,
			WasmConfig:            &wasmNodeConfig,
			TXCounterStoreService: runtime.NewKVStoreService(wasmStoreKey),
			FairBlockKeeper:       app.FairBlockKeeper,
			GasAbstractionKeeper:  app.GasAbstractionKeeper,
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
