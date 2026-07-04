package types

import errorsmod "cosmossdk.io/errors"

var (
	ErrAccountNotFound       = errorsmod.Register(ModuleName, 2, "abstract account not found")
	ErrAccountExists         = errorsmod.Register(ModuleName, 3, "abstract account already exists")
	ErrInvalidAccountType    = errorsmod.Register(ModuleName, 4, "invalid account type")
	ErrSpendingLimitExceeded = errorsmod.Register(ModuleName, 5, "spending limit exceeded")
	ErrSessionKeyExpired     = errorsmod.Register(ModuleName, 6, "session key expired")
	ErrMaxSessionKeys        = errorsmod.Register(ModuleName, 7, "maximum session keys reached")
	ErrInvalidSpendingRule   = errorsmod.Register(ModuleName, 8, "invalid spending rule")
	ErrModuleDisabled        = errorsmod.Register(ModuleName, 9, "abstract account module is disabled")

	// v3.1.84 authenticator permission enforcement (WAL-CRIT-1 / WAL-HIGH-1).
	// ErrPermissionDenied: an authenticator-authorized tx contained a message the
	// authenticator is not scoped for (or a non-delegable key-management message,
	// or an unknown message type — fail-closed). ErrAuthenticatorReplay: the
	// authenticator signature's replay binding (chain-id/account/pubkey/nonce)
	// did not match. Clients (QoreX/dashboard) decode these codes into typed
	// errors (ErrPermissionDenied / ErrSpendLimitExceeded) for user-facing copy.
	ErrPermissionDenied    = errorsmod.Register(ModuleName, 10, "authenticator not permitted for this action")
	ErrAuthenticatorReplay = errorsmod.Register(ModuleName, 11, "authenticator signature replay binding mismatch")
)
