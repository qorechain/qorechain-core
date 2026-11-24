package wasmfilter

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtestdata "github.com/CosmWasm/wasmd/x/wasm/keeper/testdata"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	wasmvm "github.com/CosmWasm/wasmvm/v2"
	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"
)

// fakeVM stands in for wasmvm: it holds code by checksum and counts the calls
// that reach it. Methods it does not implement panic through the nil embedded
// interface, so a test fails loudly if the decorator calls something else.
type fakeVM struct {
	wasmtypes.WasmEngine
	codes   map[string][]byte
	reached map[string]int
	getCode int
}

func newFakeVM() *fakeVM { return &fakeVM{codes: map[string][]byte{}, reached: map[string]int{}} }

func (v *fakeVM) put(code []byte) wasmvm.Checksum {
	sum := sha256.Sum256(code)
	v.codes[string(sum[:])] = code
	return sum[:]
}

func (v *fakeVM) GetCode(c wasmvm.Checksum) (wasmvm.WasmCode, error) {
	v.getCode++
	code, ok := v.codes[string(c)]
	if !ok {
		return nil, errors.New("no such code")
	}
	return code, nil
}

func (v *fakeVM) StoreCodeUnchecked(code wasmvm.WasmCode) (wasmvm.Checksum, error) {
	v.reached["StoreCodeUnchecked"]++
	return v.put(code), nil
}

func (v *fakeVM) StoreCode(code wasmvm.WasmCode, _ uint64) (wasmvm.Checksum, uint64, error) {
	v.reached["StoreCode"]++
	return v.put(code), 0, nil
}

func (v *fakeVM) Pin(wasmvm.Checksum) error { v.reached["Pin"]++; return nil }

func (v *fakeVM) Instantiate(wasmvm.Checksum, wasmvmtypes.Env, wasmvmtypes.MessageInfo, []byte, wasmvm.KVStore, wasmvm.GoAPI, wasmvm.Querier, wasmvm.GasMeter, uint64, wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
	v.reached["Instantiate"]++
	return &wasmvmtypes.ContractResult{}, 7, nil
}

func (v *fakeVM) Execute(wasmvm.Checksum, wasmvmtypes.Env, wasmvmtypes.MessageInfo, []byte, wasmvm.KVStore, wasmvm.GoAPI, wasmvm.Querier, wasmvm.GasMeter, uint64, wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
	v.reached["Execute"]++
	return &wasmvmtypes.ContractResult{}, 7, nil
}

func (v *fakeVM) Query(wasmvm.Checksum, wasmvmtypes.Env, []byte, wasmvm.KVStore, wasmvm.GoAPI, wasmvm.Querier, wasmvm.GasMeter, uint64, wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
	v.reached["Query"]++
	return &wasmvmtypes.QueryResult{}, 7, nil
}

func (v *fakeVM) MigrateWithInfo(wasmvm.Checksum, wasmvmtypes.Env, []byte, wasmvmtypes.MigrateInfo, wasmvm.KVStore, wasmvm.GoAPI, wasmvm.Querier, wasmvm.GasMeter, uint64, wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
	v.reached["MigrateWithInfo"]++
	return &wasmvmtypes.ContractResult{}, 7, nil
}

func (v *fakeVM) Sudo(wasmvm.Checksum, wasmvmtypes.Env, []byte, wasmvm.KVStore, wasmvm.GoAPI, wasmvm.Querier, wasmvm.GasMeter, uint64, wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
	v.reached["Sudo"]++
	return &wasmvmtypes.ContractResult{}, 7, nil
}

func (v *fakeVM) Reply(wasmvm.Checksum, wasmvmtypes.Env, wasmvmtypes.Reply, wasmvm.KVStore, wasmvm.GoAPI, wasmvm.Querier, wasmvm.GasMeter, uint64, wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
	v.reached["Reply"]++
	return &wasmvmtypes.ContractResult{}, 7, nil
}

func (v *fakeVM) IBCPacketReceive(wasmvm.Checksum, wasmvmtypes.Env, wasmvmtypes.IBCPacketReceiveMsg, wasmvm.KVStore, wasmvm.GoAPI, wasmvm.Querier, wasmvm.GasMeter, uint64, wasmvmtypes.UFraction) (*wasmvmtypes.IBCReceiveResult, uint64, error) {
	v.reached["IBCPacketReceive"]++
	return &wasmvmtypes.IBCReceiveResult{}, 7, nil
}

func testCtx(t *testing.T) sdk.Context {
	t.Helper()
	return testutil.DefaultContext(storetypes.NewKVStoreKey("wasmfilter-test"), storetypes.NewTransientStoreKey("wasmfilter-test-t"))
}

