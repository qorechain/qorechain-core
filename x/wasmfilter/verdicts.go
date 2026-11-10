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
	if ioutils.IsGzip(code) {
		raw, uerr := ioutils.Uncompress(code, int64(wasmtypes.MaxWasmSize))
		if uerr != nil {
			return false, nil
		}
		code = raw
	}
	return true, Check(code)
}

// Verdicts caches Check results per code checksum. The result depends on the
// bytes only, which the checksum names, so the cache never changes an answer:
// it only saves re-parsing a stored code on every call. Safe for concurrent
// use (queries run alongside block execution).
type Verdicts struct {
	mu  sync.RWMutex
	byC map[string]error
}

// NewVerdicts returns an empty cache.
func NewVerdicts() *Verdicts { return &Verdicts{byC: map[string]error{}} }

// Get returns the cached verdict for a checksum.
func (v *Verdicts) Get(checksum []byte) (ok bool, verdict error) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	verdict, ok = v.byC[hex.EncodeToString(checksum)]
	return ok, verdict
}

// Put records the verdict for a checksum.
func (v *Verdicts) Put(checksum []byte, err error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.byC[hex.EncodeToString(checksum)] = err
}
