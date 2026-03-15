package rpc

// TokenAccountInfo describes parsed SPL Token account data.
type TokenAccountInfo struct {
	Mint            string `json:"mint"`
	Owner           string `json:"owner"`
	Amount          string `json:"amount"`
	Decimals        uint8  `json:"decimals"`
	Delegate        string `json:"delegate,omitempty"`
	DelegatedAmount string `json:"delegatedAmount,omitempty"`
	State           string `json:"state"`
	IsNative        bool   `json:"isNative"`
	CloseAuthority  string `json:"closeAuthority,omitempty"`
}

// MintInfo describes parsed SPL Token mint data.
type MintInfo struct {
	MintAuthority   string `json:"mintAuthority"`
	Supply          string `json:"supply"`
	Decimals        uint8  `json:"decimals"`
	IsInitialized   bool   `json:"isInitialized"`
	FreezeAuthority string `json:"freezeAuthority,omitempty"`
}

// SimulateTransactionResult is the result of simulateTransaction.
type SimulateTransactionResult struct {
	Context ContextResult  `json:"context"`
	Value   *SimulateValue `json:"value"`
}

// SimulateValue holds the simulated execution output.
type SimulateValue struct {
	Err           interface{} `json:"err"`
	Logs          []string    `json:"logs"`
	Accounts      interface{} `json:"accounts"`
	UnitsConsumed uint64      `json:"unitsConsumed"`
}

// SendTransactionResult is the transaction signature hash.
type SendTransactionResult string

// ProgramAccountResult is a single entry in getProgramAccounts response.
type ProgramAccountResult struct {
	Pubkey  string       `json:"pubkey"`
	Account *AccountInfo `json:"account"`
}

// BlockhashResult is the result of getRecentBlockhash/getLatestBlockhash.
type BlockhashResult struct {
	Context ContextResult  `json:"context"`
	Value   *BlockhashValue `json:"value"`
}

// BlockhashValue holds blockhash and validity information.
type BlockhashValue struct {
	Blockhash            string `json:"blockhash"`
	LastValidBlockHeight uint64 `json:"lastValidBlockHeight"`
}

// FeeResult is the result of getFeeForMessage.
type FeeResult struct {
	Context ContextResult `json:"context"`
	Value   uint64        `json:"value"`
}
