package app

import (
	"net/http"
)

// moduleRESTAliases maps REST paths that clients and older docs use to the
// routes the module gateways actually serve. Each target is a query; nothing
// else is rewritten. Node-local: before it each alias answered 501 Not
// Implemented, which is what the gateway returns for an unknown path.
var moduleRESTAliases = map[string]string{
	// The agent status RPC is served at /v1/agent.
	"/qorechain/rlconsensus/v1/agent_status": "/qorechain/rlconsensus/v1/agent",
	"/qorechain/rlconsensus/v1/status":       "/qorechain/rlconsensus/v1/agent",
	// The reward RPC is /v1/reward; some docs listed /v1/rewards.
	"/qorechain/rlconsensus/v1/rewards": "/qorechain/rlconsensus/v1/reward",
	// Gas abstraction has one configuration record; the accepted tokens are
	// part of it.
	"/qorechain/gasabstraction/v1/params":          "/qorechain/gasabstraction/v1/config",
	"/qorechain/gasabstraction/v1/accepted_tokens": "/qorechain/gasabstraction/v1/config",
	// x/ai keeps its parameters in its config record.
	"/qorechain/ai/v1/params": "/qorechain/ai/v1/config",
}

// moduleRESTAlias serves every path in moduleRESTAliases from its target route.
func moduleRESTAlias(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if target, ok := moduleRESTAliases[r.URL.Path]; ok {
			r2 := r.Clone(r.Context())
			r2.URL.Path = target
			r2.URL.RawPath = ""
			next.ServeHTTP(w, r2)
			return
		}
		next.ServeHTTP(w, r)
	})
}
