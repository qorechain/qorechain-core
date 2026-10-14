package app

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckBuildFlavor(t *testing.T) {
	for _, chainID := range []string{CosmosChainIDMainnet, CosmosChainIDTestnet} {
		err := CheckBuildFlavor(chainID, false)
		if FullBuild {
			require.NoError(t, err, chainID)
		} else {
			require.ErrorContains(t, err, "community build", chainID)
			require.ErrorContains(t, err, FlagCommunityUnenforcedPQC, chainID)
		}
		// The explicit opt-in always starts.
		require.NoError(t, CheckBuildFlavor(chainID, true), chainID)
	}
	// Devnets and an unknown chain-id are never refused.
	require.NoError(t, CheckBuildFlavor("qorechain-devnet", false))
	require.NoError(t, CheckBuildFlavor("", false))
}

func TestNodeChainIDReadsFlagThenGenesis(t *testing.T) {
	home := homeWithGenesis(t, CosmosChainIDMainnet)
	require.Equal(t, CosmosChainIDMainnet, NodeChainID(mockAppOpts{"home": home}))
	require.Equal(t, "other-1", NodeChainID(mockAppOpts{"home": home, "chain-id": "other-1"}))
	require.Equal(t, "", NodeChainID(mockAppOpts{}))
}
