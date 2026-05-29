package app

import (
	"encoding/json"
	"fmt"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	feemarkettypes "github.com/cosmos/evm/x/feemarket/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

// QorDenomMetadata returns the bank denom metadata for the native QoreChain
// token. cosmos/evm's x/vm InitGenesis derives the EVM coin decimals from this
// metadata (the exponent of the denom unit whose name equals Display), so the
// Display unit MUST sit at BaseDecimals (6).
func QorDenomMetadata() banktypes.Metadata {
	return banktypes.Metadata{
		Description: "The native staking, governance and gas token of QoreChain.",
		Base:        BaseDenom,    // uqor
		Display:     DisplayDenom, // qor
		Name:        "QoreChain",
		Symbol:      "QOR",
		DenomUnits: []*banktypes.DenomUnit{
			{Denom: BaseDenom, Exponent: 0, Aliases: []string{"microqor"}},
			{Denom: DisplayDenom, Exponent: BaseDecimals}, // qor = 10^6 uqor
		},
	}
}

// ApplyQoreChainDenoms rewrites the denom-dependent portions of a default
// genesis app state so the chain is configured for the native uqor token:
//   - bank: registers the uqor denom metadata
//   - x/vm (evm): points the EVM coin at uqor (6-decimal base) with the aqor
//     18-decimal extended denom used by x/precisebank
//
// staking/mint/gov denoms are handled separately via sdk.DefaultBondDenom.
func ApplyQoreChainDenoms(cdc codec.JSONCodec, appState map[string]json.RawMessage) error {
	if raw, ok := appState[banktypes.ModuleName]; ok {
		var bankGen banktypes.GenesisState
		if err := cdc.UnmarshalJSON(raw, &bankGen); err != nil {
			return fmt.Errorf("unmarshal bank genesis: %w", err)
		}
		bankGen.DenomMetadata = []banktypes.Metadata{QorDenomMetadata()}
		appState[banktypes.ModuleName] = cdc.MustMarshalJSON(&bankGen)
	}

	if raw, ok := appState[evmtypes.ModuleName]; ok {
		var evmGen evmtypes.GenesisState
		if err := cdc.UnmarshalJSON(raw, &evmGen); err != nil {
			return fmt.Errorf("unmarshal evm genesis: %w", err)
		}
		evmGen.Params.EvmDenom = BaseDenom // uqor
		evmGen.Params.ExtendedDenomOptions = &evmtypes.ExtendedDenomOptions{
			ExtendedDenom: ExtendedDenom, // aqor
		}
		appState[evmtypes.ModuleName] = cdc.MustMarshalJSON(&evmGen)
	}

	// Calibrate the fee market for the 6-decimal uqor base denom. The
	// cosmos/evm default base_fee (1e9) is tuned for 18-decimal chains and makes
	// a transfer cost ~400 QOR. At 0.1 uqor/gas a typical transfer (~100k gas)
	// costs ~0.01 QOR. Fees are gas-proportional (not a hard %-of-value floor).
	if raw, ok := appState[feemarkettypes.ModuleName]; ok {
		var fmGen feemarkettypes.GenesisState
		if err := cdc.UnmarshalJSON(raw, &fmGen); err != nil {
			return fmt.Errorf("unmarshal feemarket genesis: %w", err)
		}
		feeRate := sdkmath.LegacyNewDecWithPrec(1, 1) // 0.1 uqor per gas
		fmGen.Params.BaseFee = feeRate
		fmGen.Params.MinGasPrice = feeRate
		appState[feemarkettypes.ModuleName] = cdc.MustMarshalJSON(&fmGen)
	}

	return nil
}

// PatchGenesisFileDenoms loads a genesis file, applies the QoreChain denom
// configuration to its app state, and writes it back. Used by the wrapped
// `init` command so freshly generated genesis files are chain-ready.
func PatchGenesisFileDenoms(cdc codec.JSONCodec, genFile string) error {
	appGenesis, err := genutiltypes.AppGenesisFromFile(genFile)
	if err != nil {
		return err
	}

	var appState map[string]json.RawMessage
	if err := json.Unmarshal(appGenesis.AppState, &appState); err != nil {
		return fmt.Errorf("unmarshal app_state: %w", err)
	}

	if err := ApplyQoreChainDenoms(cdc, appState); err != nil {
		return err
	}

	appGenesis.AppState, err = json.Marshal(appState)
	if err != nil {
		return fmt.Errorf("marshal app_state: %w", err)
	}

	return appGenesis.SaveAs(genFile)
}
