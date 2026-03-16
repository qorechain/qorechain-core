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
	"bytes"
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

// ExecuteV2 runs a BPF program with full Solana-compatible account context.
func (e *FFIExecutor) ExecuteV2(program []byte, accounts []types.SVMAccount, metas []types.AccountMeta,
	instructionData []byte, programID [32]byte,
	computeBudget uint64, blockTime int64) (*types.ExecutionResult, error) {

	if len(program) == 0 {
		return nil, fmt.Errorf("empty program bytecode")
	}

	// Serialize accounts into BPF input format.
	inputBuf := types.SerializeAccountsForBPF(accounts, metas, instructionData, programID)

	// Calculate result buffer size: at least 64 KiB, or 2x the total account data.
	totalDataSize := 0
	for _, acc := range accounts {
		totalDataSize += len(acc.Data)
	}
	resultBufCap := 65536
	if totalDataSize*2 > resultBufCap {
		resultBufCap = totalDataSize * 2
	}
	resultBuf := make([]byte, resultBufCap)
	var resultLen C.size_t

	ret := C.qore_svm_execute_v2(
		(*C.uint8_t)(unsafe.Pointer(&program[0])),
		C.size_t(len(program)),
		(*C.uint8_t)(unsafe.Pointer(&inputBuf[0])),
		C.size_t(len(inputBuf)),
		C.uint64_t(computeBudget),
		C.int64_t(blockTime),
		(*C.uint8_t)(unsafe.Pointer(&resultBuf[0])),
		C.size_t(resultBufCap),
		&resultLen,
		nil, // callback_ctx
		nil, // sysvar_callback — uses Rust defaults
	)

	actualLen := int(resultLen)
	if actualLen > resultBufCap {
		actualLen = resultBufCap
	}

	if ret < 0 {
		if actualLen > 0 {
			nullIdx := bytes.IndexByte(resultBuf[:actualLen], 0)
			jsonEnd := actualLen
			if nullIdx >= 0 {
				jsonEnd = nullIdx
			}
			var summary resultSummaryV2
			if err := json.Unmarshal(resultBuf[:jsonEnd], &summary); err == nil && summary.Error != "" {
				return nil, fmt.Errorf("SVM v2 execution failed: %s (code %d)", summary.Error, ret)
			}
		}
		return nil, fmt.Errorf("SVM v2 execution failed: error code %d", ret)
	}

	return parseV2Result(resultBuf, actualLen)
}

// ExecuteNative runs a native program directly (no BPF interpretation).
func (e *FFIExecutor) ExecuteNative(programID [32]byte, accounts []types.SVMAccount, metas []types.AccountMeta,
	instructionData []byte, blockTime int64) (*types.ExecutionResult, error) {

	// Serialize accounts into BPF input format.
	inputBuf := types.SerializeAccountsForBPF(accounts, metas, instructionData, programID)

	// Result buffer — native programs typically return small results.
	resultBufCap := 65536
	resultBuf := make([]byte, resultBufCap)
	var resultLen C.size_t

	ret := C.qore_svm_execute_native(
		(*C.uint8_t)(unsafe.Pointer(&programID[0])),
		(*C.uint8_t)(unsafe.Pointer(&inputBuf[0])),
		C.size_t(len(inputBuf)),
		C.int64_t(blockTime),
		(*C.uint8_t)(unsafe.Pointer(&resultBuf[0])),
		C.size_t(resultBufCap),
		&resultLen,
	)

	actualLen := int(resultLen)
	if actualLen > resultBufCap {
		actualLen = resultBufCap
	}

	if ret < 0 {
		if actualLen > 0 {
			nullIdx := bytes.IndexByte(resultBuf[:actualLen], 0)
			jsonEnd := actualLen
			if nullIdx >= 0 {
				jsonEnd = nullIdx
			}
			var summary resultSummaryV2
			if err := json.Unmarshal(resultBuf[:jsonEnd], &summary); err == nil && summary.Error != "" {
				return nil, fmt.Errorf("SVM native execution failed: %s (code %d)", summary.Error, ret)
			}
		}
		return nil, fmt.Errorf("SVM native execution failed: error code %d", ret)
	}

	return parseV2Result(resultBuf, actualLen)
}

// parseV2Result parses the v2/native result buffer format:
// JSON header (null-terminated) followed by optional binary modified accounts.
func parseV2Result(resultBuf []byte, actualLen int) (*types.ExecutionResult, error) {
	if actualLen == 0 {
		return &types.ExecutionResult{Success: true}, nil
	}

	// Find null terminator separating JSON header from binary data.
	nullIdx := bytes.IndexByte(resultBuf[:actualLen], 0)
	jsonEnd := actualLen
	if nullIdx >= 0 {
		jsonEnd = nullIdx
	}

	var summary resultSummaryV2
	if err := json.Unmarshal(resultBuf[:jsonEnd], &summary); err != nil {
		return &types.ExecutionResult{
			Success:    true,
			ReturnData: resultBuf[:actualLen],
		}, nil
	}

	result := &types.ExecutionResult{
		Success:          summary.Status == "ok",
		ComputeUnitsUsed: summary.CU,
		Error:            summary.Error,
	}

	// Parse modified accounts from binary data after the null byte.
	if nullIdx >= 0 && nullIdx+1 < actualLen {
		modified, err := types.DeserializeModifiedAccounts(resultBuf[nullIdx+1 : actualLen])
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize modified accounts: %w", err)
		}
		result.ModifiedAccounts = modified
	}

	return result, nil
}

// resultSummaryV2 mirrors the JSON structure returned by the Rust v2/native execute calls.
type resultSummaryV2 struct {
	Status      string `json:"status"`
	CU          uint64 `json:"cu"`
	Logs        int    `json:"logs"`
	Error       string `json:"error"`
	NumModified int    `json:"num_modified"`
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
