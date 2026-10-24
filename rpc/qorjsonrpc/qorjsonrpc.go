// Package qorjsonrpc serves the qor_ namespace on the EVM JSON-RPC server.
//
// The namespace is node-local and read-only: every method answers from a
// gRPC query through the server's client context, the same path the REST
// gateway uses, so nothing here can change consensus. It is served when the
// operator lists "qor" in app.toml [json-rpc] api.
//
// cosmos-evm accepts one creator per namespace, so the methods of every
// module are collected here and registered once; a module adds its service
// with AddService before the server starts.
package qorjsonrpc

import (
	"sync"

	"github.com/ethereum/go-ethereum/rpc"

	evmmempool "github.com/cosmos/evm/mempool"
	evmrpc "github.com/cosmos/evm/rpc"
	"github.com/cosmos/evm/rpc/stream"
	servertypes "github.com/cosmos/evm/server/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/server"
)

// Namespace is the JSON-RPC namespace this package serves.
const Namespace = "qor"

const apiVersion = "1.0"

// ServiceFactory builds one service of the namespace. Its exported methods
// become qor_<method>, with the first letter lower-cased.
type ServiceFactory func(clientCtx client.Context) interface{}

var (
	mu         sync.Mutex
	factories  []ServiceFactory
	registered bool
)

// AddService adds a service to the namespace. Call it before Register.
func AddService(f ServiceFactory) {
	mu.Lock()
	defer mu.Unlock()
	factories = append(factories, f)
}

// Register registers the namespace with the EVM JSON-RPC server. A second
// call is a no-op.
func Register() error {
	mu.Lock()
	defer mu.Unlock()
	if registered {
		return nil
	}
	if err := evmrpc.RegisterAPINamespace(Namespace, createAPIs); err != nil {
		return err
	}
	registered = true
	return nil
}

func createAPIs(
	_ *server.Context,
	clientCtx client.Context,
	_ *stream.RPCStream,
	_ bool,
	_ servertypes.EVMTxIndexer,
	_ *evmmempool.ExperimentalEVMMempool,
) []rpc.API {
	mu.Lock()
	defer mu.Unlock()
	apis := make([]rpc.API, 0, len(factories))
	for _, f := range factories {
		apis = append(apis, rpc.API{
			Namespace: Namespace,
			Version:   apiVersion,
			Service:   f(clientCtx),
			Public:    true,
		})
	}
	return apis
}
