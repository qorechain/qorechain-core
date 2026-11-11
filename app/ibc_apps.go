package app

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	gogogrpc "github.com/cosmos/gogoproto/grpc"
	"github.com/spf13/cast"
	"google.golang.org/grpc"

	corestore "cosmossdk.io/core/store"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/store/prefix"
	"cosmossdk.io/store/snapshots"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/module"

	packetforward "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v10/packetforward"
	ratelimit "github.com/cosmos/ibc-apps/modules/rate-limiting/v10"
	ratelimitkeeper "github.com/cosmos/ibc-apps/modules/rate-limiting/v10/keeper"
	ratelimittypes "github.com/cosmos/ibc-apps/modules/rate-limiting/v10/types"
	ibcwasm "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10"
	ibcwasmkeeper "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/keeper"
	ibcwasmtypes "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	ibcapi "github.com/cosmos/ibc-go/v10/modules/core/api"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"

	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
)

// IBC applications added in v3.2.6: the ICS-20 rate limiter, Packet Forward
// Middleware and the 08-wasm light client.
//
// WHERE THEIR STATE LIVES. None of the three gets a store of its own. Every
// mounted store is part of the app hash from the first block, empty or not, so
// a store mounted by this binary would give every block it replays below the
// v3.2.6 plan height a different app hash than the one the network committed
// (the reason x/stakingtier has no store either, see stakingtiertypes.HostStoreKey).
// A store loader adding the stores at the plan height fixes the live upgrade but
// not that replay, and not a node that replays across the height in one run.
// Each module therefore keeps its records in the ICS-20 transfer store, under a
// prefix of its own. The transfer module's own keys are "params",
// "totalEscrowForDenom/..." and the one-byte prefixes 0x01..0x03, it only ever
// iterates under those prefixes, and no key it writes starts with "qore/".
// Each hosted module reads and iterates only its own prefix: what it sees is
// exactly the store it would have had.
var (
	rateLimitHostPrefix     = []byte("qore/ratelimit/")
	packetForwardHostPrefix = []byte("qore/packetforward/")
	wasmClientHostPrefix    = []byte("qore/08-wasm/")
)

// wasmClientQueryAllowList is what an 08-wasm light client contract may query
// through the gRPC router: the client, consensus state and connection reads
// IBC light clients need, all deterministic and gas-metered. Nothing else.
var wasmClientQueryAllowList = []string{
	"/ibc.core.client.v1.Query/ClientState",
	"/ibc.core.client.v1.Query/ConsensusState",
	"/ibc.core.connection.v1.Query/Connection",
}

// hostedStoreService is a KVStoreService over one prefix of an existing store.
type hostedStoreService struct {
	key    storetypes.StoreKey
	prefix []byte
}

var _ corestore.KVStoreService = hostedStoreService{}

func (s hostedStoreService) OpenKVStore(ctx context.Context) corestore.KVStore {
	return coreKVStore{prefix.NewStore(sdk.UnwrapSDKContext(ctx).KVStore(s.key), s.prefix)}
}

// coreKVStore presents a store.KVStore through the core API, the same way the
// runtime's store service does.
type coreKVStore struct{ s storetypes.KVStore }

func (c coreKVStore) Get(key []byte) ([]byte, error) { return c.s.Get(key), nil }
func (c coreKVStore) Has(key []byte) (bool, error)   { return c.s.Has(key), nil }
func (c coreKVStore) Set(key, value []byte) error {
	c.s.Set(key, value)
	return nil
}

func (c coreKVStore) Delete(key []byte) error {
	c.s.Delete(key)
	return nil
}

func (c coreKVStore) Iterator(start, end []byte) (corestore.Iterator, error) {
	return c.s.Iterator(start, end), nil
}

func (c coreKVStore) ReverseIterator(start, end []byte) (corestore.Iterator, error) {
	return c.s.ReverseIterator(start, end), nil
}

// featureSwitch answers whether a protocol feature is on, off the gas meter
// (ProtocolFeatureEnabled does not charge), so gas is the same on both sides.
func (app *QoreChainApp) featureSwitch(name string) func(sdk.Context) bool {
	return func(ctx sdk.Context) bool {
		return app.PQCKeeper != nil && app.PQCKeeper.ProtocolFeatureEnabled(ctx, name)
	}
}

// wasmClientDataDir is where the 08-wasm light client VM keeps its code and
// caches: under the node's data directory, next to the chain data a reset or a
// state-sync restore replaces, and apart from x/wasm's directory. Two wasmvm
// instances on one directory would fight over its lock file.
func wasmClientDataDir(appOpts servertypes.AppOptions) string {
	home := cast.ToString(appOpts.Get(flags.FlagHome))
	if home == "" {
		home = DefaultNodeHome
	}
	return filepath.Join(home, "data")
}

