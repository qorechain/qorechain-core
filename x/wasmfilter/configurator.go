package wasmfilter

import (
	"fmt"

	gogogrpc "github.com/cosmos/gogoproto/grpc"
	"google.golang.org/grpc"

	"github.com/cosmos/cosmos-sdk/types/module"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
)

// WasmMsgServiceName is the full name of wasmd's Msg service.
const WasmMsgServiceName = "cosmwasm.wasm.v1.Msg"

// Configurator hands the wasm module a message server registrar that wraps
// wasmd's Msg service implementation with DecorateMsgServer. Query services
// and migrations are registered unchanged.
type Configurator struct {
	module.Configurator
	F *Filter
}

func (c Configurator) MsgServer() gogogrpc.Server {
	return registrar{inner: c.Configurator.MsgServer(), f: c.F}
}

type registrar struct {
	inner gogogrpc.Server
	f     *Filter
}

func (r registrar) RegisterService(sd *grpc.ServiceDesc, ss any) {
	if sd.ServiceName == WasmMsgServiceName {
		impl, ok := ss.(wasmtypes.MsgServer)
		if !ok {
			panic(fmt.Sprintf("wasmfilter: %s implementation is %T, not a wasm MsgServer", sd.ServiceName, ss))
		}
		ss = r.f.DecorateMsgServer(impl)
	}
	r.inner.RegisterService(sd, ss)
}
