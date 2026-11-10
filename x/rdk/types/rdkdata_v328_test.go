package types

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Records written before v3.2.8 keep their bytes: the new rollup, batch,
// params and genesis fields are omitted while unset.
func TestRDKDataFieldsOmittedWhenUnset(t *testing.T) {
	for name, v := range map[string]any{
		"params":  DefaultParams(),
		"genesis": DefaultGenesisState(),
		"rollup":  RollupConfig{RollupID: "r"},
		"batch":   SettlementBatch{RollupID: "r"},
	} {
		bz, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		for _, field := range []string{"sequencing", "inbox_start", "inbox_end", "posted_data", "inboxes",
			"max_posted_data_bytes", "posted_data_fee_per_byte", "posted_data_retention_blocks"} {
			if bytes.Contains(bz, []byte(`"`+field+`"`)) {
				t.Fatalf("%s carries %s while unset: %s", name, field, bz)
			}
		}
	}
	if p := (&MsgUpdateParams{MaxRollups: 1}).ToParams(); p.HasPostedDataParams() {
		t.Fatalf("a message without the posted-data fields sets them: %+v", p)
	}
}

func TestPostedDataParamsDefaultsAndBounds(t *testing.T) {
	p := DefaultParams()
	if p.PostedDataMaxBytes() != 128<<10 || p.PostedDataMaxBytesPerBlock() != 512<<10 ||
		p.PostedDataFee() != 10 || p.PostedDataRetention() != 201600 {
		t.Fatalf("defaults: %d %d %d %d", p.PostedDataMaxBytes(), p.PostedDataMaxBytesPerBlock(),
			p.PostedDataFee(), p.PostedDataRetention())
	}
	if err := p.ValidateUpdate(); err != nil {
		t.Fatalf("default params: %v", err)
	}
	for _, c := range []struct {
		name string
		mod  func(*Params)
		ok   bool
	}{
		{"per post at the ceiling with a block cap", func(p *Params) { p.MaxPostedDataBytes = 1 << 20; p.MaxPostedDataBytesPerBlock = 1 << 20 }, true},
		{"per post above the ceiling", func(p *Params) { p.MaxPostedDataBytes = 1<<20 + 1 }, false},
		{"per post above the default block cap", func(p *Params) { p.MaxPostedDataBytes = 1 << 20 }, false},
		{"block cap below the post cap", func(p *Params) { p.MaxPostedDataBytesPerBlock = 1000 }, false},
		{"block cap above its ceiling", func(p *Params) { p.MaxPostedDataBytesPerBlock = 8<<20 + 1 }, false},
		{"fee at its ceiling", func(p *Params) { p.PostedDataFeePerByte = MaxPostedDataFeePerByte }, true},
		{"fee above its ceiling", func(p *Params) { p.PostedDataFeePerByte = MaxPostedDataFeePerByte + 1 }, false},
		{"retention too short", func(p *Params) { p.PostedDataRetentionBlocks = 99 }, false},
		{"retention minimum", func(p *Params) { p.PostedDataRetentionBlocks = 100 }, true},
	} {
		q := DefaultParams()
		c.mod(&q)
		err := q.ValidateUpdate()
		if (err == nil) != c.ok {
			t.Fatalf("%s: err=%v", c.name, err)
		}
		if err != nil && !errors.Is(err, ErrInvalidParams) {
			t.Fatalf("%s: %v is not ErrInvalidParams", c.name, err)
		}
	}
	if BillableBytes(1) != 1024 || BillableBytes(1024) != 1024 || BillableBytes(5000) != 5000 {
		t.Fatal("billable bytes")
	}
}

