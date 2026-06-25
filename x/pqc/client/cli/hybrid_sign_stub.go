//go:build !full

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// pqcSignerCommands in the community (non-full) build returns placeholders that
// explain the FFI-backed signer is only available in the full (validator) build.
func pqcSignerCommands() []*cobra.Command {
	mk := func(use, short string) *cobra.Command {
		return &cobra.Command{
			Use:   use,
			Short: short + " (requires the full build)",
			RunE: func(*cobra.Command, []string) error {
				return fmt.Errorf("%q requires the full (validator) build with the PQC FFI library; "+
					"the community build cannot produce Dilithium signatures", use)
			},
		}
	}
	return []*cobra.Command{
		mk("gen-key [name]", "Generate and store a Dilithium-5 key"),
		mk("cosign [unsigned-tx.json]", "PQC+classical sign a tx and broadcast"),
	}
}
