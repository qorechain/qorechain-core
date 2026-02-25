package types

import (
	"time"

	sdkmath "cosmossdk.io/math"
)

// ChainType identifies the type of external chain.
type ChainType string

const (
	ChainTypeIBC      ChainType = "ibc"
	ChainTypeEVM      ChainType = "evm"
	ChainTypeSolana   ChainType = "solana"
	ChainTypeTON      ChainType = "ton"
	ChainTypeSui      ChainType = "sui"
	ChainTypeAptos    ChainType = "aptos"
	ChainTypeBitcoin  ChainType = "bitcoin"
	ChainTypeNEAR     ChainType = "near"
	ChainTypeCardano  ChainType = "cardano"
	ChainTypePolkadot ChainType = "polkadot"
	ChainTypeTezos    ChainType = "tezos"
	ChainTypeTron     ChainType = "tron"
)

// BridgeStatus represents the current status of a bridge.
type BridgeStatus string

const (
	BridgeStatusActive  BridgeStatus = "active"
	BridgeStatusPaused  BridgeStatus = "paused"
	BridgeStatusPending BridgeStatus = "pending"
)

// OperationStatus represents the status of a bridge operation.
type OperationStatus string

const (
	OpStatusPending   OperationStatus = "pending"
	OpStatusAttested  OperationStatus = "attested"
	OpStatusExecuted  OperationStatus = "executed"
	OpStatusFailed    OperationStatus = "failed"
	OpStatusChallenged OperationStatus = "challenged"
)

// OperationType represents the type of bridge operation.
type OperationType string

const (
	OpTypeDeposit    OperationType = "deposit"
	OpTypeWithdrawal OperationType = "withdrawal"
)

// BridgeConfig holds the global bridge configuration.
type BridgeConfig struct {
	MinValidators       int      `json:"min_validators"`
	AttestationThreshold int     `json:"attestation_threshold"` // e.g., 7 of 10
	ChallengePeriodSecs  int64   `json:"challenge_period_secs"` // 24 hours for large transfers
	LargeTransferThreshold string `json:"large_transfer_threshold"` // Amount above which challenge period applies
	Enabled              bool    `json:"enabled"`
}

// DefaultBridgeConfig returns default bridge configuration.
func DefaultBridgeConfig() BridgeConfig {
	return BridgeConfig{
		MinValidators:          3,
		AttestationThreshold:   7,
		ChallengePeriodSecs:    86400, // 24 hours
		LargeTransferThreshold: "100000000000", // 100,000 QOR equivalent in uqor
		Enabled:                true,
	}
}

// ChainConfig holds the configuration for a specific external chain.
type ChainConfig struct {
	ChainID          string       `json:"chain_id"`
	Name             string       `json:"name"`
	ChainType        ChainType    `json:"chain_type"`
	BridgeContract   string       `json:"bridge_contract,omitempty"` // Contract address on external chain
	Status           BridgeStatus `json:"status"`
	MinConfirmations int          `json:"min_confirmations"` // Required confirmations on source chain
	SupportedAssets  []string     `json:"supported_assets"`
	MaxSingleTransfer string     `json:"max_single_transfer"` // Max single transfer in base denom
	DailyLimit       string       `json:"daily_limit"`
}

