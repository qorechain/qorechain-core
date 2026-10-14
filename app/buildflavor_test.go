package app

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckBuildFlavor(t *testing.T) {
	for _, chainID := range []string{CosmosChainIDMainnet, CosmosChainIDTestnet} {
		err := CheckBuildFlavor(chainID, "", false)
		if FullBuild {
			require.NoError(t, err, chainID)
		} else {
			require.ErrorContains(t, err, "community build", chainID)
			require.ErrorContains(t, err, FlagCommunityUnenforcedPQC, chainID)
		}
		// The explicit opt-in always starts.
		require.NoError(t, CheckBuildFlavor(chainID, "", true), chainID)
	}
	// Devnets and an unknown chain-id are never refused.
	require.NoError(t, CheckBuildFlavor("qorechain-devnet", "", false))
	require.NoError(t, CheckBuildFlavor("", "", false))
}

func TestNodeChainIDReadsFlagThenGenesis(t *testing.T) {
	home := homeWithGenesis(t, CosmosChainIDMainnet)
	require.Equal(t, CosmosChainIDMainnet, NodeChainID(mockAppOpts{"home": home}))
	require.Equal(t, "other-1", NodeChainID(mockAppOpts{"home": home, "chain-id": "other-1"}))
	require.Equal(t, "", NodeChainID(mockAppOpts{}))
}

// A local devnet that reuses the public chain-id but has its own genesis must
// start (the community docker-compose quick start).
func TestLocalDevnetReusingPublicChainIDStarts(t *testing.T) {
	require.False(t, IsPublicNetwork("qorechain-diana", "2026-10-15T08:00:00Z"))
	require.True(t, IsPublicNetwork("qorechain-diana", "2026-07-01T20:28:30.435519983Z"))
	require.True(t, IsPublicNetwork("qorechain-vladi", "2026-06-29T17:03:09.395855717Z"))
	require.True(t, IsPublicNetwork("qorechain-vladi", ""))
	require.False(t, IsPublicNetwork("qorechain-devnet", ""))
	require.NoError(t, CheckBuildFlavor("qorechain-diana", "2026-10-15T08:00:00Z", false))
}
