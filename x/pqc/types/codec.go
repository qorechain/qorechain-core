package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
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

	// PQCHybridSignature is a TX extension option carried in
	// TxBody.extension_options under HybridSigTypeURL
	// ("/qorechain.pqc.v1.PQCHybridSignature"). It MUST be registered as a
	// TxExtensionOptionI implementation, otherwise the SDK tx decoder rejects
	// every PQC-signed tx with "unable to resolve type URL". The Any payload is
	// the proto-marshaled signature (see hybrid_proto.go); the ante handler
	// proto-decodes it.
	registry.RegisterImplementations((*txtypes.TxExtensionOptionI)(nil),
		&PQCHybridSignature{},
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
	cdc.RegisterConcrete(&PQCHybridSignature{}, "pqc/PQCHybridSignature", nil)
}