// The accumulator and range hash are the documented byte layouts, and both
// name the chain and the rollup.
func TestInboxHashes(t *testing.T) {
	dh := PostedDataHash([]byte("tx"))
	want := sha256.Sum256([]byte("tx"))
	if !bytes.Equal(dh, want[:]) {
		t.Fatal("posted data hash is not SHA-256")
	}
	acc := InboxAccumulate("qorechain-diana", "r1", nil, 1, 7, "qor1poster", dh)

	h := sha256.New()
	h.Write([]byte("qorechain/rdk/inbox/v1"))
	lp := func(b []byte) {
		var n [4]byte
		binary.BigEndian.PutUint32(n[:], uint32(len(b)))
		h.Write(n[:])
		h.Write(b)
	}
	u64 := func(v uint64) {
		var b [8]byte
		binary.BigEndian.PutUint64(b[:], v)
		h.Write(b[:])
	}
	lp([]byte("qorechain-diana"))
	lp([]byte("r1"))
	h.Write(make([]byte, 32))
	u64(1)
	u64(7)
	lp([]byte("qor1poster"))
	h.Write(dh)
	if !bytes.Equal(acc, h.Sum(nil)) {
		t.Fatal("accumulator layout differs from the documented one")
	}
	if bytes.Equal(acc, InboxAccumulate("qorechain-vladi", "r1", nil, 1, 7, "qor1poster", dh)) {
		t.Fatal("the accumulator does not name the chain")
	}
	if bytes.Equal(acc, InboxAccumulate("qorechain-diana", "r2", nil, 1, 7, "qor1poster", dh)) {
		t.Fatal("the accumulator does not name the rollup")
	}
	r := InboxRangeHash("qorechain-diana", "r1", 0, 1, nil, acc)
	if bytes.Equal(r, InboxRangeHash("qorechain-diana", "r1", 0, 1, EmptyInboxAcc, acc)) == false {
		t.Fatal("an empty accumulator must stand for 32 zero bytes")
	}
	if bytes.Equal(r, InboxRangeHash("qorechain-diana", "r1", 1, 1, nil, acc)) {
		t.Fatal("the range hash does not name its start")
	}
	if bytes.Equal(r, InboxRangeHash("qorechain-vladi", "r1", 0, 1, nil, acc)) {
		t.Fatal("the range hash does not name the chain")
	}
}

// Only zk (snark) and based-settlement rollups may be based; only a based zk
// (snark) rollup takes batches from anyone.
func TestBasedEligibility(t *testing.T) {
	cfg := func(mode SettlementMode, sys ProofSystem) RollupConfig {
		seq := SequencerDedicated
		if mode == SettlementBased {
			seq = SequencerBased
		}
		return RollupConfig{RollupID: "r", BlockTimeMs: 1, MaxTxPerBlock: 1, StakeAmount: 1, SettlementMode: mode,
			SequencerConfig: SequencerConfig{Mode: seq}, ProofConfig: ProofConfig{System: sys}}
	}
	for _, c := range []struct {
		c        RollupConfig
		eligible bool
		open     bool
	}{
		{cfg(SettlementZK, ProofSystemSNARK), true, true},
		{cfg(SettlementBased, ProofSystemNone), true, false},
		{cfg(SettlementZK, ProofSystemSTARK), false, false},
		{cfg(SettlementOptimistic, ProofSystemFraud), false, false},
		{cfg(SettlementSovereign, ProofSystemNone), false, false},
	} {
		if err := c.c.Validate(); err != nil {
			t.Fatalf("%s/%s not based: %v", c.c.SettlementMode, c.c.ProofConfig.System, err)
		}
		if c.c.PermissionlessBatches() {
			t.Fatalf("%s/%s: open submission without based sequencing", c.c.SettlementMode, c.c.ProofConfig.System)
		}
		b := c.c
		b.Sequencing = SequencingBased
		err := b.Validate()
		if (err == nil) != c.eligible {
			t.Fatalf("%s/%s based: %v", b.SettlementMode, b.ProofConfig.System, err)
		}
		if err != nil && !errors.Is(err, ErrBasedSequencingUnsupported) {
			t.Fatalf("%s/%s: %v", b.SettlementMode, b.ProofConfig.System, err)
		}
		if b.PermissionlessBatches() != c.open {
			t.Fatalf("%s/%s based: permissionless=%v", b.SettlementMode, b.ProofConfig.System, b.PermissionlessBatches())
		}
	}
	bad := cfg(SettlementZK, ProofSystemSNARK)
	bad.Sequencing = "shared"
	if err := bad.Validate(); err == nil {
		t.Fatal("an unknown sequencing mode validated")
	}
}

