package cmd

import (
	"sort"
	"strings"
	"testing"

	gogoproto "github.com/cosmos/gogoproto/proto"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"

	"github.com/qorechain/qorechain-core/app"
	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
)

// rootClientRegistry runs the root command's pre-run (which builds the client
// context the REST gateway serves with) and returns that context's registry.
func rootClientRegistry(t *testing.T) codectypes.InterfaceRegistry {
	t.Helper()
	root := NewRootCmd()
	root.PersistentFlags().String(flags.FlagHome, t.TempDir(), "node home")
	var reg codectypes.InterfaceRegistry
	root.AddCommand(&cobra.Command{Use: "probe-registry", RunE: func(cmd *cobra.Command, _ []string) error {
		reg = client.GetClientContextFromCmd(cmd).InterfaceRegistry
		return nil
	}})
	root.SetArgs([]string{"probe-registry"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if reg == nil {
		t.Fatal("no client context registry")
	}
	return reg
}

// qorechainMsgResponses lists the response type of every QoreChain Msg service
// method whose request the registry accepts as an sdk.Msg.
func qorechainMsgResponses(t *testing.T, reg codectypes.InterfaceRegistry) []string {
	t.Helper()
	msgs := map[string]bool{}
	for _, url := range reg.ListImplementations(sdk.MsgInterfaceProtoName) {
		msgs[strings.TrimPrefix(url, "/")] = true
	}
	var out []string
	gogoproto.HybridResolver.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		if !strings.HasPrefix(string(fd.Package()), "qorechain.") {
			return true
		}
		for i := 0; i < fd.Services().Len(); i++ {
			ms := fd.Services().Get(i).Methods()
			for j := 0; j < ms.Len(); j++ {
				if msgs[string(ms.Get(j).Input().FullName())] {
					out = append(out, string(ms.Get(j).Output().FullName()))
				}
			}
		}
		return true
	})
	return out
}

// REST POST /cosmos/tx/v1beta1/simulate failed with "unable to resolve type
// URL /qorechain.pqc.v1.MsgOpenEVMWindowResponse": the custom modules never
// registered their Msg response types. Every one of them must now survive the
// gateway's path (binary decode with interface unpacking, then proto-JSON) on
// both the app registry and the client registry the gateway uses.
func TestSimulateResponseRendersEveryCustomMsgResponse(t *testing.T) {
	for name, reg := range map[string]codectypes.InterfaceRegistry{
		"app":    sharedTestApp(t).InterfaceRegistry(),
		"client": rootClientRegistry(t),
	} {
		t.Run(name, func(t *testing.T) {
			resps := qorechainMsgResponses(t, reg)
			if len(resps) < 40 {
				t.Fatalf("only %d QoreChain Msg responses found; the module set did not register", len(resps))
			}
			seenWindow := false
			cdc := codec.NewProtoCodec(reg)
			for _, full := range resps {
				if full == "qorechain.pqc.v1.MsgOpenEVMWindowResponse" {
					seenWindow = true
				}
				typ := gogoproto.MessageType(full)
				if typ == nil {
					t.Fatalf("%s has no Go type", full)
				}
				msg, err := reg.Resolve("/" + full)
				if err != nil {
					t.Fatalf("%s: %v", full, err)
				}
				anyResp, err := codectypes.NewAnyWithValue(msg)
				if err != nil {
					t.Fatal(err)
				}
				sim := &txtypes.SimulateResponse{
					GasInfo: &sdk.GasInfo{GasWanted: 200000, GasUsed: 129362},
					Result:  &sdk.Result{MsgResponses: []*codectypes.Any{anyResp}},
				}
				bz, err := cdc.Marshal(sim)
				if err != nil {
					t.Fatal(err)
				}
				var back txtypes.SimulateResponse
				if err := cdc.Unmarshal(bz, &back); err != nil {
					t.Fatalf("%s: decode: %v", full, err)
				}
				js, err := cdc.MarshalJSON(&back)
				if err != nil {
					t.Fatalf("%s: render: %v", full, err)
				}
				if !strings.Contains(string(js), `"@type":"/`+full+`"`) {
					t.Fatalf("%s: rendered without its type: %s", full, js)
				}
			}
			if !seenWindow {
				t.Fatal("MsgOpenEVMWindowResponse (the reported case) is not in the set")
			}
		})
	}
	// The reported case on its own, as the gateway would show it.
	reg := rootClientRegistry(t)
	anyResp, _ := codectypes.NewAnyWithValue(&pqctypes.MsgOpenEVMWindowResponse{})
	js, err := codec.NewProtoCodec(reg).MarshalJSON(&txtypes.SimulateResponse{
		GasInfo: &sdk.GasInfo{}, Result: &sdk.Result{MsgResponses: []*codectypes.Any{anyResp}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(js), "MsgOpenEVMWindowResponse") {
		t.Fatalf("unexpected rendering: %s", js)
	}
}

// The registration adds response types only: the set of Msgs a tx may carry
// is exactly what it was, so tx decoding (consensus) is unchanged.
func TestMsgResponseRegistrationAddsNoMsg(t *testing.T) {
	reg := codectypes.NewInterfaceRegistry()
	sdk.RegisterInterfaces(reg)
	txtypes.RegisterInterfaces(reg)
	pqctypes.RegisterInterfaces(reg)
	before := reg.ListImplementations(sdk.MsgInterfaceProtoName)
	beforeResp := len(reg.ListImplementations("cosmos.tx.v1beta1.MsgResponse"))
	app.RegisterMsgResponseTypes(reg)
	after := reg.ListImplementations(sdk.MsgInterfaceProtoName)
	sort.Strings(before)
	sort.Strings(after)
	if strings.Join(before, ",") != strings.Join(after, ",") {
		t.Fatalf("sdk.Msg set changed:\nbefore %v\nafter  %v", before, after)
	}
	if got, want := len(reg.ListImplementations("cosmos.tx.v1beta1.MsgResponse")), beforeResp+len(qorechainMsgResponses(t, reg)); got != want {
		t.Fatalf("expected one response per pqc Msg service method: have %d, want %d", got, want)
	}
	if _, err := reg.Resolve("/qorechain.pqc.v1.MsgOpenEVMWindowResponse"); err != nil {
		t.Fatal(err)
	}
	// Idempotent.
	app.RegisterMsgResponseTypes(reg)
}
