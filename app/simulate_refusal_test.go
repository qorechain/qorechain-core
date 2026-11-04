package app

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	gateway "github.com/cosmos/gogogateway"
	"github.com/cosmos/gogoproto/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/cosmos/cosmos-sdk/client"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"

	stakingtiertypes "github.com/qorechain/qorechain-core/x/stakingtier/types"
)

func refusalServer(simErr error) simulateRefusalServer {
	reg := codectypes.NewInterfaceRegistry()
	sim := func([]byte) (sdk.GasInfo, *sdk.Result, error) {
		if simErr != nil {
			return sdk.GasInfo{GasWanted: 200_000, GasUsed: 51_234}, nil, simErr
		}
		return sdk.GasInfo{GasWanted: 200_000, GasUsed: 51_234}, &sdk.Result{}, nil
	}
	return simulateRefusalServer{
		ServiceServer: authtx.NewTxServer(client.Context{}, sim, reg),
		clientCtx:     client.Context{}, simulate: sim, registry: reg,
	}
}

// A ClaimBonus with nothing accrued is refused in simulation with
// stakingtier 12; the refusal used to reach the client as text only. It keeps
// its gRPC code and text and carries codespace and code in a TxResponse detail.
func TestSimulateRefusalKeepsCodespaceAndCode(t *testing.T) {
	cause := stakingtiertypes.ErrNothingToClaim.Wrap("failed to execute message; message index: 0")
	_, err := refusalServer(cause).Simulate(context.Background(), &txtypes.SimulateRequest{TxBytes: []byte{1}})
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("not a status: %v", err)
	}
	if st.Code() != codes.Unknown || !strings.Contains(st.Message(), "no bonus to claim") ||
		!strings.Contains(st.Message(), "with gas used: '51234'") {
		t.Fatalf("status changed: %v %q", st.Code(), st.Message())
	}
	p := st.Proto()
	if len(p.Details) != 1 || p.Details[0].TypeUrl != "/cosmos.base.abci.v1beta1.TxResponse" {
		t.Fatalf("details: %v", p.Details)
	}
	var res sdk.TxResponse
	if err := proto.Unmarshal(p.Details[0].Value, &res); err != nil {
		t.Fatal(err)
	}
	if res.Codespace != stakingtiertypes.ModuleName || res.Code != 12 || res.GasUsed != 51_234 ||
		res.GasWanted != 200_000 || !strings.Contains(res.RawLog, "no bonus to claim") {
		t.Fatalf("detail: %+v", res)
	}

	// The REST error body renders the detail as JSON a client can read.
	bz, err := (&gateway.JSONPb{EmitDefaults: true, OrigName: true, AnyResolver: codectypes.NewInterfaceRegistry()}).Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var body struct {
		Code    int `json:"code"`
		Details []struct {
			Type      string `json:"@type"`
			Codespace string `json:"codespace"`
			Code      uint32 `json:"code"`
		} `json:"details"`
	}
	if err := json.Unmarshal(bz, &body); err != nil {
		t.Fatalf("%v: %s", err, bz)
	}
	if body.Code != int(codes.Unknown) || len(body.Details) != 1 || body.Details[0].Codespace != "stakingtier" || body.Details[0].Code != 12 {
		t.Fatalf("REST body: %s", bz)
	}
}

// A successful simulation and a request refused before simulating (no tx
// bytes) are answered exactly as by the stock service.
func TestSimulateRefusalLeavesTheRestAlone(t *testing.T) {
	res, err := refusalServer(nil).Simulate(context.Background(), &txtypes.SimulateRequest{TxBytes: []byte{1}})
	if err != nil || res.GasInfo.GasUsed != 51_234 {
		t.Fatalf("success: %v %+v", err, res)
	}
	_, err = refusalServer(errors.New("unused")).Simulate(context.Background(), &txtypes.SimulateRequest{})
	st, _ := status.FromError(err)
	if st.Code() != codes.InvalidArgument || len(st.Proto().Details) != 0 {
		t.Fatalf("argument error changed: %v", err)
	}
	// An unregistered error is reported as the delivered result would: the
	// undefined codespace, code 1.
	_, err = refusalServer(errors.New("boom")).Simulate(context.Background(), &txtypes.SimulateRequest{TxBytes: []byte{1}})
	st, _ = status.FromError(err)
	var r sdk.TxResponse
	if d := st.Proto().Details; len(d) != 1 || proto.Unmarshal(d[0].Value, &r) != nil || r.Code != 1 {
		t.Fatalf("unregistered error: %v %+v", err, r)
	}
	// No gas limit in simulation: gas wanted is the maximum uint64, reported as 0.
	if got := asInt64(^uint64(0)); got != 0 {
		t.Fatalf("unbounded gas wanted reported as %d", got)
	}
}
