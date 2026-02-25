package types

import "cosmossdk.io/errors"

var (
	ErrInvalidParams      = errors.Register(ModuleName, 2, "invalid inflation params")
	ErrModuleDisabled     = errors.Register(ModuleName, 3, "inflation module is disabled")
	ErrMintFailed         = errors.Register(ModuleName, 4, "failed to mint epoch emission")
	ErrInvalidEpochLength = errors.Register(ModuleName, 5, "invalid epoch length")
)
