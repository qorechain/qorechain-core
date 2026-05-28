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

	// New ChainTypes added in v2.24.0 (cross-network expansion §3.4).
	// Each requires a dedicated bridge handler in the extended build —
	// EVM-family chains share evm_bridge.go via per-chain config injection,
	// while these architectures need their own deposit/withdrawal verifiers.
	ChainTypeStarknet ChainType = "starknet" // Cairo VM L2
	ChainTypeXRPL     ChainType = "xrpl"     // XRP Ledger UNL consensus
	ChainTypeStellar  ChainType = "stellar"  // Stellar Consensus Protocol
	ChainTypeHedera   ChainType = "hedera"   // Hashgraph; HCS subscription model
	ChainTypeAlgorand ChainType = "algorand" // Pure Proof-of-Stake
)

// AllChainTypes returns every supported ChainType. Used by validation and
// CLI surfaces — order is stable for deterministic output.
func AllChainTypes() []ChainType {
	return []ChainType{
		ChainTypeIBC,
		ChainTypeEVM,
		ChainTypeSolana,
		ChainTypeTON,
		ChainTypeSui,
		ChainTypeAptos,
		ChainTypeBitcoin,
		ChainTypeNEAR,
		ChainTypeCardano,
		ChainTypePolkadot,
		ChainTypeTezos,
		ChainTypeTron,
		ChainTypeStarknet,
		ChainTypeXRPL,
		ChainTypeStellar,
		ChainTypeHedera,
		ChainTypeAlgorand,
	}
}

// IsValidChainType returns true if t is one of the supported ChainTypes.
// Used by msg validation and config import to reject unknown chain types
// at the boundary before any keeper code runs.
func IsValidChainType(t ChainType) bool {
	for _, ct := range AllChainTypes() {
		if ct == t {
			return true
		}
	}
	return false
}

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
	OpStatusPending    OperationStatus = "pending"
	OpStatusAttested   OperationStatus = "attested"
	OpStatusExecuting  OperationStatus = "executing"
	OpStatusExecuted   OperationStatus = "executed"
	OpStatusFailed     OperationStatus = "failed"
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
	MinValidators          int    `json:"min_validators"`
	AttestationThreshold   int    `json:"attestation_threshold"`    // e.g., 7 of 10
	ChallengePeriodSecs    int64  `json:"challenge_period_secs"`    // 24 hours for large transfers
	LargeTransferThreshold string `json:"large_transfer_threshold"` // Amount above which challenge period applies
	Enabled                bool   `json:"enabled"`
}

// DefaultBridgeConfig returns default bridge configuration.
func DefaultBridgeConfig() BridgeConfig {
	return BridgeConfig{
		MinValidators:          10,
		AttestationThreshold:   7,
		ChallengePeriodSecs:    86400,          // 24 hours
		LargeTransferThreshold: "100000000000", // 100,000 QOR equivalent in uqor
		Enabled:                true,
	}
}

// ChainArchitecture disambiguates the IBC flavour of a `ChainTypeIBC`
// chain: legacy IBC (classic) vs IBC Eureka v2 (next-gen). Non-IBC
// chains use ChainArchEmpty.
type ChainArchitecture string

const (
	// ChainArchEmpty is the default for non-IBC chains.
	ChainArchEmpty ChainArchitecture = ""

	// ChainArchIBCClassic is the legacy IBC stack. Used for chains
	// onboarded before IBC Eureka v2 (the 7 baseline IBC chains:
	// cosmoshub, osmosis, noble, celestia, stride, akash, babylon).
	ChainArchIBCClassic ChainArchitecture = "ibc_classic"

	// ChainArchIBCEurekaV2 is the next-generation IBC stack. New
	// IBC chains added from v3.0.0 forward default to Eureka v2.
	// Existing IBC chains can be migrated by governance proposal.
	ChainArchIBCEurekaV2 ChainArchitecture = "ibc_eureka_v2"
)

