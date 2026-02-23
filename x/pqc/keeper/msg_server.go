//go:build proprietary

package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/pqc/types"
)

type msgServer struct {
	keeper Keeper
}

// NewMsgServer returns an implementation of the MsgServer interface.
func NewMsgServer(keeper Keeper) *msgServer {
	return &msgServer{keeper: keeper}
}

// RegisterPQCKey handles the legacy MsgRegisterPQCKey message.
// Defaults to Dilithium-5 (AlgorithmID=1) for backward compatibility.
func (s *msgServer) RegisterPQCKey(goCtx context.Context, msg *types.MsgRegisterPQCKey) error {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if s.keeper.HasPQCAccount(ctx, msg.Sender) {
		return types.ErrAccountAlreadyExists
	}

	info := types.PQCAccountInfo{
		Address:         msg.Sender,
		PublicKey:       msg.DilithiumPubkey,
		AlgorithmID:     types.AlgorithmDilithium5, // Legacy default
		ECDSAPubkey:     msg.ECDSAPubkey,
		KeyType:         msg.KeyType,
		CreatedAtHeight: ctx.BlockHeight(),
	}

	if err := s.keeper.SetPQCAccount(ctx, info); err != nil {
		return err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"pqc_key_registered",
		sdk.NewAttribute("address", msg.Sender),
		sdk.NewAttribute("key_type", msg.KeyType),
		sdk.NewAttribute("algorithm_id", types.AlgorithmDilithium5.String()),
	))

	return nil
}

// RegisterPQCKeyV2 handles the algorithm-aware MsgRegisterPQCKeyV2 message.
func (s *msgServer) RegisterPQCKeyV2(goCtx context.Context, msg *types.MsgRegisterPQCKeyV2) error {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if s.keeper.HasPQCAccount(ctx, msg.Sender) {
		return types.ErrAccountAlreadyExists
	}

	// Verify algorithm exists and is active
	algo, err := s.keeper.GetAlgorithm(ctx, msg.AlgorithmID)
	if err != nil {
		return err
	}
	if algo.Status != types.StatusActive {
		return types.ErrAlgorithmNotActive.Wrapf("algorithm %s is %s", algo.Name, algo.Status)
	}

	// Validate key size matches algorithm spec
	if msg.KeyType != types.KeyTypeClassicalOnly {
		if uint32(len(msg.PublicKey)) != algo.PublicKeySize {
			return types.ErrInvalidKeyLength.Wrapf(
				"expected %d bytes for %s public key, got %d",
				algo.PublicKeySize, algo.Name, len(msg.PublicKey),
			)
		}
	}

	info := types.PQCAccountInfo{
		Address:         msg.Sender,
		PublicKey:       msg.PublicKey,
		AlgorithmID:     msg.AlgorithmID,
		ECDSAPubkey:     msg.ECDSAPubkey,
		KeyType:         msg.KeyType,
		CreatedAtHeight: ctx.BlockHeight(),
	}

	if err := s.keeper.SetPQCAccount(ctx, info); err != nil {
		return err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"pqc_key_registered",
		sdk.NewAttribute("address", msg.Sender),
		sdk.NewAttribute("key_type", msg.KeyType),
		sdk.NewAttribute("algorithm_id", msg.AlgorithmID.String()),
	))

	return nil
}

