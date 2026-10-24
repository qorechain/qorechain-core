package app

import (
	"reflect"
	"strings"

	gogoproto "github.com/cosmos/gogoproto/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
)

// RegisterMsgResponseTypes registers, as tx.MsgResponse implementations, the
// response type of every Msg service method whose request is already a
// registered sdk.Msg on registry.
//
// The SDK modules do this through msgservice.RegisterMsgServiceDesc. The
// QoreChain modules register their Msgs by hand and never registered the
// responses, so anything that has to unpack a TxMsgData or a simulate result
// (the REST gateway's POST /cosmos/tx/v1beta1/simulate, the JSON rendering of
// a tx result) failed with "unable to resolve type URL
// /qorechain.pqc.v1.MsgOpenEVMWindowResponse", while ABCI simulate, which
// returns raw bytes, worked.
//
// Node-local: the state machine packs responses with NewAnyWithValue, which
// needs no registry, and no request type is added here, so the set of
// decodable transactions is unchanged.
func RegisterMsgResponseTypes(registry codectypes.InterfaceRegistry) {
	msgs := map[string]bool{}
	for _, url := range registry.ListImplementations(sdk.MsgInterfaceProtoName) {
		msgs[strings.TrimPrefix(url, "/")] = true
	}
	gogoproto.HybridResolver.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		services := fd.Services()
		for i := 0; i < services.Len(); i++ {
			methods := services.Get(i).Methods()
			for j := 0; j < methods.Len(); j++ {
				md := methods.Get(j)
				if !msgs[string(md.Input().FullName())] {
					continue
				}
				name := string(md.Output().FullName())
				if _, err := registry.Resolve("/" + name); err == nil {
					continue
				}
				typ := gogoproto.MessageType(name)
				if typ == nil || typ.Kind() != reflect.Ptr {
					continue
				}
				resp, ok := reflect.New(typ.Elem()).Interface().(gogoproto.Message)
				if !ok {
					continue
				}
				registry.RegisterImplementations((*txtypes.MsgResponse)(nil), resp)
			}
		}
		return true
	})
}
