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

// fakeClusterClient answers ClusterInfo only; any other call panics through the
// nil embedded interface, which would fail the test.
type fakeClusterClient struct {
	QueryClient
	calls int
}

func (f *fakeClusterClient) ClusterInfo(_ context.Context, _ *QueryClusterInfoRequest, _ ...grpc.CallOption) (*QueryClusterInfoResponse, error) {
	f.calls++
	return &QueryClusterInfoResponse{
		ChainId:      "qorechain-vladi",
		GenesisHash:  pinnedMainnetGenesisHash,
		Caip2ChainId: pinnedMainnetCAIP2,
	}, nil
}

// Explorers and docs read the identity over REST at a fixed path.
func TestClusterInfoRESTRoute(t *testing.T) {
	mux := runtime.NewServeMux()
	fc := &fakeClusterClient{}
	if err := RegisterQueryHandlerClient(context.Background(), mux, fc); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/qorechain/svm/v1/cluster_info", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	if fc.calls != 1 {
		t.Fatalf("ClusterInfo called %d times", fc.calls)
	}
	body := rec.Body.String()
	if !strings.Contains(body, pinnedMainnetGenesisHash) || !strings.Contains(body, pinnedMainnetCAIP2) {
		t.Fatalf("REST body missing identity: %s", body)
	}
}
