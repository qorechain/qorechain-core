package rpc

// RPCRequest is a standard JSON-RPC 2.0 request.
type RPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      interface{}   `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params,omitempty"`
}

// RPCResponse is a standard JSON-RPC 2.0 response.
type RPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

// RPCError is a JSON-RPC 2.0 error object.
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Standard JSON-RPC error codes.
const (
	ErrCodeParse          = -32700
	ErrCodeInvalidRequest = -32600
	ErrCodeMethodNotFound = -32601
	ErrCodeInvalidParams  = -32602
	ErrCodeInternal       = -32603
)

// AccountInfo represents a Solana-compatible account info response.
type AccountInfo struct {
	Data       []string `json:"data"`       // [base64-encoded, "base64"]
	Executable bool     `json:"executable"`
	Lamports   uint64   `json:"lamports"`
	Owner      string   `json:"owner"`
	RentEpoch  uint64   `json:"rentEpoch"`
}

// GetAccountInfoResult is the result of getAccountInfo.
type GetAccountInfoResult struct {
	Context ContextResult `json:"context"`
	Value   *AccountInfo  `json:"value"`
}

// ContextResult provides the slot context for responses.
type ContextResult struct {
	Slot uint64 `json:"slot"`
}

// GetBalanceResult is the result of getBalance.
type GetBalanceResult struct {
	Context ContextResult `json:"context"`
	Value   uint64        `json:"value"`
}

// ProgramAccountEntry represents one entry in getProgramAccounts.
type ProgramAccountEntry struct {
	Account AccountInfo `json:"account"`
	Pubkey  string      `json:"pubkey"`
}
