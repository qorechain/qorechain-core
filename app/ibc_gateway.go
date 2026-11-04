package app

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"

	"github.com/cosmos/cosmos-sdk/client"

	clientv2types "github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
)

// registerIBCClientV2Gateway mounts the REST routes of the IBC v2 client query
// service, /ibc/core/client/v2/counterparty_info/{client_id} and
// /ibc/core/client/v2/config/{client_id}. The ibc core module serves the
// service over gRPC but its RegisterGRPCGatewayRoutes (ibc-go v10.7) mounts
// the client v1, connection, channel v1 and channel v2 gateways only, so these
// two answered 501 over REST, and a relayer setting up an IBC v2 path reads
// the counterparty from there. Node-local: a query route, no consensus change.
func registerIBCClientV2Gateway(clientCtx client.Context, mux *runtime.ServeMux) {
	if err := clientv2types.RegisterQueryHandlerClient(context.Background(), mux, clientv2types.NewQueryClient(clientCtx)); err != nil {
		panic(err)
	}
}
