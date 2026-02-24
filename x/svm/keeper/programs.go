//go:build proprietary

package keeper

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/svm/types"
)

// DeployProgram validates and deploys a BPF ELF binary. It creates both the
// executable program account and the data account that holds the bytecode,
// then stores the program metadata.
//
// Returns the deterministic 32-byte program address derived from the deployer
// and bytecode hash.
func (k *Keeper) DeployProgram(ctx sdk.Context, deployer [32]byte, bytecode []byte) ([32]byte, error) {
	params := k.GetParams(ctx)
	if !params.Enabled {
		return [32]byte{}, types.ErrSVMDisabled
	}

	// Validate bytecode size.
	if uint64(len(bytecode)) > params.MaxProgramSize {
		return [32]byte{}, types.ErrInvalidBytecode.Wrapf(
			"bytecode size %d exceeds max %d", len(bytecode), params.MaxProgramSize)
	}

	// Validate the ELF binary via the executor if available.
	if k.executor != nil {
		if err := k.executor.ValidateProgram(bytecode); err != nil {
			return [32]byte{}, types.ErrInvalidBytecode.Wrapf("ELF validation failed: %v", err)
		}
	}

	// Derive deterministic program address: SHA-256(deployer || bytecodeHash).
	bcHash := sha256.Sum256(bytecode)
	seed := make([]byte, 0, 64)
	seed = append(seed, deployer[:]...)
	seed = append(seed, bcHash[:]...)
	programAddr := sha256.Sum256(seed)

	// Ensure program does not already exist.
	if k.HasAccount(ctx, programAddr) {
		return [32]byte{}, types.ErrAccountAlreadyExists.Wrapf(
			"program %s already deployed", types.Base58Encode(programAddr))
	}

	// Derive data account address: SHA-256(programAddr || "data").
	dataSeed := make([]byte, 0, 36)
	dataSeed = append(dataSeed, programAddr[:]...)
	dataSeed = append(dataSeed, []byte("data")...)
	dataAddr := sha256.Sum256(dataSeed)

	currentSlot := k.GetCurrentSlot(ctx)

	// Create the data account that holds the raw bytecode.
	dataAccount := &types.SVMAccount{
		Address:    dataAddr,
		Lamports:   k.GetMinimumBalance(uint64(len(bytecode))),
		DataLen:    uint64(len(bytecode)),
		Data:       bytecode,
		Owner:      programAddr,
		Executable: false,
		RentEpoch:  0,
	}
	if err := k.SetAccount(ctx, dataAccount); err != nil {
		return [32]byte{}, fmt.Errorf("failed to create data account: %w", err)
	}

	// Create the executable program account (no data, just a marker).
	programAccount := &types.SVMAccount{
		Address:    programAddr,
		Lamports:   1,
		DataLen:    0,
		Data:       []byte{},
		Owner:      types.SystemProgramAddress,
		Executable: true,
		RentEpoch:  0,
	}
	if err := k.SetAccount(ctx, programAccount); err != nil {
		return [32]byte{}, fmt.Errorf("failed to create program account: %w", err)
	}

	// Store program metadata.
	meta := types.ProgramMeta{
		ProgramAddress:   programAddr,
		UpgradeAuthority: deployer,
		DeploySlot:       currentSlot,
		LastDeploySlot:   currentSlot,
		DataAccount:      dataAddr,
	}
	if err := k.SetProgramMeta(ctx, meta); err != nil {
		return [32]byte{}, fmt.Errorf("failed to store program metadata: %w", err)
	}

	k.logger.Info("program deployed",
		"program", types.Base58Encode(programAddr),
		"deployer", types.Base58Encode(deployer),
		"bytecode_len", len(bytecode),
		"slot", currentSlot,
	)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"svm_deploy_program",
		sdk.NewAttribute("program_address", types.Base58Encode(programAddr)),
		sdk.NewAttribute("deployer", types.Base58Encode(deployer)),
		sdk.NewAttribute("bytecode_size", fmt.Sprintf("%d", len(bytecode))),
	))

	return programAddr, nil
}

// GetProgramMeta reads program metadata from the KVStore.
func (k *Keeper) GetProgramMeta(ctx sdk.Context, addr [32]byte) (*types.ProgramMeta, error) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ProgramKey(addr))
	if bz == nil {
		return nil, types.ErrProgramNotFound.Wrapf("program %s not found",
			types.Base58Encode(addr))
	}
	var meta types.ProgramMeta
	if err := json.Unmarshal(bz, &meta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal program metadata: %w", err)
	}
	return &meta, nil
}

// SetProgramMeta writes program metadata to the KVStore.
func (k *Keeper) SetProgramMeta(ctx sdk.Context, meta types.ProgramMeta) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("failed to marshal program metadata: %w", err)
	}
	store.Set(types.ProgramKey(meta.ProgramAddress), bz)
	return nil
}

// GetAllProgramMetas returns all program metadata entries from the store.
func (k *Keeper) GetAllProgramMetas(ctx sdk.Context) []types.ProgramMeta {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.ProgramKeyPrefix)
	defer iter.Close()

	var metas []types.ProgramMeta
	for ; iter.Valid(); iter.Next() {
		var meta types.ProgramMeta
		if err := json.Unmarshal(iter.Value(), &meta); err != nil {
			k.logger.Error("failed to unmarshal program meta during iteration", "error", err)
			continue
		}
		metas = append(metas, meta)
	}
	return metas
}

// LoadProgramBytecode fetches the bytecode for a deployed program by reading
// the data account referenced in the program metadata.
func (k *Keeper) LoadProgramBytecode(ctx sdk.Context, programAddr [32]byte) ([]byte, error) {
	meta, err := k.GetProgramMeta(ctx, programAddr)
	if err != nil {
		return nil, err
	}
	dataAcc, err := k.GetAccount(ctx, meta.DataAccount)
	if err != nil {
		return nil, fmt.Errorf("failed to load data account for program %s: %w",
			types.Base58Encode(programAddr), err)
	}
	if len(dataAcc.Data) == 0 {
		return nil, types.ErrInvalidBytecode.Wrapf(
			"data account %s has no bytecode", types.Base58Encode(meta.DataAccount))
	}
	return dataAcc.Data, nil
}
