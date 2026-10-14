package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"cosmossdk.io/log"
	circuittypes "cosmossdk.io/x/circuit/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"

	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
)

// ---- module account least privilege (QSR-2026-0532) -------------------------

type memAccounts struct {
	accs  map[string]sdk.AccountI
	perms map[string][]string // what lazy creation would grant (the app_config table)
	next  uint64
}

func newMemAccounts() *memAccounts {
	m := &memAccounts{accs: map[string]sdk.AccountI{}, perms: map[string][]string{}}
	for _, a := range moduleAccPerms {
		m.perms[a.Account] = a.Permissions
	}
	return m
}

func (m *memAccounts) GetAccount(_ context.Context, addr sdk.AccAddress) sdk.AccountI {
	return m.accs[string(addr)]
}

func (m *memAccounts) GetModuleAccount(_ context.Context, name string) sdk.ModuleAccountI {
	addr := authtypes.NewModuleAddress(name)
	if acc, ok := m.accs[string(addr)]; ok {
		return acc.(sdk.ModuleAccountI)
	}
	macc := authtypes.NewEmptyModuleAccount(name, m.perms[name]...)
	macc.AccountNumber = m.next
	m.next++
	m.accs[string(addr)] = macc
	return macc
}

// Keyed by the module name rather than GetAddress, which re-parses the stored
// bech32 string with the process-wide prefix that other tests switch.
func (m *memAccounts) SetModuleAccount(_ context.Context, macc sdk.ModuleAccountI) {
	m.accs[string(authtypes.NewModuleAddress(macc.GetName()))] = macc
}

func svTestCtx() sdk.Context {
	return sdk.Context{}.WithEventManager(sdk.NewEventManager()).WithLogger(log.NewNopLogger())
}

func TestRestrictModuleAccountPermissions(t *testing.T) {
	ak := newMemAccounts()
	// diana: amm and xqore exist with the old grants; rdk is missing (mainnet).
	ak.GetModuleAccount(context.Background(), "amm")
	ak.GetModuleAccount(context.Background(), "xqore")
	// A squatted non-module account at a listed address must not halt the chain.
	squat := authtypes.NewModuleAddress("rdk")
	ak.accs[string(squat)] = authtypes.NewBaseAccountWithAddress(squat)

	if err := restrictModuleAccountPermissions(svTestCtx(), ak); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string][]string{"amm": {authtypes.Burner}, "xqore": {authtypes.Burner}} {
		got := ak.accs[string(authtypes.NewModuleAddress(name))].(*authtypes.ModuleAccount)
		if !slices.Equal(got.Permissions, want) {
			t.Errorf("%s: permissions %v, want %v", name, got.Permissions, want)
		}
		if got.HasPermission(authtypes.Minter) {
			t.Errorf("%s still mints", name)
		}
	}
	if _, ok := ak.accs[string(squat)].(*authtypes.BaseAccount); !ok {
		t.Fatal("the squatted rdk address was rewritten")
	}

	// mainnet shape: none exists. The step creates them with the reduced set.
	ak = newMemAccounts()
	if err := restrictModuleAccountPermissions(svTestCtx(), ak); err != nil {
		t.Fatal(err)
	}
	rdk := ak.accs[string(authtypes.NewModuleAddress("rdk"))].(*authtypes.ModuleAccount)
	if len(rdk.Permissions) != 0 {
		t.Fatalf("rdk permissions %v, want none", rdk.Permissions)
	}
	// Idempotent: a second run changes nothing.
	before := ak.next
	if err := restrictModuleAccountPermissions(svTestCtx(), ak); err != nil {
		t.Fatal(err)
	}
	if ak.next != before {
		t.Fatal("second run created accounts")
	}
}

// Every Minter grant that survives the v3.2.3 step belongs to a module with a
// MintCoins caller. The stored grants are what bank checks; the table entries
// for the three restricted accounts only matter for lazy creation below the
// plan height (see leastPrivilegeModulePerms).
func TestEveryMinterGrantHasAMintCaller(t *testing.T) {
	effective := map[string][]string{}
	for _, a := range moduleAccPerms {
		effective[a.Account] = a.Permissions
	}
	for _, e := range leastPrivilegeModulePerms {
		effective[e.name] = e.perms
	}
	// Module -> who calls MintCoins with that module name.
	callers := map[string]string{
		"mint":        "x/mint (stock)",
		"transfer":    "ibc-go transfer vouchers",
		"evm":         "cosmos/evm",
		"erc20":       "cosmos/evm x/erc20",
		"precisebank": "cosmos/evm x/precisebank",
		"inflation":   "x/inflation epoch emission",
		"bridge":      "x/bridge inbound deposits (QSR-2026-0225)",
	}
	for name, perms := range effective {
		if slices.Contains(perms, authtypes.Minter) {
			if _, ok := callers[name]; !ok {
				t.Errorf("module %q keeps Minter with no MintCoins caller", name)
			}
		}
	}
	for name := range callers {
		if !slices.Contains(effective[name], authtypes.Minter) {
			t.Errorf("module %q mints but has no Minter grant", name)
		}
	}

	// The restricted modules must not have grown a MintCoins (or, for rdk, a
	// BurnCoins) caller. The proprietary sources are scanned when the full
	// build's tree is given (QORE_HANDLER_ROOTS, as the full test run sets it).
	roots := []string{".."}
	if r := os.Getenv("QORE_HANDLER_ROOTS"); r != "" {
		roots = append(roots, strings.Split(r, string(os.PathListSeparator))...)
	}
	call := regexp.MustCompile(`\.(MintCoins|BurnCoins)\(`)
	for _, root := range roots {
		for _, e := range leastPrivilegeModulePerms {
			dir := filepath.Join(root, "x", e.name)
			_ = filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
					return nil
				}
				b, _ := os.ReadFile(p)
				for _, m := range call.FindAllStringSubmatch(string(b), -1) {
					need := authtypes.Minter
					if m[1] == "BurnCoins" {
						need = authtypes.Burner
					}
					if !slices.Contains(e.perms, need) {
						t.Errorf("%s calls %s but module %q loses %s at v3.2.3", p, m[1], e.name, need)
					}
				}
				return nil
			})
		}
	}
}

