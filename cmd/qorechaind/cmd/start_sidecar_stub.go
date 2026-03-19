//go:build !full

package cmd

import "cosmossdk.io/log"

// SidecarStartHook is a no-op in public builds.
type SidecarStartHook struct{}

func NewSidecarStartHook(_ log.Logger) *SidecarStartHook {
	return &SidecarStartHook{}
}

func (h *SidecarStartHook) Start() error { return nil }
func (h *SidecarStartHook) Stop() error  { return nil }
