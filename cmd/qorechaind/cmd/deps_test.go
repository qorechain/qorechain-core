package cmd

import (
	"strconv"
	"strings"
	"testing"

	"github.com/cometbft/cometbft/blocksync"
	"github.com/cometbft/cometbft/p2p"
	cmtversion "github.com/cometbft/cometbft/version"
	"google.golang.org/grpc"
)

// The consensus engine added in v0.38.24 the pre-unmarshal p2p message filter
// the blocksync reactor uses to refuse unsolicited BlockResponses and
// oversized commits (QSR-2026-0060). Without it this does not compile.
var _ p2p.MsgBytesFilter = (*blocksync.Reactor)(nil)

// Minimum dependency versions carrying security fixes, checked against the
// version constants compiled into each module (test binaries carry no module
// build info). A go.mod edit that drops below one of them fails here instead
// of shipping silently. Engine v0.38.26 is deliberately not required: it
// changes consensus params validation and signature size limits and can only
// arrive with a coordinated upgrade.
func TestSecurityDependencyVersionsPinned(t *testing.T) {
	for _, dep := range []struct{ name, got, min string }{
		{"consensus engine", cmtversion.TMCoreSemVer, "v0.38.25"}, // QSR-2026-0060
		{"grpc-go", grpc.Version, "v1.83.1"},                      // QSR-2026-0319, GHSA-vp52-pcj8-j9qc
	} {
		if semverLess(dep.got, dep.min) {
			t.Errorf("%s is %s, below %s which carries the security fix", dep.name, dep.got, dep.min)
		}
	}
}

// semverLess compares the numeric major.minor.patch of two vX.Y.Z versions;
// pre-release and build suffixes are ignored.
func semverLess(a, b string) bool {
	pa, pb := semverParts(a), semverParts(b)
	for i := range pa {
		if pa[i] != pb[i] {
			return pa[i] < pb[i]
		}
	}
	return false
}

func semverParts(v string) [3]int {
	var out [3]int
	v = strings.TrimPrefix(v, "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	for i, p := range strings.SplitN(v, ".", 3) {
		out[i], _ = strconv.Atoi(p)
	}
	return out
}

func TestSemverLess(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		less bool
	}{
		{"v0.38.21", "v0.38.25", true},
		{"v0.38.25", "v0.38.25", false},
		{"v0.38.26", "v0.38.25", false},
		{"v1.79.3", "v1.83.1", true},
		{"v1.83.2", "v1.83.1", false},
		{"v0.39.0-rc1", "v0.38.25", false},
	} {
		if got := semverLess(tc.a, tc.b); got != tc.less {
			t.Errorf("semverLess(%s, %s) = %v, want %v", tc.a, tc.b, got, tc.less)
		}
	}
}
