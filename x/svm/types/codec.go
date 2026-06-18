package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// The SVM Msg types (MsgDeployProgram, MsgCreateAccount, MsgExecuteProgram,
// MsgRegisterSVMPQCKey) and the SvmAccountMeta sub-message are generated from
// proto/qorechain/svm/v1/tx.proto (see tx.pb.go). ValidateBasic methods live
// in msgs_validate.go.

// RegisterInterfaces registers the SVM module's interface types.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgDeployProgram{},
		&MsgExecuteProgram{},
		&MsgCreateAccount{},
		&MsgRegisterSVMPQCKey{},
	)
}

// RegisterLegacyAminoCodec registers the SVM module's types with the legacy amino codec.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgDeployProgram{}, "svm/MsgDeployProgram", nil)
	cdc.RegisterConcrete(&MsgExecuteProgram{}, "svm/MsgExecuteProgram", nil)
	cdc.RegisterConcrete(&MsgCreateAccount{}, "svm/MsgCreateAccount", nil)
	cdc.RegisterConcrete(&MsgRegisterSVMPQCKey{}, "svm/MsgRegisterSVMPQCKey", nil)
}
