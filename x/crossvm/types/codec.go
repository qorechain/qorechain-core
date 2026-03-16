package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
)

// RegisterInterfaces registers the crossvm module's interface types.
// MsgCrossVMCall and MsgProcessQueue are registered as amino concrete types
// below. Full protobuf interface registration (sdk.Msg implementations) will
// be added once .proto code generation is in place.
func RegisterInterfaces(_ codectypes.InterfaceRegistry) {
	// No proto-generated sdk.Msg implementations yet; amino registration
	// in RegisterLegacyAminoCodec covers JSON/amino serialization paths.
}

// RegisterLegacyAminoCodec registers the crossvm module's types with the legacy amino codec.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgCrossVMCall{}, "crossvm/MsgCrossVMCall", nil)
	cdc.RegisterConcrete(&MsgProcessQueue{}, "crossvm/MsgProcessQueue", nil)
}
