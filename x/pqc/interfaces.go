package pqc

import (
	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/pqc/types"
)

// PQCClient is the interface for all post-quantum cryptographic operations.
// The production implementation calls into the Rust FFI library (libqorepqc).
type PQCClient interface {
	// Legacy algorithm-specific operations (backward compatibility)
	DilithiumKeygen() (pubkey []byte, privkey []byte, err error)
	DilithiumSign(privkey []byte, message []byte) (signature []byte, err error)
	DilithiumVerify(pubkey []byte, message []byte, signature []byte) (bool, error)

	MLKEMKeygen() (pubkey []byte, privkey []byte, err error)
	MLKEMEncapsulate(pubkey []byte) (ciphertext []byte, sharedSecret []byte, err error)
	MLKEMDecapsulate(privkey []byte, ciphertext []byte) (sharedSecret []byte, err error)

	GenerateRandomBeacon(seed []byte, epoch uint64) ([]byte, error)

	// Algorithm-aware operations (v0.6.0)
	Keygen(algorithmID types.AlgorithmID) (pubkey []byte, privkey []byte, err error)
	Sign(algorithmID types.AlgorithmID, privkey []byte, message []byte) (signature []byte, err error)
	Verify(algorithmID types.AlgorithmID, pubkey []byte, message []byte, signature []byte) (bool, error)
	AlgorithmInfo(algorithmID types.AlgorithmID) (pubkeySize, privkeySize, outputSize uint32, err error)
	ListAlgorithms() ([]types.AlgorithmID, error)

	Version() string
	Algorithms() string
}

// PQCVerifier is the interface for algorithm-specific signature verification.
// Registered at app startup; the ante decorator dispatches to the correct
// verifier by looking up the account's algorithm ID.
type PQCVerifier interface {
	// Verify checks a PQC signature against a public key and message.
	Verify(pubkey []byte, message []byte, signature []byte) (bool, error)
	// Algorithm returns the algorithm ID this verifier handles.
	Algorithm() types.AlgorithmID
}

// PQCKeeper is the interface for the x/pqc module's keeper.
// Used by the ante handler and other modules (e.g., x/bridge).
type PQCKeeper interface {
	PQCClient() PQCClient
	Logger() log.Logger

	// Params
	GetParams(ctx sdk.Context) types.Params
	SetParams(ctx sdk.Context, params types.Params) error

	// Account management
	GetPQCAccount(ctx sdk.Context, address string) (types.PQCAccountInfo, bool)
	HasPQCAccount(ctx sdk.Context, address string) bool
	SetPQCAccount(ctx sdk.Context, info types.PQCAccountInfo) error
	IncrementPQCVerifications(ctx sdk.Context)
	IncrementClassicalFallbacks(ctx sdk.Context)
	GetStats(ctx sdk.Context) types.PQCStats
	SetStats(ctx sdk.Context, stats types.PQCStats)

	// Algorithm registry (v0.6.0)
	RegisterAlgorithm(ctx sdk.Context, algo types.AlgorithmInfo) error
	GetAlgorithm(ctx sdk.Context, id types.AlgorithmID) (types.AlgorithmInfo, error)
	ListAlgorithms(ctx sdk.Context) []types.AlgorithmInfo
	UpdateAlgorithmStatus(ctx sdk.Context, id types.AlgorithmID, status types.AlgorithmStatus) error
	GetActiveSignatureAlgorithms(ctx sdk.Context) []types.AlgorithmInfo
	GetActiveKEMAlgorithms(ctx sdk.Context) []types.AlgorithmInfo

	// Migration (v0.6.0)
	GetMigration(ctx sdk.Context, fromID types.AlgorithmID) (types.MigrationInfo, bool)
	SetMigration(ctx sdk.Context, migration types.MigrationInfo) error
	DeleteMigration(ctx sdk.Context, fromID types.AlgorithmID)

	// Genesis
	InitGenesis(ctx sdk.Context, gs types.GenesisState)
	ExportGenesis(ctx sdk.Context) *types.GenesisState
}
