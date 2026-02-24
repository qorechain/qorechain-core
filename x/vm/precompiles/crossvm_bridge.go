//go:build proprietary

package precompiles

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	crossvmmod "github.com/qorechain/qorechain-core/x/crossvm"
	crossvmtypes "github.com/qorechain/qorechain-core/x/crossvm/types"
)

// CrossVMBridgePrecompile enables EVM contracts to call CosmWasm contracts
// via synchronous cross-VM message passing.
type CrossVMBridgePrecompile struct {
	crossvmKeeper crossvmmod.CrossVMKeeper
}

// NewCrossVMBridgePrecompile creates a new cross-VM bridge precompile instance.
func NewCrossVMBridgePrecompile(keeper crossvmmod.CrossVMKeeper) *CrossVMBridgePrecompile {
	return &CrossVMBridgePrecompile{crossvmKeeper: keeper}
}

func (p *CrossVMBridgePrecompile) Address() common.Address { return CrossVMBridgeAddress }

func (p *CrossVMBridgePrecompile) RequiredGas(_ []byte) uint64 { return 50_000 }

func (p *CrossVMBridgePrecompile) Run(evm *vm.EVM, contract *vm.Contract, _ bool) ([]byte, error) {
	// Decode: executeCrossVMCall(uint8 targetVM, string targetContract, bytes payload)
	targetVM, targetContract, payload, err := DecodeCrossVMCallInput(contract.Input)
	if err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}

	ctx, err := getSDKContext(evm)
	if err != nil {
		return nil, fmt.Errorf("context unavailable: %w", err)
	}

	// Map target VM type
	var vmType crossvmtypes.VMType
	switch targetVM {
	case 0:
		vmType = crossvmtypes.VMTypeEVM
	case 1:
		vmType = crossvmtypes.VMTypeCosmWasm
	default:
		return nil, fmt.Errorf("unsupported target VM type: %d", targetVM)
	}

	msg := crossvmtypes.CrossVMMessage{
		SourceVM:       crossvmtypes.VMTypeEVM,
		TargetVM:       vmType,
		Sender:         contract.Caller().Hex(),
		TargetContract: targetContract,
		Payload:        payload,
	}

	resp, err := p.crossvmKeeper.ExecuteSyncCall(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("cross-VM call failed: %w", err)
	}

	return resp.Data, nil
}
