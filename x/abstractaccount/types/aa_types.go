package types

import "time"

// AbstractAccountConfig holds the module configuration.
type AbstractAccountConfig struct {
	Enabled             bool  `json:"enabled"`
	MaxSessionKeys      int   `json:"max_session_keys"`
	MaxSpendingRules    int   `json:"max_spending_rules"`
	DefaultSessionTTL   int64 `json:"default_session_ttl"` // seconds
}

// DefaultAbstractAccountConfig returns default configuration.
func DefaultAbstractAccountConfig() AbstractAccountConfig {
	return AbstractAccountConfig{
		Enabled:           false,
		MaxSessionKeys:    10,
		MaxSpendingRules:  5,
		DefaultSessionTTL: 86400, // 24 hours
	}
}

// AbstractAccount represents a smart-contract backed account.
type AbstractAccount struct {
	Address          string         `json:"address"`
	ContractAddress  string         `json:"contract_address"`
	AccountType      string         `json:"account_type"` // multisig, social_recovery, session_based
	SpendingRules    []SpendingRule `json:"spending_rules"`
	SessionKeys      []SessionKey   `json:"session_keys"`
	CreatedAt        time.Time      `json:"created_at"`
	Owner            string         `json:"owner"`
}

// SpendingRule defines spending limits for an abstract account.
type SpendingRule struct {
	ID            string   `json:"id"`
	DailyLimit    int64    `json:"daily_limit"`    // in base denom units
	PerTxLimit    int64    `json:"per_tx_limit"`   // in base denom units
	AllowedDenoms []string `json:"allowed_denoms"` // empty = all denoms
	Enabled       bool     `json:"enabled"`
}

// SessionKey represents a temporary key with limited permissions.
type SessionKey struct {
	Key         string    `json:"key"`
	Expiry      time.Time `json:"expiry"`
	Permissions []string  `json:"permissions"` // send, delegate, vote, etc.
	Label       string    `json:"label"`
	CreatedAt   time.Time `json:"created_at"`
}

// IsExpired checks if a session key has expired.
func (sk SessionKey) IsExpired(now time.Time) bool {
	return now.After(sk.Expiry)
}
