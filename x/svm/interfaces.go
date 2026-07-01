package svm

import (
	"cosmossdk.io/log"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/svm/types"
)

// SVMKeeper defines the interface for the SVM module keeper.
// Both the full and stub implementations satisfy this interface.
type SVMKeeper interface {
	// GetAccount retrieves an SVM account by its 32-byte address.
	GetAccount(ctx sdk.Context, addr [32]byte) (*types.SVMAccount, error)

	// SetAccount stores or updates an SVM account.
	SetAccount(ctx sdk.Context, account *types.SVMAccount) error

	// DeleteAccount removes an SVM account (used by rent collection).
	DeleteAccount(ctx sdk.Context, addr [32]byte) error

	// DeployProgram deploys a BPF ELF binary and returns the program address.
	DeployProgram(ctx sdk.Context, deployer [32]byte, bytecode []byte) ([32]byte, error)

	// ExecuteProgram executes an instruction on a deployed program.
	ExecuteProgram(ctx sdk.Context, programID [32]byte, instruction []byte,
		accounts []types.AccountMeta, signers [][32]byte) (*types.ExecutionResult, error)

	// SetBankKeeper wires the x/bank keeper so wallet-account balances are
	// backed by native QOR (one balance, unified across Cosmos/EVM/SVM).
	// Called once during app init, after the keeper is constructed.
	SetBankKeeper(bk types.BankKeeper)

	// GetNativeLamports returns an SVM address's spendable balance in lamports,
	// read from x/bank (uqor × LamportsPerUqor) plus any sub-uqor dust ledger.
	GetNativeLamports(ctx sdk.Context, svmAddr [32]byte) uint64

	// FaucetCreditNative credits native lamports from the svm module account
	// (testnet faucet). Requires the module account to be funded; never mints.
	FaucetCreditNative(ctx sdk.Context, svmAddr [32]byte, lamports uint64) error

	// SetAuthenticatorResolver wires the x/abstractaccount resolver so foreign
	// wallet keys (e.g. Phantom ed25519) map to the canonical account they act for.
	SetAuthenticatorResolver(r types.AuthenticatorResolver)

	// ResolveAuthenticatedSigner verifies a foreign-scheme signature and returns
	// the SVM address of the canonical account it authenticates (requires the
	// "svm"/"all" permission), so a Phantom-signed action drives the user's
	// single unified account. ok=false if unresolved/invalid/unauthorized.
	ResolveAuthenticatedSigner(ctx sdk.Context, scheme string, pubkey, msg, sig []byte) ([32]byte, bool)

	// SVMToCosmosAddr converts a 32-byte SVM address to a native address.
	SVMToCosmosAddr(svmAddr [32]byte) sdk.AccAddress

	// CosmosToSVMAddr looks up the SVM address mapped to a native address.
	CosmosToSVMAddr(cosmosAddr sdk.AccAddress) ([32]byte, error)

	// CollectRent collects rent from a non-exempt account.
	CollectRent(ctx sdk.Context, addr [32]byte) error

	// GetMinimumBalance returns the minimum lamports for rent exemption.
	GetMinimumBalance(dataLen uint64) uint64

	// GetCurrentSlot returns the current SVM slot number.
	GetCurrentSlot(ctx sdk.Context) uint64

	// BeginBlock advances the slot to the chain height and records the block
	// hash into the recent-blockhash window (replay protection).
	BeginBlock(ctx sdk.Context)

	// IsRecentBlockhash reports whether the hash is within the last 150 slots.
	IsRecentBlockhash(ctx sdk.Context, hash []byte) bool

	// GetLatestBlockhash returns the most recent recorded block hash + its slot.
	GetLatestBlockhash(ctx sdk.Context) ([]byte, uint64)

	// GetSignaturesForAddress returns up to limit recent transaction signatures
	// involving the given address, newest first.
	GetSignaturesForAddress(ctx sdk.Context, addr [32]byte, limit int) []string

	// GetSVMTransaction returns the stored transaction record for a signature.
	GetSVMTransaction(ctx sdk.Context, signature string) (*types.SVMTxRecord, bool)

	// GetParams returns the module parameters.
	GetParams(ctx sdk.Context) types.Params

	// SetParams updates the module parameters.
	SetParams(ctx sdk.Context, params types.Params) error

	// InitGenesis initializes the module's state from genesis.
	InitGenesis(ctx sdk.Context, gs types.GenesisState)

	// ExportGenesis exports the module's current state.
	ExportGenesis(ctx sdk.Context) *types.GenesisState

	// IterateAccounts iterates over all SVM accounts.
	IterateAccounts(ctx sdk.Context, cb func(types.SVMAccount) bool)

	// GetAllAccounts returns all SVM accounts.
	GetAllAccounts(ctx sdk.Context) []types.SVMAccount

	// Logger returns the module's logger.
	Logger() log.Logger
}

// SVMExecutor is an alias for the BPF execution engine interface defined in the
// types package. It is re-exported here for backward compatibility.
type SVMExecutor = types.SVMExecutor
