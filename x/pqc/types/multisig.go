package types

import (
	"bytes"
	"encoding/binary"
	"math/bits"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/gogoproto/proto"
)

// k-of-n hybrid post-quantum multisig (v3.2.7, FeaturePQCMultisig).
//
// A classical multisig account (an SDK LegacyAminoPubKey threshold key) cannot
// hold one ML-DSA key that k of its n members share, and the single-signer
// hybrid path verifies exactly one post-quantum signature against one
// registered key. So under HybridRequired such an account could send nothing
// but its own key registration. This file is the stateless half of the fix:
// the registry record, the extension's bounds and the bytes members sign. The
// ante verification is in the full build (x/pqc/ante_multisig.go).
// Design: testnet/handoff-v327/multisig-design.md.

// Errors of the multisig registry. Codes from 320 keep clear of the ranges
// the other v3.2.x items use.
var (
	ErrMultisigNotEnabled      = errors.Register(ModuleName, 320, "the PQC multisig registry is not enabled on this chain yet")
	ErrInvalidMultisig         = errors.Register(ModuleName, 321, "invalid PQC multisig member set")
	ErrMalformedMultisigSig    = errors.Register(ModuleName, 322, "malformed PQC multisig signature extension")
	ErrMultisigThreshold       = errors.Register(ModuleName, 323, "PQC multisig signature below the threshold")
	ErrMultisigSigInvalid      = errors.Register(ModuleName, 324, "PQC multisig member signature verification failed")
	ErrMultisigSigRequired     = errors.Register(ModuleName, 325, "PQC multisig signature required")
	ErrMultisigNotMultisigAcct = errors.Register(ModuleName, 326, "only a classical multisig account may register a PQC multisig member set")
	ErrMultisigConflict        = errors.Register(ModuleName, 327, "account holds the other kind of PQC registration")
	ErrMultisigMembersMismatch = errors.Register(ModuleName, 328, "PQC multisig member set does not match the classical multisig")
	ErrMultisigMemberKey       = errors.Register(ModuleName, 329, "PQC multisig member has no usable registered key")
)

// MultisigSigTypeURL is the type URL of the PQCMultisigSignature extension.
const MultisigSigTypeURL = "/qorechain.pqc.v1.PQCMultisigSignature"

// MaxPQCMultisigMembers caps n. Every member signature is a 4,627-byte
// ML-DSA-87 signature verified separately, so n bounds both the extension's
// size (about 74 KB at 16) and the verification work of one transaction. The
// classical multisig allows 32 members; a set of more than 16 post-quantum
// members has no use case we know of and doubles the worst case.
const MaxPQCMultisigMembers = 16

// MaxPQCMultisigBitmapLen is the longest bitmap any registered set can need.
const MaxPQCMultisigBitmapLen = (MaxPQCMultisigMembers + 7) / 8

// MultisigBitmapLen is the bitmap length of a set of n members.
func MultisigBitmapLen(n int) int { return (n + 7) / 8 }

// MultisigAlgorithm is the only algorithm a member set may use. Its sizes are
// fixed, so every bound below is exact. Another algorithm needs its own switch.
const MultisigAlgorithm = AlgorithmDilithium5

// mldsa87PublicKeySize and mldsa87SignatureSize are ML-DSA-87's (FIPS 204).
const (
	mldsa87PublicKeySize = 2592
	mldsa87SignatureSize = 4627
)

// MultisigPrefix is the KV prefix of member sets, one per multisig account.
var MultisigPrefix = []byte("pqc/multisig/")

// MultisigKey returns the store key of an account's member set.
func MultisigKey(addr []byte) []byte {
	return append(append([]byte{}, MultisigPrefix...), addr...)
}

// PQCMultisigInfo is a multisig account's registered member set as stored.
// It is a new record under a new prefix, so no historical bytes depend on its
// JSON shape.
//
// The set is tied to the classical multisig: Members[i] is the account of
// the multisig's classical member i and Threshold is the classical
// threshold. No member key is stored: a member's signature is verified
// against the single-signer key its own account holds at that moment, so a
// member's rotation takes effect at once and its old key stops counting the
// moment it is replaced.
type PQCMultisigInfo struct {
	Address         string      `json:"address"`
	AlgorithmID     AlgorithmID `json:"algorithm_id"`
	Threshold       uint32      `json:"threshold"`
	Members         []string    `json:"members"`
	CreatedAtHeight int64       `json:"created_at_height"`
	UpdatedAtHeight int64       `json:"updated_at_height"`
	// Version is 1 for the first registration and grows with every
	// re-registration.
	Version uint64 `json:"version"`
}

