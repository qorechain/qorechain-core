package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RegisterInterfaces registers the module's interface types.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRegisterPQCKey{},
		&MsgRegisterPQCKeyV2{},
		&MsgMigratePQCKey{},
		&MsgAddAlgorithm{},
		&MsgDeprecateAlgorithm{},
		&MsgDisableAlgorithm{},
	)
}

// RegisterLegacyAminoCodec registers the module's types on the amino codec.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgRegisterPQCKey{}, "pqc/MsgRegisterPQCKey", nil)
	cdc.RegisterConcrete(&MsgRegisterPQCKeyV2{}, "pqc/MsgRegisterPQCKeyV2", nil)
	cdc.RegisterConcrete(&MsgMigratePQCKey{}, "pqc/MsgMigratePQCKey", nil)
	cdc.RegisterConcrete(&MsgAddAlgorithm{}, "pqc/MsgAddAlgorithm", nil)
	cdc.RegisterConcrete(&MsgDeprecateAlgorithm{}, "pqc/MsgDeprecateAlgorithm", nil)
	cdc.RegisterConcrete(&MsgDisableAlgorithm{}, "pqc/MsgDisableAlgorithm", nil)
}
