package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RegisterInterfaces registers the crossvm module's interface types.
func RegisterInterfaces(reg codectypes.InterfaceRegistry) {
	reg.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCrossVMCall{},
		&MsgProcessQueue{},
	)
}

// RegisterLegacyAminoCodec registers the crossvm module's types with the legacy amino codec.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgCrossVMCall{}, "crossvm/MsgCrossVMCall", nil)
	cdc.RegisterConcrete(&MsgProcessQueue{}, "crossvm/MsgProcessQueue", nil)
}
