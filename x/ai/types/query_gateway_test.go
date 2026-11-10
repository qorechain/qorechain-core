package types

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
)

type fakeAIQuery struct {
	QueryClient
	calls map[string]int
}

func (f *fakeAIQuery) Config(context.Context, *QueryConfigRequest, ...grpc.CallOption) (*QueryConfigResponse, error) {
	f.calls["config"]++
	return &QueryConfigResponse{Config: &ConfigView{AnomalyAction: "flag"}}, nil
}

func (f *fakeAIQuery) Stats(context.Context, *QueryStatsRequest, ...grpc.CallOption) (*QueryStatsResponse, error) {
	f.calls["stats"]++
	return &QueryStatsResponse{Stats: &StatsView{TxsRouted: 3}}, nil
}

func (f *fakeAIQuery) FlaggedTx(_ context.Context, r *QueryFlaggedTxRequest, _ ...grpc.CallOption) (*QueryFlaggedTxResponse, error) {
	f.calls["flagged:"+r.TxHash]++
	return &QueryFlaggedTxResponse{}, nil
}

// The x/ai queries are served over REST. Before the HTTP annotations every
// /qorechain/ai/v1/... path answered 501 Not Implemented.
func TestAIQueryGatewayRoutes(t *testing.T) {
	q := &fakeAIQuery{calls: map[string]int{}}
	mux := runtime.NewServeMux()
	if err := RegisterQueryHandlerClient(context.Background(), mux, q); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/qorechain/ai/v1/config", "/qorechain/ai/v1/stats", "/qorechain/ai/v1/flagged/ABCD"} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK || !strings.HasPrefix(rec.Body.String(), "{") {
			t.Fatalf("%s: %d %s", path, rec.Code, rec.Body.String())
		}
	}
	if q.calls["config"] != 1 || q.calls["stats"] != 1 || q.calls["flagged:ABCD"] != 1 {
		t.Fatalf("calls %v", q.calls)
	}
}
