//go:build proprietary

package keeper

import (
	"crypto/sha256"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/svm/types"
)

// MsgServer wraps the keeper and implements message handler methods.
type MsgServer struct {
	keeper *Keeper
}

// NewMsgServerImpl returns a new MsgServer backed by the given keeper.
func NewMsgServerImpl(keeper *Keeper) *MsgServer {
	return &MsgServer{keeper: keeper}
}

// HandleMsgDeployProgram validates and deploys a BPF program to the SVM runtime.
func (s *MsgServer) HandleMsgDeployProgram(ctx sdk.Context, msg *types.MsgDeployProgram) ([32]byte, error) {
	if err := msg.ValidateBasic(); err != nil {
		return [32]byte{}, err
	}

	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return [32]byte{}, types.ErrInvalidAddress.Wrapf("invalid sender: %v", err)
	}

	// Derive the deployer SVM address from the native sender address.
	deployerSVM, err := s.keeper.cosmosToSVMAddrFromStore(ctx, senderAddr)
	if err != nil {
		// If the sender has no SVM account yet, derive a deterministic one.
		var derived [32]byte
		copy(derived[:20], senderAddr)
		deployerSVM = derived
	}

	programAddr, err := s.keeper.DeployProgram(ctx, deployerSVM, msg.Bytecode)
	if err != nil {
		return [32]byte{}, err
	}

	s.keeper.logger.Info("MsgDeployProgram handled",
		"sender", msg.Sender,
		"program", types.Base58Encode(programAddr),
	)

	return programAddr, nil
}

// HandleMsgExecuteProgram executes an instruction on a deployed SVM program.
func (s *MsgServer) HandleMsgExecuteProgram(ctx sdk.Context, msg *types.MsgExecuteProgram) (*types.ExecutionResult, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, types.ErrInvalidAddress.Wrapf("invalid sender: %v", err)
	}

	// Resolve the sender's SVM address for signer verification.
	senderSVM, err := s.keeper.cosmosToSVMAddrFromStore(ctx, senderAddr)
	if err != nil {
		// If the sender has no SVM mapping, derive a deterministic address.
		var derived [32]byte
		copy(derived[:20], senderAddr)
		senderSVM = derived
	}

	signers := [][32]byte{senderSVM}

	result, err := s.keeper.ExecuteProgram(ctx, msg.ProgramID, msg.Data, msg.Accounts, signers)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// HandleMsgCreateAccount creates a new SVM data account with the specified space
// and initial lamports.
func (s *MsgServer) HandleMsgCreateAccount(ctx sdk.Context, msg *types.MsgCreateAccount) error {
	if err := msg.ValidateBasic(); err != nil {
		return err
	}

	params := s.keeper.GetParams(ctx)
	if !params.Enabled {
		return types.ErrSVMDisabled
	}

	if msg.Space > params.MaxAccountDataSize {
		return types.ErrInvalidBytecode.Wrapf(
			"requested space %d exceeds max %d", msg.Space, params.MaxAccountDataSize)
	}

	// Generate a deterministic address from the sender and owner.
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return types.ErrInvalidAddress.Wrapf("invalid sender: %v", err)
	}
	addrSeed := make([]byte, 0, 52)
	addrSeed = append(addrSeed, senderAddr...)
	addrSeed = append(addrSeed, msg.Owner[:]...)
	newAddr := sha256.Sum256(addrSeed)

	// Check that the derived address does not already exist.
	if s.keeper.HasAccount(ctx, newAddr) {
		return types.ErrAccountAlreadyExists.Wrapf("account %s already exists",
			types.Base58Encode(newAddr))
	}

	// Validate minimum balance for rent exemption.
	minBalance := computeMinimumBalance(msg.Space, params.LamportsPerByte, params.RentExemptionMulti)
	if msg.Lamports < minBalance {
		return types.ErrInsufficientLamports.Wrapf(
			"lamports %d below rent-exempt minimum %d for %d bytes",
			msg.Lamports, minBalance, msg.Space)
	}

	account := &types.SVMAccount{
		Address:    newAddr,
		Lamports:   msg.Lamports,
		DataLen:    msg.Space,
		Data:       make([]byte, msg.Space),
		Owner:      msg.Owner,
		Executable: false,
		RentEpoch:  0,
	}

	if err := s.keeper.SetAccount(ctx, account); err != nil {
		return fmt.Errorf("failed to create account: %w", err)
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"svm_create_account",
		sdk.NewAttribute("address", types.Base58Encode(newAddr)),
		sdk.NewAttribute("owner", types.Base58Encode(msg.Owner)),
		sdk.NewAttribute("space", fmt.Sprintf("%d", msg.Space)),
		sdk.NewAttribute("lamports", fmt.Sprintf("%d", msg.Lamports)),
	))

	s.keeper.logger.Info("MsgCreateAccount handled",
		"sender", msg.Sender,
		"account", types.Base58Encode(newAddr),
		"space", msg.Space,
	)

	return nil
}