// DefaultChainConfigs returns the default supported chain configurations.
func DefaultChainConfigs() []ChainConfig {
	return []ChainConfig{
		{
			ChainID:          "ethereum",
			Name:             "Ethereum",
			ChainType:        ChainTypeEVM,
			BridgeContract:   "0x0000000000000000000000000000000000000000", // Placeholder
			Status:           BridgeStatusPending,
			MinConfirmations: 12,
			SupportedAssets:  []string{"ETH", "USDC", "USDT", "WBTC"},
			MaxSingleTransfer: "1000000000000", // 1M QOR equivalent
			DailyLimit:       "10000000000000",
		},
		{
			ChainID:          "solana",
			Name:             "Solana",
			ChainType:        ChainTypeSolana,
			BridgeContract:   "", // Program ID placeholder
			Status:           BridgeStatusPending,
			MinConfirmations: 32,
			SupportedAssets:  []string{"SOL", "USDC"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:       "10000000000000",
		},
		{
			ChainID:          "ton",
			Name:             "TON",
			ChainType:        ChainTypeTON,
			BridgeContract:   "",
			Status:           BridgeStatusPending,
			MinConfirmations: 10,
			SupportedAssets:  []string{"TON", "USDT"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:       "10000000000000",
		},
		{
			ChainID:          "bsc",
			Name:             "BNB Smart Chain",
			ChainType:        ChainTypeEVM,
			BridgeContract:   "0x0000000000000000000000000000000000000000",
			Status:           BridgeStatusPending,
			MinConfirmations: 15,
			SupportedAssets:  []string{"BNB", "USDC", "USDT"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:       "10000000000000",
		},
		{
			ChainID:          "avalanche",
			Name:             "Avalanche C-Chain",
			ChainType:        ChainTypeEVM,
			BridgeContract:   "0x0000000000000000000000000000000000000000",
			Status:           BridgeStatusPending,
			MinConfirmations: 12,
			SupportedAssets:  []string{"AVAX", "USDC"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:       "10000000000000",
		},
		{
			ChainID:          "polygon",
			Name:             "Polygon PoS",
			ChainType:        ChainTypeEVM,
			BridgeContract:   "0x0000000000000000000000000000000000000000",
			Status:           BridgeStatusPending,
			MinConfirmations: 128,
			SupportedAssets:  []string{"POL", "USDC", "USDT", "WETH"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:       "10000000000000",
		},
		{
			ChainID:          "arbitrum",
			Name:             "Arbitrum One",
			ChainType:        ChainTypeEVM,
			BridgeContract:   "0x0000000000000000000000000000000000000000",
			Status:           BridgeStatusPending,
			MinConfirmations: 64,
			SupportedAssets:  []string{"ETH", "USDC", "ARB", "USDT"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:       "10000000000000",
		},
		{
			ChainID:          "sui",
			Name:             "Sui",
			ChainType:        ChainTypeSui,
			BridgeContract:   "",
			Status:           BridgeStatusPending,
			MinConfirmations: 3,
			SupportedAssets:  []string{"SUI", "USDC"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:       "10000000000000",
		},
		// v1.2.0: 9 new chain connections
		{
			ChainID:          "optimism",
			Name:             "Optimism",
			ChainType:        ChainTypeEVM,
			BridgeContract:   "0x0000000000000000000000000000000000000000",
			Status:           BridgeStatusPending,
			MinConfirmations: 10,
			SupportedAssets:  []string{"ETH", "USDC", "OP"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:       "10000000000000",
		},
		{
			ChainID:          "base",
			Name:             "Base",
			ChainType:        ChainTypeEVM,
			BridgeContract:   "0x0000000000000000000000000000000000000000",
			Status:           BridgeStatusPending,
			MinConfirmations: 10,
			SupportedAssets:  []string{"ETH", "USDC"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:       "10000000000000",
		},
		{
			ChainID:          "aptos",
			Name:             "Aptos",
			ChainType:        ChainTypeAptos,
			BridgeContract:   "",
			Status:           BridgeStatusPending,
			MinConfirmations: 6,
			SupportedAssets:  []string{"APT", "USDC"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:       "10000000000000",
		},
		{
			ChainID:          "bitcoin",
			Name:             "Bitcoin",
			ChainType:        ChainTypeBitcoin,
			BridgeContract:   "",
			Status:           BridgeStatusPending,
			MinConfirmations: 6,
			SupportedAssets:  []string{"BTC"},
			MaxSingleTransfer: "500000000000", // Lower limit for BTC
			DailyLimit:       "5000000000000",
		},
		{
			ChainID:          "near",
			Name:             "NEAR Protocol",
			ChainType:        ChainTypeNEAR,
			BridgeContract:   "",
			Status:           BridgeStatusPending,
			MinConfirmations: 3,
			SupportedAssets:  []string{"NEAR", "USDC"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:       "10000000000000",
		},
		{
			ChainID:          "cardano",
			Name:             "Cardano",
			ChainType:        ChainTypeCardano,
			BridgeContract:   "",
			Status:           BridgeStatusPending,
			MinConfirmations: 15,
			SupportedAssets:  []string{"ADA"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:       "10000000000000",
		},
		{
			ChainID:          "polkadot",
			Name:             "Polkadot",
			ChainType:        ChainTypePolkadot,
			BridgeContract:   "",
			Status:           BridgeStatusPending,
			MinConfirmations: 12,
			SupportedAssets:  []string{"DOT"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:       "10000000000000",
		},
		{
			ChainID:          "tezos",
			Name:             "Tezos",
			ChainType:        ChainTypeTezos,
			BridgeContract:   "",
			Status:           BridgeStatusPending,
			MinConfirmations: 2,
			SupportedAssets:  []string{"XTZ"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:       "10000000000000",
		},
		{
			ChainID:          "tron",
			Name:             "TRON",
			ChainType:        ChainTypeTron,
			BridgeContract:   "",
			Status:           BridgeStatusPending,
			MinConfirmations: 20,
			SupportedAssets:  []string{"TRX", "USDT"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:       "10000000000000",
		},
	}
}

// BridgeValidator represents a registered bridge validator.
type BridgeValidator struct {
	Address         string   `json:"address"`
	PQCPubkey       []byte   `json:"pqc_pubkey"`
	SupportedChains []string `json:"supported_chains"`
	Reputation      float64  `json:"reputation"`
	Active          bool     `json:"active"`
	RegisteredAt    int64    `json:"registered_at"` // Block height
}

// BridgeOperation represents a single bridge transfer operation.
type BridgeOperation struct {
	ID               string            `json:"id"`
	Type             OperationType     `json:"type"`
	SourceChain      string            `json:"source_chain"`
	DestChain        string            `json:"dest_chain"`
	Sender           string            `json:"sender"`
	Receiver         string            `json:"receiver"`
	Asset            string            `json:"asset"`
	Amount           string            `json:"amount"`
	SourceTxHash     string            `json:"source_tx_hash,omitempty"`
	Status           OperationStatus   `json:"status"`
	Attestations     []Attestation     `json:"attestations"`
	PQCCommitment    []byte            `json:"pqc_commitment,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	CompletedAt      *time.Time        `json:"completed_at,omitempty"`
	ChallengeEndTime *time.Time        `json:"challenge_end_time,omitempty"`
}

// Attestation represents a bridge validator's attestation for an operation.
type Attestation struct {
	Validator    string `json:"validator"`
	EventType    string `json:"event_type"` // "deposit" | "withdrawal_complete"
	TxHash       string `json:"tx_hash"`
	PQCSignature []byte `json:"pqc_signature"`
	Timestamp    int64  `json:"timestamp"` // Block height
}

// LockedAmount tracks locked and minted amounts for a chain/asset pair.
type LockedAmount struct {
	Chain       string `json:"chain"`
	Asset       string `json:"asset"`
	TotalLocked string `json:"total_locked"` // sdkmath.Int as string
	TotalMinted string `json:"total_minted"` // sdkmath.Int as string
}

// CircuitBreakerState holds per-chain circuit breaker state.
type CircuitBreakerState struct {
	Chain              string    `json:"chain"`
	MaxSingleTransfer  string    `json:"max_single_transfer"` // sdkmath.Int as string
	DailyLimit         string    `json:"daily_limit"`
	CurrentDaily       string    `json:"current_daily"`
	LastResetHeight    int64     `json:"last_reset_height"`
	Paused             bool      `json:"paused"`
	PausedReason       string    `json:"paused_reason,omitempty"`
	PausedAt           *time.Time `json:"paused_at,omitempty"`
}

// BridgeRouteEstimate represents an AI-optimized route estimate.
type BridgeRouteEstimate struct {
	SourceChain    string           `json:"source_chain"`
	DestChain      string           `json:"dest_chain"`
	Asset          string           `json:"asset"`
	Amount         string           `json:"amount"`
	EstimatedFee   string           `json:"estimated_fee"`
	EstimatedTime  int64            `json:"estimated_time_seconds"`
	Route          []string         `json:"route"` // Path through intermediate chains
	Confidence     float64          `json:"confidence"`
	SecurityScore  float64          `json:"security_score"`
}

// ParseAmount safely parses an sdkmath.Int from a string.
func ParseAmount(s string) (sdkmath.Int, error) {
	i, ok := sdkmath.NewIntFromString(s)
	if !ok {
		return sdkmath.Int{}, ErrInvalidAmount.Wrapf("cannot parse amount: %s", s)
	}
	return i, nil
}
