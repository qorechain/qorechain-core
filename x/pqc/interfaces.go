package pqc

import (
	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/pqc/types"
)

// PQCClient is the interface for all post-quantum cryptographic operations.
// The production implementation calls into the Rust FFI library (libqorepqc).
type PQCClient interface {
	DilithiumKeygen() (pubkey []byte, privkey []byte, err error)
	DilithiumSign(privkey []byte, message []byte) (signature []byte, err error)
	DilithiumVerify(pubkey []byte, message []byte, signature []byte) (bool, error)

	MLKEMKeygen() (pubkey []byte, privkey []byte, err error)
	MLKEMEncapsulate(pubkey []byte) (ciphertext []byte, sharedSecret []byte, err error)
	MLKEMDecapsulate(privkey []byte, ciphertext []byte) (sharedSecret []byte, err error)

	GenerateRandomBeacon(seed []byte, epoch uint64) ([]byte, error)

	Version() string
	Algorithms() string
}

// PQCKeeper is the interface for the x/pqc module's keeper.
// Used by the ante handler and other modules (e.g., x/bridge).
type PQCKeeper interface {
	PQCClient() PQCClient
	Logger() log.Logger

	GetParams(ctx sdk.Context) types.Params
	SetParams(ctx sdk.Context, params types.Params) error
	GetPQCAccount(ctx sdk.Context, address string) (types.PQCAccountInfo, bool)
	HasPQCAccount(ctx sdk.Context, address string) bool
	SetPQCAccount(ctx sdk.Context, info types.PQCAccountInfo) error
	IncrementPQCVerifications(ctx sdk.Context)
	IncrementClassicalFallbacks(ctx sdk.Context)
	GetStats(ctx sdk.Context) types.PQCStats
	SetStats(ctx sdk.Context, stats types.PQCStats)

	InitGenesis(ctx sdk.Context, gs types.GenesisState)
	ExportGenesis(ctx sdk.Context) *types.GenesisState
}