// MigratePQCKey handles dual-signature key migration.
func (s *msgServer) MigratePQCKey(goCtx context.Context, msg *types.MsgMigratePQCKey) error {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Get existing account
	acct, found := s.keeper.GetPQCAccount(ctx, msg.Sender)
	if !found {
		return types.ErrAccountNotFound
	}

	// Verify old key matches
	if string(acct.PublicKey) != string(msg.OldPublicKey) {
		return types.ErrKeyMigrationFailed.Wrap("old public key does not match registered key")
	}

	// Check migration is active for the account's current algorithm
	migration, hasMigration := s.keeper.GetMigration(ctx, acct.AlgorithmID)
	if !hasMigration {
		return types.ErrMigrationNotActive.Wrapf("no active migration for algorithm %s", acct.AlgorithmID)
	}

	// Verify the new algorithm matches the migration target
	if msg.NewAlgorithmID != migration.ToAlgorithmID {
		return types.ErrKeyMigrationFailed.Wrapf(
			"migration target is %s, not %s",
			migration.ToAlgorithmID, msg.NewAlgorithmID,
		)
	}

	// Verify new algorithm is active
	newAlgo, err := s.keeper.GetAlgorithm(ctx, msg.NewAlgorithmID)
	if err != nil {
		return err
	}
	if newAlgo.Status != types.StatusActive {
		return types.ErrAlgorithmNotActive.Wrapf("target algorithm %s is %s", newAlgo.Name, newAlgo.Status)
	}

	// Verify both signatures using the FFI client
	pqcClient := s.keeper.PQCClient()

	// Verify old signature with old algorithm
	migrationMsg := []byte("migrate:" + msg.Sender)
	oldValid, err := pqcClient.Verify(acct.AlgorithmID, msg.OldPublicKey, migrationMsg, msg.OldSignature)
	if err != nil {
		return types.ErrDualSigInvalid.Wrap(err.Error())
	}
	if !oldValid {
		return types.ErrDualSigInvalid.Wrap("old key signature verification failed")
	}

	// Verify new signature with new algorithm
	newValid, err := pqcClient.Verify(msg.NewAlgorithmID, msg.NewPublicKey, migrationMsg, msg.NewSignature)
	if err != nil {
		return types.ErrDualSigInvalid.Wrap(err.Error())
	}
	if !newValid {
		return types.ErrDualSigInvalid.Wrap("new key signature verification failed")
	}

	// Update account with new key
	acct.PublicKey = msg.NewPublicKey
	acct.AlgorithmID = msg.NewAlgorithmID
	acct.MigrationPublicKey = nil
	acct.MigrationAlgorithmID = 0

	if err := s.keeper.SetPQCAccount(ctx, acct); err != nil {
		return err
	}

	// Update migration counters
	migration.MigratedAccounts++
	if migration.RemainingAccounts > 0 {
		migration.RemainingAccounts--
	}
	if err := s.keeper.SetMigration(ctx, migration); err != nil {
		return err
	}

	// Update stats
	stats := s.keeper.GetStats(ctx)
	stats.TotalKeyMigrations++
	stats.TotalDualSigVerifies += 2
	s.keeper.SetStats(ctx, stats)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"pqc_key_migrated",
		sdk.NewAttribute("address", msg.Sender),
		sdk.NewAttribute("from_algorithm", acct.AlgorithmID.String()),
		sdk.NewAttribute("to_algorithm", msg.NewAlgorithmID.String()),
	))

	return nil
}

// AddAlgorithm handles governance-approved algorithm addition.
func (s *msgServer) AddAlgorithm(goCtx context.Context, msg *types.MsgAddAlgorithm) error {
	ctx := sdk.UnwrapSDKContext(goCtx)

	algo := msg.Algorithm
	algo.AddedAtHeight = ctx.BlockHeight()
	algo.Status = types.StatusActive

	return s.keeper.RegisterAlgorithm(ctx, algo)
}

// DeprecateAlgorithm starts a migration period for an algorithm.
func (s *msgServer) DeprecateAlgorithm(goCtx context.Context, msg *types.MsgDeprecateAlgorithm) error {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Verify algorithm exists
	algo, err := s.keeper.GetAlgorithm(ctx, msg.AlgorithmID)
	if err != nil {
		return err
	}
	if algo.Status != types.StatusActive {
		return types.ErrAlgorithmNotActive.Wrapf("algorithm %s is already %s", algo.Name, algo.Status)
	}

	// Verify replacement algorithm exists and is active
	replacement, err := s.keeper.GetAlgorithm(ctx, msg.ReplacementAlgID)
	if err != nil {
		return err
	}
	if replacement.Status != types.StatusActive {
		return types.ErrAlgorithmNotActive.Wrapf("replacement algorithm %s is not active", replacement.Name)
	}

	// Check no existing migration for this algorithm
	if _, exists := s.keeper.GetMigration(ctx, msg.AlgorithmID); exists {
		return types.ErrMigrationActive
	}

	// Set algorithm to migrating status
	if err := s.keeper.UpdateAlgorithmStatus(ctx, msg.AlgorithmID, types.StatusMigrating); err != nil {
		return err
	}

	// Create migration record
	migrationBlocks := msg.MigrationBlocks
	if migrationBlocks <= 0 {
		migrationBlocks = types.DefaultMigrationBlocks
	}

	migration := types.MigrationInfo{
		FromAlgorithmID: msg.AlgorithmID,
		ToAlgorithmID:   msg.ReplacementAlgID,
		StartHeight:     ctx.BlockHeight(),
		EndHeight:       ctx.BlockHeight() + migrationBlocks,
	}

	return s.keeper.SetMigration(ctx, migration)
}

// DisableAlgorithm emergency-disables an algorithm.
func (s *msgServer) DisableAlgorithm(goCtx context.Context, msg *types.MsgDisableAlgorithm) error {
	ctx := sdk.UnwrapSDKContext(goCtx)

	algo, err := s.keeper.GetAlgorithm(ctx, msg.AlgorithmID)
	if err != nil {
		return err
	}

	if algo.Status == types.StatusDisabled {
		return types.ErrAlgorithmDisabled.Wrapf("algorithm %s is already disabled", algo.Name)
	}

	if err := s.keeper.UpdateAlgorithmStatus(ctx, msg.AlgorithmID, types.StatusDisabled); err != nil {
		return err
	}

	// Clean up any active migration
	s.keeper.DeleteMigration(ctx, msg.AlgorithmID)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"pqc_algorithm_disabled",
		sdk.NewAttribute("algorithm_id", msg.AlgorithmID.String()),
		sdk.NewAttribute("reason", msg.Reason),
	))

	return nil
}
