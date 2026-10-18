package types

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	inflationtypes "github.com/qorechain/qorechain-core/x/inflation/types"
)

func bps(n int64) math.LegacyDec { return math.LegacyNewDec(n) }

func TestDefaultParamsAreTheDecidedValues(t *testing.T) {
	p := DefaultParams()
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	for months, want := range map[uint32]uint32{3: 150, 6: 300, 12: 500} {
		got, ok := p.TierBonus(months)
		if !ok || got != want {
			t.Fatalf("%d months: got %d (%v), want %d", months, got, ok, want)
		}
	}
	if _, ok := p.TierBonus(9); ok {
		t.Fatal("9 months is not a tier")
	}
	if p.LightNodeBonusBps != 200 || p.ValidatorBonusBps != 300 || p.MaxTotalAPYBps != 2500 ||
		p.EarlyUnlockPenaltyBps != 500 || !p.Enabled {
		t.Fatalf("defaults drifted: %+v", p)
	}
}

func TestParamsValidation(t *testing.T) {
	cases := map[string]func(*Params){
		"no tiers":          func(p *Params) { p.LockTiers = nil },
		"unknown term":      func(p *Params) { p.LockTiers[0].Months = 4 },
		"unsorted terms":    func(p *Params) { p.LockTiers[0], p.LockTiers[1] = p.LockTiers[1], p.LockTiers[0] },
		"duplicate term":    func(p *Params) { p.LockTiers[1].Months = 3 },
		"tier over 100%":    func(p *Params) { p.LockTiers[2].BonusBps = 10_001 },
		"light node > 100%": func(p *Params) { p.LightNodeBonusBps = 10_001 },
		"validator > 100%":  func(p *Params) { p.ValidatorBonusBps = 10_001 },
		"penalty > 100%":    func(p *Params) { p.EarlyUnlockPenaltyBps = 10_001 },
		"zero ceiling":      func(p *Params) { p.MaxTotalAPYBps = 0 },
		"ceiling > 100%":    func(p *Params) { p.MaxTotalAPYBps = 10_001 },
	}
	for name, mutate := range cases {
		p := DefaultParams()
		mutate(&p)
		if err := p.Validate(); err == nil {
			t.Fatalf("%s: accepted", name)
		}
	}
	// Governance may stop offering a term.
	p := DefaultParams()
	p.LockTiers = p.LockTiers[1:]
	if err := p.Validate(); err != nil {
		t.Fatalf("dropping a term must be allowed: %v", err)
	}
}

// Each tier on its own, and the stacking rule.
func TestLockBonusStacking(t *testing.T) {
	p := DefaultParams()
	for months, want := range map[uint32]uint32{3: 150, 6: 300, 12: 500} {
		tier, _ := p.TierBonus(months)
		if got := LockBonusBps(p, tier, false, false); got != want {
			t.Fatalf("%d months alone: %d, want %d", months, got, want)
		}
		if got := LockBonusBps(p, tier, true, false); got != want+200 {
			t.Fatalf("%d months + light node: %d", months, got)
		}
		if got := LockBonusBps(p, tier, false, true); got != want+300 {
			t.Fatalf("%d months + validator: %d", months, got)
		}
		if got := LockBonusBps(p, tier, true, true); got != want+500 {
			t.Fatalf("%d months + both: %d", months, got)
		}
	}
}

// The 25% ceiling cuts the bonus to the room above base yield.
func TestCappedBonus(t *testing.T) {
	// 15% base, 12-month lock + light node + validator = 10 points: 25% exactly.
	if got := CappedBonusBps(1000, bps(1500), 2500); !got.Equal(bps(1000)) {
		t.Fatalf("at the ceiling: %s", got)
	}
	// 18% base: only 7 points of room.
	if got := CappedBonusBps(1000, bps(1800), 2500); !got.Equal(bps(700)) {
		t.Fatalf("over the ceiling: %s, want 700", got)
	}
	// Base already above the ceiling: no bonus, never negative.
	if got := CappedBonusBps(500, bps(2600), 2500); !got.IsZero() {
		t.Fatalf("base above ceiling: %s", got)
	}
	// Fractional base.
	if got := CappedBonusBps(500, math.LegacyMustNewDecFromStr("2100.5"), 2500); !got.Equal(math.LegacyMustNewDecFromStr("399.5")) {
		t.Fatalf("fractional room: %s", got)
	}
	if got := CappedBonusBps(500, math.LegacyDec{}, 2500); !got.Equal(bps(500)) {
		t.Fatalf("nil base reads as zero: %s", got)
	}
}