// ChainConfig holds the configuration for a specific external chain.
//
// IBC-specific fields (IBCChannelID, IBCPortID, IBCConnectionID,
// EurekaClientType, Architecture) are only populated for chains where
// ChainType == ChainTypeIBC; non-IBC chains leave them empty and the
// JSON omitempty markers keep them out of the wire representation.
type ChainConfig struct {
	ChainID           string       `json:"chain_id"`
	Name              string       `json:"name"`
	ChainType         ChainType    `json:"chain_type"`
	BridgeContract    string       `json:"bridge_contract,omitempty"` // Contract address on external chain
	Status            BridgeStatus `json:"status"`
	MinConfirmations  int          `json:"min_confirmations"` // Required confirmations on source chain
	SupportedAssets   []string     `json:"supported_assets"`
	MaxSingleTransfer string       `json:"max_single_transfer"` // Max single transfer in base denom
	DailyLimit        string       `json:"daily_limit"`

	// IBC-specific (v2.35.0+).
	Architecture     ChainArchitecture `json:"architecture,omitempty"`
	IBCChannelID     string            `json:"ibc_channel_id,omitempty"`
	IBCPortID        string            `json:"ibc_port_id,omitempty"`
	IBCConnectionID  string            `json:"ibc_connection_id,omitempty"`
	EurekaClientType string            `json:"eureka_client_type,omitempty"` // e.g. "tendermint", "solomachine"
}

// IsValidChainArchitecture returns true if a is a recognised value.
func IsValidChainArchitecture(a ChainArchitecture) bool {
	switch a {
	case ChainArchEmpty, ChainArchIBCClassic, ChainArchIBCEurekaV2:
		return true
	}
	return false
}

