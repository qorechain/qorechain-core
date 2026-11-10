package qledger

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"errors"
	"testing"

	"github.com/cloudflare/circl/sign/mldsa/mldsa44"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/rdk/types"
)

const (
	testChain  = "qorechain-qledger-test"
	testRollup = "ledger-roll"
)

type edUser struct {
	pub  ed25519.PublicKey
	priv ed25519.PrivateKey
}

func newEd(seed byte) edUser {
	priv := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{seed}, 32))
	return edUser{pub: priv.Public().(ed25519.PublicKey), priv: priv}
}

func (u edUser) id() [32]byte { return AccountID(SchemeEd25519, u.pub) }

func (u edUser) transfer(nonce uint64, to [32]byte, amt uint64) []byte {
	tx := Tx{Scheme: SchemeEd25519, PubKey: u.pub, Kind: KindTransfer, Nonce: nonce, To: to, Amount: amt}
	tx.SignEd25519(testChain, testRollup, u.priv)
	return tx.Encode()
}

func (u edUser) withdraw(nonce uint64, rcpt string, amt uint64) []byte {
	tx := Tx{Scheme: SchemeEd25519, PubKey: u.pub, Kind: KindWithdraw, Nonce: nonce, Recipient: rcpt, Amount: amt}
	tx.SignEd25519(testChain, testRollup, u.priv)
	return tx.Encode()
}

func deposit(id [32]byte, amt uint64) Entry {
	return Entry{Height: 5, Poster: DepositPosterPrefix + "someone", Payload: EncodeDeposit(id, amt)}
}

func post(payload []byte) Entry { return Entry{Height: 6, Poster: "poster", Payload: payload} }

var env = Env{ChainID: testChain, RollupID: testRollup}

// rangeOf builds the accumulator chain and range hash a based batch over
// entries starting at start (from accFrom) would carry.
func rangeOf(start uint64, accFrom []byte, entries []Entry) []byte {
	acc := accFrom
	for i, e := range entries {
		acc = types.InboxAccumulate(testChain, testRollup, acc, start+uint64(i)+1, e.Height, e.Poster, types.PostedDataHash(e.Payload))
	}
	return types.InboxRangeHash(testChain, testRollup, start, start+uint64(len(entries)), accFrom, acc)
}