func TestBaseAPYAndCommission(t *testing.T) {
	bonded := math.NewInt(1_000_000_000_000)
	reach := math.LegacyMustNewDecFromStr("0.47")
	// target_apy minting for a day at 1667 bps gives a base of 1667 bps.
	day := int64(86_400)
	emission := inflationtypes.TargetAPYEmission(bonded, 1667, reach, day)
	base := BaseAPYBps(emission, bonded, reach, day)
	if diff := base.Sub(bps(1667)).Abs(); diff.GT(math.LegacyMustNewDecFromStr("0.01")) {
		t.Fatalf("base %s, want about 1667", base)
	}
	net := NetOfCommission(bps(1667), math.LegacyMustNewDecFromStr("0.10"))
	if !net.Equal(math.LegacyMustNewDecFromStr("1500.3")) {
		t.Fatalf("net %s, want 1500.3", net)
	}
	if !NetOfCommission(bps(1000), math.LegacyMustNewDecFromStr("1.5")).IsZero() {
		t.Fatal("commission over 1 leaves nothing")
	}
	if !BaseAPYBps(emission, bonded, reach, 0).IsZero() || !BaseAPYBps(emission, math.ZeroInt(), reach, day).IsZero() {
		t.Fatal("degenerate inputs give zero")
	}
}

func TestBonusOwedAndProRata(t *testing.T) {
	tokens := math.NewInt(1_000_000_000) // 1,000 QOR
	// 5% for a full year is 50 QOR.
	if got := BonusOwed(tokens, bps(500), SecondsPerYear); !got.Equal(math.NewInt(50_000_000)) {
		t.Fatalf("a year at 5%%: %s", got)
	}
	// Half a year.
	if got := BonusOwed(tokens, bps(500), SecondsPerYear/2); !got.Equal(math.NewInt(25_000_000)) {
		t.Fatalf("half a year at 5%%: %s", got)
	}
	if got := BonusOwed(tokens, bps(0), SecondsPerYear); !got.IsZero() {
		t.Fatalf("zero rate: %s", got)
	}
	// Pro rata: 60 of 100 funded.
	if got := ProRata(math.NewInt(30), math.NewInt(60), math.NewInt(100)); !got.Equal(math.NewInt(18)) {
		t.Fatalf("pro rata: %s", got)
	}
	if got := ProRata(math.NewInt(30), math.NewInt(100), math.NewInt(100)); !got.Equal(math.NewInt(30)) {
		t.Fatalf("fully funded: %s", got)
	}
	if got := ProRata(math.NewInt(30), math.ZeroInt(), math.NewInt(100)); !got.IsZero() {
		t.Fatalf("nothing funded: %s", got)
	}
}

func TestLockEnd(t *testing.T) {
	start := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	for months, want := range map[uint32]time.Time{
		3:  time.Date(2027, 1, 18, 12, 0, 0, 0, time.UTC),
		6:  time.Date(2027, 4, 18, 12, 0, 0, 0, time.UTC),
		12: time.Date(2027, 10, 18, 12, 0, 0, 0, time.UTC),
	} {
		if got := LockEnd(start, months); !got.Equal(want) {
			t.Fatalf("%d months: %s, want %s", months, got, want)
		}
	}
}

