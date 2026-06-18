package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MsgGrantLicense, MsgRevokeLicense, MsgSuspendLicense and MsgResumeLicense are
// generated from proto/qorechain/license/v1/tx.proto (see tx.pb.go). The
// ValidateBasic methods below are attached to those generated types.

const (
	TypeMsgGrantLicense   = "grant_license"
	TypeMsgRevokeLicense  = "revoke_license"
	TypeMsgSuspendLicense = "suspend_license"
	TypeMsgResumeLicense  = "resume_license"
)

func (msg MsgGrantLicense) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if msg.Grantee == "" {
		return fmt.Errorf("grantee cannot be empty")
	}
	if !IsValidFeatureID(msg.FeatureID) {
		return fmt.Errorf("invalid feature ID: %s", msg.FeatureID)
	}
	if msg.ExpiresAt < 0 {
		return fmt.Errorf("expires_at cannot be negative")
	}
	return nil
}

func (msg MsgRevokeLicense) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if msg.Grantee == "" {
		return fmt.Errorf("grantee cannot be empty")
	}
	if msg.FeatureID == "" {
		return fmt.Errorf("feature_id cannot be empty")
	}
	return nil
}

func (msg MsgSuspendLicense) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if msg.Grantee == "" {
		return fmt.Errorf("grantee cannot be empty")
	}
	if msg.FeatureID == "" {
		return fmt.Errorf("feature_id cannot be empty")
	}
	return nil
}

func (msg MsgResumeLicense) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if msg.Grantee == "" {
		return fmt.Errorf("grantee cannot be empty")
	}
	if msg.FeatureID == "" {
		return fmt.Errorf("feature_id cannot be empty")
	}
	return nil
}
