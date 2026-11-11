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
	// PDASigners are the program derived addresses the top-level program
	// signed for with sol_invoke_signed during this execution, as the runtime
	// derived them from the program id it holds (svm-cpi-host-privileges).
	// Empty without that switch. Not part of any encoded output.
	PDASigners [][32]byte `json:"-"`
	// Attested is true when the runtime reported the attestation of
	// svm-cpi-composable: InvokedPrograms (every program that ran, the
	// top-level one first) and PDAPairs ({program, pda} for every PDA a program
	// signed for), both from host memory. Not part of any encoded output.
	Attested        bool          `json:"-"`
	InvokedPrograms [][32]byte    `json:"-"`
	PDAPairs        [][2][32]byte `json:"-"`
}
