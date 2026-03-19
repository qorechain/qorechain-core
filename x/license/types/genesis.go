package types

import "fmt"

type GenesisState struct {
	Licenses  []License `json:"licenses"`
	Authority string    `json:"authority"`
}

func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Licenses:  []License{},
		Authority: "",
	}
}

func (gs GenesisState) Validate() error {
	seen := make(map[string]bool)
	for _, l := range gs.Licenses {
		key := l.Grantee + "/" + l.FeatureID
		if seen[key] {
			return fmt.Errorf("duplicate license: %s", key)
		}
		seen[key] = true
		if l.Grantee == "" {
			return fmt.Errorf("license grantee cannot be empty")
		}
		if !IsValidFeatureID(l.FeatureID) {
			return fmt.Errorf("invalid feature ID in genesis: %s", l.FeatureID)
		}
	}
	return nil
}
