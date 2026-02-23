//go:build !proprietary

package pqc

import (
	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/pqc/types"
)

// stubPQCClient is a no-op implementation of PQCClient for public builds.
type stubPQCClient struct{}

func (stubPQCClient) DilithiumKeygen() ([]byte, []byte, error) {
	return make([]byte, 2592), make([]byte, 4896), nil
}
func (stubPQCClient) DilithiumSign(_ []byte, _ []byte) ([]byte, error) {
	return make([]byte, 4627), nil
}
func (stubPQCClient) DilithiumVerify(_ []byte, _ []byte, _ []byte) (bool, error) {
	return false, nil
}
func (stubPQCClient) MLKEMKeygen() ([]byte, []byte, error) {
	return make([]byte, 1568), make([]byte, 3168), nil
}
func (stubPQCClient) MLKEMEncapsulate(_ []byte) ([]byte, []byte, error) {
	return make([]byte, 1568), make([]byte, 32), nil
}
func (stubPQCClient) MLKEMDecapsulate(_ []byte, _ []byte) ([]byte, error) {
	return make([]byte, 32), nil
}
func (stubPQCClient) GenerateRandomBeacon(_ []byte, _ uint64) ([]byte, error) {
	return make([]byte, 32), nil
}

// Algorithm-aware stubs (v0.6.0)
func (stubPQCClient) Keygen(_ types.AlgorithmID) ([]byte, []byte, error) {
	return nil, nil, types.ErrInvalidAlgorithm.Wrap("PQC not available in community build")
}
func (stubPQCClient) Sign(_ types.AlgorithmID, _ []byte, _ []byte) ([]byte, error) {
	return nil, types.ErrInvalidAlgorithm.Wrap("PQC not available in community build")
}
func (stubPQCClient) Verify(_ types.AlgorithmID, _ []byte, _ []byte, _ []byte) (bool, error) {
	return false, nil
}
func (stubPQCClient) AlgorithmInfo(_ types.AlgorithmID) (uint32, uint32, uint32, error) {
	return 0, 0, 0, types.ErrInvalidAlgorithm.Wrap("PQC not available in community build")
}
func (stubPQCClient) ListAlgorithms() ([]types.AlgorithmID, error) {
	return []types.AlgorithmID{types.AlgorithmDilithium5, types.AlgorithmMLKEM1024}, nil
}
func (stubPQCClient) Version() string    { return "stub-0.6.0" }
func (stubPQCClient) Algorithms() string { return "dilithium5,mlkem1024 (stub)" }

// NewStubPQCClient returns a no-op PQCClient for public builds.
func NewStubPQCClient() PQCClient {
	return stubPQCClient{}
}

// StubKeeper is a no-op implementation of PQCKeeper for public builds.
type StubKeeper struct {
	client PQCClient
	logger log.Logger
}

// NewStubKeeper creates a new stub PQC keeper.
func NewStubKeeper(logger log.Logger) *StubKeeper {
	return &StubKeeper{
		client: stubPQCClient{},
		logger: logger.With("module", types.ModuleName),
	}
}

func (k *StubKeeper) PQCClient() PQCClient               { return k.client }
func (k *StubKeeper) Logger() log.Logger                  { return k.logger }
func (k *StubKeeper) GetParams(_ sdk.Context) types.Params { return types.DefaultParams() }
func (k *StubKeeper) SetParams(_ sdk.Context, _ types.Params) error { return nil }
func (k *StubKeeper) GetPQCAccount(_ sdk.Context, _ string) (types.PQCAccountInfo, bool) {
	return types.PQCAccountInfo{}, false
}
func (k *StubKeeper) HasPQCAccount(_ sdk.Context, _ string) bool                { return false }
func (k *StubKeeper) SetPQCAccount(_ sdk.Context, _ types.PQCAccountInfo) error { return nil }
func (k *StubKeeper) IncrementPQCVerifications(_ sdk.Context)                   {}
func (k *StubKeeper) IncrementClassicalFallbacks(_ sdk.Context)                 {}
func (k *StubKeeper) GetStats(_ sdk.Context) types.PQCStats                     { return types.PQCStats{} }
func (k *StubKeeper) SetStats(_ sdk.Context, _ types.PQCStats)                  {}

// Algorithm registry stubs (v0.6.0)
func (k *StubKeeper) RegisterAlgorithm(_ sdk.Context, _ types.AlgorithmInfo) error { return nil }
func (k *StubKeeper) GetAlgorithm(_ sdk.Context, id types.AlgorithmID) (types.AlgorithmInfo, error) {
	// Return default info for built-in algorithms
	switch id {
	case types.AlgorithmDilithium5:
		return types.DefaultDilithium5Info(), nil
	case types.AlgorithmMLKEM1024:
		return types.DefaultMLKEM1024Info(), nil
	default:
		return types.AlgorithmInfo{}, types.ErrInvalidAlgorithm.Wrapf("algorithm %d not found", id)
	}
}
func (k *StubKeeper) ListAlgorithms(_ sdk.Context) []types.AlgorithmInfo {
	return []types.AlgorithmInfo{types.DefaultDilithium5Info(), types.DefaultMLKEM1024Info()}
}
func (k *StubKeeper) UpdateAlgorithmStatus(_ sdk.Context, _ types.AlgorithmID, _ types.AlgorithmStatus) error {
	return nil
}
func (k *StubKeeper) GetActiveSignatureAlgorithms(_ sdk.Context) []types.AlgorithmInfo {
	return []types.AlgorithmInfo{types.DefaultDilithium5Info()}
}
func (k *StubKeeper) GetActiveKEMAlgorithms(_ sdk.Context) []types.AlgorithmInfo {
	return []types.AlgorithmInfo{types.DefaultMLKEM1024Info()}
}

// Migration stubs (v0.6.0)
func (k *StubKeeper) GetMigration(_ sdk.Context, _ types.AlgorithmID) (types.MigrationInfo, bool) {
	return types.MigrationInfo{}, false
}
func (k *StubKeeper) SetMigration(_ sdk.Context, _ types.MigrationInfo) error { return nil }
func (k *StubKeeper) DeleteMigration(_ sdk.Context, _ types.AlgorithmID)      {}

// Genesis
func (k *StubKeeper) InitGenesis(_ sdk.Context, _ types.GenesisState) {}
func (k *StubKeeper) ExportGenesis(_ sdk.Context) *types.GenesisState {
	return types.DefaultGenesisState()
}
