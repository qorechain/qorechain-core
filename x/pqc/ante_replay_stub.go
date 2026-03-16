//go:build !full

package pqc

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// PQCReplayGuard is a pass-through stub for public builds.
// In the full build, this provides anti-replay protection for PQC-signed transactions.
type PQCReplayGuard struct{}

// NewPQCReplayGuard creates a new anti-replay guard (stub).
func NewPQCReplayGuard(_ time.Duration) PQCReplayGuard {
	return PQCReplayGuard{}
}

// DefaultPQCReplayGuard creates a PQCReplayGuard with the default drift tolerance (stub).
func DefaultPQCReplayGuard() PQCReplayGuard {
	return PQCReplayGuard{}
}

func (PQCReplayGuard) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	return next(ctx, tx, simulate)
}
