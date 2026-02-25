package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
)

// RegisterInterfaces registers the rlconsensus module's interface types.
func RegisterInterfaces(_ codectypes.InterfaceRegistry) {
	// TODO: Register sdk.Msg implementations once proto definitions are generated.
}

// RegisterLegacyAminoCodec registers the rlconsensus module's types with the legacy amino codec.
func RegisterLegacyAminoCodec(_ *codec.LegacyAmino) {
	// TODO: Register concrete types once proto definitions are generated.
}
