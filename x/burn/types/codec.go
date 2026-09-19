package types

import (
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

// RegisterInterfaces registers the burn module's messages with the interface
// registry. Without this the message cannot be decoded on chain, which is a
// failure that only shows up when a proposal executes.
func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil), &MsgReleaseStrandedRewards{})
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
