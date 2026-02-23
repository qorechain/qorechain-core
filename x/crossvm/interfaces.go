package crossvm

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/crossvm/types"
)

// CrossVMKeeper defines the interface for the cross-VM module keeper.
// Both the proprietary and stub implementations satisfy this interface.
type CrossVMKeeper interface {
	// GetParams returns the module parameters.
	GetParams(ctx sdk.Context) types.Params

	// SetParams updates the module parameters.
	SetParams(ctx sdk.Context, params types.Params) error

	// SubmitMessage submits a new cross-VM message for processing.
	SubmitMessage(ctx sdk.Context, msg types.CrossVMMessage) (string, error)

	// GetMessage retrieves a cross-VM message by ID.
	GetMessage(ctx sdk.Context, id string) (types.CrossVMMessage, bool)

	// GetPendingMessages returns all pending messages in the queue.
	GetPendingMessages(ctx sdk.Context) []types.CrossVMMessage

	// ProcessQueue processes pending cross-VM messages.
	ProcessQueue(ctx sdk.Context) error

	// ExecuteSyncCall performs a synchronous cross-VM call (used by precompile).
	ExecuteSyncCall(ctx sdk.Context, msg types.CrossVMMessage) (types.CrossVMResponse, error)

	// InitGenesis initializes the module's state from genesis.
	InitGenesis(ctx sdk.Context, gs types.GenesisState)

	// ExportGenesis exports the module's current state.
	ExportGenesis(ctx sdk.Context) *types.GenesisState
}
