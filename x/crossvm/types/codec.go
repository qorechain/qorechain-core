package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
)

// RegisterInterfaces registers the crossvm module's interface types.
// Proto-generated message types will be registered here once .proto files are added.
func RegisterInterfaces(_ codectypes.InterfaceRegistry) {
	// TODO: Register MsgCrossVMCall and MsgProcessQueue once proto definitions exist.
}

// RegisterLegacyAminoCodec registers the crossvm module's types with the legacy amino codec.
func RegisterLegacyAminoCodec(_ *codec.LegacyAmino) {
	// TODO: Register concrete types once proto definitions exist.
}
