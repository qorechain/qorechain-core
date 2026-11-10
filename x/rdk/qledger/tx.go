package qledger

import (
	"crypto/sha256"
	"encoding/binary"

	"github.com/cloudflare/circl/sign/mldsa/mldsa44"
)

// SchemeMLDSA44 (FIPS 204 ML-DSA-44) is the only signature scheme of a
// qledger-v1 account. Any other scheme byte (2 was reserved for Ed25519 in
// the first draft and refused by decision of 2026-11-11) does not parse, so
// such a post is a no-op.
const SchemeMLDSA44 byte = 1

// Transaction kinds.
const (
	KindTransfer byte = 1
	KindWithdraw byte = 2
)

const (
	txVersion = 1
	// MaxRecipientLen bounds a withdrawal recipient (a bech32 address).
	MaxRecipientLen = 90
	// mldsaContext is the FIPS 204 context string of a qledger-v1 signature.
	mldsaContext = "qledger/v1"
	// SigGasMLDSA44 is charged per signature checked.
	SigGasMLDSA44 = 30_000
)

func pkLen(s byte) int {
	if s == SchemeMLDSA44 {
		return mldsa44.PublicKeySize
	}
	return -1
}

func sigLen(s byte) int {
	if s == SchemeMLDSA44 {
		return mldsa44.SignatureSize
	}
	return -1
}

// AccountID is the id of the account controlled by a public key.
func AccountID(scheme byte, pubKey []byte) [32]byte {
	h := sha256.New()
	h.Write([]byte("qledger/v1/account"))
	h.Write([]byte{scheme})
	h.Write(pubKey)
	var id [32]byte
	copy(id[:], h.Sum(nil))
	return id
}

// Tx is one qledger-v1 transaction:
//
//	version=1 | scheme | public_key | kind | nonce (u64 BE) | body | signature
//	transfer body: to[32] | amount (u64 BE)
//	withdraw body: len (u8) | recipient (bech32) | amount (u64 BE)
type Tx struct {
	Scheme    byte
	PubKey    []byte
	Kind      byte
	Nonce     uint64
	To        [32]byte // transfer
	Recipient string   // withdraw
	Amount    uint64
	Sig       []byte
}

// From is the sending account.
func (tx Tx) From() [32]byte { return AccountID(tx.Scheme, tx.PubKey) }

// Unsigned is the encoding without the signature.
func (tx Tx) Unsigned() []byte {
	b := []byte{txVersion, tx.Scheme}
	b = append(b, tx.PubKey...)
	b = append(b, tx.Kind)
	b = binary.BigEndian.AppendUint64(b, tx.Nonce)
	switch tx.Kind {
	case KindTransfer:
		b = append(b, tx.To[:]...)
	case KindWithdraw:
		b = append(b, byte(len(tx.Recipient)))
		b = append(b, tx.Recipient...)
	}
	return binary.BigEndian.AppendUint64(b, tx.Amount)
}

// Encode is the transaction as posted.
func (tx Tx) Encode() []byte { return append(tx.Unsigned(), tx.Sig...) }

// SigningMessage is what an account signs: the transaction bound to one
// chain and one rollup.
func SigningMessage(chainID, rollupID string, unsigned []byte) []byte {
	h := sha256.New()
	h.Write([]byte("qledger/v1/tx"))
	var n [4]byte
	binary.BigEndian.PutUint32(n[:], uint32(len(chainID)))
	h.Write(n[:])
	h.Write([]byte(chainID))
	binary.BigEndian.PutUint32(n[:], uint32(len(rollupID)))
	h.Write(n[:])
	h.Write([]byte(rollupID))
	h.Write(unsigned)
	return h.Sum(nil)
}

// SignMLDSA44 signs tx with key (deterministic signing).
func (tx *Tx) SignMLDSA44(chainID, rollupID string, key *mldsa44.PrivateKey) error {
	sig := make([]byte, mldsa44.SignatureSize)
	if err := mldsa44.SignTo(key, SigningMessage(chainID, rollupID, tx.Unsigned()), []byte(mldsaContext), false, sig); err != nil {
		return err
	}
	tx.Sig = sig
	return nil
}

// DecodeTx parses a posted payload. ok is false for anything that is not
// exactly one well-formed transaction.
func DecodeTx(b []byte) (Tx, bool) {
	var tx Tx
	if len(b) < 2 || b[0] != txVersion {
		return tx, false
	}
	tx.Scheme = b[1]
	pl, sl := pkLen(tx.Scheme), sigLen(tx.Scheme)
	if pl < 0 {
		return tx, false
	}
	p := 2
	if len(b) < p+pl+1+8 {
		return tx, false
	}
	tx.PubKey = append([]byte(nil), b[p:p+pl]...)
	p += pl
	tx.Kind = b[p]
	p++
	tx.Nonce = binary.BigEndian.Uint64(b[p : p+8])
	p += 8
	switch tx.Kind {
	case KindTransfer:
		if len(b) < p+32 {
			return tx, false
		}
		copy(tx.To[:], b[p:p+32])
		p += 32
	case KindWithdraw:
		if len(b) < p+1 {
			return tx, false
		}
		n := int(b[p])
		p++
		if n == 0 || n > MaxRecipientLen || len(b) < p+n {
			return tx, false
		}
		tx.Recipient = string(b[p : p+n])
		p += n
	default:
		return tx, false
	}
	if len(b) != p+8+sl {
		return tx, false
	}
	tx.Amount = binary.BigEndian.Uint64(b[p : p+8])
	p += 8
	tx.Sig = append([]byte(nil), b[p:]...)
	return tx, true
}

// VerifySignature checks the transaction's signature for chainID/rollupID.
func (tx Tx) VerifySignature(chainID, rollupID string) bool {
	msg := SigningMessage(chainID, rollupID, tx.Unsigned())
	switch tx.Scheme {
	case SchemeMLDSA44:
		if len(tx.PubKey) != mldsa44.PublicKeySize || len(tx.Sig) != mldsa44.SignatureSize {
			return false
		}
		var buf [mldsa44.PublicKeySize]byte
		copy(buf[:], tx.PubKey)
		var pk mldsa44.PublicKey
		pk.Unpack(&buf)
		return mldsa44.Verify(&pk, msg, []byte(mldsaContext), tx.Sig)
	}
	return false
}

func sigGas(byte) uint64 { return SigGasMLDSA44 }

// Deposits are inbox entries the chain writes itself (MsgDepositToRollup),
// under a poster that no signer address can be.
const (
	DepositPosterPrefix = "deposit:"
	depositMagic        = "QLDP"
	depositLen          = 4 + 32 + 8
)

// EncodeDeposit is the payload of a deposit entry.
func EncodeDeposit(account [32]byte, amount uint64) []byte {
	b := append([]byte(depositMagic), account[:]...)
	return binary.BigEndian.AppendUint64(b, amount)
}

// DecodeDeposit parses a deposit payload.
func DecodeDeposit(b []byte) ([32]byte, uint64, bool) {
	var id [32]byte
	if len(b) != depositLen || string(b[:4]) != depositMagic {
		return id, 0, false
	}
	copy(id[:], b[4:36])
	return id, binary.BigEndian.Uint64(b[36:]), true
}
