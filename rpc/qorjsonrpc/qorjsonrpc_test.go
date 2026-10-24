package qorjsonrpc

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/rpc"

	evmrpc "github.com/cosmos/evm/rpc"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/server"

	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
)

// Registration makes "qor" a namespace the EVM JSON-RPC server will serve
// when app.toml lists it, carrying the KEM service. Registering twice is
// harmless (the command package registers at start-up).
func TestRegisterServesTheQorNamespace(t *testing.T) {
	if err := Register(); err != nil {
		t.Fatal(err)
	}
	if err := Register(); err != nil {
		t.Fatalf("second Register: %v", err)
	}
	apis := evmrpc.GetRPCAPIs(server.NewDefaultContext(), client.Context{}, nil, false, nil, []string{Namespace}, nil)
	if len(apis) == 0 {
		t.Fatal("the qor namespace yields no API")
	}
	var kem bool
	for _, a := range apis {
		if a.Namespace != Namespace {
			t.Fatalf("an API of namespace %q was registered under qor", a.Namespace)
		}
		if _, ok := a.Service.(*KEMAPI); ok {
			kem = true
		}
	}
	if !kem {
		t.Fatal("the KEM service is not part of the qor namespace")
	}
}

// qor_getKEMKey over a real JSON-RPC round trip: the method name, the hex
// encoding of the key, and the not-found answer.
func TestGetKEMKeyOverJSONRPC(t *testing.T) {
	pub := []byte{0xde, 0xad, 0xbe, 0xef}
	api := &KEMAPI{query: func(_ context.Context, req *pqctypes.QueryKEMKeyRequest) (*pqctypes.QueryKEMKeyResponse, error) {
		if req.Address != "0x00000000000000000000000000000000000000aa" {
			return &pqctypes.QueryKEMKeyResponse{Found: false}, nil
		}
		return &pqctypes.QueryKEMKeyResponse{Found: true, Key: &pqctypes.KEMKeyView{
			Address: "qor1example", AlgorithmId: 2, AlgorithmName: "mlkem1024", PublicKey: pub, RegisteredHeight: 42,
		}}, nil
	}}
	srv := rpc.NewServer()
	if err := srv.RegisterName(Namespace, api); err != nil {
		t.Fatal(err)
	}
	c := rpc.DialInProc(srv)
	defer c.Close()

	var got map[string]interface{}
	if err := c.Call(&got, "qor_getKEMKey", "0x00000000000000000000000000000000000000aa"); err != nil {
		t.Fatal(err)
	}
	if got["found"] != true || got["public_key"] != "0xdeadbeef" || got["algorithm_name"] != "mlkem1024" ||
		got["address"] != "qor1example" || got["registered_height"] != float64(42) {
		t.Fatalf("unexpected answer %v", got)
	}

	got = nil
	if err := c.Call(&got, "qor_getKEMKey", "qor1nobody"); err != nil {
		t.Fatal(err)
	}
	if got["found"] != false || got["public_key"] != nil {
		t.Fatalf("unexpected not-found answer %v", got)
	}
	if err := c.Call(&got, "qor_getKEMKey", ""); err == nil {
		t.Fatal("an empty address was answered")
	}
}
