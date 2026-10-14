package cmd

import (
	"bytes"
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/require"
)

// A freshly written app.toml closes the node keyring to JSON-RPC callers
// (QSR-2026-0486/0562), leaves the txpool namespace off (QSR-2026-0079) and
// bounds query gas (QSR-2026-0419).
func TestAppTomlRPCHardeningDefaults(t *testing.T) {
	tmplStr, cfg := initAppConfig("")
	qc := cfg.(QoreChainConfig)
	require.False(t, qc.JSONRPC.AllowInsecureUnlock)
	require.Equal(t, []string{"eth", "net", "web3"}, qc.JSONRPC.API)
	require.Equal(t, DefaultQueryGasLimit, qc.Config.QueryGasLimit)
	require.NotZero(t, qc.Config.QueryGasLimit)

	tmpl, err := template.New("app").Parse(tmplStr)
	require.NoError(t, err)
	var buf bytes.Buffer
	require.NoError(t, tmpl.Execute(&buf, cfg))
	out := buf.String()
	for _, want := range []string{
		"allow-insecure-unlock = false",
		`api = "eth,net,web3"`,
		`query-gas-limit = "50000000"`,
	} {
		require.True(t, strings.Contains(out, want), "app.toml lacks %q", want)
	}
}
