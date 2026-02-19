package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
)

// RegisterInterfaces registers the module's interface types.
func RegisterInterfaces(_ codectypes.InterfaceRegistry) {
	// PQC message types will be registered when protobuf definitions are added.
	// For now, the module uses plain JSON genesis and keeper operations.
}

// RegisterLegacyAminoCodec registers the module's types on the amino codec.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgRegisterPQCKey{}, "pqc/MsgRegisterPQCKey", nil)
}
