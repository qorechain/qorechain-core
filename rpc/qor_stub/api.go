//go:build !proprietary

// Package qor_stub provides stub implementations of the qor_ JSON-RPC namespace.
// All methods return "not available in public build" errors.
package qor_stub

import "fmt"

var errNotAvailable = fmt.Errorf("qor_ namespace is not available in the public build")

// QorAPI is a stub implementation of the qor_ JSON-RPC namespace.
type QorAPI struct{}

// NewQorAPI creates a stub QorAPI.
func NewQorAPI() *QorAPI {
	return &QorAPI{}
}

type StubResult struct {
	Error string `json:"error"`
}

func (api *QorAPI) GetPQCKeyStatus(_ string) (*StubResult, error) {
	return nil, errNotAvailable
}

func (api *QorAPI) GetAIRiskScore(_ string) (*StubResult, error) {
	return nil, errNotAvailable
}

func (api *QorAPI) GetCrossVMMessage(_ string) (*StubResult, error) {
	return nil, errNotAvailable
}

func (api *QorAPI) GetReputationScore(_ string) (*StubResult, error) {
	return nil, errNotAvailable
}

func (api *QorAPI) GetLayerInfo(_ string) (*StubResult, error) {
	return nil, errNotAvailable
}

func (api *QorAPI) GetBridgeStatus(_ string) (*StubResult, error) {
	return nil, errNotAvailable
}

func (api *QorAPI) GetRLAgentStatus() (*StubResult, error) {
	return nil, errNotAvailable
}

func (api *QorAPI) GetRLObservation() (*StubResult, error) {
	return nil, errNotAvailable
}

func (api *QorAPI) GetRLReward() (*StubResult, error) {
	return nil, errNotAvailable
}

func (api *QorAPI) GetPoolClassification(_ string) (*StubResult, error) {
	return nil, errNotAvailable
}
