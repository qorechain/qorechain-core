package wasmfilter

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"sync"

	errorsmod "cosmossdk.io/errors"

	"github.com/CosmWasm/wasmd/x/wasm/ioutils"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
)

// ModuleName is the codespace of the filter's refusals.
const ModuleName = "wasmfilter"

// ErrCodeRefused is what a refused upload or a refused call of stored code
// returns. The wrapped text names the reason (one of the Err* values).
var ErrCodeRefused = errorsmod.Register(ModuleName, 2, "wasm code refused by the code filter")

// CheckUpload checks bytes as MsgStoreCode carries them: gzip-wrapped bytes
// are unpacked exactly as wasmd does before compiling them (same detection,
// same size limit). It returns (false, nil) when the bytes cannot be
// unpacked: wasmd refuses those itself with its own error, and the filter does
// not change which error that is.
func CheckUpload(code []byte) (checked bool, err error) {
	raw, ok := unpackUpload(code)
	if !ok {
		return false, nil
	}
	return true, Check(raw)
}

// CheckUploadFloats is CheckFloats on bytes as MsgStoreCode carries them.
func CheckUploadFloats(code []byte) error {
	raw, ok := unpackUpload(code)
	if !ok {
		return nil
	}
	return CheckFloats(raw)
}

func unpackUpload(code []byte) ([]byte, bool) {
	if ioutils.IsGzip(code) {
		raw, uerr := ioutils.Uncompress(code, int64(wasmtypes.MaxWasmSize))
		if uerr != nil {
			return nil, false
		}
		return raw, true
	}
	return code, true
}

// Verdict is what the filter knows about one stored code: Check's result and
// whether the module contains float operators.
type Verdict struct {
	Err    error
	Floats error // CheckFloats's result
}

// Verdicts caches the verdicts per code checksum. They depend on the bytes
// only, which the checksum names, so the cache never changes an answer: it
// only saves re-parsing a stored code on every call. Safe for concurrent use
// (queries run alongside block execution).
type Verdicts struct {
	mu  sync.RWMutex
	byC map[string]Verdict
}

// NewVerdicts returns an empty cache.
func NewVerdicts() *Verdicts { return &Verdicts{byC: map[string]Verdict{}} }

// Get returns the cached verdict for a checksum.
func (v *Verdicts) Get(checksum []byte) (ok bool, verdict Verdict) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	verdict, ok = v.byC[hex.EncodeToString(checksum)]
	return ok, verdict
}

// Put records the verdict for a checksum.
func (v *Verdicts) Put(checksum []byte, verdict Verdict) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.byC[hex.EncodeToString(checksum)] = verdict
}

// Upload pricing and bounds from the upload-bounds switch (msg_server.go).
const (
	// CompileGasPerByte is wasmd's compile price at its default multiplier
	// (wasmvm charges 3 x 140,000 VM gas per byte; the engine keeps it at 3
	// SDK gas per byte whatever the multiplier).
	CompileGasPerByte = 3
	// UploadStorageGasPerByte is charged on the uncompressed size: the
	// module is kept on every node uncompressed, while wasmd's own charge
	// follows the compressed bytes a transaction carries. An 800 KiB module
	// costs about 15.6M gas in all, well within a 50M transaction.
	UploadStorageGasPerByte = 15
	// UploadInspectGasPerByte prices unpacking and inspecting the module.
	UploadInspectGasPerByte = 1
	// UploadGasMargin covers wasmd's own fixed costs around the store.
	UploadGasMargin = 200_000
	// MaxUploadCustomBytes bounds the custom sections (names, debug info) of
	// an uploaded module. Custom sections compress to almost nothing in a
	// transaction and are kept forever; release builds carry none, and the
	// largest stored on the testnet is 67,089 bytes. Code already stored is
	// never refused for them.
	MaxUploadCustomBytes = 128 * 1024
)

// ErrCustomSections is the refusal of an upload whose custom sections are
// too large.
var ErrCustomSections = errors.New("custom sections too large")

// UploadInspection is what one pass over an upload found.
type UploadInspection struct {
	Raw   []byte
	Stats Stats
	Err   error // Check's verdict
}

// InspectUpload unpacks an upload once (as wasmd does) and inspects it in
// one pass; false when it cannot be unpacked (wasmd refuses that itself).
func InspectUpload(code []byte) (UploadInspection, bool) {
	raw, ok := unpackUpload(code)
	if !ok {
		return UploadInspection{}, false
	}
	st, err := Inspect(raw)
	return UploadInspection{Raw: raw, Stats: st, Err: err}, true
}

// DeclaredUploadSize is the size an upload says it unpacks to, without
// unpacking it: gzip's size field (the length modulo 2^32, which a module
// under wasmd's size limit fits) or the length of raw bytes.
func DeclaredUploadSize(code []byte) (uint64, bool) {
	if ioutils.IsGzip(code) {
		if len(code) < 18 {
			return 0, false
		}
		return uint64(binary.LittleEndian.Uint32(code[len(code)-4:])), true
	}
	return uint64(len(code)), true
}
