package app

import (
	"errors"
	"reflect"
	"testing"

	"cosmossdk.io/log"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	pqcmod "github.com/qorechain/qorechain-core/x/pqc"
	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
)

type gateFakePQC struct {
	pqcmod.PQCKeeper
	enabled map[string]bool
}

func (f gateFakePQC) ProtocolFeatureEnabled(_ sdk.Context, name string) bool { return f.enabled[name] }

type step struct {
	name string
	log  *[]string
	err  error
}

func (s step) AnteHandle(ctx sdk.Context, tx sdk.Tx, sim bool, next sdk.AnteHandler) (sdk.Context, error) {
	*s.log = append(*s.log, s.name)
	if s.err != nil {
		return ctx, s.err
	}
	return next(ctx, tx, sim)
}

func TestFeatureGatedDecorators(t *testing.T) {
	ctx := sdk.NewContext(nil, cmtproto.Header{Height: 7}, false, log.NewNopLogger())
	var got []string
	next := func(c sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
		got = append(got, "next")
		return c, nil
	}
	gate := func(on bool, inner ...sdk.AnteDecorator) featureGatedDecorators {
		return featureGatedDecorators{
			pqc:     gateFakePQC{enabled: map[string]bool{pqctypes.FeatureEVMAnteParity: on}},
			feature: pqctypes.FeatureEVMAnteParity,
			inner:   inner,
		}
	}

	// Off: the inner decorators do not run; the chain continues as if the
	// gate were not there (blocks below the switch replay unchanged).
	if _, err := gate(false, step{"ai", &got, nil}, step{"burn", &got, nil}).AnteHandle(ctx, nil, false, next); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"next"}) {
		t.Fatalf("off: %v", got)
	}

	// On: they run in order, then the chain continues.
	got = nil
	if _, err := gate(true, step{"ai", &got, nil}, step{"burn", &got, nil}).AnteHandle(ctx, nil, false, next); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"ai", "burn", "next"}) {
		t.Fatalf("on: %v", got)
	}

	// On, first refuses: nothing after it runs (a rate-limited EVM tx is not
	// counted toward the milestone burns).
	got = nil
	refused := errors.New("rate limited")
	if _, err := gate(true, step{"ai", &got, refused}, step{"burn", &got, nil}).AnteHandle(ctx, nil, false, next); !errors.Is(err, refused) {
		t.Fatalf("got %v", err)
	}
	if !reflect.DeepEqual(got, []string{"ai"}) {
		t.Fatalf("refused: %v", got)
	}
}
