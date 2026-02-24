//go:build !proprietary

package ffi

import (
	"github.com/qorechain/qorechain-core/x/svm/types"
)

// StubExecutor is a no-op SVMExecutor for the public build.
type StubExecutor struct{}

func NewStubExecutor() *StubExecutor {
	return &StubExecutor{}
}

func (e *StubExecutor) Execute(_ []byte, _ []byte, _ []types.SVMAccount,
	_ uint64) (*types.ExecutionResult, error) {
	return nil, types.ErrSVMDisabled.Wrap("SVM executor not available in community build")
}

func (e *StubExecutor) ValidateProgram(_ []byte) error {
	return types.ErrSVMDisabled.Wrap("SVM executor not available in community build")
}

func (e *StubExecutor) Close() {}
