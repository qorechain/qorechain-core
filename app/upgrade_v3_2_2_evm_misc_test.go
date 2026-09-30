package app

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
)

type codeHashLog struct{ deleted []common.Address }

func (l *codeHashLog) DeleteCodeHash(_ sdk.Context, addr common.Address) {
	l.deleted = append(l.deleted, addr)
}

// newUpgradeAccountKeeper is an auth keeper on an in-memory store with the
// application's real module-account permission table.
func newUpgradeAccountKeeper(t *testing.T) (authkeeper.AccountKeeper, sdk.Context) {
	t.Helper()
	reg := codectypes.NewInterfaceRegistry()
	authtypes.RegisterInterfaces(reg)
	vestingtypes.RegisterInterfaces(reg)
	cryptocodec.RegisterInterfaces(reg)
	cdc := codec.NewProtoCodec(reg)

	perms := map[string][]string{}
	for _, p := range moduleAccPerms {
		perms[p.Account] = p.Permissions
	}
	key := storetypes.NewKVStoreKey(authtypes.StoreKey)
	ctx := testutil.DefaultContext(key, storetypes.NewTransientStoreKey("t_auth"))
	ak := authkeeper.NewAccountKeeper(cdc, runtime.NewKVStoreService(key), authtypes.ProtoBaseAccount,
		perms, addresscodec.NewBech32Codec("qor"), "qor", authtypes.NewModuleAddress("gov").String())
	return ak, ctx
}

func countEvents(ctx sdk.Context, typ string) int {
	n := 0
	for _, ev := range ctx.EventManager().Events() {
		if ev.Type == typ {
			n++
		}
	}
	return n
}

// QSR-2026-0304: after the v3.2.2 step every module account in the table is a
// stored ModuleAccount with its permissions, whatever was at its address
// before, and existing accounts keep their account numbers.
func TestMaterialiseModuleAccounts(t *testing.T) {
	ak, ctx := newUpgradeAccountKeeper(t)
	names := moduleAccountNames(moduleAccPerms)

	// The state the live networks are in: the modules that have run are
	// materialised, the others have nothing stored. Plus an ordinary account
	// planted at xqore the way the EVM did it (no key, sequence 0), holding a
	// code hash, and a vesting account planted at amm.
	for _, name := range []string{"fee_collector", "distribution", "mint", "bonded_tokens_pool", "evm", "erc20", "burn", "svm"} {
		ak.GetModuleAccount(ctx, name)
	}
	user := ak.NewAccountWithAddress(ctx, sdk.AccAddress([]byte("an-ordinary-account!")))
	ak.SetAccount(ctx, user)

	xqore := authtypes.NewModuleAddress("xqore")
	planted := ak.NewAccountWithAddress(ctx, xqore)
	ak.SetAccount(ctx, planted)
	plantedNumber := planted.GetAccountNumber()

	amm := authtypes.NewModuleAddress("amm")
	ammBase := ak.NewAccountWithAddress(ctx, amm).(*authtypes.BaseAccount)
	vesting, err := vestingtypes.NewDelayedVestingAccount(ammBase, sdk.NewCoins(sdk.NewInt64Coin("uqor", 1)), 1)
	if err != nil {
		t.Fatal(err)
	}
	ak.SetAccount(ctx, vesting)

	before := map[string]uint64{}
	for _, name := range names {
		if acc := ak.GetAccount(ctx, authtypes.NewModuleAddress(name)); acc != nil {
			before[name] = acc.GetAccountNumber()
		}
	}
	userNumber := ak.GetAccount(ctx, user.GetAddress()).GetAccountNumber()

	evm := &codeHashLog{}
	if err := materialiseModuleAccounts(ctx, ak, evm, names); err != nil {
		t.Fatal(err)
	}

	for _, p := range moduleAccPerms {
		acc := ak.GetAccount(ctx, authtypes.NewModuleAddress(p.Account))
		macc, ok := acc.(sdk.ModuleAccountI)
		if !ok {
			t.Fatalf("%s: %T is not a module account", p.Account, acc)
		}
		if macc.GetName() != p.Account {
			t.Fatalf("%s: module account named %q", p.Account, macc.GetName())
		}
		if strings.Join(macc.GetPermissions(), ",") != strings.Join(p.Permissions, ",") {
			t.Fatalf("%s: permissions %v, want %v", p.Account, macc.GetPermissions(), p.Permissions)
		}
		if n, had := before[p.Account]; had && macc.GetAccountNumber() != n {
			t.Fatalf("%s: account number moved from %d to %d", p.Account, n, macc.GetAccountNumber())
		}
		// The SDK's own accessor no longer panics for any module.
		if got := ak.GetModuleAccount(ctx, p.Account); got == nil {
			t.Fatalf("%s: GetModuleAccount returned nil", p.Account)
		}
	}
	if got := ak.GetAccount(ctx, xqore).GetAccountNumber(); got != plantedNumber {
		t.Fatalf("repaired xqore account number %d, want %d", got, plantedNumber)
	}
	if got := ak.GetAccount(ctx, user.GetAddress()).GetAccountNumber(); got != userNumber {
		t.Fatal("an ordinary account was touched")
	}
	// Table order: xqore comes before amm.
	if len(evm.deleted) != 2 || evm.deleted[0] != common.BytesToAddress(xqore) || evm.deleted[1] != common.BytesToAddress(amm) {
		t.Fatalf("code hash cleanup: %v", evm.deleted)
	}
	if got := countEvents(ctx, EventTypeModuleAccountRepaired); got != 2 {
		t.Fatalf("%d repair events, want 2 (xqore, amm)", got)
	}
	if got, want := countEvents(ctx, EventTypeModuleAccountMaterialised), len(names)-len(before); got != want {
		t.Fatalf("%d materialise events, want %d", got, want)
	}

	// Idempotent: a second run changes nothing and says nothing.
	ctx = ctx.WithEventManager(sdk.NewEventManager())
	evm.deleted = nil
	if err := materialiseModuleAccounts(ctx, ak, evm, names); err != nil {
		t.Fatal(err)
	}
	if len(ctx.EventManager().Events()) != 0 || len(evm.deleted) != 0 {
		t.Fatal("second run was not a no-op")
	}
}

// An account with a public key at a module address cannot be a planted one;
// the step refuses to convert it instead of guessing.
func TestMaterialiseModuleAccountsRefusesKeyedAccount(t *testing.T) {
	ak, ctx := newUpgradeAccountKeeper(t)
	acc := ak.NewAccountWithAddress(ctx, authtypes.NewModuleAddress("bridge"))
	if err := acc.SetPubKey(secp256k1.GenPrivKey().PubKey()); err != nil {
		t.Fatal(err)
	}
	ak.SetAccount(ctx, acc)
	err := materialiseModuleAccounts(ctx, ak, &codeHashLog{}, moduleAccountNames(moduleAccPerms))
	if err == nil || !strings.Contains(err.Error(), "public key") {
		t.Fatalf("keyed account converted (err %v)", err)
	}
}

// A name missing from the auth keeper's permission table is a wiring error the
// upgrade must not paper over.
func TestMaterialiseModuleAccountsUnknownModule(t *testing.T) {
	ak, ctx := newUpgradeAccountKeeper(t)
	if err := materialiseModuleAccounts(ctx, ak, &codeHashLog{}, []string{"no-such-module"}); err == nil {
		t.Fatal("unknown module accepted")
	}
}
