package cmd

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"text/template"

	"github.com/spf13/viper"

	evmmempool "github.com/cosmos/evm/mempool"

	"github.com/qorechain/qorechain-core/app"
)

// baseapp must hold the evicting wrapper (so CheckTx inserts and block
// removals go through it), while GetMempool must still return the unified
// pool itself: the cosmos/evm start command type-asserts it to
// *ExperimentalEVMMempool to wire the event bus and the JSON-RPC txpool, and a
// failed assertion there stops the node from starting.
func TestMempoolWiring(t *testing.T) {
	a := sharedTestApp(t)
	if got := fmt.Sprintf("%T", a.Mempool()); got != "*app.evictingMempool" {
		t.Fatalf("baseapp mempool is %s, want the evicting wrapper", got)
	}
	m, ok := a.GetMempool().(*evmmempool.ExperimentalEVMMempool)
	if !ok || m == nil || m != a.EVMMempool {
		t.Fatalf("GetMempool is %T, want the app's *ExperimentalEVMMempool", a.GetMempool())
	}
}

// The [qorechain] section of a freshly written app.toml carries the defaults
// and reads back as them.
func TestAppTomlQoreChainSection(t *testing.T) {
	tmplStr, cfg := initAppConfig("")
	tmpl, err := template.New("app").Parse(tmplStr)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, cfg); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{
		"[qorechain]",
		"cosmos-mempool-max-txs = 5000",
		`prepare-proposal-budget = "1s"`,
		"prepare-proposal-max-skips = 64",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("app.toml lacks %q", want)
		}
	}

	v := viper.New()
	v.SetConfigType("toml")
	if err := v.ReadConfig(strings.NewReader(out)); err != nil {
		t.Fatalf("rendered app.toml does not parse: %v", err)
	}
	if got := app.ReadMempoolConfig(v); got != app.DefaultMempoolConfig() {
		t.Fatalf("read back %+v, want %+v", got, app.DefaultMempoolConfig())
	}
}