// ---- IBC allowed clients backstop (QSR-2026-0346) ----------------------------

type memClientParams struct {
	p    clienttypes.Params
	sets int
}

func (m *memClientParams) GetParams(sdk.Context) clienttypes.Params { return m.p }
func (m *memClientParams) SetParams(_ sdk.Context, p clienttypes.Params) {
	m.p = p
	m.sets++
}

func TestPinIBCAllowedClients(t *testing.T) {
	m := &memClientParams{p: clienttypes.DefaultParams()}
	pinIBCAllowedClients(svTestCtx(), m)
	if !slices.Equal(m.p.AllowedClients, []string{"07-tendermint"}) {
		t.Fatalf("allowed clients %v", m.p.AllowedClients)
	}
	if m.p.IsAllowedClient("09-localhost") {
		t.Fatal("09-localhost still allowed")
	}
	// Governance already chose a list: left alone.
	m = &memClientParams{p: clienttypes.NewParams("07-tendermint", "08-wasm")}
	pinIBCAllowedClients(svTestCtx(), m)
	if m.sets != 0 {
		t.Fatal("a governance-chosen list was overwritten")
	}
}

// ---- vesting creation (QSR-2026-0402) -----------------------------------------

func TestCheckVestingCreation(t *testing.T) {
	ctx := svTestCtx()
	from := sdk.AccAddress([]byte("from________________"))
	victim := sdk.AccAddress([]byte("victim______________"))
	gov := authtypes.NewModuleAddress(govtypes.ModuleName)
	periodic := &vestingtypes.MsgCreatePeriodicVestingAccount{}
	locked := &vestingtypes.MsgCreatePermanentLockedAccount{}

	stored := map[string]circuittypes.Permissions{}
	reader := func(_ sdk.Context, addr []byte) (circuittypes.Permissions, bool, error) {
		p, ok := stored[string(addr)]
		return p, ok, nil
	}

	for _, msg := range []sdk.Msg{periodic, locked, &vestingtypes.MsgCreateVestingAccount{}} {
		if err := checkVestingCreation(ctx, reader, msg, from, victim); err == nil {
			t.Fatalf("%T for another address accepted from a plain account", msg)
		}
		if err := checkVestingCreation(ctx, reader, msg, from, from); err != nil {
			t.Fatalf("%T for the sender itself refused: %v", msg, err)
		}
		if err := checkVestingCreation(ctx, reader, msg, gov, victim); err != nil {
			t.Fatalf("%T from governance refused: %v", msg, err)
		}
	}

	// An escrow granted LEVEL_SOME_MSGS over the periodic message may deliver
	// periodic schedules, and nothing else.
	stored[string(from)] = circuittypes.Permissions{
		Level:         circuittypes.Permissions_LEVEL_SOME_MSGS,
		LimitTypeUrls: []string{sdk.MsgTypeURL(periodic)},
	}
	if err := checkVestingCreation(ctx, reader, periodic, from, victim); err != nil {
		t.Fatalf("escrow with a periodic grant refused: %v", err)
	}
	if err := checkVestingCreation(ctx, reader, locked, from, victim); err == nil {
		t.Fatal("a periodic grant also admitted a permanent lock")
	}
	stored[string(from)] = circuittypes.Permissions{Level: circuittypes.Permissions_LEVEL_SUPER_ADMIN}
	if err := checkVestingCreation(ctx, reader, locked, from, victim); err != nil {
		t.Fatalf("circuit super admin refused: %v", err)
	}

	boom := errors.New("store")
	failing := func(sdk.Context, []byte) (circuittypes.Permissions, bool, error) {
		return circuittypes.Permissions{}, false, boom
	}
	if err := checkVestingCreation(ctx, failing, locked, from, victim); !errors.Is(err, boom) {
		t.Fatalf("store error swallowed: %v", err)
	}
}

func TestVestingCreationGateFollowsTheSwitch(t *testing.T) {
	ctx := svTestCtx()
	from := sdk.AccAddress([]byte("from________________"))
	victim := sdk.AccAddress([]byte("victim______________"))
	msg := &vestingtypes.MsgCreateVestingAccount{}

	app := &QoreChainApp{PQCKeeper: nil}
	if err := app.vestingCreationGate(ctx, msg, from, victim); err != nil {
		t.Fatalf("no feature keeper: %v", err)
	}
	off := &QoreChainApp{PQCKeeper: gateFakePQC{enabled: map[string]bool{}}}
	if err := off.vestingCreationGate(ctx, msg, from, victim); err != nil {
		t.Fatalf("switch off must keep the old behaviour: %v", err)
	}
	// Switch on: the rule applies (paths that need no circuit read).
	on := &QoreChainApp{PQCKeeper: gateFakePQC{enabled: map[string]bool{pqctypes.FeatureVestingCreateSelfOnly: true}}}
	if err := on.vestingCreationGate(ctx, msg, from, from); err != nil {
		t.Fatalf("self creation refused: %v", err)
	}
	if err := on.vestingCreationGate(ctx, msg, authtypes.NewModuleAddress(govtypes.ModuleName), victim); err != nil {
		t.Fatalf("governance refused: %v", err)
	}
}