// newWasmClientKeeper builds the 08-wasm keeper on its own VM.
//
// This VM is not behind the CosmWasm code filter (x/wasmfilter, wired into
// x/wasm's engine only), so the filter's guarantee does not cover light
// client code. That is deliberate: the code it runs is stored by governance
// alone and only from FeatureIBCWasmLightClient, so nothing an account
// uploads reaches it. The VM keeps no
// memory cache (ibcwasmtypes.MemoryCacheSize is 0) and runs with the shared
// contract memory limit, so the extra instance costs little when idle. Light
// client code reaches the VM only through MsgStoreCode, which the keeper accepts
// from the governance authority alone, and only once FeatureIBCWasmLightClient
// is on (gatedConfigurator); InitGenesis can carry code only into a new chain.
func (app *QoreChainApp) newWasmClientKeeper(appOpts servertypes.AppOptions, transferStoreKey storetypes.StoreKey, authority string) ibcwasmkeeper.Keeper {
	plugins := ibcwasmkeeper.QueryPlugins{
		Stargate: ibcwasmkeeper.AcceptListStargateQuerier(wasmClientQueryAllowList, app.GRPCQueryRouter()),
	}
	return ibcwasmkeeper.NewKeeperWithConfig(
		app.appCodec,
		hostedStoreService{key: transferStoreKey, prefix: wasmClientHostPrefix},
		app.IBCKeeper.ClientKeeper,
		authority,
		ibcwasmtypes.DefaultWasmConfig(wasmClientDataDir(appOpts)),
		app.GRPCQueryRouter(),
		ibcwasmkeeper.WithQueryPlugins(&plugins),
	)
}

// registerWasmClientSnapshotter adds the 08-wasm extension to state-sync
// snapshots: light client code lives in the VM's directory, not in the
// multistore, so a restored node would otherwise hold every checksum and none
// of the code. Node-local.
func (app *QoreChainApp) registerWasmClientSnapshotter(manager *snapshots.Manager) error {
	if manager == nil {
		return nil
	}
	if err := manager.RegisterExtensions(ibcwasmkeeper.NewWasmSnapshotter(app.CommitMultiStore(), &app.WasmClientKeeper)); err != nil {
		return fmt.Errorf("register 08-wasm snapshot extension: %w", err)
	}
	return nil
}

// packetForwardTimeout is how long a forwarded hop may stay unacknowledged
// before it times out, when the memo does not say. PFM's own default.
const packetForwardTimeout = 10 * time.Minute

// ---------------------------------------------------------------------------
// Switches in the transfer stack
// ---------------------------------------------------------------------------

// gatedIBCModule routes every callback to `with` once its feature is on and to
// `without` before. `without` is the stack exactly as it was without the
// layer, so below the switch a packet takes the v3.2.5 path, byte for byte.
type gatedIBCModule struct {
	on      func(sdk.Context) bool
	with    porttypes.IBCModule
	without porttypes.IBCModule
}

var (
	_ porttypes.IBCModule             = gatedIBCModule{}
	_ porttypes.PacketDataUnmarshaler = gatedIBCModule{}
)

func (g gatedIBCModule) pick(ctx sdk.Context) porttypes.IBCModule {
	if g.on(ctx) {
		return g.with
	}
	return g.without
}

func (g gatedIBCModule) OnChanOpenInit(ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID, channelID string, counterparty channeltypes.Counterparty, version string) (string, error) {
	return g.pick(ctx).OnChanOpenInit(ctx, order, connectionHops, portID, channelID, counterparty, version)
}

func (g gatedIBCModule) OnChanOpenTry(ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID, channelID string, counterparty channeltypes.Counterparty, counterpartyVersion string) (string, error) {
	return g.pick(ctx).OnChanOpenTry(ctx, order, connectionHops, portID, channelID, counterparty, counterpartyVersion)
}

func (g gatedIBCModule) OnChanOpenAck(ctx sdk.Context, portID, channelID, counterpartyChannelID, counterpartyVersion string) error {
	return g.pick(ctx).OnChanOpenAck(ctx, portID, channelID, counterpartyChannelID, counterpartyVersion)
}

func (g gatedIBCModule) OnChanOpenConfirm(ctx sdk.Context, portID, channelID string) error {
	return g.pick(ctx).OnChanOpenConfirm(ctx, portID, channelID)
}