// DefaultChainConfigs returns the default supported chain configurations.
func DefaultChainConfigs() []ChainConfig {
	return []ChainConfig{
		{
			ChainID:           "ethereum",
			Name:              "Ethereum",
			ChainType:         ChainTypeEVM,
			BridgeContract:    "0x0000000000000000000000000000000000000000", // Placeholder
			Status:            BridgeStatusPending,
			MinConfirmations:  12,
			SupportedAssets:   []string{"ETH", "USDC", "USDT", "WBTC"},
			MaxSingleTransfer: "1000000000000", // 1M QOR equivalent
			DailyLimit:        "10000000000000",
		},
		{
			ChainID:           "solana",
			Name:              "Solana",
			ChainType:         ChainTypeSolana,
			BridgeContract:    "", // Program ID placeholder
			Status:            BridgeStatusPending,
			MinConfirmations:  32,
			SupportedAssets:   []string{"SOL", "USDC"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:        "10000000000000",
		},
		{
			ChainID:           "ton",
			Name:              "TON",
			ChainType:         ChainTypeTON,
			BridgeContract:    "",
			Status:            BridgeStatusPending,
			MinConfirmations:  10,
			SupportedAssets:   []string{"TON", "USDT"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:        "10000000000000",
		},
		{
			ChainID:           "bsc",
			Name:              "BNB Smart Chain",
			ChainType:         ChainTypeEVM,
			BridgeContract:    "0x0000000000000000000000000000000000000000",
			Status:            BridgeStatusPending,
			MinConfirmations:  15,
			SupportedAssets:   []string{"BNB", "USDC", "USDT"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:        "10000000000000",
		},
		{
			ChainID:           "avalanche",
			Name:              "Avalanche C-Chain",
			ChainType:         ChainTypeEVM,
			BridgeContract:    "0x0000000000000000000000000000000000000000",
			Status:            BridgeStatusPending,
			MinConfirmations:  12,
			SupportedAssets:   []string{"AVAX", "USDC"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:        "10000000000000",
		},
		{
			ChainID:           "polygon",
			Name:              "Polygon PoS",
			ChainType:         ChainTypeEVM,
			BridgeContract:    "0x0000000000000000000000000000000000000000",
			Status:            BridgeStatusPending,
			MinConfirmations:  128,
			SupportedAssets:   []string{"POL", "USDC", "USDT", "WETH"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:        "10000000000000",
		},
		{
			ChainID:           "arbitrum",
			Name:              "Arbitrum One",
			ChainType:         ChainTypeEVM,
			BridgeContract:    "0x0000000000000000000000000000000000000000",
			Status:            BridgeStatusPending,
			MinConfirmations:  64,
			SupportedAssets:   []string{"ETH", "USDC", "ARB", "USDT"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:        "10000000000000",
		},
		{
			ChainID:           "sui",
			Name:              "Sui",
			ChainType:         ChainTypeSui,
			BridgeContract:    "",
			Status:            BridgeStatusPending,
			MinConfirmations:  3,
			SupportedAssets:   []string{"SUI", "USDC"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:        "10000000000000",
		},
		// v1.2.0: 9 new chain connections
		{
			ChainID:           "optimism",
			Name:              "Optimism",
			ChainType:         ChainTypeEVM,
			BridgeContract:    "0x0000000000000000000000000000000000000000",
			Status:            BridgeStatusPending,
			MinConfirmations:  10,
			SupportedAssets:   []string{"ETH", "USDC", "OP"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:        "10000000000000",
		},
		{
			ChainID:           "base",
			Name:              "Base",
			ChainType:         ChainTypeEVM,
			BridgeContract:    "0x0000000000000000000000000000000000000000",
			Status:            BridgeStatusPending,
			MinConfirmations:  10,
			SupportedAssets:   []string{"ETH", "USDC"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:        "10000000000000",
		},
		{
			ChainID:           "aptos",
			Name:              "Aptos",
			ChainType:         ChainTypeAptos,
			BridgeContract:    "",
			Status:            BridgeStatusPending,
			MinConfirmations:  6,
			SupportedAssets:   []string{"APT", "USDC"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:        "10000000000000",
		},
		{
			ChainID:           "bitcoin",
			Name:              "Bitcoin",
			ChainType:         ChainTypeBitcoin,
			BridgeContract:    "",
			Status:            BridgeStatusPending,
			MinConfirmations:  6,
			SupportedAssets:   []string{"BTC"},
			MaxSingleTransfer: "500000000000", // Lower limit for BTC
			DailyLimit:        "5000000000000",
		},
		{
			ChainID:           "near",
			Name:              "NEAR Protocol",
			ChainType:         ChainTypeNEAR,
			BridgeContract:    "",
			Status:            BridgeStatusPending,
			MinConfirmations:  3,
			SupportedAssets:   []string{"NEAR", "USDC"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:        "10000000000000",
		},
		{
			ChainID:           "cardano",
			Name:              "Cardano",
			ChainType:         ChainTypeCardano,
			BridgeContract:    "",
			Status:            BridgeStatusPending,
			MinConfirmations:  15,
			SupportedAssets:   []string{"ADA"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:        "10000000000000",
		},
		{
			ChainID:           "polkadot",
			Name:              "Polkadot",
			ChainType:         ChainTypePolkadot,
			BridgeContract:    "",
			Status:            BridgeStatusPending,
			MinConfirmations:  12,
			SupportedAssets:   []string{"DOT"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:        "10000000000000",
		},
		{
			ChainID:           "tezos",
			Name:              "Tezos",
			ChainType:         ChainTypeTezos,
			BridgeContract:    "",
			Status:            BridgeStatusPending,
			MinConfirmations:  2,
			SupportedAssets:   []string{"XTZ"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:        "10000000000000",
		},
		{
			ChainID:           "tron",
			Name:              "TRON",
			ChainType:         ChainTypeTron,
			BridgeContract:    "",
			Status:            BridgeStatusPending,
			MinConfirmations:  20,
			SupportedAssets:   []string{"TRX", "USDT"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:        "10000000000000",
		},

		// ----- New chain configs added in v2.25.0 (cross-network expansion §3.4) -----

		// EVM L2 (ZK rollups)
		{
			ChainID:           "zksync_era",
			Name:              "zkSync Era",
			ChainType:         ChainTypeEVM,
			BridgeContract:    "0x0000000000000000000000000000000000000000",
			Status:            BridgeStatusPending,
			MinConfirmations:  12,
			SupportedAssets:   []string{"ETH", "USDC"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:        "10000000000000",
		},
		{
			ChainID:           "linea",
			Name:              "Linea",
			ChainType:         ChainTypeEVM,
			BridgeContract:    "0x0000000000000000000000000000000000000000",
			Status:            BridgeStatusPending,
			MinConfirmations:  12,
			SupportedAssets:   []string{"ETH", "USDC"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:        "10000000000000",
		},
		{
			ChainID:           "scroll",
			Name:              "Scroll",
			ChainType:         ChainTypeEVM,
			BridgeContract:    "0x0000000000000000000000000000000000000000",
			Status:            BridgeStatusPending,
			MinConfirmations:  12,
			SupportedAssets:   []string{"ETH", "USDC"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:        "10000000000000",
		},

		// Cairo VM L2
		{
			ChainID:           "starknet",
			Name:              "Starknet",
			ChainType:         ChainTypeStarknet,
			BridgeContract:    "",
			Status:            BridgeStatusPending,
			MinConfirmations:  12,
			SupportedAssets:   []string{"ETH", "STRK", "USDC"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:        "10000000000000",
		},

		// EVM L2 (Optimistic) — yield-bearing
		{
			ChainID:           "blast",
			Name:              "Blast",
			ChainType:         ChainTypeEVM,
			BridgeContract:    "0x0000000000000000000000000000000000000000",
			Status:            BridgeStatusPending,
			MinConfirmations:  10,
			SupportedAssets:   []string{"ETH", "USDB"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:        "10000000000000",
		},

		// EVM L2
		{
			ChainID:           "mantle",
			Name:              "Mantle",
			ChainType:         ChainTypeEVM,
			BridgeContract:    "0x0000000000000000000000000000000000000000",
			Status:            BridgeStatusPending,
			MinConfirmations:  10,
			SupportedAssets:   []string{"MNT", "USDC"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:        "10000000000000",
		},

		// HyperEVM L1 — derivatives-focused
		{
			ChainID:           "hyperliquid",
			Name:              "Hyperliquid",
			ChainType:         ChainTypeEVM,
			BridgeContract:    "0x0000000000000000000000000000000000000000",
			Status:            BridgeStatusPending,
			MinConfirmations:  10,
			SupportedAssets:   []string{"USDC"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:        "10000000000000",
		},

		// EVM L1 (Proof-of-Liquidity)
		{
			ChainID:           "berachain",
			Name:              "Berachain",
			ChainType:         ChainTypeEVM,
			BridgeContract:    "0x0000000000000000000000000000000000000000",
			Status:            BridgeStatusPending,
			MinConfirmations:  10,
			SupportedAssets:   []string{"BERA", "HONEY"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:        "10000000000000",
		},

		// EVM L1
		{
			ChainID:           "sonic",
			Name:              "Sonic",
			ChainType:         ChainTypeEVM,
			BridgeContract:    "0x0000000000000000000000000000000000000000",
			Status:            BridgeStatusPending,
			MinConfirmations:  10,
			SupportedAssets:   []string{"S", "USDC"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:        "10000000000000",
		},

		// Parallel EVM L1 (Cosmos-based; dual EVM + IBC)
		{
			ChainID:           "sei",
			Name:              "Sei",
			ChainType:         ChainTypeEVM,
			BridgeContract:    "0x0000000000000000000000000000000000000000",
			Status:            BridgeStatusPending,
			MinConfirmations:  10,
			SupportedAssets:   []string{"SEI", "USDC"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:        "10000000000000",
		},

		// Parallel EVM L1 (high finality)
		{
			ChainID:           "monad",
			Name:              "Monad",
			ChainType:         ChainTypeEVM,
			BridgeContract:    "0x0000000000000000000000000000000000000000",
			Status:            BridgeStatusPending,
			MinConfirmations:  30,
			SupportedAssets:   []string{"MON", "USDC"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:        "10000000000000",
		},

		// EVM L1 (stablecoin-focused; BTC anchor)
		{
			ChainID:           "plasma",
			Name:              "Plasma",
			ChainType:         ChainTypeEVM,
			BridgeContract:    "0x0000000000000000000000000000000000000000",
			Status:            BridgeStatusPending,
			MinConfirmations:  10,
			SupportedAssets:   []string{"XPL", "USDT"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:        "10000000000000",
		},

		// XRP Ledger
		{
			ChainID:           "xrpl",
			Name:              "XRP Ledger",
			ChainType:         ChainTypeXRPL,
			BridgeContract:    "",
			Status:            BridgeStatusPending,
			MinConfirmations:  4,
			SupportedAssets:   []string{"XRP", "RLUSD"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:        "10000000000000",
		},

		// Stellar
		{
			ChainID:           "stellar",
			Name:              "Stellar",
			ChainType:         ChainTypeStellar,
			BridgeContract:    "",
			Status:            BridgeStatusPending,
			MinConfirmations:  5,
			SupportedAssets:   []string{"XLM", "USDC"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:        "10000000000000",
		},

		// Hedera
		{
			ChainID:           "hedera",
			Name:              "Hedera",
			ChainType:         ChainTypeHedera,
			BridgeContract:    "",
			Status:            BridgeStatusPending,
			MinConfirmations:  4,
			SupportedAssets:   []string{"HBAR", "USDC"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:        "10000000000000",
		},

		// Algorand
		{
			ChainID:           "algorand",
			Name:              "Algorand",
			ChainType:         ChainTypeAlgorand,
			BridgeContract:    "",
			Status:            BridgeStatusPending,
			MinConfirmations:  4,
			SupportedAssets:   []string{"ALGO", "USDC"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:        "10000000000000",
		},

		// Injective (IBC — cheapest path; reuses IBC handler)
		{
			ChainID:           "injective",
			Name:              "Injective",
			ChainType:         ChainTypeIBC,
			BridgeContract:    "",
			Status:            BridgeStatusPending,
			MinConfirmations:  1,
			SupportedAssets:   []string{"INJ", "USDC"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:        "10000000000000",
		},

		// Filecoin FVM (EVM-compatible since 2023)
		{
			ChainID:           "filecoin",
			Name:              "Filecoin (FVM)",
			ChainType:         ChainTypeEVM,
			BridgeContract:    "0x0000000000000000000000000000000000000000",
			Status:            BridgeStatusPending,
			MinConfirmations:  10,
			SupportedAssets:   []string{"FIL", "USDC"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:        "10000000000000",
		},

		// Cronos (EVM L1)
		{
			ChainID:           "cronos",
			Name:              "Cronos",
			ChainType:         ChainTypeEVM,
			BridgeContract:    "0x0000000000000000000000000000000000000000",
			Status:            BridgeStatusPending,
			MinConfirmations:  12,
			SupportedAssets:   []string{"CRO", "USDC"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:        "10000000000000",
		},

		// Kaia (Klaytn + Finschia merged)
		{
			ChainID:           "kaia",
			Name:              "Kaia",
			ChainType:         ChainTypeEVM,
			BridgeContract:    "0x0000000000000000000000000000000000000000",
			Status:            BridgeStatusPending,
			MinConfirmations:  10,
			SupportedAssets:   []string{"KAIA", "USDC"},
			MaxSingleTransfer: "1000000000000",
			DailyLimit:        "10000000000000",
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
	ID               string          `json:"id"`
	Type             OperationType   `json:"type"`
	SourceChain      string          `json:"source_chain"`
	DestChain        string          `json:"dest_chain"`
	Sender           string          `json:"sender"`
	Receiver         string          `json:"receiver"`
	Asset            string          `json:"asset"`
	Amount           string          `json:"amount"`
	SourceTxHash     string          `json:"source_tx_hash,omitempty"`
	Status           OperationStatus `json:"status"`
	Attestations     []Attestation   `json:"attestations"`
	PQCCommitment    []byte          `json:"pqc_commitment,omitempty"`
	RetryCount       int             `json:"retry_count"`
	CreatedAt        time.Time       `json:"created_at"`
	CompletedAt      *time.Time      `json:"completed_at,omitempty"`
	ChallengeEndTime *time.Time      `json:"challenge_end_time,omitempty"`
}

// WrappedDenom returns the bridge-wrapped denomination for a given asset and
// source chain, e.g. "bridge/ethereum/USDC".
func WrappedDenom(asset, sourceChain string) string {
	return "bridge/" + sourceChain + "/" + asset
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
	Chain             string     `json:"chain"`
	MaxSingleTransfer string     `json:"max_single_transfer"` // sdkmath.Int as string
	DailyLimit        string     `json:"daily_limit"`
	CurrentDaily      string     `json:"current_daily"`
	LastResetHeight   int64      `json:"last_reset_height"`
	LastResetTime     *time.Time `json:"last_reset_time,omitempty"`
	Paused            bool       `json:"paused"`
	PausedReason      string     `json:"paused_reason,omitempty"`
	PausedAt          *time.Time `json:"paused_at,omitempty"`
}

// BridgeRouteEstimate represents an AI-optimized route estimate.
type BridgeRouteEstimate struct {
	SourceChain   string   `json:"source_chain"`
	DestChain     string   `json:"dest_chain"`
	Asset         string   `json:"asset"`
	Amount        string   `json:"amount"`
	EstimatedFee  string   `json:"estimated_fee"`
	EstimatedTime int64    `json:"estimated_time_seconds"`
	Route         []string `json:"route"` // Path through intermediate chains
	Confidence    float64  `json:"confidence"`
	SecurityScore float64  `json:"security_score"`
}

// ParseAmount safely parses an sdkmath.Int from a string.
func ParseAmount(s string) (sdkmath.Int, error) {
	i, ok := sdkmath.NewIntFromString(s)
	if !ok {
		return sdkmath.Int{}, ErrInvalidAmount.Wrapf("cannot parse amount: %s", s)
	}
	return i, nil
}