var badCode = mkModule([][]byte{t0, tParam}, body(0x41, 0x01, 0x41, 0x00, 0x04, 0x01, 0x0B, 0x1A, 0x0B))

// call runs every decorated entry point the fake VM implements and returns
// how many of them refused.
func call(e *Engine, checksum wasmvm.Checksum, q wasmvm.Querier) (refused int, errs []error) {
	var env wasmvmtypes.Env
	record := func(err error) {
		if err != nil {
			refused++
			errs = append(errs, err)
		}
	}
	_, _, err := e.Instantiate(checksum, env, wasmvmtypes.MessageInfo{}, nil, nil, wasmvm.GoAPI{}, q, nil, 0, wasmvmtypes.UFraction{})
	record(err)
	_, _, err = e.Execute(checksum, env, wasmvmtypes.MessageInfo{}, nil, nil, wasmvm.GoAPI{}, q, nil, 0, wasmvmtypes.UFraction{})
	record(err)
	_, _, err = e.Query(checksum, env, nil, nil, wasmvm.GoAPI{}, q, nil, 0, wasmvmtypes.UFraction{})
	record(err)
	// wasmd hands MigrateWithInfo a pointer to its query handler.
	if h, ok := q.(wasmkeeper.QueryHandler); ok {
		_, _, err = e.MigrateWithInfo(checksum, env, nil, wasmvmtypes.MigrateInfo{}, nil, wasmvm.GoAPI{}, &h, nil, 0, wasmvmtypes.UFraction{})
	} else {
		_, _, err = e.MigrateWithInfo(checksum, env, nil, wasmvmtypes.MigrateInfo{}, nil, wasmvm.GoAPI{}, q, nil, 0, wasmvmtypes.UFraction{})
	}
	record(err)
	_, _, err = e.Sudo(checksum, env, nil, nil, wasmvm.GoAPI{}, q, nil, 0, wasmvmtypes.UFraction{})
	record(err)
	_, _, err = e.Reply(checksum, env, wasmvmtypes.Reply{}, nil, wasmvm.GoAPI{}, q, nil, 0, wasmvmtypes.UFraction{})
	record(err)
	_, _, err = e.IBCPacketReceive(checksum, env, wasmvmtypes.IBCPacketReceiveMsg{}, nil, wasmvm.GoAPI{}, q, nil, 0, wasmvmtypes.UFraction{})
	record(err)
	return refused, errs
}

const entryPoints = 7

func TestEngineRefusesFailingCodeOnlyWhenOn(t *testing.T) {
	ctx := testCtx(t)
	on := true
	f := New(func(sdk.Context) bool { return on })
	vm := newFakeVM()
	e := f.DecorateEngine(vm).(*Engine)
	bad := vm.put(badCode)
	q := wasmkeeper.NewQueryHandler(ctx, nil, nil, nil)

	refused, errs := call(e, bad, q)
	if refused != entryPoints {
		t.Fatalf("switch on: %d of %d entry points refused", refused, entryPoints)
	}
	for _, err := range errs {
		if !errors.Is(err, ErrCodeRefused) {
			t.Fatalf("refusal %v is not ErrCodeRefused", err)
		}
	}
	if len(vm.reached) != 0 {
		t.Fatalf("refused calls reached the VM: %v", vm.reached)
	}

	on = false
	if refused, _ := call(e, bad, q); refused != 0 {
		t.Fatalf("switch off: %d entry points refused", refused)
	}
	if got := len(vm.reached); got != entryPoints {
		t.Fatalf("switch off: %d entry points reached the VM", got)
	}
	// The verdict was read from the VM once and then cached.
	if vm.getCode != 1 {
		t.Fatalf("GetCode called %d times", vm.getCode)
	}
}

func TestEnginePassingCodeNeverReadsTheSwitch(t *testing.T) {
	f := New(func(sdk.Context) bool { panic("switch read for a passing code") })
	vm := newFakeVM()
	e := f.DecorateEngine(vm).(*Engine)
	good := vm.put(wasmtestdata.ReflectContractWasm())
	// No usable context at all: passing code still runs.
	if refused, _ := call(e, good, nil); refused != 0 {
		t.Fatal("passing code refused")
	}
	if refused, _ := call(e, good, wasmkeeper.NewQueryHandler(testCtx(t), nil, nil, nil)); refused != 0 {
		t.Fatal("passing code refused")
	}
}

