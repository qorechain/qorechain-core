package pqc

import (
	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
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

	// EVM-lane authorisation window. ConsumeEVMWindow admits one EVM transaction
	// against the sender's window or refuses it; see x/pqc/types/evm_window.go.
	GetEVMWindow(ctx sdk.Context, addr sdk.AccAddress) (types.EVMAuthWindow, bool)
	SetEVMWindow(ctx sdk.Context, addr sdk.AccAddress, w types.EVMAuthWindow) error
	DeleteEVMWindow(ctx sdk.Context, addr sdk.AccAddress)
	ConsumeEVMWindow(ctx sdk.Context, addr sdk.AccAddress, outflow sdkmath.Int) error
	IncrementPQCVerifications(ctx sdk.Context)
	IncrementClassicalFallbacks(ctx sdk.Context)
	IncrementMLKEMOperations(ctx sdk.Context)
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

	// Hybrid signature methods (v1.1.0)
	GetHybridSignatureMode(ctx sdk.Context) types.HybridSignatureMode
	IncrementHybridVerifications(ctx sdk.Context)

	// Protocol features: behaviour switches for fixes that live in patched
	// dependencies. See types/features.go.
	EnableProtocolFeature(ctx sdk.Context, name string) error
	ProtocolFeatureEnabled(ctx sdk.Context, name string) bool

	// Genesis
	InitGenesis(ctx sdk.Context, gs types.GenesisState)
	ExportGenesis(ctx sdk.Context) *types.GenesisState
}
