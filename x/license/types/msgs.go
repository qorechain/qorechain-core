package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	TypeMsgGrantLicense   = "grant_license"
	TypeMsgRevokeLicense  = "revoke_license"
	TypeMsgSuspendLicense = "suspend_license"
	TypeMsgResumeLicense  = "resume_license"
)

type MsgGrantLicense struct {
	Authority string `json:"authority"`
	Grantee   string `json:"grantee"`
	FeatureID string `json:"feature_id"`
	ExpiresAt int64  `json:"expires_at"`
	Metadata  string `json:"metadata"`
}

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

type MsgRevokeLicense struct {
	Authority string `json:"authority"`
	Grantee   string `json:"grantee"`
	FeatureID string `json:"feature_id"`
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

type MsgSuspendLicense struct {
	Authority string `json:"authority"`
	Grantee   string `json:"grantee"`
	FeatureID string `json:"feature_id"`
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

type MsgResumeLicense struct {
	Authority string `json:"authority"`
	Grantee   string `json:"grantee"`
	FeatureID string `json:"feature_id"`
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
