//go:build proprietary

package ffi

/*
#cgo darwin,arm64 LDFLAGS: -L${SRCDIR}/../../../lib/darwin_arm64 -lqoresvm
#cgo darwin,amd64 LDFLAGS: -L${SRCDIR}/../../../lib/darwin_amd64 -lqoresvm
#cgo linux,amd64 LDFLAGS: -L${SRCDIR}/../../../lib/linux_amd64 -lqoresvm
#cgo linux,arm64 LDFLAGS: -L${SRCDIR}/../../../lib/linux_arm64 -lqoresvm
#include "bridge.h"
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"unsafe"

	"github.com/qorechain/qorechain-core/x/svm/types"
)

// resultBufSize is the default buffer for JSON-encoded execution results.
const resultBufSize = 4096

// FFIExecutor implements SVMExecutor via the Rust qoresvm library.
type FFIExecutor struct {
	handle unsafe.Pointer
}

// NewFFIExecutor creates a new BPF executor backed by the Rust runtime.
func NewFFIExecutor(computeBudget uint64) *FFIExecutor {
	handle := C.qore_svm_init(C.uint64_t(computeBudget))
	return &FFIExecutor{handle: handle}
}

// Execute runs a BPF program with the given instruction and accounts.
func (e *FFIExecutor) Execute(program []byte, instruction []byte, accounts []types.SVMAccount,
	computeBudget uint64) (*types.ExecutionResult, error) {

	if len(program) == 0 {
		return nil, fmt.Errorf("empty program bytecode")
	}

	// Build the input buffer. The Rust side expects a mutable buffer so we
	// make a copy of the instruction data to avoid mutating the caller's slice.
	var inputPtr *C.uint8_t
	var inputLen C.size_t
	if len(instruction) > 0 {
		inputCopy := make([]byte, len(instruction))
		copy(inputCopy, instruction)
		inputPtr = (*C.uint8_t)(unsafe.Pointer(&inputCopy[0]))
		inputLen = C.size_t(len(inputCopy))
	}

	// Allocate result buffer for the JSON-encoded execution summary.
	resultBuf := make([]byte, resultBufSize)
	resultCap := C.size_t(resultBufSize)

	ret := C.qore_svm_execute(
		(*C.uint8_t)(unsafe.Pointer(&program[0])),
		C.size_t(len(program)),
		inputPtr,
		inputLen,
		C.uint64_t(computeBudget),
		(*C.uint8_t)(unsafe.Pointer(&resultBuf[0])),
		&resultCap,
	)

	if ret < 0 {
		// Try to extract an error message from the result buffer.
		actualLen := int(resultCap)
		if actualLen > resultBufSize {
			actualLen = resultBufSize
		}
		if actualLen > 0 {
			var summary resultSummary
			if err := json.Unmarshal(resultBuf[:actualLen], &summary); err == nil && summary.Error != "" {
				return nil, fmt.Errorf("SVM execution failed: %s (code %d)", summary.Error, ret)
			}
		}
		return nil, fmt.Errorf("SVM execution failed: error code %d", ret)
	}

	// Parse the JSON-encoded result summary.
	actualLen := int(resultCap)
	if actualLen > resultBufSize {
		actualLen = resultBufSize
	}

	result := &types.ExecutionResult{
		Success:          true,
		ComputeUnitsUsed: 0,
	}

	if actualLen > 0 {
		var summary resultSummary
		if err := json.Unmarshal(resultBuf[:actualLen], &summary); err != nil {
			// JSON parsing failed — return raw data as return_data.
			result.ReturnData = resultBuf[:actualLen]
			return result, nil
		}
		result.Success = summary.Status == "ok"
		result.ComputeUnitsUsed = summary.CU
		if summary.Error != "" {
			result.Error = summary.Error
		}
	}

	return result, nil
}

// ValidateProgram verifies a BPF ELF binary is well-formed.
func (e *FFIExecutor) ValidateProgram(bytecode []byte) error {
	if len(bytecode) == 0 {
		return fmt.Errorf("empty bytecode")
	}

	ret := C.qore_svm_validate_elf(
		(*C.uint8_t)(unsafe.Pointer(&bytecode[0])),
		C.size_t(len(bytecode)),
	)
	if ret != 0 {
		return fmt.Errorf("ELF validation failed: error code %d", ret)
	}
	return nil
}

// Close releases the executor resources.
func (e *FFIExecutor) Close() {
	if e.handle != nil {
		C.qore_svm_free(e.handle)
		e.handle = nil
	}
}

// Version returns the qoresvm library version string.
func (e *FFIExecutor) Version() string {
	return C.GoString(C.qore_svm_version())
}

// resultSummary mirrors the JSON structure returned by the Rust execute call.
type resultSummary struct {
	Status string `json:"status"`
	CU     uint64 `json:"cu"`
	Logs   int    `json:"logs"`
	Error  string `json:"error"`
}
