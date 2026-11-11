package wasmfilter

import (
	"encoding/hex"
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
