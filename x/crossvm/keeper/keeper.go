//go:build proprietary

package keeper

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	evmkeeper "github.com/cosmos/evm/x/vm/keeper"

	"github.com/qorechain/qorechain-core/x/crossvm/types"
)

// SVMCallFunc is the callback signature for routing cross-VM calls to the SVM runtime.
// It is set once during app initialization via SetSVMCallHandler.
type SVMCallFunc func(ctx sdk.Context, targetContract string, payload []byte, sender string) ([]byte, error)

// svmCallHandler is the package-level SVM call handler.
// Set during app initialization; read-only after startup.
var svmCallHandler SVMCallFunc

// SetSVMCallHandler registers the SVM call handler for cross-VM routing.
// Must be called during app initialization, before any blocks are processed.
func SetSVMCallHandler(fn SVMCallFunc) {
	svmCallHandler = fn
}

// Keeper manages the cross-VM message store and orchestrates calls between VMs.
type Keeper struct {
	cdc            codec.Codec
	storeKey       storetypes.StoreKey
	evmKeeper      *evmkeeper.Keeper
	wasmContractKp *wasmkeeper.PermissionedKeeper
	logger         log.Logger
}

func NewKeeper(
	cdc codec.Codec,
	storeKey storetypes.StoreKey,
	evmKeeper *evmkeeper.Keeper,
	wasmKeeper *wasmkeeper.Keeper,
	logger log.Logger,
) Keeper {
	return Keeper{
		cdc:            cdc,
		storeKey:       storeKey,
		evmKeeper:      evmKeeper,
		wasmContractKp: wasmkeeper.NewDefaultPermissionKeeper(wasmKeeper),
		logger:         logger.With("module", types.ModuleName),
	}
}

func (k Keeper) Logger() log.Logger {
	return k.logger
}

// GetParams returns the module parameters from the store.
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get([]byte(types.ParamsKey))
	if bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	if err := json.Unmarshal(bz, &params); err != nil {
		k.logger.Error("failed to unmarshal crossvm params", "error", err)
		return types.DefaultParams()
	}
	return params
}

// SetParams stores the module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) error {
	if err := params.Validate(); err != nil {
		return err
	}
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to marshal crossvm params: %w", err)
	}
	store.Set([]byte(types.ParamsKey), bz)
	return nil
}

// generateMessageID creates a deterministic message ID from the message content and block height.
func generateMessageID(msg types.CrossVMMessage, height int64) string {
	data := fmt.Sprintf("%s:%s:%s:%s:%d", msg.SourceVM, msg.TargetVM, msg.TargetContract, msg.Sender, height)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:16])
}

// SubmitMessage stores a new cross-VM message and adds it to the queue.
func (k Keeper) SubmitMessage(ctx sdk.Context, msg types.CrossVMMessage) (string, error) {
	params := k.GetParams(ctx)
	if !params.Enabled {
		return "", types.ErrUnsupportedVM.Wrap("cross-VM module is disabled")
	}
	if uint64(len(msg.Payload)) > params.MaxMessageSize {
		return "", types.ErrMessageTooLarge.Wrapf("payload %d exceeds max %d", len(msg.Payload), params.MaxMessageSize)
	}

	msg.ID = generateMessageID(msg, ctx.BlockHeight())
	msg.Status = types.StatusPending
	msg.CreatedHeight = ctx.BlockHeight()

	if err := k.storeMessage(ctx, msg); err != nil {
		return "", err
	}
	if err := k.enqueueMessage(ctx, msg.ID); err != nil {
		return "", err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeCrossVMRequest,
		sdk.NewAttribute(types.AttributeKeyMessageID, msg.ID),
		sdk.NewAttribute(types.AttributeKeySourceVM, string(msg.SourceVM)),
		sdk.NewAttribute(types.AttributeKeyTargetVM, string(msg.TargetVM)),
		sdk.NewAttribute(types.AttributeKeyTargetContract, msg.TargetContract),
		sdk.NewAttribute(types.AttributeKeySender, msg.Sender),
	))

	return msg.ID, nil
}

