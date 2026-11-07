package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"

	ratelimittypes "github.com/cosmos/ibc-apps/modules/rate-limiting/v10/types"
)

type fakeRateLimitQuery struct {
	ratelimittypes.QueryClient
	allCalls int
}

func (f *fakeRateLimitQuery) AllRateLimits(context.Context, *ratelimittypes.QueryAllRateLimitsRequest, ...grpc.CallOption) (*ratelimittypes.QueryAllRateLimitsResponse, error) {
	f.allCalls++
	return &ratelimittypes.QueryAllRateLimitsResponse{}, nil
}

// The rate-limit module serves its REST routes under
// /Stride-Labs/ibc-rate-limiting/ratelimit/..., so the IBC-style path
// /ibc/apps/ratelimit/v1/ratelimits answered 501 Not Implemented on both
// networks. The IBC-style paths now reach the same handlers.
func TestRateLimitRESTAlias(t *testing.T) {
	q := &fakeRateLimitQuery{}
	mux := runtime.NewServeMux()
	if err := ratelimittypes.RegisterQueryHandlerClient(context.Background(), mux, q); err != nil {
		t.Fatal(err)
	}
	h := rateLimitRESTAlias(mux)

	for _, path := range []string{
		"/ibc/apps/ratelimit/v1/ratelimits",
		"/Stride-Labs/ibc-rate-limiting/ratelimit/ratelimits",
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK || !strings.HasPrefix(rec.Body.String(), "{") {
			t.Fatalf("%s: %d %s", path, rec.Code, rec.Body.String())
		}
	}
	if q.allCalls != 2 {
		t.Fatalf("AllRateLimits called %d times, want 2", q.allCalls)
	}

	// Other paths are untouched.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ibc/apps/other/v1/x", nil))
	if rec.Code != http.StatusNotImplemented && rec.Code != http.StatusNotFound {
		t.Fatalf("unrelated path: %d", rec.Code)
	}
}
