//go:build proprietary

package precompiles

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/cosmos/evm/x/vm/statedb"

	sdk "github.com/cosmos/cosmos-sdk/types"
	pqcmod "github.com/qorechain/qorechain-core/x/pqc"
)

// PQCKeyStatusPrecompile queries PQC key registration status for an address.
// Allows Solidity contracts to check whether an address has a PQC key registered.
type PQCKeyStatusPrecompile struct {
	pqcKeeper pqcmod.PQCKeeper
}

// NewPQCKeyStatusPrecompile creates a new PQC key status precompile instance.
func NewPQCKeyStatusPrecompile(keeper pqcmod.PQCKeeper) *PQCKeyStatusPrecompile {
	return &PQCKeyStatusPrecompile{pqcKeeper: keeper}
}

func (p *PQCKeyStatusPrecompile) Address() common.Address { return PQCKeyStatusAddress }

func (p *PQCKeyStatusPrecompile) RequiredGas(_ []byte) uint64 { return 2_500 }

func (p *PQCKeyStatusPrecompile) Run(evm *vm.EVM, contract *vm.Contract, _ bool) ([]byte, error) {
	addr, err := DecodePQCKeyStatusInput(contract.Input)
	if err != nil {
		return EncodePQCKeyStatusOutput(false, 0, nil)
	}

	// Get SDK context from EVM StateDB
	ctx, err := getSDKContext(evm)
	if err != nil {
		return EncodePQCKeyStatusOutput(false, 0, nil)
	}

	// Convert EVM address to bech32 for keeper lookup
	bech32Addr := sdk.AccAddress(addr.Bytes()).String()

	info, found := p.pqcKeeper.GetPQCAccount(ctx, bech32Addr)
	if !found {
		return EncodePQCKeyStatusOutput(false, 0, nil)
	}

	return EncodePQCKeyStatusOutput(true, uint8(info.AlgorithmID), info.PublicKey)
}

// getSDKContext extracts the SDK context from the EVM's StateDB.
// This is the standard pattern in QoreChain EVM for stateful precompiles.
func getSDKContext(evm *vm.EVM) (sdk.Context, error) {
	sdb, ok := evm.StateDB.(*statedb.StateDB)
	if !ok {
		return sdk.Context{}, errors.New("not running in EVM context")
	}
	return sdb.GetCacheContext()
}
