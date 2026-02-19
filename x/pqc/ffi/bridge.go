package ffi

/*
#cgo darwin,arm64 LDFLAGS: -L${SRCDIR}/../../../lib/darwin_arm64 -lqorepqc
#cgo darwin,amd64 LDFLAGS: -L${SRCDIR}/../../../lib/darwin_amd64 -lqorepqc
#cgo linux,amd64 LDFLAGS: -L${SRCDIR}/../../../lib/linux_amd64 -lqorepqc
#cgo linux,arm64 LDFLAGS: -L${SRCDIR}/../../../lib/linux_arm64 -lqorepqc
#include "bridge.h"
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// Dilithium-5 key/signature sizes (from pqcrypto-dilithium v0.5.0).
const (
	DilithiumPubkeySize  = 2592
	DilithiumPrivkeySize = 4896
	DilithiumSigSize     = 4627
)

// ML-KEM-1024 sizes (from pqcrypto-kyber v0.8.1).
const (
	MLKEMPubkeySize       = 1568
	MLKEMPrivkeySize      = 3168
	MLKEMCiphertextSize   = 1568
	MLKEMSharedSecretSize = 32
)

// DefaultBeaconOutputLen is the default random beacon output length.
const DefaultBeaconOutputLen = 32

// PQCClient is the interface for all PQC operations via the Rust FFI library.
type PQCClient interface {
	DilithiumKeygen() (pubkey []byte, privkey []byte, err error)
	DilithiumSign(privkey []byte, message []byte) (signature []byte, err error)
	DilithiumVerify(pubkey []byte, message []byte, signature []byte) (bool, error)

	MLKEMKeygen() (pubkey []byte, privkey []byte, err error)
	MLKEMEncapsulate(pubkey []byte) (ciphertext []byte, sharedSecret []byte, err error)
	MLKEMDecapsulate(privkey []byte, ciphertext []byte) (sharedSecret []byte, err error)

	GenerateRandomBeacon(seed []byte, epoch uint64) ([]byte, error)

	Version() string
	Algorithms() string
}

// FFIClient implements PQCClient by calling into libqorepqc via cgo.
type FFIClient struct{}

// NewFFIClient creates a new FFI-based PQC client.
func NewFFIClient() *FFIClient {
	return &FFIClient{}
}

func (c *FFIClient) DilithiumKeygen() ([]byte, []byte, error) {
	pk := make([]byte, DilithiumPubkeySize)
	sk := make([]byte, DilithiumPrivkeySize)
	pkLen := C.size_t(DilithiumPubkeySize)
	skLen := C.size_t(DilithiumPrivkeySize)

	ret := C.qore_dilithium_keygen(
		(*C.uint8_t)(unsafe.Pointer(&pk[0])),
		&pkLen,
		(*C.uint8_t)(unsafe.Pointer(&sk[0])),
		&skLen,
	)
	if ret != 0 {
		return nil, nil, fmt.Errorf("dilithium keygen failed: error code %d", ret)
	}

	return pk[:pkLen], sk[:skLen], nil
}

func (c *FFIClient) DilithiumSign(privkey []byte, message []byte) ([]byte, error) {
	sig := make([]byte, DilithiumSigSize)
	sigLen := C.size_t(DilithiumSigSize)

	ret := C.qore_dilithium_sign(
		(*C.uint8_t)(unsafe.Pointer(&privkey[0])),
		C.size_t(len(privkey)),
		(*C.uint8_t)(unsafe.Pointer(&message[0])),
		C.size_t(len(message)),
		(*C.uint8_t)(unsafe.Pointer(&sig[0])),
		&sigLen,
	)
	if ret != 0 {
		return nil, fmt.Errorf("dilithium sign failed: error code %d", ret)
	}

	return sig[:sigLen], nil
}

func (c *FFIClient) DilithiumVerify(pubkey []byte, message []byte, signature []byte) (bool, error) {
	ret := C.qore_dilithium_verify(
		(*C.uint8_t)(unsafe.Pointer(&pubkey[0])),
		C.size_t(len(pubkey)),
		(*C.uint8_t)(unsafe.Pointer(&message[0])),
		C.size_t(len(message)),
		(*C.uint8_t)(unsafe.Pointer(&signature[0])),
		C.size_t(len(signature)),
	)

	if ret < 0 {
		return false, fmt.Errorf("dilithium verify failed: error code %d", ret)
	}
	return ret == 1, nil
}

func (c *FFIClient) MLKEMKeygen() ([]byte, []byte, error) {
	pk := make([]byte, MLKEMPubkeySize)
	sk := make([]byte, MLKEMPrivkeySize)
	pkLen := C.size_t(MLKEMPubkeySize)
	skLen := C.size_t(MLKEMPrivkeySize)

	ret := C.qore_mlkem_keygen(
		(*C.uint8_t)(unsafe.Pointer(&pk[0])),
		&pkLen,
		(*C.uint8_t)(unsafe.Pointer(&sk[0])),
		&skLen,
	)
	if ret != 0 {
		return nil, nil, fmt.Errorf("mlkem keygen failed: error code %d", ret)
	}

	return pk[:pkLen], sk[:skLen], nil
}

func (c *FFIClient) MLKEMEncapsulate(pubkey []byte) ([]byte, []byte, error) {
	ct := make([]byte, MLKEMCiphertextSize)
	ss := make([]byte, MLKEMSharedSecretSize)
	ctLen := C.size_t(MLKEMCiphertextSize)
	ssLen := C.size_t(MLKEMSharedSecretSize)

	ret := C.qore_mlkem_encapsulate(
		(*C.uint8_t)(unsafe.Pointer(&pubkey[0])),
		C.size_t(len(pubkey)),
		(*C.uint8_t)(unsafe.Pointer(&ct[0])),
		&ctLen,
		(*C.uint8_t)(unsafe.Pointer(&ss[0])),
		&ssLen,
	)
	if ret != 0 {
		return nil, nil, fmt.Errorf("mlkem encapsulate failed: error code %d", ret)
	}

	return ct[:ctLen], ss[:ssLen], nil
}

func (c *FFIClient) MLKEMDecapsulate(privkey []byte, ciphertext []byte) ([]byte, error) {
	ss := make([]byte, MLKEMSharedSecretSize)
	ssLen := C.size_t(MLKEMSharedSecretSize)

	ret := C.qore_mlkem_decapsulate(
		(*C.uint8_t)(unsafe.Pointer(&privkey[0])),
		C.size_t(len(privkey)),
		(*C.uint8_t)(unsafe.Pointer(&ciphertext[0])),
		C.size_t(len(ciphertext)),
		(*C.uint8_t)(unsafe.Pointer(&ss[0])),
		&ssLen,
	)
	if ret != 0 {
		return nil, fmt.Errorf("mlkem decapsulate failed: error code %d", ret)
	}

	return ss[:ssLen], nil
}

func (c *FFIClient) GenerateRandomBeacon(seed []byte, epoch uint64) ([]byte, error) {
	output := make([]byte, DefaultBeaconOutputLen)

	var seedPtr *C.uint8_t
	if len(seed) > 0 {
		seedPtr = (*C.uint8_t)(unsafe.Pointer(&seed[0]))
	} else {
		// Use a single zero-byte to avoid passing nil to C
		empty := []byte{0}
		seedPtr = (*C.uint8_t)(unsafe.Pointer(&empty[0]))
	}

	ret := C.qore_random_beacon(
		seedPtr,
		C.size_t(len(seed)),
		C.uint64_t(epoch),
		(*C.uint8_t)(unsafe.Pointer(&output[0])),
		C.size_t(DefaultBeaconOutputLen),
	)
	if ret != 0 {
		return nil, fmt.Errorf("random beacon failed: error code %d", ret)
	}

	return output, nil
}

func (c *FFIClient) Version() string {
	return C.GoString(C.qore_pqc_version())
}

func (c *FFIClient) Algorithms() string {
	return C.GoString(C.qore_pqc_algorithms())
}
