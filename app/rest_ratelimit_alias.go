package app

import (
	"net/http"
	"strings"
)

const (
	// rateLimitIBCPrefix is the IBC-style REST prefix clients expect for the
	// IBC rate-limit module (as /ibc/apps/transfer/v1/... for transfer).
	rateLimitIBCPrefix = "/ibc/apps/ratelimit/v1/"
	// rateLimitGatewayPrefix is where the module's generated gateway serves
	// the same queries (ibc-apps rate-limiting v10 query.proto).
	rateLimitGatewayPrefix = "/Stride-Labs/ibc-rate-limiting/ratelimit/"
)

// rateLimitRESTAlias serves the IBC rate-limit queries under the IBC-style
// prefix as well. The module registers its REST routes only under
// /Stride-Labs/ibc-rate-limiting/ratelimit/..., so
// /ibc/apps/ratelimit/v1/ratelimits answered 501 Not Implemented. The alias
// rewrites the path and hands the request to the same gateway; queries only.
func rateLimitRESTAlias(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rest, ok := strings.CutPrefix(r.URL.Path, rateLimitIBCPrefix); ok {
			r2 := r.Clone(r.Context())
			r2.URL.Path = rateLimitGatewayPrefix + rest
			r2.URL.RawPath = ""
			next.ServeHTTP(w, r2)
			return
		}
		next.ServeHTTP(w, r)
	})
}