func (g gatedIBCModule) OnChanCloseInit(ctx sdk.Context, portID, channelID string) error {
	return g.pick(ctx).OnChanCloseInit(ctx, portID, channelID)
}

func (g gatedIBCModule) OnChanCloseConfirm(ctx sdk.Context, portID, channelID string) error {
	return g.pick(ctx).OnChanCloseConfirm(ctx, portID, channelID)
}

func (g gatedIBCModule) OnRecvPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, relayer sdk.AccAddress) ibcexported.Acknowledgement {
	return g.pick(ctx).OnRecvPacket(ctx, channelVersion, packet, relayer)
}

func (g gatedIBCModule) OnAcknowledgementPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, acknowledgement []byte, relayer sdk.AccAddress) error {
	return g.pick(ctx).OnAcknowledgementPacket(ctx, channelVersion, packet, acknowledgement, relayer)
}

func (g gatedIBCModule) OnTimeoutPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, relayer sdk.AccAddress) error {
	return g.pick(ctx).OnTimeoutPacket(ctx, channelVersion, packet, relayer)
}

func (g gatedIBCModule) UnmarshalPacketData(ctx sdk.Context, portID, channelID string, bz []byte) (any, string, error) {
	u, ok := g.pick(ctx).(porttypes.PacketDataUnmarshaler)
	if !ok {
		return nil, "", errorsmod.Wrap(sdkerrors.ErrInvalidType, "underlying application cannot unmarshal packet data")
	}
	return u.UnmarshalPacketData(ctx, portID, channelID, bz)
}

// gatedICS4Wrapper is the send side of the same switch: the transfer keeper's
// outgoing packets and the acknowledgements written for received ones go
// through `with` once the feature is on and straight to `without` before.
type gatedICS4Wrapper struct {
	on      func(sdk.Context) bool
	with    porttypes.ICS4Wrapper
	without porttypes.ICS4Wrapper
}

var _ porttypes.ICS4Wrapper = gatedICS4Wrapper{}

func (g gatedICS4Wrapper) pick(ctx sdk.Context) porttypes.ICS4Wrapper {
	if g.on(ctx) {
		return g.with
	}
	return g.without
}

func (g gatedICS4Wrapper) SendPacket(ctx sdk.Context, sourcePort, sourceChannel string, timeoutHeight clienttypes.Height, timeoutTimestamp uint64, data []byte) (uint64, error) {
	return g.pick(ctx).SendPacket(ctx, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
}

func (g gatedICS4Wrapper) WriteAcknowledgement(ctx sdk.Context, packet ibcexported.PacketI, ack ibcexported.Acknowledgement) error {
	return g.pick(ctx).WriteAcknowledgement(ctx, packet, ack)
}

func (g gatedICS4Wrapper) GetAppVersion(ctx sdk.Context, portID, channelID string) (string, bool) {
	return g.pick(ctx).GetAppVersion(ctx, portID, channelID)
}

// gatedIBCModuleV2 is gatedIBCModule for the IBC v2 router.
type gatedIBCModuleV2 struct {
	on      func(sdk.Context) bool
	with    ibcapi.IBCModule
	without ibcapi.IBCModule
}

var (
	_ ibcapi.IBCModule             = gatedIBCModuleV2{}
	_ ibcapi.PacketDataUnmarshaler = gatedIBCModuleV2{}
)

func (g gatedIBCModuleV2) pick(ctx sdk.Context) ibcapi.IBCModule {
	if g.on(ctx) {
		return g.with
	}
	return g.without
}

func (g gatedIBCModuleV2) OnSendPacket(ctx sdk.Context, sourceClient, destinationClient string, sequence uint64, payload channeltypesv2.Payload, signer sdk.AccAddress) error {
	return g.pick(ctx).OnSendPacket(ctx, sourceClient, destinationClient, sequence, payload, signer)
}

func (g gatedIBCModuleV2) OnRecvPacket(ctx sdk.Context, sourceClient, destinationClient string, sequence uint64, payload channeltypesv2.Payload, relayer sdk.AccAddress) channeltypesv2.RecvPacketResult {
	return g.pick(ctx).OnRecvPacket(ctx, sourceClient, destinationClient, sequence, payload, relayer)
}

func (g gatedIBCModuleV2) OnTimeoutPacket(ctx sdk.Context, sourceClient, destinationClient string, sequence uint64, payload channeltypesv2.Payload, relayer sdk.AccAddress) error {
	return g.pick(ctx).OnTimeoutPacket(ctx, sourceClient, destinationClient, sequence, payload, relayer)
}

func (g gatedIBCModuleV2) OnAcknowledgementPacket(ctx sdk.Context, sourceClient, destinationClient string, sequence uint64, acknowledgement []byte, payload channeltypesv2.Payload, relayer sdk.AccAddress) error {
	return g.pick(ctx).OnAcknowledgementPacket(ctx, sourceClient, destinationClient, sequence, acknowledgement, payload, relayer)
}

// UnmarshalPacketData has no context to read the switch from; both sides end
// in the same transfer application, so the answer does not depend on it.
func (g gatedIBCModuleV2) UnmarshalPacketData(payload channeltypesv2.Payload) (any, error) {
	u, ok := g.without.(ibcapi.PacketDataUnmarshaler)
	if !ok {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidType, "underlying application cannot unmarshal packet data")
	}
	return u.UnmarshalPacketData(payload)
}

