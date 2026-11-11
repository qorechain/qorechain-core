package wasmfilter

import (
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmvm "github.com/CosmWasm/wasmvm/v2"
	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"
)

// growVM is a fakeVM whose Execute reports that the call grew its memory to
// 512 pages, as the patched VM does.
type growVM struct{ *fakeVM }

func (v growVM) Execute(c wasmvm.Checksum, env wasmvmtypes.Env, info wasmvmtypes.MessageInfo, msg []byte, s wasmvm.KVStore, api wasmvm.GoAPI, q wasmvm.Querier, gm wasmvm.GasMeter, limit uint64, d wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
	if r, ok := q.(wasmvmtypes.MemoryPagesReporter); ok {
		r.ReportMemoryPages(VMMemoryLimitPages)
	}
	return v.fakeVM.Execute(c, env, info, msg, s, api, q, gm, limit, d)
}

// orderBudget records the order of the budget's calls; Used aborts (a
// clamped call that ran out) and ChargeCallGrowth may panic (out of gas).
type orderBudget struct {
	calls     []string
	growPanic any
}

func (b *orderBudget) Limit(sdk.Context, uint64) (uint64, bool, error) {
	b.calls = append(b.calls, "limit")
	return 100, true, nil
}

func (b *orderBudget) Used(sdk.Context, uint64, uint64, uint64, bool) {
	b.calls = append(b.calls, "used")
	panic(BudgetAbort{Reason: "clamped call ran out"})
}

func (b *orderBudget) ChargeCallSetup(sdk.Context, uint64, uint64) error {
	b.calls = append(b.calls, "setup")
	return nil
}

func (b *orderBudget) ChargeCallGrowth(_ sdk.Context, units, _ uint64) error {
	b.calls = append(b.calls, "growth")
	if b.growPanic != nil {
		panic(b.growPanic)
	}
	return nil
}

// Review 15 F2: a call's growth is charged before the clamped-call abort
// of its own VM gas is raised (its units recorded first), and that abort
// is still raised when the growth charge panics (out of gas).
func TestGrowthChargedBeforeClampedAbort(t *testing.T) {
	ctx := testCtx(t)
	for _, growPanic := range []any{nil, "out of gas"} {
		bud := &orderBudget{growPanic: growPanic}
		f := New(func(sdk.Context) bool { return true }).WithBudget(bud).WithCallSetupPricing(func(sdk.Context) bool { return true })
		vm := newFakeVM()
		e := f.DecorateEngine(growVM{vm}).(*Engine)
		sum := vm.put(memModule(17, 0, true, 0, 0))
		q := wasmkeeper.NewQueryHandler(ctx, nil, nil, nil)
		var got any
		func() {
			defer func() { got = recover() }()
			_, _, _ = e.Execute(sum, wasmvmtypes.Env{}, wasmvmtypes.MessageInfo{}, nil, nil, wasmvm.GoAPI{}, q, nil, 0, wasmvmtypes.UFraction{})
		}()
		if order := strings.Join(bud.calls, ","); order != "setup,limit,used,growth" {
			t.Fatalf("growth panic %v: order %s", growPanic, order)
		}
		if _, ok := got.(BudgetAbort); !ok {
			t.Fatalf("growth panic %v: recovered %v, want the clamped abort", growPanic, got)
		}
	}
}