// GetMessage retrieves a cross-VM message by ID.
func (k Keeper) GetMessage(ctx sdk.Context, id string) (types.CrossVMMessage, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.MessageStoreKey(id))
	if bz == nil {
		return types.CrossVMMessage{}, false
	}
	msg, err := types.UnmarshalCrossVMMessage(bz)
	if err != nil {
		k.logger.Error("failed to unmarshal cross-VM message", "id", id, "error", err)
		return types.CrossVMMessage{}, false
	}
	return *msg, true
}

// GetPendingMessages returns all pending messages in the queue.
func (k Keeper) GetPendingMessages(ctx sdk.Context) []types.CrossVMMessage {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, []byte(types.QueueKeyPrefix))
	defer iter.Close()

	var messages []types.CrossVMMessage
	for ; iter.Valid(); iter.Next() {
		msgID := string(iter.Value())
		msg, found := k.GetMessage(ctx, msgID)
		if found && msg.Status == types.StatusPending {
			messages = append(messages, msg)
		}
	}
	return messages
}

// ProcessQueue processes all pending cross-VM messages.
func (k Keeper) ProcessQueue(ctx sdk.Context) error {
	params := k.GetParams(ctx)
	pending := k.GetPendingMessages(ctx)

	for _, msg := range pending {
		// Check for timeout
		if ctx.BlockHeight()-msg.CreatedHeight > params.QueueTimeoutBlocks {
			msg.Status = types.StatusTimedOut
			msg.Error = "message timed out"
			_ = k.storeMessage(ctx, msg)
			k.removeFromQueue(ctx, msg.ID)

			ctx.EventManager().EmitEvent(sdk.NewEvent(
				types.EventTypeCrossVMTimeout,
				sdk.NewAttribute(types.AttributeKeyMessageID, msg.ID),
			))
			continue
		}

		// Execute the cross-VM call
		resp, err := k.executeCrossVMCall(ctx, msg)
		if err != nil {
			msg.Status = types.StatusFailed
			msg.Error = err.Error()
			msg.ExecutedHeight = ctx.BlockHeight()
			_ = k.storeMessage(ctx, msg)
		} else {
			msg.Status = types.StatusExecuted
			msg.Response = resp.Data
			msg.ExecutedHeight = ctx.BlockHeight()
			_ = k.storeMessage(ctx, msg)
		}

		k.removeFromQueue(ctx, msg.ID)

		ctx.EventManager().EmitEvent(sdk.NewEvent(
			types.EventTypeCrossVMResponse,
			sdk.NewAttribute(types.AttributeKeyMessageID, msg.ID),
			sdk.NewAttribute(types.AttributeKeyStatus, string(msg.Status)),
		))
	}
	return nil
}

// ExecuteSyncCall performs a synchronous cross-VM call (used by the EVM precompile).
func (k Keeper) ExecuteSyncCall(ctx sdk.Context, msg types.CrossVMMessage) (types.CrossVMResponse, error) {
	params := k.GetParams(ctx)
	if !params.Enabled {
		return types.CrossVMResponse{}, types.ErrUnsupportedVM.Wrap("cross-VM module is disabled")
	}
	if uint64(len(msg.Payload)) > params.MaxMessageSize {
		return types.CrossVMResponse{}, types.ErrMessageTooLarge
	}

	return k.executeCrossVMCall(ctx, msg)
}

// executeCrossVMCall routes a cross-VM call to the appropriate target VM.
func (k Keeper) executeCrossVMCall(ctx sdk.Context, msg types.CrossVMMessage) (types.CrossVMResponse, error) {
	switch msg.TargetVM {
	case types.VMTypeCosmWasm:
		return k.callCosmWasm(ctx, msg)
	case types.VMTypeEVM:
		return k.callEVM(ctx, msg)
	case types.VMTypeSVM:
		return k.callSVM(ctx, msg)
	default:
		return types.CrossVMResponse{}, types.ErrUnsupportedVM.Wrapf("unknown target VM: %s", msg.TargetVM)
	}
}