// HandleMsgRegisterSVMPQCKey registers a Dilithium-5 public key for an
// SVM account, enabling optional post-quantum signature verification.
func (s *MsgServer) HandleMsgRegisterSVMPQCKey(ctx sdk.Context, msg *types.MsgRegisterSVMPQCKey) error {
	if err := msg.ValidateBasic(); err != nil {
		return err
	}

	// Verify the SVM account exists.
	account, err := s.keeper.GetAccount(ctx, msg.SVMAddr)
	if err != nil {
		return err
	}

	// Verify sender owns the SVM account: the sender's derived native address
	// must match the account's reverse-mapped native address.
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return types.ErrInvalidAddress.Wrapf("invalid sender: %v", err)
	}
	accountCosmosAddr := types.SVMToCosmosAddress(account.Address)
	senderSVM, lookupErr := s.keeper.cosmosToSVMAddrFromStore(ctx, senderAddr)
	if lookupErr != nil || senderSVM != msg.SVMAddr {
		// Also check if the sender's native address matches the account's derived address.
		if !sdk.AccAddress(accountCosmosAddr).Equals(senderAddr) {
			return types.ErrInvalidAccountOwner.Wrapf(
				"sender %s does not own SVM account %s", msg.Sender, types.Base58Encode(msg.SVMAddr))
		}
	}

	// Validate the PQC public key via the PQC keeper.
	if s.keeper.pqcKeeper != nil {
		pqcClient := s.keeper.pqcKeeper.PQCClient()
		if pqcClient != nil {
			// Perform a basic size check (Dilithium-5 public key = 2592 bytes).
			if len(msg.PQCPubKey) != 2592 {
				return types.ErrInvalidSignature.Wrapf(
					"expected 2592-byte Dilithium-5 public key, got %d", len(msg.PQCPubKey))
			}
		}
	}

	// Store the PQC key in the account's data field as a tagged prefix.
	// Convention: first byte = key type (0x01 = Dilithium-5), followed by pubkey.
	pqcData := make([]byte, 1+len(msg.PQCPubKey))
	pqcData[0] = 0x01 // Dilithium-5 tag
	copy(pqcData[1:], msg.PQCPubKey)

	// For now we overwrite data with the PQC key. Future iterations may
	// use a separate PQC key registry.
	account.Data = pqcData
	account.DataLen = uint64(len(pqcData))

	if err := s.keeper.SetAccount(ctx, account); err != nil {
		return fmt.Errorf("failed to update account with PQC key: %w", err)
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"svm_register_pqc_key",
		sdk.NewAttribute("svm_address", types.Base58Encode(msg.SVMAddr)),
		sdk.NewAttribute("pqc_key_size", fmt.Sprintf("%d", len(msg.PQCPubKey))),
	))

	s.keeper.logger.Info("PQC key registered for SVM account",
		"svm_address", types.Base58Encode(msg.SVMAddr),
		"sender", msg.Sender,
	)

	return nil
}
