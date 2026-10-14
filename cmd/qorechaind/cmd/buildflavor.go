package cmd

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/server"

	"github.com/qorechain/qorechain-core/app"
)

// wireBuildFlavorGuard adds --community-unenforced-pqc to the start command
// and runs app.CheckBuildFlavor before anything else in its PreRunE, so a
// community build refuses to start on a public network unless the operator
// opts in. Every community-build start logs a warning.
func wireBuildFlavorGuard(rootCmd *cobra.Command) {
	for _, c := range rootCmd.Commands() {
		if c.Name() != "start" {
			continue
		}
		c.Flags().Bool(app.FlagCommunityUnenforcedPQC, false,
			"Start a community build (no PQC verification) on a public QoreChain network anyway")
		prev := c.PreRunE
		c.PreRunE = func(cmd *cobra.Command, args []string) error {
			override, _ := cmd.Flags().GetBool(app.FlagCommunityUnenforcedPQC)
			serverCtx := server.GetServerContextFromCmd(cmd)
			chainID := app.NodeChainID(serverCtx.Viper)
			if !app.FullBuild {
				serverCtx.Logger.Warn("community build: PQC signatures are not verified and the "+
					"full overlay is absent; use it for local devnets only",
					"chain_id", chainID, "override", override)
			}
			if err := app.CheckBuildFlavor(chainID, app.NodeGenesisTime(serverCtx.Viper), override); err != nil {
				return err
			}
			if prev != nil {
				return prev(cmd, args)
			}
			return nil
		}
	}
}