// Validate checks a record's shape.
func (m PQCMultisigInfo) Validate() error {
	if _, err := sdk.AccAddressFromBech32(m.Address); err != nil {
		return ErrInvalidMultisig.Wrapf("address: %v", err)
	}
	if m.AlgorithmID != MultisigAlgorithm {
		return ErrInvalidMultisig.Wrapf("member keys must be %s, got %s", MultisigAlgorithm, m.AlgorithmID)
	}
	n := len(m.Members)
	if n == 0 || n > MaxPQCMultisigMembers {
		return ErrInvalidMultisig.Wrapf("a member set has 1 to %d members, got %d", MaxPQCMultisigMembers, n)
	}
	if m.Threshold == 0 || int(m.Threshold) > n {
		return ErrInvalidMultisig.Wrapf("threshold must be between 1 and the %d members, got %d", n, m.Threshold)
	}
	seen := map[string]bool{}
	for i, a := range m.Members {
		if _, err := sdk.AccAddressFromBech32(a); err != nil {
			return ErrInvalidMultisig.Wrapf("member %d: %v", i, err)
		}
		if seen[a] {
			return ErrInvalidMultisig.Wrapf("member %s listed twice", a)
		}
		seen[a] = true
	}
	return nil
}

// ValidateMultisigSet is the stateless rule for a member set: the multisig
// algorithm, 1 <= k <= n <= MaxPQCMultisigMembers, every key of the exact
// size, none degenerate, no two equal. Two equal keys would let one holder
// count twice toward k.
func ValidateMultisigSet(algo AlgorithmID, threshold uint32, keys [][]byte) error {
	if algo != MultisigAlgorithm {
		return ErrInvalidMultisig.Wrapf("member keys must be %s (algorithm %d), got %s",
			MultisigAlgorithm, MultisigAlgorithm, algo)
	}
	n := len(keys)
	if n == 0 || n > MaxPQCMultisigMembers {
		return ErrInvalidMultisig.Wrapf("a member set has 1 to %d keys, got %d", MaxPQCMultisigMembers, n)
	}
	if threshold == 0 || int(threshold) > n {
		return ErrInvalidMultisig.Wrapf("threshold must be between 1 and the %d members, got %d", n, threshold)
	}
	seen := make(map[string]int, n)
	for i, k := range keys {
		if len(k) != mldsa87PublicKeySize {
			return ErrInvalidMultisig.Wrapf("member %d: an ML-DSA-87 public key is %d bytes, got %d",
				i, mldsa87PublicKeySize, len(k))
		}
		if IsDegeneratePublicKey(algo, k) {
			return ErrWeakPQCKey.Wrapf("member %d: the key's t1 part is all zero, so anyone can forge its signatures", i)
		}
		if j, dup := seen[string(k)]; dup {
			return ErrInvalidMultisig.Wrapf("members %d and %d have the same key; one holder would count twice", j, i)
		}
		seen[string(k)] = i
	}
	return nil
}

// ValidateBasic for MsgRegisterPQCMultisig. The message type is new in
// v3.2.7, so no historical block can carry it and this check needs no switch.
func (msg *MsgRegisterPQCMultisig) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return err
	}
	if err := ValidateMultisigSet(msg.AlgorithmID, msg.Threshold, msg.PublicKeys); err != nil {
		return err
	}
	if len(msg.ProofsOfPossession) != len(msg.PublicKeys) {
		return ErrInvalidMultisig.Wrapf("one proof of possession per member key: %d keys, %d proofs",
			len(msg.PublicKeys), len(msg.ProofsOfPossession))
	}
	for i, p := range msg.ProofsOfPossession {
		// Empty: the member's registered key was proven already (checked
		// by the keeper against the member's record).
		if len(p) != 0 && len(p) != mldsa87SignatureSize {
			return ErrInvalidProofOfPossession.Wrapf("member %d: an ML-DSA-87 signature is %d bytes, got %d",
				i, mldsa87SignatureSize, len(p))
		}
	}
	return nil
}

