package types

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// Every google.api.http route in the proto must exist in the generated
// gateway, or the REST endpoint answers 501 while gRPC serves it. The EVM
// authorisation window shipped that way: the query was in the proto and in
// the generated client, but the gateway file had never been regenerated, so
// no wallet could read a window back (reported by the QoreX work, and the
// same shape as QSR-2026-0236).
func TestEveryProtoRouteIsInTheGateway(t *testing.T) {
	proto, err := os.ReadFile("../../../proto/qorechain/pqc/v1/query.proto")
	if err != nil {
		t.Skipf("proto not readable here: %v", err)
	}
	gw, err := os.ReadFile("query.pb.gw.go")
	if err != nil {
		t.Fatal(err)
	}
	routes := regexp.MustCompile(`\(google\.api\.http\)\.get\s*=\s*"([^"]+)"`).FindAllStringSubmatch(string(proto), -1)
	if len(routes) == 0 {
		t.Fatal("no http routes found in the proto")
	}
	for _, m := range routes {
		// The gateway spells a route as its path segments; compare on the
		// distinctive one rather than on the whole string.
		seg := ""
		for _, s := range strings.Split(m[1], "/") {
			if s != "" && !strings.HasPrefix(s, "{") && s != "qorechain" && s != "pqc" && s != "v1" {
				seg = s
			}
		}
		if seg == "" {
			continue
		}
		if !strings.Contains(string(gw), `"`+seg+`"`) {
			t.Errorf("route %s is in the proto but not in the generated gateway", m[1])
		}
	}
}
