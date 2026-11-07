package cmd

import (
	"bytes"
	"strings"
	"testing"
	"text/template"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/CosmWasm/wasmd/x/wasm"
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

// A freshly written app.toml bounds tx simulation: without
// [wasm] simulation_gas_limit, and with block max_gas -1, simulate ran on an
// infinite gas meter. The section is read back by wasmd's ReadNodeConfig.
func TestAppTomlWasmSimulationGasLimit(t *testing.T) {
	tmplStr, cfg := initAppConfig("")
	qc := cfg.(QoreChainConfig)
	require.Equal(t, DefaultWasmSimulationGasLimit, qc.Wasm.SimulationGasLimit)

	tmpl, err := template.New("app").Parse(tmplStr)
	require.NoError(t, err)
	var buf bytes.Buffer
	require.NoError(t, tmpl.Execute(&buf, cfg))
	out := buf.String()
	for _, want := range []string{
		"[wasm]",
		"simulation_gas_limit = 50000000",
		"query_gas_limit = 3000000",
		"memory_cache_size = 100",
	} {
		require.True(t, strings.Contains(out, want), "app.toml lacks %q", want)
	}

	v := viper.New()
	v.SetConfigType("toml")
	require.NoError(t, v.ReadConfig(strings.NewReader(out)))
	nodeCfg, err := wasm.ReadNodeConfig(v)
	require.NoError(t, err)
	require.NotNil(t, nodeCfg.SimulationGasLimit)
	require.Equal(t, DefaultWasmSimulationGasLimit, *nodeCfg.SimulationGasLimit)
	require.Equal(t, uint64(3_000_000), nodeCfg.SmartQueryGasLimit)
	require.Equal(t, uint32(100), nodeCfg.MemoryCacheSize)
}