// ValidateStateless is the bound every node applies to the extension before
// reading state or verifying anything: one algorithm, a bitmap no longer than
// the largest set can need, between 1 and MaxPQCMultisigMembers signatures of
// the exact size, as many signatures as set bits, and no signature twice.
// Everything that can make a transaction expensive to judge is refused here.
func (sig PQCMultisigSignature) ValidateStateless() error {
	if sig.AlgorithmID != MultisigAlgorithm {
		return ErrMalformedMultisigSig.Wrapf("algorithm must be %s, got %s", MultisigAlgorithm, sig.AlgorithmID)
	}
	if len(sig.Bitmap) == 0 || len(sig.Bitmap) > MaxPQCMultisigBitmapLen {
		return ErrMalformedMultisigSig.Wrapf("bitmap is 1 to %d bytes, got %d", MaxPQCMultisigBitmapLen, len(sig.Bitmap))
	}
	n := len(sig.Signatures)
	if n == 0 || n > MaxPQCMultisigMembers {
		return ErrMalformedMultisigSig.Wrapf("1 to %d member signatures, got %d", MaxPQCMultisigMembers, n)
	}
	set := 0
	for _, b := range sig.Bitmap {
		set += bits.OnesCount8(b)
	}
	if set != n {
		return ErrMalformedMultisigSig.Wrapf("bitmap marks %d members but %d signatures are attached", set, n)
	}
	seen := make(map[string]struct{}, n)
	for i, s := range sig.Signatures {
		if len(s) != mldsa87SignatureSize {
			return ErrMalformedMultisigSig.Wrapf("signature %d: an ML-DSA-87 signature is %d bytes, got %d",
				i, mldsa87SignatureSize, len(s))
		}
		if _, dup := seen[string(s)]; dup {
			return ErrMalformedMultisigSig.Wrapf("signature %d repeats an earlier one", i)
		}
		seen[string(s)] = struct{}{}
	}
	return nil
}

// Signers returns the member indices the bitmap marks, ascending, for a set
// of n members. The bitmap must be exactly MultisigBitmapLen(n) bytes with no
// bit at or beyond n, so one set of signers has one encoding.
func (sig PQCMultisigSignature) Signers(n int) ([]int, error) {
	if len(sig.Bitmap) != MultisigBitmapLen(n) {
		return nil, ErrMalformedMultisigSig.Wrapf("a set of %d members has a %d-byte bitmap, got %d",
			n, MultisigBitmapLen(n), len(sig.Bitmap))
	}
	var out []int
	for i := 0; i < len(sig.Bitmap)*8; i++ {
		if sig.Bitmap[i/8]&(1<<(uint(i)%8)) == 0 {
			continue
		}
		if i >= n {
			return nil, ErrMalformedMultisigSig.Wrapf("bitmap marks member %d of a set of %d", i, n)
		}
		out = append(out, i)
	}
	return out, nil
}

// MultisigBitmap encodes member indices as a bitmap for a set of n members.
func MultisigBitmap(n int, members []int) ([]byte, error) {
	bm := make([]byte, MultisigBitmapLen(n))
	for _, i := range members {
		if i < 0 || i >= n {
			return nil, ErrMalformedMultisigSig.Wrapf("member %d is outside a set of %d", i, n)
		}
		if bm[i/8]&(1<<(uint(i)%8)) != 0 {
			return nil, ErrMalformedMultisigSig.Wrapf("member %d given twice", i)
		}
		bm[i/8] |= 1 << (uint(i) % 8)
	}
	return bm, nil
}

// MultisigSignBytesDomain is the domain tag of the bytes each member signs.
const MultisigSignBytesDomain = "qorechain-pqc-multisig-v1"

