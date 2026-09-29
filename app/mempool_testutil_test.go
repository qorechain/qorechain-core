package app

import (
	"context"
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	protov2 "google.golang.org/protobuf/proto"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
)

// memTx is a minimal transaction for mempool and proposal tests. Its encoding
// is its id followed by padding, so its size on the wire is chosen by the test.
type memTx struct {
	id      int
	signer  sdk.AccAddress
	seq     uint64
	gas     uint64
	size    int
	fee     int64 // priority: fee per gas
	invalid bool  // the ante rejects it whatever the state
	msgs    []sdk.Msg
}

func (t *memTx) GetMsgs() []sdk.Msg                    { return t.msgs }
func (t *memTx) GetMsgsV2() ([]protov2.Message, error) { return nil, nil }
func (t *memTx) GetGas() uint64                        { return t.gas }

func addr(i int) sdk.AccAddress {
	b := make([]byte, 20)
	binary.BigEndian.PutUint32(b[16:], uint32(i))
	b[0] = 1
	return b
}

// memTxSigners extracts the single signer and sequence of a memTx.
type memTxSigners struct{}

func (memTxSigners) GetSigners(tx sdk.Tx) ([]sdkmempool.SignerData, error) {
	t, ok := tx.(*memTx)
	if !ok {
		return nil, fmt.Errorf("unexpected tx %T", tx)
	}
	return []sdkmempool.SignerData{sdkmempool.NewSignerData(t.signer, t.seq)}, nil
}

func memTxPriority() sdkmempool.TxPriority[math.Int] {
	return sdkmempool.TxPriority[math.Int]{
		GetTxPriority: func(_ context.Context, tx sdk.Tx) math.Int {
			return math.NewInt(tx.(*memTx).fee)
		},
		Compare:  func(a, b math.Int) int { return a.BigInt().Cmp(b.BigInt()) },
		MinValue: math.ZeroInt(),
	}
}

func newMemPool(maxTx int) *sdkmempool.PriorityNonceMempool[math.Int] {
	return sdkmempool.NewPriorityMempool(sdkmempool.PriorityNonceMempoolConfig[math.Int]{
		TxPriority:      memTxPriority(),
		SignerExtractor: memTxSigners{},
		MaxTx:           maxTx,
	})
}

// memVerifier stands in for baseapp's proposal verification. Like the real
// ante in PrepareProposal/ProcessProposal it is sequential: a tx is valid when
// its sequence is the signer's next one in the state built up so far.
type memVerifier struct {
	mtx      sync.Mutex
	registry map[string]*memTx
	initial  map[string]uint64
	state    map[string]uint64
	delay    time.Duration

	verifyCalls int
	rejected    map[int]bool
}

func newMemVerifier(txs []*memTx) *memVerifier {
	v := &memVerifier{registry: map[string]*memTx{}, initial: map[string]uint64{}, rejected: map[int]bool{}}
	for _, tx := range txs {
		v.registry[string(v.encode(tx))] = tx
	}
	v.reset()
	return v
}

func (v *memVerifier) reset() {
	v.state = map[string]uint64{}
	for k, s := range v.initial {
		v.state[k] = s
	}
}

func (v *memVerifier) encode(tx *memTx) []byte {
	bz := make([]byte, tx.size)
	copy(bz, fmt.Sprintf("tx-%d;", tx.id))
	return bz
}

func (v *memVerifier) verify(tx *memTx) error {
	v.verifyCalls++
	if v.delay > 0 {
		time.Sleep(v.delay)
	}
	k := tx.signer.String()
	if tx.invalid || v.state[k] != tx.seq {
		v.rejected[tx.id] = true
		return fmt.Errorf("tx %d rejected", tx.id)
	}
	v.state[k] = tx.seq + 1
	return nil
}

func (v *memVerifier) PrepareProposalVerifyTx(tx sdk.Tx) ([]byte, error) {
	v.mtx.Lock()
	defer v.mtx.Unlock()
	if err := v.verify(tx.(*memTx)); err != nil {
		return nil, err
	}
	return v.encode(tx.(*memTx)), nil
}

func (v *memVerifier) ProcessProposalVerifyTx(bz []byte) (sdk.Tx, error) {
	tx, err := v.TxDecode(bz)
	if err != nil {
		return nil, err
	}
	v.mtx.Lock()
	defer v.mtx.Unlock()
	if err := v.verify(tx.(*memTx)); err != nil {
		return nil, err
	}
	return tx, nil
}

func (v *memVerifier) TxDecode(bz []byte) (sdk.Tx, error) {
	tx, ok := v.registry[string(bz)]
	if !ok {
		return nil, fmt.Errorf("unknown tx bytes")
	}
	return tx, nil
}

func (v *memVerifier) TxEncode(tx sdk.Tx) ([]byte, error) {
	return v.encode(tx.(*memTx)), nil
}
