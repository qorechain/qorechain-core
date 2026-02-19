package types

import "cosmossdk.io/errors"

var (
	ErrAnomalyDetected = errors.Register(ModuleName, 2, "transaction flagged as anomalous")
	ErrTxRejected      = errors.Register(ModuleName, 3, "transaction rejected by AI engine")
	ErrInvalidConfig   = errors.Register(ModuleName, 4, "invalid AI configuration")
	ErrSidecarTimeout  = errors.Register(ModuleName, 5, "AI sidecar timeout")
)
