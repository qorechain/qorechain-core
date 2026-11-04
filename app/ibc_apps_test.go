package app

import (
	"testing"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"

	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
)

// A hosted module sees only its own prefix of the host store, and the host's
// own keys are untouched by it.
func TestHostedStoreServiceIsolation(t *testing.T) {
	key := storetypes.NewKVStoreKey("transfer")
	ctx := testutil.DefaultContext(key, storetypes.NewTransientStoreKey("t"))
	host := ctx.KVStore(key)
	host.Set([]byte("params"), []byte("host"))
	host.Set([]byte{0x03, 'd'}, []byte("denom"))

	a := hostedStoreService{key: key, prefix: rateLimitHostPrefix}.OpenKVStore(ctx)
	b := hostedStoreService{key: key, prefix: packetForwardHostPrefix}.OpenKVStore(ctx)
	if err := a.Set([]byte("params"), []byte("a")); err != nil {
		t.Fatal(err)
	}
	if err := b.Set([]byte("k"), []byte("b")); err != nil {
		t.Fatal(err)
	}
	if got := host.Get([]byte("params")); string(got) != "host" {
		t.Fatalf("host params overwritten: %q", got)
	}
	if got := host.Get(append(append([]byte(nil), rateLimitHostPrefix...), "params"...)); string(got) != "a" {
		t.Fatalf("hosted key not under its prefix: %q", got)
	}
	it, err := a.Iterator(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer it.Close()
	n := 0
	for ; it.Valid(); it.Next() {
		n++
		if string(it.Key()) != "params" {
			t.Fatalf("hosted iterator sees %q", it.Key())
		}
	}
	if n != 1 {
		t.Fatalf("hosted iterator saw %d keys", n)
	}
	if ok, _ := b.Has([]byte("params")); ok {
		t.Fatal("one hosted module sees another's keys")
	}
}

type recordingModule struct {
	porttypes.IBCModule
	name string
	got  *string
}

func (m recordingModule) OnRecvPacket(sdk.Context, string, channeltypes.Packet, sdk.AccAddress) ibcexported.Acknowledgement {
	*m.got = m.name
	return channeltypes.NewResultAcknowledgement([]byte{1})
}

// The switch routes a packet past the layer below it and through it above.
func TestGatedIBCModuleFollowsTheSwitch(t *testing.T) {
	var got string
	on := false
	g := gatedIBCModule{
		on:      func(sdk.Context) bool { return on },
		with:    recordingModule{name: "with", got: &got},
		without: recordingModule{name: "without", got: &got},
	}
	g.OnRecvPacket(sdk.Context{}, "", channeltypes.Packet{}, nil)
	if got != "without" {
		t.Fatalf("below the switch the packet went %s", got)
	}
	on = true
	g.OnRecvPacket(sdk.Context{}, "", channeltypes.Packet{}, nil)
	if got != "with" {
		t.Fatalf("above the switch the packet went %s", got)
	}
}
