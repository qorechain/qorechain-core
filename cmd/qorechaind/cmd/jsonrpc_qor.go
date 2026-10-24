package cmd

import (
	"github.com/qorechain/qorechain-core/rpc/qorjsonrpc"
)

// The qor_ JSON-RPC namespace (qor_getKEMKey and any service added to
// rpc/qorjsonrpc) is registered with the EVM JSON-RPC server before it
// starts. It is served only where app.toml [json-rpc] api lists "qor".
func init() {
	if err := qorjsonrpc.Register(); err != nil {
		panic(err)
	}
}
