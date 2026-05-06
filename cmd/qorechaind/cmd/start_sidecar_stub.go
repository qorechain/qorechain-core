//go:build !full

package cmd

import (
	"cosmossdk.io/log"

	"github.com/spf13/cobra"
)

// SidecarStartHook is a no-op in public builds.
type SidecarStartHook struct{}

func NewSidecarStartHook(_ log.Logger) *SidecarStartHook {
	return &SidecarStartHook{}
}

func (h *SidecarStartHook) Start() error { return nil }
func (h *SidecarStartHook) Stop() error  { return nil }

// WireSidecarHooks is a no-op in public builds. In extended builds it
// wraps the start command's PreRunE to start the sidecar orchestrator
// alongside the node and registers a signal handler to stop it on
// SIGINT/SIGTERM.
func WireSidecarHooks(_ *cobra.Command) {}
