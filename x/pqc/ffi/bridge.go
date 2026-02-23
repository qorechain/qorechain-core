//go:build proprietary

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

	"github.com/qorechain/qorechain-core/x/pqc/types"
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

// Maximum buffer sizes for algorithm-aware operations.
// These are generous upper bounds; the FFI layer returns actual sizes.
const (
	MaxPubkeySize  = 4096
	MaxPrivkeySize = 8192
	MaxSigSize     = 8192
)

// PQCClient is the interface for all PQC operations via the Rust FFI library.
type PQCClient interface {
	// Legacy algorithm-specific operations (backward compatibility)
	DilithiumKeygen() (pubkey []byte, privkey []byte, err error)
	DilithiumSign(privkey []byte, message []byte) (signature []byte, err error)
	DilithiumVerify(pubkey []byte, message []byte, signature []byte) (bool, error)

	MLKEMKeygen() (pubkey []byte, privkey []byte, err error)
	MLKEMEncapsulate(pubkey []byte) (ciphertext []byte, sharedSecret []byte, err error)
	MLKEMDecapsulate(privkey []byte, ciphertext []byte) (sharedSecret []byte, err error)

	GenerateRandomBeacon(seed []byte, epoch uint64) ([]byte, error)

	// Algorithm-aware operations (v0.6.0)
	Keygen(algorithmID types.AlgorithmID) (pubkey []byte, privkey []byte, err error)
	Sign(algorithmID types.AlgorithmID, privkey []byte, message []byte) (signature []byte, err error)
	Verify(algorithmID types.AlgorithmID, pubkey []byte, message []byte, signature []byte) (bool, error)
	AlgorithmInfo(algorithmID types.AlgorithmID) (pubkeySize, privkeySize, outputSize uint32, err error)
	ListAlgorithms() ([]types.AlgorithmID, error)

	Version() string
	Algorithms() string
}

// FFIClient implements PQCClient by calling into libqorepqc via cgo.
type FFIClient struct{}

// NewFFIClient creates a new FFI-based PQC client.
func NewFFIClient() *FFIClient {
	return &FFIClient{}
}

// ---- Legacy algorithm-specific operations ----

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

// ---- Algorithm-aware operations (v0.6.0) ----

// Keygen generates a keypair using the specified algorithm.
func (c *FFIClient) Keygen(algorithmID types.AlgorithmID) ([]byte, []byte, error) {
	pk := make([]byte, MaxPubkeySize)
	sk := make([]byte, MaxPrivkeySize)
	pkLen := C.size_t(MaxPubkeySize)
	skLen := C.size_t(MaxPrivkeySize)

	ret := C.qore_pqc_keygen(
		C.uint32_t(algorithmID),
		(*C.uint8_t)(unsafe.Pointer(&pk[0])),
		&pkLen,
		(*C.uint8_t)(unsafe.Pointer(&sk[0])),
		&skLen,
	)
	if ret != 0 {
		return nil, nil, fmt.Errorf("pqc keygen failed for algorithm %s: error code %d", algorithmID, ret)
	}

	return pk[:pkLen], sk[:skLen], nil
}

// Sign creates a signature using the specified algorithm.
func (c *FFIClient) Sign(algorithmID types.AlgorithmID, privkey []byte, message []byte) ([]byte, error) {
	sig := make([]byte, MaxSigSize)
	sigLen := C.size_t(MaxSigSize)

	var msgPtr *C.uint8_t
	if len(message) > 0 {
		msgPtr = (*C.uint8_t)(unsafe.Pointer(&message[0]))
	} else {
		empty := []byte{0}
		msgPtr = (*C.uint8_t)(unsafe.Pointer(&empty[0]))
	}

	ret := C.qore_pqc_sign(
		C.uint32_t(algorithmID),
		(*C.uint8_t)(unsafe.Pointer(&privkey[0])),
		C.size_t(len(privkey)),
		msgPtr,
		C.size_t(len(message)),
		(*C.uint8_t)(unsafe.Pointer(&sig[0])),
		&sigLen,
	)
	if ret != 0 {
		return nil, fmt.Errorf("pqc sign failed for algorithm %s: error code %d", algorithmID, ret)
	}

	return sig[:sigLen], nil
}

// Verify checks a signature using the specified algorithm.
// Returns true if valid, false if invalid.
func (c *FFIClient) Verify(algorithmID types.AlgorithmID, pubkey []byte, message []byte, signature []byte) (bool, error) {
	var msgPtr *C.uint8_t
	if len(message) > 0 {
		msgPtr = (*C.uint8_t)(unsafe.Pointer(&message[0]))
	} else {
		empty := []byte{0}
		msgPtr = (*C.uint8_t)(unsafe.Pointer(&empty[0]))
	}

	ret := C.qore_pqc_verify(
		C.uint32_t(algorithmID),
		(*C.uint8_t)(unsafe.Pointer(&pubkey[0])),
		C.size_t(len(pubkey)),
		msgPtr,
		C.size_t(len(message)),
		(*C.uint8_t)(unsafe.Pointer(&signature[0])),
		C.size_t(len(signature)),
	)

	if ret < 0 {
		return false, fmt.Errorf("pqc verify failed for algorithm %s: error code %d", algorithmID, ret)
	}
	return ret == 1, nil
}

// AlgorithmInfo queries the key/output sizes for a given algorithm.
func (c *FFIClient) AlgorithmInfo(algorithmID types.AlgorithmID) (uint32, uint32, uint32, error) {
	var pubkeySize, privkeySize, outputSize C.uint32_t

	ret := C.qore_pqc_algorithm_info(
		C.uint32_t(algorithmID),
		&pubkeySize,
		&privkeySize,
		&outputSize,
	)
	if ret != 0 {
		return 0, 0, 0, fmt.Errorf("pqc algorithm_info failed for algorithm %s: error code %d", algorithmID, ret)
	}

	return uint32(pubkeySize), uint32(privkeySize), uint32(outputSize), nil
}

// ListAlgorithms returns the IDs of all algorithms supported by the FFI library.
func (c *FFIClient) ListAlgorithms() ([]types.AlgorithmID, error) {
	// First call with nil to get the count
	var count C.size_t
	ret := C.qore_pqc_list_algorithms(nil, &count)
	if ret != 0 {
		return nil, fmt.Errorf("pqc list_algorithms count failed: error code %d", ret)
	}

	if count == 0 {
		return nil, nil
	}

	// Second call to get the actual IDs
	ids := make([]C.uint32_t, count)
	ret = C.qore_pqc_list_algorithms(&ids[0], &count)
	if ret != 0 {
		return nil, fmt.Errorf("pqc list_algorithms failed: error code %d", ret)
	}

	result := make([]types.AlgorithmID, count)
	for i := C.size_t(0); i < count; i++ {
		result[i] = types.AlgorithmID(ids[i])
	}

	return result, nil
}
