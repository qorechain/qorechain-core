package types

import "cosmossdk.io/errors"

var (
	ErrInvalidParams        = errors.Register(ModuleName, 2, "invalid stakingtier params")
	ErrFeatureNotActive     = errors.Register(ModuleName, 3, "staking tiers are not active on this chain yet")
	ErrDisabled             = errors.Register(ModuleName, 4, "staking tiers are disabled by governance")
	ErrInvalidTerm          = errors.Register(ModuleName, 5, "lock term is not offered")
	ErrInvalidAmount        = errors.Register(ModuleName, 6, "invalid lock amount")
	ErrInsufficientUnlocked = errors.Register(ModuleName, 7, "not enough unlocked delegation")
	ErrLockNotFound         = errors.Register(ModuleName, 8, "lock not found")
	ErrNotLockOwner         = errors.Register(ModuleName, 9, "lock belongs to another delegator")
	ErrLockExpired          = errors.Register(ModuleName, 10, "lock has already expired")
	ErrTooManyLocks         = errors.Register(ModuleName, 11, "too many open locks")
	ErrNothingToClaim       = errors.Register(ModuleName, 12, "no bonus to claim")
	ErrLockedStake          = errors.Register(ModuleName, 13, "stake is locked")
)
