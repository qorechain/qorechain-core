package types

// ProgramMeta stores metadata about a deployed SVM program.
type ProgramMeta struct {
	ProgramAddress   [32]byte `json:"program_address"`
	UpgradeAuthority [32]byte `json:"upgrade_authority"` // zero = immutable
	DeploySlot       uint64   `json:"deploy_slot"`
	LastDeploySlot   uint64   `json:"last_deploy_slot"`
	DataAccount      [32]byte `json:"data_account"` // where BPF bytecode is stored
}

// ExecutionResult holds the outcome of an SVM program execution.
type ExecutionResult struct {
	Success          bool         `json:"success"`
	ReturnData       []byte       `json:"return_data"`
	ComputeUnitsUsed uint64       `json:"compute_units_used"`
	Logs             []string     `json:"logs"`
	ModifiedAccounts []SVMAccount `json:"modified_accounts"`
	Error            string       `json:"error,omitempty"`
}
