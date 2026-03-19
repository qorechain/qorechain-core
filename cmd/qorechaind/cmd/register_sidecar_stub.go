//go:build !full

package cmd

import "github.com/spf13/cobra"

func init() {
	registerSidecarCmd = func(_ *cobra.Command) {}
}
