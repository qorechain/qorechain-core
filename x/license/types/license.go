package types

import "encoding/json"

type License struct {
	Grantee   string `json:"grantee"`
	FeatureID string `json:"feature_id"`
	ExpiresAt int64  `json:"expires_at"`
	GrantedAt int64  `json:"granted_at"`
	GrantedBy string `json:"granted_by"`
	Suspended bool   `json:"suspended"`
	Metadata  string `json:"metadata"`
}

func (l License) IsExpired(currentHeight int64) bool {
	return l.ExpiresAt > 0 && l.ExpiresAt <= currentHeight
}

func (l License) IsActive(currentHeight int64) bool {
	return !l.Suspended && !l.IsExpired(currentHeight)
}

func (l License) Marshal() ([]byte, error) {
	return json.Marshal(l)
}

func Unmarshal(bz []byte) (License, error) {
	var l License
	err := json.Unmarshal(bz, &l)
	return l, err
}
