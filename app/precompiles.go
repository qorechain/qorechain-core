//go:build proprietary

package app

import (
	"fmt"
	"maps"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	bankprecompile "github.com/cosmos/evm/precompiles/bank"
	"github.com/cosmos/evm/precompiles/bech32"
	cmn "github.com/cosmos/evm/precompiles/common"
	distprecompile "github.com/cosmos/evm/precompiles/distribution"
	evidenceprecompile "github.com/cosmos/evm/precompiles/evidence"
	govprecompile "github.com/cosmos/evm/precompiles/gov"
	ics20precompile "github.com/cosmos/evm/precompiles/ics20"
	"github.com/cosmos/evm/precompiles/p256"
	slashingprecompile "github.com/cosmos/evm/precompiles/slashing"
	stakingprecompile "github.com/cosmos/evm/precompiles/staking"
	erc20Keeper "github.com/cosmos/evm/x/erc20/keeper"
	transferkeeper "github.com/cosmos/evm/x/ibc/transfer/keeper"
	evmkeeper "github.com/cosmos/evm/x/vm/keeper"
	channelkeeper "github.com/cosmos/ibc-go/v10/modules/core/04-channel/keeper"

	evidencekeeper "cosmossdk.io/x/evidence/keeper"

	"github.com/cosmos/cosmos-sdk/codec"
	distributionkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

	aimod "github.com/qorechain/qorechain-core/x/ai"
	crossvmmod "github.com/qorechain/qorechain-core/x/crossvm"
	pqcmod "github.com/qorechain/qorechain-core/x/pqc"
	qoreprecompiles "github.com/qorechain/qorechain-core/x/vm/precompiles"
)

const bech32PrecompileBaseGas = 6_000

// newAvailableStaticPrecompiles returns the full set of static precompiled contracts
// including standard QoreChain EVM precompiles and QoreChain custom precompiles
// (PQC, AI, RL consensus params, and CrossVM bridge).
func newAvailableStaticPrecompiles(
	stakingKeeper stakingkeeper.Keeper,
	distributionKeeper distributionkeeper.Keeper,
	bankKeeper cmn.BankKeeper,
	erc20Keeper erc20Keeper.Keeper,
	transferKeeper transferkeeper.Keeper,
	channelKeeper *channelkeeper.Keeper,
	evmKeeper *evmkeeper.Keeper,
	govKeeper govkeeper.Keeper,
	slashingKeeper slashingkeeper.Keeper,
	evidenceKeeper evidencekeeper.Keeper,
	cdc codec.Codec,
	// QoreChain custom keepers
	pqcKeeper pqcmod.PQCKeeper,
	aiKeeper aimod.AIKeeper,
	crossvmKeeper crossvmmod.CrossVMKeeper,
	rlProvider qoreprecompiles.RLConsensusParamsProvider,
) map[common.Address]vm.PrecompiledContract {
	precompiles := maps.Clone(vm.PrecompiledContractsBerlin)

	// Stateless precompiles
	p256Precompile := &p256.Precompile{}
	bech32Precompile, err := bech32.NewPrecompile(bech32PrecompileBaseGas)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate bech32 precompile: %w", err))
	}
	precompiles[p256Precompile.Address()] = p256Precompile
	precompiles[bech32Precompile.Address()] = bech32Precompile

	// Stateful precompiles — standard QoreChain EVM
	stakingPrecompile, err := stakingprecompile.NewPrecompile(stakingKeeper, bankKeeper)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate staking precompile: %w", err))
	}
	precompiles[stakingPrecompile.Address()] = stakingPrecompile

	distributionPrecompile, err := distprecompile.NewPrecompile(distributionKeeper, bankKeeper, stakingKeeper, evmKeeper)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate distribution precompile: %w", err))
	}
	precompiles[distributionPrecompile.Address()] = distributionPrecompile

	ibcTransferPrecompile, err := ics20precompile.NewPrecompile(stakingKeeper, bankKeeper, transferKeeper, channelKeeper, evmKeeper)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate ICS20 precompile: %w", err))
	}
	precompiles[ibcTransferPrecompile.Address()] = ibcTransferPrecompile

	bankPrecompile, err := bankprecompile.NewPrecompile(bankKeeper, erc20Keeper)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate bank precompile: %w", err))
	}
	precompiles[bankPrecompile.Address()] = bankPrecompile

	govPrecompile, err := govprecompile.NewPrecompile(govKeeper, bankKeeper, cdc)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate gov precompile: %w", err))
	}
	precompiles[govPrecompile.Address()] = govPrecompile

	slashingPrecompile, err := slashingprecompile.NewPrecompile(slashingKeeper, bankKeeper)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate slashing precompile: %w", err))
	}
	precompiles[slashingPrecompile.Address()] = slashingPrecompile

	evidencePrecompile, err := evidenceprecompile.NewPrecompile(evidenceKeeper, bankKeeper)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate evidence precompile: %w", err))
	}
	precompiles[evidencePrecompile.Address()] = evidencePrecompile

	// QoreChain custom precompiles — PQC, AI, RL, CrossVM
	pqcVerify := qoreprecompiles.NewPQCVerifyPrecompile(pqcKeeper)
	precompiles[pqcVerify.Address()] = pqcVerify

	pqcKeyStatus := qoreprecompiles.NewPQCKeyStatusPrecompile(pqcKeeper)
	precompiles[pqcKeyStatus.Address()] = pqcKeyStatus

	aiRiskScore := qoreprecompiles.NewAIRiskScorePrecompile(aiKeeper)
	precompiles[aiRiskScore.Address()] = aiRiskScore

	aiAnomalyCheck := qoreprecompiles.NewAIAnomalyCheckPrecompile(aiKeeper)
	precompiles[aiAnomalyCheck.Address()] = aiAnomalyCheck

	rlConsensusParams := qoreprecompiles.NewRLConsensusParamsPrecompile(rlProvider)
	precompiles[rlConsensusParams.Address()] = rlConsensusParams

	crossvmBridge := qoreprecompiles.NewCrossVMBridgePrecompile(crossvmKeeper)
	precompiles[crossvmBridge.Address()] = crossvmBridge

	return precompiles
}

// registerEVMPrecompiles sets the static precompiles on the EVM keeper.
func (app *QoreChainApp) registerEVMPrecompiles() {
	app.EVMKeeper.WithStaticPrecompiles(
		newAvailableStaticPrecompiles(
			*app.StakingKeeper,
			app.DistrKeeper,
			app.PreciseBankKeeper,
			app.Erc20Keeper,
			app.TransferKeeper,
			app.IBCKeeper.ChannelKeeper,
			app.EVMKeeper,
			*app.GovKeeper,
			app.SlashingKeeper,
			app.EvidenceKeeper,
			app.appCodec,
			// QoreChain custom keepers
			app.PQCKeeper,
			app.AIKeeper,
			app.CrossVMKeeper,
			qoreprecompiles.DefaultStaticRLProvider(),
		),
	)
}
