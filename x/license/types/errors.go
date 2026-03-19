package types

import "cosmossdk.io/errors"

var (
	ErrLicenseNotFound  = errors.Register(ModuleName, 2, "license not found")
	ErrLicenseExists    = errors.Register(ModuleName, 3, "license already exists")
	ErrLicenseExpired   = errors.Register(ModuleName, 4, "license has expired")
	ErrLicenseSuspended = errors.Register(ModuleName, 5, "license is suspended")
	ErrInvalidFeatureID = errors.Register(ModuleName, 6, "invalid feature ID")
	ErrUnauthorized     = errors.Register(ModuleName, 7, "unauthorized: not module authority")
	ErrInvalidAddress   = errors.Register(ModuleName, 8, "invalid address")
)
