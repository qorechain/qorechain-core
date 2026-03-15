//go:build proprietary

package ffi

import (
	"sync"
)

// Sysvar ID constants used by the sysvar callback mechanism.
// These match the IDs expected by the Rust runtime.
const (
	SysvarClockID uint32 = 0
	SysvarRentID  uint32 = 1
)

// SyscallHandler is a function that handles a Go-side syscall invoked by the
// BPF program during execution. The Rust runtime will call back into Go when
// a registered syscall ID is encountered.
type SyscallHandler func(data []byte) ([]byte, error)

// syscallRegistry holds registered Go-side syscall handlers keyed by ID.
var (
	syscallMu       sync.RWMutex
	syscallHandlers = make(map[uint32]SyscallHandler)
)

// RegisterSyscallHandler registers a Go function to handle a specific syscall ID.
// If a handler for the given ID already exists it will be replaced.
func RegisterSyscallHandler(id uint32, handler SyscallHandler) {
	syscallMu.Lock()
	defer syscallMu.Unlock()
	syscallHandlers[id] = handler
}

// UnregisterSyscallHandler removes the handler for a specific syscall ID.
func UnregisterSyscallHandler(id uint32) {
	syscallMu.Lock()
	defer syscallMu.Unlock()
	delete(syscallHandlers, id)
}

// ClearSyscallHandlers removes all registered handlers.
func ClearSyscallHandlers() {
	syscallMu.Lock()
	defer syscallMu.Unlock()
	syscallHandlers = make(map[uint32]SyscallHandler)
}

// lookupSyscallHandler returns the handler for the given ID, or nil.
func lookupSyscallHandler(id uint32) SyscallHandler {
	syscallMu.RLock()
	defer syscallMu.RUnlock()
	return syscallHandlers[id]
}