// MultisigSignBytes is what every member of a multisig account's set signs
// for one transaction. Every member signs the same bytes, so the members can
// sign independently and in any order, before anyone knows who else will.
//
//	"qorechain-pqc-multisig-v1" ‖ BE64(len(chainID)) ‖ chainID ‖
//	  BE64(account_number) ‖ BE64(sequence) ‖
//	  BE32(len(address)) ‖ address ‖
//	  BE32(len(B0)) ‖ B0 ‖ BE32(len(A0)) ‖ A0
//
// B0 is the TxBody serialised with every PQC extension option removed (the
// single-signer hybrid one and this one), so a signature does not cover the
// bytes that carry it. A0 is the AuthInfo serialised with its signer_infos
// removed: the fee (amount, gas limit, payer, granter) and the tip stay
// covered, while the classical multisig's mode info and bitmap, which are only
// known once the classical signatures are combined, do not have to be.
//
// What binds the transaction to one account and one moment is stated
// explicitly instead: the chain id, the account number and the sequence are
// those the chain holds for the multisig account when it verifies (the
// classical signature check, later in the same ante, requires the
// transaction's own sequence to be the same one), and the address is the raw
// account address. A signature therefore verifies on one network, for one
// account, at one sequence: it cannot be replayed after the account's next
// transaction, on another chain, or for another account with the same members.
func MultisigSignBytes(chainID string, accountNumber, sequence uint64, address, b0, a0 []byte) []byte {
	out := make([]byte, 0, len(MultisigSignBytesDomain)+8+len(chainID)+16+4+len(address)+4+len(b0)+4+len(a0))
	out = append(out, MultisigSignBytesDomain...)
	out = binary.BigEndian.AppendUint64(out, uint64(len(chainID)))
	out = append(out, chainID...)
	out = binary.BigEndian.AppendUint64(out, accountNumber)
	out = binary.BigEndian.AppendUint64(out, sequence)
	out = binary.BigEndian.AppendUint32(out, uint32(len(address)))
	out = append(out, address...)
	out = binary.BigEndian.AppendUint32(out, uint32(len(b0)))
	out = append(out, b0...)
	out = binary.BigEndian.AppendUint32(out, uint32(len(a0)))
	out = append(out, a0...)
	return out
}

// MultisigMemberPossessionSignBytes is what a member key signs to prove its
// holder consents to be a member of this multisig account on this chain. Its
// domain differs from the single-key proof's (ProofOfPossessionSignBytes), so
// neither proof is accepted as the other.
func MultisigMemberPossessionSignBytes(chainID string, algorithmID uint32, account string, pub []byte) []byte {
	var buf bytes.Buffer
	buf.WriteString("qorechain-pqc-multisig-pop-v1|")
	buf.WriteString(chainID)
	buf.WriteByte('|')
	buf.Write(binary.BigEndian.AppendUint32(nil, algorithmID))
	buf.WriteByte('|')
	buf.WriteString(account)
	buf.WriteByte('|')
	buf.Write(pub)
	return buf.Bytes()
}

// IsMultisigRegistrationMsg reports whether msg registers a member set.
func IsMultisigRegistrationMsg(msg sdk.Msg) bool {
	_, ok := msg.(*MsgRegisterPQCMultisig)
	return ok
}

// MultisigBodyB0 is the B0 of MultisigSignBytes for a decoded body: the body
// re-serialised without the hybrid or the multisig extension option, every
// other field and extension kept in order. The chain and the client both call
// it, on the body they decoded, so they agree on the bytes.
func MultisigBodyB0(body *txtypes.TxBody) ([]byte, error) {
	cp := *body
	cp.ExtensionOptions = nil
	for _, opt := range body.ExtensionOptions {
		if opt != nil && (opt.GetTypeUrl() == HybridSigTypeURL || opt.GetTypeUrl() == MultisigSigTypeURL) {
			continue
		}
		cp.ExtensionOptions = append(cp.ExtensionOptions, opt)
	}
	return proto.Marshal(&cp)
}

// MultisigAuthInfoA0 is the A0 of MultisigSignBytes: the auth info
// re-serialised without its signer infos (fee and tip kept).
func MultisigAuthInfoA0(auth *txtypes.AuthInfo) ([]byte, error) {
	cp := *auth
	cp.SignerInfos = nil
	return proto.Marshal(&cp)
}