func TestEngineFailsClosedWithoutContext(t *testing.T) {
	f := New(func(sdk.Context) bool { return false })
	vm := newFakeVM()
	e := f.DecorateEngine(vm).(*Engine)
	bad := vm.put(badCode)
	// A failing code reached through a querier that carries no context: the
	// switch cannot be read, so it is refused.
	for _, q := range []wasmvm.Querier{nil, wasmkeeper.QueryHandler{}, (*wasmkeeper.QueryHandler)(nil)} {
		if refused, _ := call(e, bad, q); refused != entryPoints {
			t.Fatalf("querier %T: %d of %d refused", q, refused, entryPoints)
		}
	}
}

func TestEngineUnknownCodeIsLeftToTheVM(t *testing.T) {
	f := New(func(sdk.Context) bool { return true })
	vm := newFakeVM()
	e := f.DecorateEngine(vm).(*Engine)
	missing := make([]byte, 32)
	if refused, _ := call(e, missing, nil); refused != 0 {
		t.Fatal("a code the node does not hold was refused by the filter")
	}
	// Not cached: once the bytes arrive, the verdict is computed.
	vm.codes[string(missing)] = badCode
	if refused, _ := call(e, missing, nil); refused != entryPoints {
		t.Fatal("code that arrived later was not checked")
	}
}

// Restoring and pinning never refuse, whatever the code and the switch.
func TestEngineNeverRefusesRestoreOrPin(t *testing.T) {
	f := New(func(sdk.Context) bool { return true })
	vm := newFakeVM()
	e := f.DecorateEngine(vm)
	sum, err := e.StoreCodeUnchecked(badCode)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Pin(sum); err != nil {
		t.Fatal(err)
	}
	if vm.reached["StoreCodeUnchecked"] != 1 || vm.reached["Pin"] != 1 {
		t.Fatalf("not passed through: %v", vm.reached)
	}
}

// ---- message server --------------------------------------------------------

type fakeMsgServer struct {
	wasmtypes.MsgServer
	reached int
}

func (s *fakeMsgServer) StoreCode(context.Context, *wasmtypes.MsgStoreCode) (*wasmtypes.MsgStoreCodeResponse, error) {
	s.reached++
	return &wasmtypes.MsgStoreCodeResponse{}, nil
}

func (s *fakeMsgServer) StoreAndInstantiateContract(context.Context, *wasmtypes.MsgStoreAndInstantiateContract) (*wasmtypes.MsgStoreAndInstantiateContractResponse, error) {
	s.reached++
	return &wasmtypes.MsgStoreAndInstantiateContractResponse{}, nil
}

func (s *fakeMsgServer) StoreAndMigrateContract(context.Context, *wasmtypes.MsgStoreAndMigrateContract) (*wasmtypes.MsgStoreAndMigrateContractResponse, error) {
	s.reached++
	return &wasmtypes.MsgStoreAndMigrateContractResponse{}, nil
}

func TestMsgServerRefusesFailingUploads(t *testing.T) {
	ctx := testCtx(t)
	on := true
	f := New(func(sdk.Context) bool { return on })
	inner := &fakeMsgServer{}
	s := f.DecorateMsgServer(inner)

	upload := func(code []byte) (refused int) {
		if _, err := s.StoreCode(ctx, &wasmtypes.MsgStoreCode{WASMByteCode: code}); err != nil {
			if !errors.Is(err, ErrCodeRefused) {
				t.Fatalf("StoreCode: %v", err)
			}
			refused++
		}
		if _, err := s.StoreAndInstantiateContract(ctx, &wasmtypes.MsgStoreAndInstantiateContract{WASMByteCode: code}); err != nil {
			refused++
		}
		if _, err := s.StoreAndMigrateContract(ctx, &wasmtypes.MsgStoreAndMigrateContract{WASMByteCode: code}); err != nil {
			refused++
		}
		return refused
	}

	if n := upload(badCode); n != 3 || inner.reached != 0 {
		t.Fatalf("on, raw: %d refused, %d reached", n, inner.reached)
	}
	if n := upload(gz(t, badCode)); n != 3 || inner.reached != 0 {
		t.Fatalf("on, gzip: %d refused, %d reached", n, inner.reached)
	}
	if n := upload(wasmtestdata.ReflectContractWasm()); n != 0 || inner.reached != 3 {
		t.Fatalf("on, good: %d refused, %d reached", n, inner.reached)
	}
	// Bytes that do not unpack are passed to wasmd, which refuses them itself.
	broken := gz(t, badCode)
	if n := upload(broken[:len(broken)-6]); n != 0 || inner.reached != 6 {
		t.Fatalf("on, broken gzip: %d refused, %d reached", n, inner.reached)
	}

	on = false
	if n := upload(badCode); n != 0 || inner.reached != 9 {
		t.Fatalf("off: %d refused, %d reached", n, inner.reached)
	}
}