// Every key this module writes lives under its prefix, which no x/inflation
// key shares.
func TestKeysStayUnderThePrefix(t *testing.T) {
	addr := sdk.AccAddress(bytes.Repeat([]byte{7}, 20))
	keys := [][]byte{
		ParamsKey, PoolKey, NextLockIDKey, LockKey(1), DelegatorIndexKey(addr, 1),
		ExpiryQueueKey(1_800_000_000, 1), ClaimableKey(addr),
	}
	for _, k := range keys {
		if !bytes.HasPrefix(k, KeyPrefix) {
			t.Fatalf("key %q is outside %q", k, KeyPrefix)
		}
	}
	for _, k := range [][]byte{inflationtypes.ParamsKey, inflationtypes.CurrentEpochKey, inflationtypes.EmissionLogPrefix} {
		if bytes.HasPrefix(k, KeyPrefix) {
			t.Fatalf("x/inflation key %q collides with %q", k, KeyPrefix)
		}
	}
	if HostStoreKey != inflationtypes.StoreKey {
		t.Fatal("the host store must be x/inflation's")
	}
	// The expiry queue orders by time.
	if bytes.Compare(ExpiryQueueKey(100, 9), ExpiryQueueKey(101, 1)) >= 0 {
		t.Fatal("expiry queue is not ordered by end time")
	}
	if bytes.Compare(ExpiryQueueKey(100, 9), ExpiryQueueEnd(100)) >= 0 || bytes.Compare(ExpiryQueueKey(101, 0), ExpiryQueueEnd(100)) < 0 {
		t.Fatal("ExpiryQueueEnd does not bound the entries due")
	}
}

func testGenesis(t *testing.T) GenesisState {
	t.Helper()
	del := sdk.AccAddress(bytes.Repeat([]byte{1}, 20))
	val := sdk.ValAddress(bytes.Repeat([]byte{2}, 20))
	gs := *DefaultGenesisState()
	gs.NextLockID = 3
	gs.Locks = []Lock{
		{ID: 1, Delegator: del.String(), Validator: val.String(), Shares: math.LegacyNewDec(1000), Months: 6,
			TierBonusBps: 300, StartHeight: 10, StartTime: 1_800_000_000, EndTime: 1_815_000_000, Accrued: math.NewInt(7)},
		{ID: 2, Delegator: del.String(), Validator: val.String(), Shares: math.LegacyNewDec(5), Months: 3,
			TierBonusBps: 150, StartHeight: 11, StartTime: 1_800_000_100, EndTime: 1_808_000_000, Accrued: math.NewInt(3), Released: true},
	}
	gs.Claimable = []Claimable{{Address: del.String(), Amount: math.NewInt(90)}}
	gs.Pool.Outstanding = math.NewInt(100)
	gs.Pool.LockCount = 1
	return gs
}