// ---------------------------------------------------------------------------
// Switches on the modules' messages
// ---------------------------------------------------------------------------

// gatedConfigurator hands a module a message server whose every method refuses
// to run until the feature is on. The refusal sits in the handler itself, so it
// holds for every dispatch path: a transaction, authz MsgExec, x/group and a
// governance proposal. Query services and migrations register unchanged.
type gatedConfigurator struct {
	module.Configurator
	on      func(sdk.Context) bool
	feature string
}

func (c gatedConfigurator) MsgServer() gogogrpc.Server {
	return gatedMsgServer{inner: c.Configurator.MsgServer(), on: c.on, feature: c.feature}
}

type gatedMsgServer struct {
	inner   gogogrpc.Server
	on      func(sdk.Context) bool
	feature string
}

func (s gatedMsgServer) RegisterService(sd *grpc.ServiceDesc, ss any) {
	gated := *sd
	gated.Methods = make([]grpc.MethodDesc, len(sd.Methods))
	for i, m := range sd.Methods {
		handler, method := m.Handler, fmt.Sprintf("/%s/%s", sd.ServiceName, m.MethodName)
		gated.Methods[i] = grpc.MethodDesc{
			MethodName: m.MethodName,
			Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
				// The router also calls the handler at registration, with a nil or
				// plain context, to learn the request type: let that through.
				if ctx == nil {
					return handler(srv, ctx, dec, interceptor)
				}
				if sdkCtx, ok := ctx.Value(sdk.SdkContextKey).(sdk.Context); ok && !s.on(sdkCtx) {
					return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest,
						"%s is not available: protocol feature %q is not enabled", method, s.feature)
				}
				return handler(srv, ctx, dec, interceptor)
			},
		}
	}
	s.inner.RegisterService(&gated, ss)
}

// genesisOrDefault lets a genesis file without the module's section validate:
// every chain that existed before v3.2.6, mainnet and testnet included, has
// such a genesis, and InitChain skips the module for it.
func genesisOrDefault(bz json.RawMessage) bool { return len(bz) == 0 }

// rateLimitAppModule is the rate limiter's module with the v3.2.6 switch on its
// messages and its begin blocker.
type rateLimitAppModule struct {
	ratelimit.AppModule
	keeper ratelimitkeeper.Keeper
	store  corestore.KVStoreService
	on     func(sdk.Context) bool
}

func newRateLimitAppModule(cdc codec.Codec, k ratelimitkeeper.Keeper, store corestore.KVStoreService, on func(sdk.Context) bool) rateLimitAppModule {
	return rateLimitAppModule{AppModule: ratelimit.NewAppModule(cdc, k), keeper: k, store: store, on: on}
}

func (m rateLimitAppModule) RegisterServices(cfg module.Configurator) {
	m.AppModule.RegisterServices(gatedConfigurator{Configurator: cfg, on: m.on, feature: pqctypes.FeatureIBCRateLimit})
}

func (m rateLimitAppModule) ValidateGenesis(cdc codec.JSONCodec, cfg client.TxEncodingConfig, bz json.RawMessage) error {
	if genesisOrDefault(bz) {
		return nil
	}
	return m.AppModule.ValidateGenesis(cdc, cfg, bz)
}

// BeginBlock keeps the hour epoch and resets the quotas whose window ended.
// Below the switch it writes nothing: the module did not exist.
func (m rateLimitAppModule) BeginBlock(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if !m.on(sdkCtx) {
		return nil
	}
	if !m.hasHourEpoch(sdkCtx) {
		// A chain whose genesis predates the module and which did not take it
		// through the v3.2.6 handler: start the epoch here, as InitGenesis would.
		initRateLimitGenesis(sdkCtx, m.keeper, *ratelimittypes.DefaultGenesis())
	}
	return m.AppModule.BeginBlock(ctx)
}

