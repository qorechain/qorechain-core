package wasmfilter

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
)

// DecorateMsgServer wraps wasmd's message server so that every message that
// carries bytecode to store (MsgStoreCode, MsgStoreAndInstantiateContract,
// MsgStoreAndMigrateContract) is refused, while the switch is on, when the
// bytecode fails Check. Messages an account sends and messages a governance
// proposal executes go through the same server, so both are covered. Every
// other message is passed through: the engine decorator covers the calls that
// run stored code.
func (f *Filter) DecorateMsgServer(inner wasmtypes.MsgServer) wasmtypes.MsgServer {
	return &msgServer{MsgServer: inner, f: f}
}

type msgServer struct {
	wasmtypes.MsgServer
	f *Filter
}

// checkUpload refuses bytecode that fails Check while the switch is on. Bytes
// that cannot be unpacked are left to wasmd, which refuses them itself.
func (s *msgServer) checkUpload(goCtx context.Context, code []byte) error {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if !s.f.on(ctx) {
		return nil
	}
	checked, err := CheckUpload(code)
	if !checked || err == nil {
		return nil
	}
	return errorsmod.Wrap(ErrCodeRefused, err.Error())
}

func (s *msgServer) StoreCode(goCtx context.Context, msg *wasmtypes.MsgStoreCode) (*wasmtypes.MsgStoreCodeResponse, error) {
	if msg != nil {
		if err := s.checkUpload(goCtx, msg.WASMByteCode); err != nil {
			return nil, err
		}
	}
	return s.MsgServer.StoreCode(goCtx, msg)
}

func (s *msgServer) StoreAndInstantiateContract(goCtx context.Context, msg *wasmtypes.MsgStoreAndInstantiateContract) (*wasmtypes.MsgStoreAndInstantiateContractResponse, error) {
	if msg != nil {
		if err := s.checkUpload(goCtx, msg.WASMByteCode); err != nil {
			return nil, err
		}
	}
	return s.MsgServer.StoreAndInstantiateContract(goCtx, msg)
}

func (s *msgServer) StoreAndMigrateContract(goCtx context.Context, msg *wasmtypes.MsgStoreAndMigrateContract) (*wasmtypes.MsgStoreAndMigrateContractResponse, error) {
	if msg != nil {
		if err := s.checkUpload(goCtx, msg.WASMByteCode); err != nil {
			return nil, err
		}
	}
	return s.MsgServer.StoreAndMigrateContract(goCtx, msg)
}
