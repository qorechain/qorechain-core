package app

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Every alias reaches its target route, with the query string kept, and other
// paths pass through unchanged.
func TestModuleRESTAliases(t *testing.T) {
	var seen []string
	h := moduleRESTAlias(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.URL.Path+"?"+r.URL.RawQuery)
	}))
	for alias, target := range moduleRESTAliases {
		seen = nil
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, alias+"?height=5", nil))
		if len(seen) != 1 || seen[0] != target+"?height=5" {
			t.Fatalf("%s reached %v, want %s", alias, seen, target)
		}
	}
	seen = nil
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/qorechain/rlconsensus/v1/agent", nil))
	if len(seen) != 1 || seen[0] != "/qorechain/rlconsensus/v1/agent?" {
		t.Fatalf("unaliased path changed: %v", seen)
	}
}
