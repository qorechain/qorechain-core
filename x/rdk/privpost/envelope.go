// Package privpost is the envelope of a post on a private (encapsulated) RDK
// rollup (v3.2.8, FeatureRDKPrivateRollups): the data encrypted once under a
// fresh content key, and the content key encapsulated with ML-KEM-1024 to
// every member's registered KEM key.
//
// The chain only parses envelopes (Parse) and checks their structure against
// the rollup's members; Seal and Open run on clients. Specification:
// testnet/handoff-v328/RDK-DATA-BASED-DESIGN.md, section 14.
package privpost

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/cloudflare/circl/kem/mlkem/mlkem1024"
)

const (
	magic   = "QPRV"
	version = 1

	// KEMCiphertextSize and EncapsulationKeySize are ML-KEM-1024's.
	KEMCiphertextSize    = mlkem1024.CiphertextSize
	EncapsulationKeySize = mlkem1024.PublicKeySize
	// WrappedKeySize is the content key under AES-256-GCM (32 + 16 tag).
	WrappedKeySize = 48
	// MinPayloadSize is the GCM tag.
	MinPayloadSize = 16
	// MaxMembers bounds a member list (an entry is about 1.7 KB).
	MaxMembers = 32
	// MaxAddressLen bounds a member address in an entry.
	MaxAddressLen = 128
)

var (
	ErrMalformed = errors.New("privpost: malformed envelope")
	ErrNotMember = errors.New("privpost: the envelope has no entry for this member")
	ErrOpen      = errors.New("privpost: the envelope does not open with this key")
)

// Entry is one member's encapsulation.
type Entry struct {
	Member        string
	Fingerprint   [32]byte // SHA-256 of the member's ML-KEM-1024 encapsulation key
	KEMCiphertext []byte
	WrappedKey    []byte
}

// Envelope is a parsed envelope.
type Envelope struct {
	Epoch      uint64
	KeyCommit  [32]byte
	Entries    []Entry
	Ciphertext []byte
	header     []byte // everything before Ciphertext: the payload's AAD
}

// Fingerprint names an encapsulation key.
func Fingerprint(ek []byte) [32]byte { return sha256.Sum256(ek) }

// Parse reads an envelope and checks every size. It does not look at
// membership; the chain does that against its member list.
func Parse(b []byte) (Envelope, error) {
	var e Envelope
	p := 0
	need := func(n int) bool { return n >= 0 && len(b)-p >= n }
	if !need(5+8+32+2) || string(b[:4]) != magic || b[4] != version {
		return e, ErrMalformed
	}
	p = 5
	e.Epoch = binary.BigEndian.Uint64(b[p:])
	p += 8
	copy(e.KeyCommit[:], b[p:p+32])
	p += 32
	n := int(binary.BigEndian.Uint16(b[p:]))
	p += 2
	if n == 0 || n > MaxMembers {
		return e, fmt.Errorf("%w: %d entries", ErrMalformed, n)
	}
	for i := 0; i < n; i++ {
		if !need(2) {
			return e, ErrMalformed
		}
		al := int(binary.BigEndian.Uint16(b[p:]))
		p += 2
		if al == 0 || al > MaxAddressLen || !need(al+32+KEMCiphertextSize+WrappedKeySize) {
			return e, ErrMalformed
		}
		var en Entry
		en.Member = string(b[p : p+al])
		p += al
		copy(en.Fingerprint[:], b[p:p+32])
		p += 32
		en.KEMCiphertext = b[p : p+KEMCiphertextSize]
		p += KEMCiphertextSize
		en.WrappedKey = b[p : p+WrappedKeySize]
		p += WrappedKeySize
		e.Entries = append(e.Entries, en)
	}
	if len(b)-p < MinPayloadSize {
		return e, fmt.Errorf("%w: payload shorter than its tag", ErrMalformed)
	}
	e.header = b[:p]
	e.Ciphertext = b[p:]
	return e, nil
}

func wrapKey(ss []byte, chainID, rollupID string, epoch uint64, member string) ([]byte, error) {
	info := []byte("qorechain/rdk/private/v1/wrap")
	for _, f := range []string{chainID, rollupID} {
		info = binary.BigEndian.AppendUint32(info, uint32(len(f)))
		info = append(info, f...)
	}
	info = binary.BigEndian.AppendUint64(info, epoch)
	info = binary.BigEndian.AppendUint32(info, uint32(len(member)))
	info = append(info, member...)
	return hkdf.Key(sha256.New, ss, nil, string(info), 32)
}

// gcm seals or opens with a single-use key and the zero nonce (every key
// here encrypts exactly one message).
func gcm(key []byte) (cipher.AEAD, error) {
	blk, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(blk)
}