// The partial tree from witnesses computes the same roots as the full
// ledger, for present and absent accounts alike.
func TestPartialTreeMatchesLedger(t *testing.T) {
	l := NewLedger()
	if l.Root() != EmptyRoot() {
		t.Fatal("empty ledger root")
	}
	var ids [][32]byte
	for i := 0; i < 20; i++ {
		id := sha256.Sum256([]byte{byte(i)})
		ids = append(ids, id)
		_ = l.Set(id, uint64(100+i), uint64(i%3))
	}
	root := l.Root()
	absent := sha256.Sum256([]byte("absent"))
	ws := []Witness{l.Witness(ids[3]), l.Witness(ids[7]), l.Witness(absent)}
	pt, err := NewPartialTree(root, ws, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range []struct {
		id       [32]byte
		bal, non uint64
	}{{ids[3], 1, 9}, {absent, 55, 0}, {ids[7], 0, 0}, {ids[3], 2, 10}} {
		if err := pt.Set(step.id, step.bal, step.non); err != nil {
			t.Fatal(err)
		}
		_ = l.Set(step.id, step.bal, step.non)
		if pt.Root() != l.Root() {
			t.Fatalf("roots diverge after setting %x", step.id[:4])
		}
	}
	if _, _, err := pt.Get(ids[1]); !errors.Is(err, ErrIncompleteWitness) {
		t.Fatalf("unopened account: %v", err)
	}
	// A witness with a wrong balance, or one that disagrees with another,
	// is refused.
	bad := l.Witness(ids[2])
	bad.Balance++
	if _, err := NewPartialTree(l.Root(), []Witness{bad}, nil); !errors.Is(err, ErrBadWitness) {
		t.Fatalf("wrong balance: %v", err)
	}
	if _, err := NewPartialTree(root, []Witness{l.Witness(ids[2])}, nil); !errors.Is(err, ErrBadWitness) {
		t.Fatalf("witness of another root: %v", err)
	}
}

// Execution: deposits, transfers, withdrawals; everything invalid is a
// no-op.
func TestExecuteRules(t *testing.T) {
	a, b := newEd(1), newEd(2)
	rcpt := sdk.AccAddress(bytes.Repeat([]byte{9}, 20)).String()
	l := NewLedger()
	entries := []Entry{
		deposit(a.id(), 1000),
		post(a.transfer(0, b.id(), 300)),
		post(a.transfer(0, b.id(), 300)),  // replay: wrong nonce
		post(a.transfer(1, b.id(), 5000)), // too much
		post(b.withdraw(0, rcpt, 100)),
		post(b.withdraw(1, "not-an-address", 1)),
		post([]byte("junk")),
		{Height: 6, Poster: DepositPosterPrefix + "x", Payload: []byte("bad deposit")},
		post(a.transfer(1, b.id(), 0)), // zero amount
	}
	res, err := Execute(env, l, entries)
	if err != nil {
		t.Fatal(err)
	}
	if bal, n, _ := l.Get(a.id()); bal != 700 || n != 1 {
		t.Fatalf("a = %d/%d", bal, n)
	}
	if bal, n, _ := l.Get(b.id()); bal != 200 || n != 1 {
		t.Fatalf("b = %d/%d", bal, n)
	}
	if len(res.Withdrawals) != 1 || res.Withdrawals[0].Amount != 100 || res.Withdrawals[0].Recipient != rcpt {
		t.Fatalf("withdrawals %+v", res.Withdrawals)
	}
	if res.StateRoot != l.Root() {
		t.Fatal("result root")
	}
	// A signature for another rollup does not verify here.
	other := Env{ChainID: testChain, RollupID: "other-roll"}
	l2 := NewLedger()
	if _, err := Execute(other, l2, []Entry{deposit(a.id(), 10), post(a.transfer(0, b.id(), 5))}); err != nil {
		t.Fatal(err)
	}
	if bal, _, _ := l2.Get(b.id()); bal != 0 {
		t.Fatal("a transaction signed for one rollup executed on another")
	}
	// Withdrawal proofs verify against the root (single and padded trees).
	ws := []Withdrawal{{0, rcpt, 5}, {1, rcpt, 6}, {2, rcpt, 7}}
	root := WithdrawalsRoot(testChain, testRollup, ws)
	for i := range ws {
		path, err := WithdrawalProof(testChain, testRollup, ws, i)
		if err != nil {
			t.Fatal(err)
		}
		cur := WithdrawalLeaf(testChain, testRollup, ws[i].Index, ws[i].Recipient, ws[i].Amount)
		for lvl, s := range path {
			var sib [32]byte
			copy(sib[:], s)
			if (i>>lvl)&1 == 0 {
				cur = pairHash(cur, sib)
			} else {
				cur = pairHash(sib, cur)
			}
		}
		if !bytes.Equal(cur[:], root) {
			t.Fatalf("withdrawal %d does not prove", i)
		}
	}
	if WithdrawalsRoot(testChain, testRollup, nil) != nil {
		t.Fatal("empty withdrawals root")
	}
}

// ML-DSA-44 accounts sign and execute like Ed25519 ones.
func TestMLDSA44Account(t *testing.T) {
	pk, sk, err := mldsa44.GenerateKey(bytes.NewReader(bytes.Repeat([]byte{7}, 64)))
	if err != nil {
		t.Fatal(err)
	}
	pkb, _ := pk.MarshalBinary()
	id := AccountID(SchemeMLDSA44, pkb)
	to := newEd(3).id()
	tx := Tx{Scheme: SchemeMLDSA44, PubKey: pkb, Kind: KindTransfer, Nonce: 0, To: to, Amount: 40}
	if err := tx.SignMLDSA44(testChain, testRollup, sk); err != nil {
		t.Fatal(err)
	}
	enc := tx.Encode()
	if len(enc) > MaxPostBytes {
		t.Fatalf("an ML-DSA-44 transaction is %d bytes, over the post cap", len(enc))
	}
	l := NewLedger()
	if _, err := Execute(env, l, []Entry{deposit(id, 100), post(enc)}); err != nil {
		t.Fatal(err)
	}
	if bal, _, _ := l.Get(to); bal != 40 {
		t.Fatalf("ML-DSA-44 transfer: %d", bal)
	}
	enc[len(enc)-1] ^= 1
	if _, ok := DecodeTx(enc); !ok {
		t.Fatal("decode")
	}
	tx2, _ := DecodeTx(enc)
	if tx2.VerifySignature(testChain, testRollup) {
		t.Fatal("a corrupted ML-DSA-44 signature verified")
	}
}

// The fraud proof: an honest batch is not fraud, a batch with a wrong
// state root or a wrong withdrawals root is, and proofs about other inputs
// or other parent states decide nothing.
func TestCheckFraud(t *testing.T) {
	a, b := newEd(1), newEd(2)
	rcpt := sdk.AccAddress(bytes.Repeat([]byte{8}, 20)).String()
	parent := NewLedger()
	_, _ = Execute(env, parent, []Entry{deposit(a.id(), 1000)})
	accFrom := sha256.Sum256([]byte("acc at start"))
	entries := []Entry{post(a.transfer(0, b.id(), 250)), post(b.withdraw(0, rcpt, 50)), post([]byte("noise"))}
	proof, honest, err := BuildFraudProof(env, parent, accFrom[:], entries)
	if err != nil {
		t.Fatal(err)
	}
	pr := parent.Root()
	claim := Claim{InboxStart: 10, InboxEnd: 13, DataHash: rangeOf(10, accFrom[:], entries), PrevStateRoot: pr[:],
		StateRoot: honest.StateRoot[:], WithdrawalsRoot: honest.WithdrawalsRoot}
	var gas uint64
	counting := Env{ChainID: testChain, RollupID: testRollup, Charge: func(g uint64) { gas += g }}
	if fraud, err := CheckFraud(counting, claim, proof); err != nil || fraud {
		t.Fatalf("honest batch: fraud=%v err=%v", fraud, err)
	}
	if gas == 0 {
		t.Fatal("no gas charged")
	}
	wrongRoot := claim
	wrongRoot.StateRoot = bytes.Repeat([]byte{1}, 32)
	if fraud, err := CheckFraud(env, wrongRoot, proof); err != nil || !fraud {
		t.Fatalf("wrong state root: fraud=%v err=%v", fraud, err)
	}
	// Stealing through the withdrawals root alone is fraud too.
	steal := claim
	steal.WithdrawalsRoot = WithdrawalsRoot(testChain, testRollup, []Withdrawal{{0, rcpt, 1000}})
	if fraud, err := CheckFraud(env, steal, proof); err != nil || !fraud {
		t.Fatalf("wrong withdrawals root: fraud=%v err=%v", fraud, err)
	}
	// Other entries than the batch consumed: not applicable.
	other := claim
	other.DataHash = rangeOf(10, accFrom[:], entries[:2])
	other.InboxEnd = 12
	if _, err := CheckFraud(env, other, proof); !errors.Is(err, ErrProofNotApplicable) {
		t.Fatalf("other inputs: %v", err)
	}
	// Witness against another parent: refused.
	otherParent := claim
	otherParent.PrevStateRoot = bytes.Repeat([]byte{2}, 32)
	if _, err := CheckFraud(env, otherParent, proof); !errors.Is(err, ErrBadWitness) {
		t.Fatalf("other parent: %v", err)
	}
	// A proof missing an account the batch touches decides nothing.
	p, _ := DecodeFraudProof(proof)
	p.Witnesses = p.Witnesses[:1]
	if _, err := CheckFraud(env, wrongRoot, p.Encode()); !errors.Is(err, ErrIncompleteWitness) {
		t.Fatalf("incomplete witness: %v", err)
	}
	// Malformed and oversized proofs are refused.
	if _, err := CheckFraud(env, claim, append(proof, 0)); err == nil {
		t.Fatal("trailing byte accepted")
	}
	if _, err := CheckFraud(env, claim, proof[:len(proof)-1]); err == nil {
		t.Fatal("truncated proof accepted")
	}
	big := FraudProof{Entries: make([]Entry, MaxBatchEntries+1)}
	if _, err := DecodeFraudProof(big.Encode()); err == nil {
		t.Fatal("too many entries accepted")
	}
}

// Withdrawal indices run across batches (the chain's replay protection is
// keyed by the leaf), each batch's tree places them at index mod its size,
// and nothing can be sent to the counter slot.
func TestWithdrawalIndicesAreGlobal(t *testing.T) {
	a := newEd(1)
	rcpt := sdk.AccAddress(bytes.Repeat([]byte{7}, 20)).String()
	l := NewLedger()
	r1, _ := Execute(env, l, []Entry{deposit(a.id(), 100), post(a.withdraw(0, rcpt, 1)), post(a.withdraw(1, rcpt, 1)),
		post(a.withdraw(2, rcpt, 1))})
	r2, _ := Execute(env, l, []Entry{post(a.withdraw(3, rcpt, 1)), post(a.withdraw(4, rcpt, 1)),
		post(a.transfer(5, WithdrawalCounterID, 50)), deposit(WithdrawalCounterID, 9)})
	if len(r1.Withdrawals) != 3 || r2.Withdrawals[0].Index != 3 || r2.Withdrawals[1].Index != 4 {
		t.Fatalf("indices %+v %+v", r1.Withdrawals, r2.Withdrawals)
	}
	if next, _, _ := l.Get(WithdrawalCounterID); next != 5 {
		t.Fatalf("counter %d after a transfer and a deposit to it", next)
	}
	if bal, n, _ := l.Get(a.id()); bal != 95 || n != 5 {
		t.Fatalf("a = %d/%d (the transfer to the counter must be a no-op)", bal, n)
	}
	// Identical withdrawals in two batches have different leaves.
	if WithdrawalLeaf(testChain, testRollup, 0, rcpt, 1) == WithdrawalLeaf(testChain, testRollup, 3, rcpt, 1) {
		t.Fatal("same leaf")
	}
	// Each proof verifies at the position the chain reads from the index.
	for _, ws := range [][]Withdrawal{r1.Withdrawals, r2.Withdrawals} {
		root := WithdrawalsRoot(testChain, testRollup, ws)
		for i, w := range ws {
			path, _ := WithdrawalProof(testChain, testRollup, ws, i)
			cur := WithdrawalLeaf(testChain, testRollup, w.Index, w.Recipient, w.Amount)
			for lvl, s := range path {
				var sib [32]byte
				copy(sib[:], s)
				if (w.Index>>uint(lvl))&1 == 0 {
					cur = pairHash(cur, sib)
				} else {
					cur = pairHash(sib, cur)
				}
			}
			if !bytes.Equal(cur[:], root) {
				t.Fatalf("withdrawal %d does not prove", w.Index)
			}
		}
	}
}
