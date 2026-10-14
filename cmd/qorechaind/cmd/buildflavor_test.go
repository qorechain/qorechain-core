package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"

	"github.com/qorechain/qorechain-core/app"
)

// QSR-2026-0044 / 0074: the start command of a community build refuses a
// public network's home unless --community-unenforced-pqc is passed. The full
// build always passes the guard.
func TestStartRefusesCommunityBuildOnPublicNetwork(t *testing.T) {
	home := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(home, "config"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(home, "config", "genesis.json"),
		[]byte(`{"chain_id":"`+app.CosmosChainIDMainnet+`"}`), 0o600))

	root := NewRootCmd()
	start, _, err := root.Find([]string{"start"})
	require.NoError(t, err)
	require.NotNil(t, start.Flags().Lookup(app.FlagCommunityUnenforcedPQC))

	srvCtx := server.NewDefaultContext()
	srvCtx.Viper.Set(flags.FlagHome, home)
	start.SetContext(context.WithValue(context.Background(), server.ServerContextKey, srvCtx))

	err = start.PreRunE(start, nil)
	if app.FullBuild {
		require.NoError(t, err)
	} else {
		require.ErrorContains(t, err, "community build")
	}

	require.NoError(t, start.Flags().Set(app.FlagCommunityUnenforcedPQC, "true"))
	require.NoError(t, start.PreRunE(start, nil))
}