// InitGenesis is the module's own, less the params write: the module keeps its
// (empty) params in a legacy x/params subspace this chain does not have, and
// reads them back as the constant types.NewParams() whatever was written.
func (m rateLimitAppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, bz json.RawMessage) []abci.ValidatorUpdate {
	var gs ratelimittypes.GenesisState
	cdc.MustUnmarshalJSON(bz, &gs)
	initRateLimitGenesis(ctx, m.keeper, gs)
	return []abci.ValidatorUpdate{}
}

func (m rateLimitAppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	if !m.hasHourEpoch(ctx) {
		// State from before the module: nothing has been written.
		return cdc.MustMarshalJSON(ratelimittypes.DefaultGenesis())
	}
	return m.AppModule.ExportGenesis(ctx, cdc)
}

// hasHourEpoch reports whether the module has been initialised: its hour epoch
// is the one record it always writes (the keeper's own read panics without it).
func (m rateLimitAppModule) hasHourEpoch(ctx sdk.Context) bool {
	ok, err := m.store.OpenKVStore(ctx).Has(ratelimittypes.HourEpochKey)
	return err == nil && ok
}

// initRateLimitGenesis mirrors ratelimitkeeper.Keeper.InitGenesis without
// SetParams (see rateLimitAppModule.InitGenesis).
func initRateLimitGenesis(ctx sdk.Context, k ratelimitkeeper.Keeper, gs ratelimittypes.GenesisState) {
	for _, rateLimit := range gs.RateLimits {
		k.SetRateLimit(ctx, rateLimit)
	}
	for _, denom := range gs.BlacklistedDenoms {
		k.AddDenomToBlacklist(ctx, denom)
	}
	for _, pair := range gs.WhitelistedAddressPairs {
		k.SetWhitelistedAddressPair(ctx, pair)
	}
	for _, id := range gs.PendingSendPacketSequenceNumbers {
		channelOrClientID, sequence, denom, err := ratelimittypes.ParsePendingPacketId(id)
		if err != nil {
			if ratelimittypes.IsLegacyPendingPacketId(id) {
				continue
			}
			panic(err.Error())
		}
		if err := k.SetPendingSendPacket(ctx, channelOrClientID, sequence, denom); err != nil {
			panic(err)
		}
	}
	for _, id := range gs.PendingRecvPacketSequenceNumbers {
		channelOrClientID, sequence, denom, err := ratelimittypes.ParsePendingPacketId(id)
		if err != nil {
			if ratelimittypes.IsLegacyPendingPacketId(id) {
				continue
			}
			panic(err.Error())
		}
		if err := k.SetPendingReceivePacket(ctx, channelOrClientID, sequence, denom); err != nil {
			panic(err)
		}
	}
	epoch := gs.HourEpoch
	if epoch.EpochNumber == 0 {
		// Not started yet: number it after the current hour and start it on the
		// hour, as the module's own InitGenesis does.
		epoch.EpochNumber = uint64(ctx.BlockTime().Hour()) //nolint:gosec // 0..23
		epoch.EpochStartTime = ctx.BlockTime().Truncate(time.Hour)
		epoch.EpochStartHeight = ctx.BlockHeight()
	}
	k.SetHourEpoch(ctx, epoch)
}

// packetForwardAppModule is PFM's module, accepting a genesis without its
// section. PFM has no messages and no params.
type packetForwardAppModule struct {
	packetforward.AppModule
}

func (m packetForwardAppModule) ValidateGenesis(cdc codec.JSONCodec, cfg client.TxEncodingConfig, bz json.RawMessage) error {
	if genesisOrDefault(bz) {
		return nil
	}
	return m.AppModule.ValidateGenesis(cdc, cfg, bz)
}

// wasmClientAppModule is the 08-wasm module with the v3.2.6 switch on its
// messages (store, migrate and remove code, all authority-only upstream).
type wasmClientAppModule struct {
	ibcwasm.AppModule
	on func(sdk.Context) bool
}

func (m wasmClientAppModule) RegisterServices(cfg module.Configurator) {
	m.AppModule.RegisterServices(gatedConfigurator{Configurator: cfg, on: m.on, feature: pqctypes.FeatureIBCWasmLightClient})
}

func (m wasmClientAppModule) ValidateGenesis(cdc codec.JSONCodec, cfg client.TxEncodingConfig, bz json.RawMessage) error {
	if genesisOrDefault(bz) {
		return nil
	}
	return m.AppModule.ValidateGenesis(cdc, cfg, bz)
}
