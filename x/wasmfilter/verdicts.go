package wasmfilter

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
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
	// Size and Functions of the code, and its conservative compiled size,
	// for pricing its load.
	Size, Functions int
	Compiled        uint64
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

	// From wasm-upload-compiled-size: an upload may define at most
	// MaxUploadFunctions functions (release contracts have 295 to 828, the
	// largest testnet code 828; code already stored is never refused for
	// it), its conservative compiled size (ConservativeCompiledSize) may be
	// at most MaxUploadCompiledSize, and it pays CompiledGasPerByte per byte
	// of that size, which every node keeps on disk. The largest release
	// contract (cw721_base, about 16.8 MB estimated) stores for about 31M
	// gas, inside the 50M transaction cap.
	MaxUploadFunctions       = 4_096
	CompiledBytesPerWasmByte = 16
	CompiledBytesPerFunction = 256
	CompiledGasPerByte       = 1
)

// ErrTooManyFunctions is the refusal of an upload with more functions than
// MaxUploadFunctions.
var ErrTooManyFunctions = errors.New("too many functions")

// EstimatedCompiledSize is the size-based floor of the compiled-size
// estimate: CompiledBytesPerWasmByte per module byte plus
// CompiledBytesPerFunction per function.
func EstimatedCompiledSize(size, functions int) uint64 {
	return uint64(size)*CompiledBytesPerWasmByte + uint64(functions)*CompiledBytesPerFunction
}

// CompiledBytesPerClass is the compiled code one operator of each class
// makes, the largest measured over the architectures validators run (x86-64
// and arm64) with a 1.25x margin; CompiledBytesPerFunctionOps is the same
// for a function's own overhead. The VM compiles for the machine it runs
// on, and x86-64 makes up to 2.3 times the code arm64 does for some
// operators (memory loads, bit counts, division by constants, if/else), so
// the weights follow the larger. Provisional x86-64 figures, from probes run
// on x86-64 under emulation, until the validator-hardware table replaces
// them.
var CompiledBytesPerClass = [NumOpClasses]uint64{
	OpOther:        20,
	OpCall:         225,
	OpCallIndirect: 366,
	OpBranch:       337,
	OpFlow:         125,
	OpLoad:         146,
	OpStore:        139,
	OpDivRem:       160,
	OpBitCount:     40,
	OpMemory:       50,
}

// CompiledBytesPerFunctionOps is a function's own compiled overhead in the
// per-operator estimate.
const CompiledBytesPerFunctionOps = 500

// ConservativeCompiledSize is the compiled size a module is charged and
// bounded by: the larger of the size-based floor and the per-operator
// estimate from its inspection (CompiledBytesPerClass). Measured compiled
// sizes of the release contracts, the testnet's codes and adversarial shapes
// are at or below it on both architectures.
func ConservativeCompiledSize(size int, st Stats) uint64 {
	floor := EstimatedCompiledSize(size, st.Functions)
	ops := uint64(st.Functions) * CompiledBytesPerFunctionOps
	for c, n := range st.ByClass {
		ops += uint64(n) * CompiledBytesPerClass[c]
	}
	return max(floor, ops)
}

// MaxUploadCompiledSize bounds an upload's conservative compiled size
// (wasm-upload-compiled-size): two such modules stay well inside a node's
// default 100 MiB memory cache. Code already stored is never refused for it.
const MaxUploadCompiledSize = 32 << 20

// ErrCompiledTooLarge is the refusal of an upload whose conservative
// compiled size is over MaxUploadCompiledSize.
var ErrCompiledTooLarge = errors.New("compiled size too large")

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

// ErrUploadSize is the refusal of a gzip upload that does not unpack to
// exactly the size it declares in one member: more bytes than declared, a
// failed integrity check, bytes after the first member or broken data.
var ErrUploadSize = errors.New("upload does not unpack to its declared size")

// unpackUploadBounded unpacks an upload reading at most limit bytes: raw
// bytes over the limit are refused; gzip bytes must form exactly one member
// whose data ends within the limit (so the end-of-member integrity check,
// which verifies the size field against the data, has run) and nothing may
// follow it (so the size field read from the end is that member's). The work
// done is bounded by limit whatever the bytes claim. Unlike wasmd's own
// unpacking it does not bound the compressed length by limit: a small honest
// module's gzip is larger than the module.
func unpackUploadBounded(code []byte, limit uint64) ([]byte, error) {
	if !ioutils.IsGzip(code) {
		if uint64(len(code)) > limit {
			return nil, ErrUploadSize
		}
		return code, nil
	}
	src := bytes.NewReader(code)
	zr, err := gzip.NewReader(src)
	if err != nil {
		return nil, ErrUploadSize
	}
	zr.Multistream(false)
	defer zr.Close()
	raw, err := io.ReadAll(io.LimitReader(zr, int64(limit)+1))
	if err != nil || uint64(len(raw)) > limit || src.Len() != 0 {
		return nil, ErrUploadSize
	}
	return raw, nil
}

// InspectUploadBounded is InspectUpload for an upload whose declared size
// has been paid for: it never unpacks more than min(declared, wasmd's size
// limit) bytes, and an upload that does not unpack to exactly its declared
// size in one member is an error (Err), not a pass to wasmd.
func InspectUploadBounded(code []byte, declared uint64) (UploadInspection, error) {
	raw, err := unpackUploadBounded(code, min(declared, uint64(wasmtypes.MaxWasmSize)))
	if err != nil {
		return UploadInspection{}, err
	}
	if uint64(len(raw)) != declared {
		return UploadInspection{}, ErrUploadSize
	}
	st, ierr := Inspect(raw)
	return UploadInspection{Raw: raw, Stats: st, Err: ierr}, nil
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