func TestRDKDataMsgValidateBasic(t *testing.T) {
	addr := sdk.AccAddress(bytes.Repeat([]byte{7}, 20)).String()
	ok := &MsgPostRollupData{Poster: addr, RollupID: "r", Data: []byte{1}}
	if err := ok.ValidateBasic(); err != nil {
		t.Fatal(err)
	}
	for name, m := range map[string]*MsgPostRollupData{
		"bad poster": {Poster: "x", RollupID: "r", Data: []byte{1}},
		"no rollup":  {Poster: addr, Data: []byte{1}},
		"no data":    {Poster: addr, RollupID: "r"},
		"too large":  {Poster: addr, RollupID: "r", Data: make([]byte, MaxPostedDataHardLimit+1)},
	} {
		if err := m.ValidateBasic(); err == nil {
			t.Fatalf("%s passed", name)
		}
	}
	for seq, want := range map[string]bool{"": true, "based": true, "shared": false} {
		err := (&MsgSetRollupSequencing{Authority: addr, RollupID: "r", Sequencing: seq}).ValidateBasic()
		if (err == nil) != want {
			t.Fatalf("sequencing %q: %v", seq, err)
		}
	}
}

func TestPostedDataKeys(t *testing.T) {
	k := PostedDataExpiryKey(1234, "roll/x", 9)
	exp, id, seq, ok := ParsePostedDataExpiryKey(k[len(PostedDataExpiryPrefix):])
	if !ok || exp != 1234 || id != "roll/x" || seq != 9 {
		t.Fatalf("parse: %d %q %d %v", exp, id, seq, ok)
	}
	// Length prefixes keep "a" and "a/b" apart.
	if bytes.HasPrefix(PostedDataKey("a/b", 1), PostedDataRollupPrefix("a")) {
		t.Fatal("rollup a's prefix covers rollup a/b")
	}
	// Expiry keys sort by height.
	if bytes.Compare(PostedDataExpiryKey(9, "z", 1), PostedDataExpiryKey(10, "a", 1)) >= 0 {
		t.Fatal("expiry keys do not sort by height")
	}
}

func TestGenesisPostedDataValidation(t *testing.T) {
	r := RollupConfig{RollupID: "r", BlockTimeMs: 1, MaxTxPerBlock: 1, StakeAmount: 1,
		SettlementMode: SettlementBased, SequencerConfig: SequencerConfig{Mode: SequencerBased},
		ProofConfig: ProofConfig{System: ProofSystemNone}}
	h := PostedDataHash([]byte("d"))
	acc := InboxAccumulate("c", "r", nil, 1, 1, "p", h)
	good := GenesisState{Params: DefaultParams(), Rollups: []RollupConfig{r},
		Inboxes:    []InboxState{{RollupID: "r", HeadSequence: 1, HeadAcc: acc}},
		PostedData: []PostedDataRecord{{RollupID: "r", Sequence: 1, DataHash: h, InboxAcc: acc, Size: 1}}}
	if err := good.Validate(); err != nil {
		t.Fatal(err)
	}
	for name, mod := range map[string]func(*GenesisState){
		"unknown rollup inbox": func(g *GenesisState) { g.Inboxes[0].RollupID = "x" },
		"duplicate inbox":      func(g *GenesisState) { g.Inboxes = append(g.Inboxes, g.Inboxes[0]) },
		"consumed above head":  func(g *GenesisState) { g.Inboxes[0].ConsumedSequence = 2 },
		"record above head":    func(g *GenesisState) { g.PostedData[0].Sequence = 2 },
		"record without inbox": func(g *GenesisState) { g.Inboxes = nil },
		"duplicate record":     func(g *GenesisState) { g.PostedData = append(g.PostedData, g.PostedData[0]) },
		"short hash":           func(g *GenesisState) { g.PostedData[0].DataHash = []byte{1} },
		"bad params":           func(g *GenesisState) { g.Params.PostedDataRetentionBlocks = 1 },
	} {
		g := good
		g.Inboxes = append([]InboxState(nil), good.Inboxes...)
		g.PostedData = append([]PostedDataRecord(nil), good.PostedData...)
		mod(&g)
		if err := g.Validate(); err == nil {
			t.Fatalf("%s validated", name)
		} else if !strings.Contains(err.Error(), "") {
			t.Fatal(err)
		}
	}
}