func TestGenesisValidateAndRoundTrip(t *testing.T) {
	if err := DefaultGenesisState().Validate(); err != nil {
		t.Fatalf("default genesis: %v", err)
	}
	gs := testGenesis(t)
	if err := gs.Validate(); err != nil {
		t.Fatalf("test genesis: %v", err)
	}
	bz, err := json.Marshal(gs)
	if err != nil {
		t.Fatal(err)
	}
	var back GenesisState
	if err := json.Unmarshal(bz, &back); err != nil {
		t.Fatal(err)
	}
	if err := back.Validate(); err != nil {
		t.Fatalf("round trip: %v", err)
	}
	bz2, _ := json.Marshal(back)
	if !bytes.Equal(bz, bz2) {
		t.Fatalf("round trip changed the bytes\n%s\n%s", bz, bz2)
	}

	bad := map[string]func(*GenesisState){
		"outstanding off":      func(g *GenesisState) { g.Pool.Outstanding = math.NewInt(99) },
		"lock count off":       func(g *GenesisState) { g.Pool.LockCount = 2 },
		"duplicate lock id":    func(g *GenesisState) { g.Locks[1].ID = 1 },
		"id not below next":    func(g *GenesisState) { g.NextLockID = 2 },
		"zero next id":         func(g *GenesisState) { g.NextLockID = 0 },
		"bad months":           func(g *GenesisState) { g.Locks[0].Months = 5 },
		"no shares":            func(g *GenesisState) { g.Locks[0].Shares = math.LegacyZeroDec() },
		"end before start":     func(g *GenesisState) { g.Locks[0].EndTime = g.Locks[0].StartTime },
		"negative accrued":     func(g *GenesisState) { g.Locks[0].Accrued = math.NewInt(-1) },
		"bad delegator":        func(g *GenesisState) { g.Locks[0].Delegator = "nope" },
		"bad validator":        func(g *GenesisState) { g.Locks[0].Validator = g.Locks[0].Delegator },
		"duplicate claimant":   func(g *GenesisState) { g.Claimable = append(g.Claimable, g.Claimable[0]) },
		"zero claimable":       func(g *GenesisState) { g.Claimable[0].Amount = math.ZeroInt() },
		"negative pool amount": func(g *GenesisState) { g.Pool.TotalPaid = math.NewInt(-1) },
		"bad params":           func(g *GenesisState) { g.Params.MaxTotalAPYBps = 0 },
	}
	for name, mutate := range bad {
		g := testGenesis(t)
		g.Locks = append([]Lock{}, g.Locks...)
		g.Claimable = append([]Claimable{}, g.Claimable...)
		mutate(&g)
		if err := g.Validate(); err == nil {
			t.Fatalf("%s: accepted", name)
		}
	}

	// Too many open locks for one delegator.
	g := testGenesis(t)
	g.Locks = nil
	g.Claimable = nil
	g.Pool = DefaultPool()
	del := sdk.AccAddress(bytes.Repeat([]byte{1}, 20)).String()
	val := sdk.ValAddress(bytes.Repeat([]byte{2}, 20)).String()
	for i := 1; i <= MaxLocksPerDelegator+1; i++ {
		g.Locks = append(g.Locks, Lock{ID: uint64(i), Delegator: del, Validator: val, Shares: math.LegacyOneDec(),
			Months: 3, TierBonusBps: 150, StartTime: 1, EndTime: 2, Accrued: math.ZeroInt()})
	}
	g.NextLockID = uint64(MaxLocksPerDelegator + 2)
	g.Pool.LockCount = uint64(MaxLocksPerDelegator + 1)
	if err := g.Validate(); err == nil {
		t.Fatal("more than MaxLocksPerDelegator open locks accepted")
	}
}

func TestMsgValidateBasic(t *testing.T) {
	del := sdk.AccAddress(bytes.Repeat([]byte{1}, 20)).String()
	val := sdk.ValAddress(bytes.Repeat([]byte{2}, 20)).String()
	ok := MsgLockDelegation{Delegator: del, Validator: val, Amount: math.NewInt(5), Months: 12}
	if err := ok.ValidateBasic(); err != nil {
		t.Fatal(err)
	}
	for name, m := range map[string]MsgLockDelegation{
		"bad months":  {Delegator: del, Validator: val, Amount: math.NewInt(5), Months: 1},
		"zero amount": {Delegator: del, Validator: val, Amount: math.ZeroInt(), Months: 3},
		"nil amount":  {Delegator: del, Validator: val, Months: 3},
		"bad val":     {Delegator: del, Validator: del, Amount: math.NewInt(5), Months: 3},
		"bad del":     {Delegator: "x", Validator: val, Amount: math.NewInt(5), Months: 3},
	} {
		if err := m.ValidateBasic(); err == nil {
			t.Fatalf("%s: accepted", name)
		}
	}
	if err := (MsgUnlockEarly{Delegator: del}).ValidateBasic(); err == nil {
		t.Fatal("lock id 0 accepted")
	}
	if err := (MsgClaimBonus{Holder: "x"}).ValidateBasic(); err == nil {
		t.Fatal("bad holder accepted")
	}
	up := MsgUpdateParams{Authority: del, LockTiers: []LockTierParam{{Months: 3, BonusBps: 150}},
		LightNodeBonusBps: 200, ValidatorBonusBps: 300, MaxTotalApyBps: 2500, EarlyUnlockPenaltyBps: 500, Enabled: true}
	if err := up.ValidateBasic(); err != nil {
		t.Fatal(err)
	}
	if p := up.ToParams(); p.MaxTotalAPYBps != 2500 || len(p.LockTiers) != 1 || !p.Enabled {
		t.Fatalf("ToParams: %+v", p)
	}
	up.MaxTotalApyBps = 0
	if err := up.ValidateBasic(); err == nil {
		t.Fatal("invalid params accepted")
	}
}
