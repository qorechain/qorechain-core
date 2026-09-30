package keeper

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"cosmossdk.io/core/comet"
	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/reputation/types"
)

type goldenVal struct {
	a []byte
	p int64
}

func (v goldenVal) Address() []byte { return v.a }
func (v goldenVal) Power() int64    { return v.p }

type goldenVote struct {
	v goldenVal
	f comet.BlockIDFlag
}

func (v goldenVote) Validator() comet.Validator        { return v.v }
func (v goldenVote) GetBlockIDFlag() comet.BlockIDFlag { return v.f }

type goldenVotes []goldenVote

func (v goldenVotes) Len() int                 { return len(v) }
func (v goldenVotes) Get(i int) comet.VoteInfo { return v[i] }

type goldenInfo struct{ votes goldenVotes }

func (g goldenInfo) GetEvidence() comet.EvidenceList { return nil }
func (g goldenInfo) GetValidatorsHash() []byte       { return nil }
func (g goldenInfo) GetProposerAddress() []byte      { return nil }
func (g goldenInfo) GetLastCommit() comet.CommitInfo { return g }
func (g goldenInfo) Round() int32                    { return 0 }
func (g goldenInfo) Votes() comet.VoteInfos          { return g.votes }

// legacyGoldenHash is the sha256 of the reputation store after the scenario
// below, computed with the v3.2.2 base code (adb2327), before reputation
// scoring v2 existed. The historical EndBlocker must keep producing exactly
// this state, so blocks below the switch replay byte-identically.
const legacyGoldenHash = "913c32a31f6f9e978c6245342b91556861c2735d3b4ebb6a14fa25f253079617"

func TestLegacyEndBlockerGolden(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	ms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := ms.LoadLatestVersion(); err != nil {
		t.Fatal(err)
	}
	k := NewKeeper(codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), storeKey, nil, log.NewNopLogger())
	// Installed but off, as on every node below the v3.2.2 plan height.
	k.SetFeatureGate(func(sdk.Context) bool { return false })
	a := func(n byte) []byte { return bytes.Repeat([]byte{n}, 20) }
	ctx := sdk.NewContext(ms, cmtproto.Header{Height: 1, ProposerAddress: a(1)}, false, log.NewNopLogger()).
		WithBlockTime(time.Unix(1_700_000_000, 0).UTC())

	k.SetValidatorReputation(ctx, types.ValidatorReputation{Address: sdk.ConsAddress(a(9)).String(), JoinedAtHeight: 1, UptimeBlocks: 7, CommunityVotes: 3, StakeScore: "0.25"})
	for h := int64(990); h < 1030; h++ {
		votes := goldenVotes{
			{goldenVal{a(1), 100}, comet.BlockIDFlagCommit},
			{goldenVal{a(2), 60}, comet.BlockIDFlagCommit},
		}
		if h%3 == 0 {
			votes = append(votes, goldenVote{goldenVal{a(3), 10}, comet.BlockIDFlagAbsent})
		} else {
			votes = append(votes, goldenVote{goldenVal{a(3), 10}, comet.BlockIDFlagNil})
		}
		if h%7 == 0 {
			votes[1].f = comet.BlockIDFlagAbsent
		}
		prop := a(1)
		if h%2 == 0 {
			prop = a(2)
		}
		bctx := ctx.WithBlockHeight(h).WithCometInfo(goldenInfo{votes: votes})
		hdr := bctx.BlockHeader()
		hdr.ProposerAddress = prop
		bctx = bctx.WithBlockHeader(hdr).WithBlockTime(time.Unix(1_700_000_000+h*2, 0).UTC())
		if err := k.EndBlocker(bctx); err != nil {
			t.Fatal(err)
		}
	}

	h := sha256.New()
	it := ctx.KVStore(storeKey).Iterator(nil, nil)
	for ; it.Valid(); it.Next() {
		h.Write(it.Key())
		h.Write([]byte{0})
		h.Write(it.Value())
		h.Write([]byte{0})
	}
	it.Close()
	if got := hex.EncodeToString(h.Sum(nil)); got != legacyGoldenHash {
		t.Fatalf("legacy EndBlocker state hash %s, want %s", got, legacyGoldenHash)
	}
}