func keyCommit(k []byte) [32]byte {
	return sha256.Sum256(append([]byte("qorechain/rdk/private/v1/key"), k...))
}

// Recipient is a member and its ML-KEM-1024 encapsulation key.
type Recipient struct {
	Member           string
	EncapsulationKey []byte
}

// Seal encrypts plaintext to recipients (in the rollup's member order) for
// chainID/rollupID at the membership epoch.
func Seal(rand io.Reader, chainID, rollupID string, epoch uint64, recipients []Recipient, plaintext []byte) ([]byte, error) {
	if len(recipients) == 0 || len(recipients) > MaxMembers {
		return nil, fmt.Errorf("privpost: %d recipients", len(recipients))
	}
	k := make([]byte, 32)
	if _, err := io.ReadFull(rand, k); err != nil {
		return nil, err
	}
	commit := keyCommit(k)
	b := append([]byte(magic), version)
	b = binary.BigEndian.AppendUint64(b, epoch)
	b = append(b, commit[:]...)
	b = binary.BigEndian.AppendUint16(b, uint16(len(recipients)))
	for _, r := range recipients {
		if len(r.Member) == 0 || len(r.Member) > MaxAddressLen {
			return nil, fmt.Errorf("privpost: member %q", r.Member)
		}
		pk, err := mlkem1024.Scheme().UnmarshalBinaryPublicKey(r.EncapsulationKey)
		if err != nil {
			return nil, fmt.Errorf("privpost: key of %s: %w", r.Member, err)
		}
		seed := make([]byte, mlkem1024.EncapsulationSeedSize)
		if _, err := io.ReadFull(rand, seed); err != nil {
			return nil, err
		}
		ct, ss, err := mlkem1024.Scheme().EncapsulateDeterministically(pk, seed)
		if err != nil {
			return nil, err
		}
		wk, err := wrapKey(ss, chainID, rollupID, epoch, r.Member)
		if err != nil {
			return nil, err
		}
		a, err := gcm(wk)
		if err != nil {
			return nil, err
		}
		fp := Fingerprint(r.EncapsulationKey)
		b = binary.BigEndian.AppendUint16(b, uint16(len(r.Member)))
		b = append(b, r.Member...)
		b = append(b, fp[:]...)
		b = append(b, ct...)
		b = a.Seal(b, make([]byte, a.NonceSize()), k, nil)
	}
	a, err := gcm(k)
	if err != nil {
		return nil, err
	}
	header := append([]byte(nil), b...)
	return a.Seal(b, make([]byte, a.NonceSize()), plaintext, header), nil
}

// DecapsulationKey reads a member's ML-KEM-1024 private key: the 64-byte
// seed (d || z) or the 3168-byte expanded form.
func DecapsulationKey(b []byte) (*mlkem1024.PrivateKey, error) {
	switch len(b) {
	case mlkem1024.KeySeedSize:
		_, sk := mlkem1024.NewKeyFromSeed(b)
		return sk, nil
	case mlkem1024.PrivateKeySize:
		var sk mlkem1024.PrivateKey
		if err := sk.Unpack(b); err != nil {
			return nil, err
		}
		return &sk, nil
	}
	return nil, fmt.Errorf("privpost: an ML-KEM-1024 key is %d (seed) or %d bytes, got %d",
		mlkem1024.KeySeedSize, mlkem1024.PrivateKeySize, len(b))
}

// Open decrypts an envelope as member with its decapsulation key.
func Open(envelope []byte, chainID, rollupID, member string, dk []byte) ([]byte, error) {
	e, err := Parse(envelope)
	if err != nil {
		return nil, err
	}
	sk, err := DecapsulationKey(dk)
	if err != nil {
		return nil, err
	}
	for _, en := range e.Entries {
		if en.Member != member {
			continue
		}
		ss := make([]byte, mlkem1024.SharedKeySize)
		sk.DecapsulateTo(ss, en.KEMCiphertext)
		wk, err := wrapKey(ss, chainID, rollupID, e.Epoch, member)
		if err != nil {
			return nil, err
		}
		a, err := gcm(wk)
		if err != nil {
			return nil, err
		}
		k, err := a.Open(nil, make([]byte, a.NonceSize()), en.WrappedKey, nil)
		if err != nil {
			return nil, ErrOpen
		}
		if c := keyCommit(k); !bytes.Equal(c[:], e.KeyCommit[:]) {
			return nil, fmt.Errorf("%w: content key does not match the envelope's commitment", ErrOpen)
		}
		pa, err := gcm(k)
		if err != nil {
			return nil, err
		}
		pt, err := pa.Open(nil, make([]byte, pa.NonceSize()), e.Ciphertext, e.header)
		if err != nil {
			return nil, ErrOpen
		}
		return pt, nil
	}
	return nil, ErrNotMember
}
