package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RegisterInterfaces registers the rlconsensus module's interface types.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSetAgentMode{},
		&MsgResumeAgent{},
		&MsgUpdatePolicy{},
		&MsgUpdateRewardWeights{},
	)
}

// RegisterLegacyAminoCodec registers the rlconsensus module's types with the legacy amino codec.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgSetAgentMode{}, "rlconsensus/MsgSetAgentMode", nil)
	cdc.RegisterConcrete(&MsgResumeAgent{}, "rlconsensus/MsgResumeAgent", nil)
	cdc.RegisterConcrete(&MsgUpdatePolicy{}, "rlconsensus/MsgUpdatePolicy", nil)
	cdc.RegisterConcrete(&MsgUpdateRewardWeights{}, "rlconsensus/MsgUpdateRewardWeights", nil)
}
