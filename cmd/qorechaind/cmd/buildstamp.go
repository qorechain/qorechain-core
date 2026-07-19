package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// QoreBuildID is a PRIVATE build stamp, set via ldflags at release time:
//
//	-X github.com/qorechain/qorechain-core/cmd/qorechaind/cmd.QoreBuildID=v3.1.87
//
// It is deliberately NOT the Cosmos SDK version.Version — that stays empty so
// the build is not fingerprintable through `qorechaind version`, `--help`, or
// the p2p/node_info application_version. The real version is visible ONLY via
// the hidden `aratavers` command below, whose name is not advertised.
var (
	QoreBuildID  = ""
	QoreCommit   = ""
	QoreDeployAt = "" // release/deploy timestamp, stamped at build time
)

// qoreBuildInfoCmd is the hidden command that reveals the private build stamp.
// Hidden: true keeps it out of `--help`; only someone who knows the name can run it.
func qoreBuildInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "aratavers",
		Short:  "Print the private QoreChain build identifier",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			id := QoreBuildID
			if id == "" {
				id = "unstamped (dev build)"
			}
			fmt.Fprintf(out, "qorechaind build: %s\n", id)
			if QoreCommit != "" {
				fmt.Fprintf(out, "commit:           %s\n", QoreCommit)
			}
			if QoreDeployAt != "" {
				fmt.Fprintf(out, "built:            %s\n", QoreDeployAt)
			}
			fmt.Fprintf(out, "go:               %s\n", runtime.Version())
			return nil
		},
	}
}

// hideVersionCommand neuters the standard `version` command: it stays present
// (so `qorechaind version` does not error) but prints nothing and is removed
// from help output. Fingerprinting the running build via the CLI is denied;
// operators who need the version use `aratavers`.
func hideVersionCommand(rootCmd *cobra.Command) {
	for _, c := range rootCmd.Commands() {
		if c.Name() == "version" {
			c.Hidden = true
			c.Run = nil
			c.RunE = func(_ *cobra.Command, _ []string) error { return nil }
			break
		}
	}
}
