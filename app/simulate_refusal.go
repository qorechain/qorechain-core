package app

import (
	"context"
	"math"

	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/gogoproto/proto"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/cosmos/cosmos-sdk/client"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
)

// simulateRefusalTypeURL names the detail a refused simulation carries.
var simulateRefusalTypeURL = "/" + proto.MessageName(&sdk.TxResponse{})

// simulateRefusalServer is the SDK tx service with one change to Simulate: a
// refused simulation keeps the codespace and code of the error that refused
// it. The stock service turns every refusal into a gRPC status whose only
// content is the error text ("... no bonus to claim with gas used: '...'"),
// so a client could only guess the registered error from its description,
// while the same transaction delivered in a block reports codespace and code.
//
// The status keeps its gRPC code and message unchanged, so a client reading
// the text sees what it saw before, and gains one detail: a
// cosmos.base.abci.v1beta1.TxResponse with codespace, code, raw_log,
// gas_wanted and gas_used, the fields a delivered result has. Over REST it is
// details[0] of the error body. Node-local: a query, no consensus change.
type simulateRefusalServer struct {
	txtypes.ServiceServer
	clientCtx client.Context
	simulate  func(txBytes []byte) (sdk.GasInfo, *sdk.Result, error)
	registry  codectypes.InterfaceRegistry
}

func (s simulateRefusalServer) Simulate(ctx context.Context, req *txtypes.SimulateRequest) (*txtypes.SimulateResponse, error) {
	// A server per request keeps the stock request handling (TxBytes or the
	// legacy Tx field, the argument errors) and hands back what the
	// simulation itself returned.
	var simErr error
	var gas sdk.GasInfo
	inner := authtx.NewTxServer(s.clientCtx, func(txBytes []byte) (sdk.GasInfo, *sdk.Result, error) {
		g, res, err := s.simulate(txBytes)
		simErr, gas = err, g
		return g, res, err
	}, s.registry)
	res, err := inner.Simulate(ctx, req)
	if err == nil || simErr == nil {
		return res, err
	}
	return nil, withRefusalDetail(err, simErr, gas)
}

// withRefusalDetail adds to the status err the codespace and code of cause,
// the error the simulation returned. err is returned unchanged when it is no
// status or the detail cannot be encoded.
func withRefusalDetail(err, cause error, gas sdk.GasInfo) error {
	st, ok := status.FromError(err)
	if !ok {
		return err
	}
	codespace, code, log := errorsmod.ABCIInfo(cause, false)
	bz, merr := proto.Marshal(&sdk.TxResponse{
		Codespace: codespace,
		Code:      code,
		RawLog:    log,
		// A simulation runs without a gas limit (gas wanted is the maximum
		// uint64), which is reported as 0 rather than wrapped negative.
		GasWanted: asInt64(gas.GasWanted),
		GasUsed:   asInt64(gas.GasUsed),
	})
	if merr != nil {
		return err
	}
	p := st.Proto()
	return status.FromProto(&spb.Status{
		Code:    p.Code,
		Message: p.Message,
		Details: append(p.Details, &anypb.Any{TypeUrl: simulateRefusalTypeURL, Value: bz}),
	}).Err()
}

func asInt64(v uint64) int64 {
	if v > math.MaxInt64 {
		return 0
	}
	return int64(v)
}