// callCosmWasm executes a CosmWasm contract call.
func (k Keeper) callCosmWasm(ctx sdk.Context, msg types.CrossVMMessage) (types.CrossVMResponse, error) {
	contractAddr, err := sdk.AccAddressFromBech32(msg.TargetContract)
	if err != nil {
		return types.CrossVMResponse{}, types.ErrInvalidTarget.Wrapf("invalid CosmWasm address: %s", err)
	}

	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return types.CrossVMResponse{}, types.ErrUnauthorized.Wrapf("invalid sender address: %s", err)
	}

	gasBefore := ctx.GasMeter().GasConsumed()
	result, err := k.wasmContractKp.Execute(ctx, contractAddr, senderAddr, msg.Payload, msg.Funds)
	gasUsed := ctx.GasMeter().GasConsumed() - gasBefore

	if err != nil {
		return types.CrossVMResponse{
			MessageID: msg.ID,
			Success:   false,
			Error:     err.Error(),
			GasUsed:   gasUsed,
		}, types.ErrWasmExecution.Wrap(err.Error())
	}

	return types.CrossVMResponse{
		MessageID: msg.ID,
		Success:   true,
		Data:      result,
		GasUsed:   gasUsed,
	}, nil
}

// callEVM executes an EVM contract call.
// This is used for CosmWasm -> EVM direction (async queue processing).
func (k Keeper) callEVM(_ sdk.Context, msg types.CrossVMMessage) (types.CrossVMResponse, error) {
	// TODO(Phase 4 follow-up): Implement EVM contract calls via EVMKeeper.
	// This requires creating an EVM message, executing it, and returning the result.
	// For now, return an error indicating this direction is not yet implemented.
	_ = k.evmKeeper
	return types.CrossVMResponse{
		MessageID: msg.ID,
		Success:   false,
		Error:     "CosmWasm -> EVM calls not yet implemented",
	}, types.ErrEVMExecution.Wrap("CosmWasm -> EVM calls not yet implemented")
}

// callSVM executes an SVM program call via the registered callback handler.
func (k Keeper) callSVM(ctx sdk.Context, msg types.CrossVMMessage) (types.CrossVMResponse, error) {
	if svmCallHandler == nil {
		return types.CrossVMResponse{
			MessageID: msg.ID,
			Success:   false,
			Error:     "SVM runtime not available",
		}, types.ErrUnsupportedVM.Wrap("SVM call handler not registered")
	}

	gasBefore := ctx.GasMeter().GasConsumed()
	result, err := svmCallHandler(ctx, msg.TargetContract, msg.Payload, msg.Sender)
	gasUsed := ctx.GasMeter().GasConsumed() - gasBefore

	if err != nil {
		return types.CrossVMResponse{
			MessageID: msg.ID,
			Success:   false,
			Error:     err.Error(),
			GasUsed:   gasUsed,
		}, nil // Return the response with error details, don't wrap
	}

	return types.CrossVMResponse{
		MessageID: msg.ID,
		Success:   true,
		Data:      result,
		GasUsed:   gasUsed,
	}, nil
}

// storeMessage persists a cross-VM message to the KV store.
func (k Keeper) storeMessage(ctx sdk.Context, msg types.CrossVMMessage) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := types.MarshalCrossVMMessage(&msg)
	if err != nil {
		return fmt.Errorf("failed to marshal cross-VM message: %w", err)
	}
	store.Set(types.MessageStoreKey(msg.ID), bz)
	return nil
}

// enqueueMessage adds a message ID to the pending queue.
func (k Keeper) enqueueMessage(ctx sdk.Context, id string) error {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.QueueStoreKey(id), []byte(id))
	return nil
}

// removeFromQueue removes a message ID from the pending queue.
func (k Keeper) removeFromQueue(ctx sdk.Context, id string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.QueueStoreKey(id))
}
