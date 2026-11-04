package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"

	"github.com/cosmos/cosmos-sdk/client"
)

// The IBC v2 client routes answered 501 (no gateway handler mounted). Once
// mounted, a request reaches the query client: here, without a node behind
// it, it fails, but not with the gateway's route-not-found answer.
func TestIBCClientV2GatewayRoutes(t *testing.T) {
	mux := runtime.NewServeMux()
	probe := func(path string) int {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		return rec.Code
	}
	paths := []string{
		"/ibc/core/client/v2/counterparty_info/07-tendermint-0",
		"/ibc/core/client/v2/config/07-tendermint-0",
	}
	for _, p := range paths {
		if code := probe(p); code != http.StatusNotFound && code != http.StatusNotImplemented {
			t.Fatalf("before mounting, GET %s = %d, want no route", p, code)
		}
	}
	registerIBCClientV2Gateway(client.Context{}, mux)
	for _, p := range paths {
		if code := probe(p); code == http.StatusNotFound || code == http.StatusNotImplemented {
			t.Fatalf("GET %s = %d: still no route", p, code)
		}
	}
}
