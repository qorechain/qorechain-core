package types

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
)

// recordingClient answers every query with an empty response and records
// what the gateway asked for.
type recordingClient struct {
	QueryClient
	apr       *QueryAprRequest
	delegator *QueryDelegatorAprRequest
}

func (c *recordingClient) Apr(_ context.Context, in *QueryAprRequest, _ ...grpc.CallOption) (*QueryAprResponse, error) {
	c.apr = in
	return &QueryAprResponse{}, nil
}

func (c *recordingClient) DelegatorApr(_ context.Context, in *QueryDelegatorAprRequest, _ ...grpc.CallOption) (*QueryDelegatorAprResponse, error) {
	c.delegator = in
	return &QueryDelegatorAprResponse{}, nil
}

// Every APR REST path reaches its query with its parameters; none answers
// 501 (the reputation module's paths once did, for want of a gateway).
func TestAprGatewayRoutes(t *testing.T) {
	c := &recordingClient{}
	mux := runtime.NewServeMux()
	if err := RegisterQueryHandlerClient(context.Background(), mux, c); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		"/qorechain/stakingtier/v1/apr",
		"/qorechain/stakingtier/v1/apr?validator=qorvaloper1v",
		"/qorechain/stakingtier/v1/apr/qor1d",
		"/qorechain/stakingtier/v1/apr/qor1d?validator=qorvaloper1v",
	} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: HTTP %d %s", path, rec.Code, rec.Body.String())
		}
	}
	if c.apr == nil || c.apr.Validator != "qorvaloper1v" {
		t.Fatalf("apr request %+v", c.apr)
	}
	if c.delegator == nil || c.delegator.Delegator != "qor1d" || c.delegator.Validator != "qorvaloper1v" {
		t.Fatalf("delegator apr request %+v", c.delegator)
	}
}
