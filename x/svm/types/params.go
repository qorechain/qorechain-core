package types

import (
	"fmt"

	"cosmossdk.io/math"
)

var (
	DefaultRentExemptionMultiDec = math.LegacyNewDec(2) // 2 years of rent
)

const (
	DefaultMaxProgramSize     uint64 = 10 * 1024 * 1024 // 10MB
	DefaultMaxAccountDataSize uint64 = 10 * 1024 * 1024 // 10MB
	DefaultComputeBudgetMax   uint64 = 1_400_000
	DefaultLamportsPerByte    uint64 = 3480
	DefaultEnabled            bool   = true
	DefaultSVMSlotOffset      int64  = 0
	DefaultSigScheme          uint8  = 0 // Ed25519
	DefaultMaxCPI             uint8  = 4
	// DefaultMaxCustodyUqor is no cap, the behaviour before the cap existed.
	// The cap is enforced only on a chain with the svm-lane-governed feature,
	// whose release handler writes a finite value.
	DefaultMaxCustodyUqor uint64 = 0
	// DefaultMaxCustodyHardUqor is no ceiling. With svm-custody-hard-ceiling,
	// governance must set a finite ceiling before the lane opens on any network
	// (the v3.2.8 handler writes one on the public testnet).
	DefaultMaxCustodyHardUqor uint64 = 0
	// DefaultMaxAccountExcessUqor is no per-account cap
	// (svm-account-excess-cap).
	DefaultMaxAccountExcessUqor uint64 = 0
)

// Params defines the configurable parameters for the SVM runtime module.
type Params struct {
	MaxProgramSize     uint64         `json:"max_program_size"`
	MaxAccountDataSize uint64         `json:"max_account_data_size"`
	ComputeBudgetMax   uint64         `json:"compute_budget_max"`
	LamportsPerByte    uint64         `json:"lamports_per_byte"`
	RentExemptionMulti math.LegacyDec `json:"rent_exemption_multi"`
	Enabled            bool           `json:"enabled"`
	SVMSlotOffset      int64          `json:"svm_slot_offset"`
	DefaultSigScheme   uint8          `json:"default_sig_scheme"`
	MaxCPI             uint8          `json:"max_cpi"`
	// MaxCustodyUqor caps the uqor balance of the svm module account; 0 is no
	// cap. omitempty keeps the stored JSON of every params value written before
	// the field existed byte-identical when it is rewritten on replay.
	MaxCustodyUqor uint64 `json:"max_custody_uqor,omitempty"`
	// MaxCustodyHardUqor is the absolute ceiling on the svm module account's
	// raw uqor balance (svm-custody-hard-ceiling); 0 is no ceiling. omitempty
	// for the same replay reason as MaxCustodyUqor.
	MaxCustodyHardUqor uint64 `json:"max_custody_hard_uqor,omitempty"`
	// MaxAccountExcessUqor caps the uqor a store account holds above its
	// rent-exempt minimum (svm-account-excess-cap); 0 is no cap.
	MaxAccountExcessUqor uint64 `json:"max_account_excess_uqor,omitempty"`
}

// DefaultParams returns a default set of SVM parameters.
func DefaultParams() Params {
	return Params{
		MaxProgramSize:       DefaultMaxProgramSize,
		MaxAccountDataSize:   DefaultMaxAccountDataSize,
		ComputeBudgetMax:     DefaultComputeBudgetMax,
		LamportsPerByte:      DefaultLamportsPerByte,
		RentExemptionMulti:   DefaultRentExemptionMultiDec,
		Enabled:              DefaultEnabled,
		SVMSlotOffset:        DefaultSVMSlotOffset,
		DefaultSigScheme:     DefaultSigScheme,
		MaxCPI:               DefaultMaxCPI,
		MaxCustodyUqor:       DefaultMaxCustodyUqor,
		MaxCustodyHardUqor:   DefaultMaxCustodyHardUqor,
		MaxAccountExcessUqor: DefaultMaxAccountExcessUqor,
	}
}

// Validate performs basic validation of SVM parameters.
func (p Params) Validate() error {
	if p.MaxProgramSize == 0 {
		return fmt.Errorf("max program size must be positive")
	}
	if p.MaxAccountDataSize == 0 {
		return fmt.Errorf("max account data size must be positive")
	}
	if p.ComputeBudgetMax == 0 {
		return fmt.Errorf("compute budget max must be positive")
	}
	if p.LamportsPerByte == 0 {
		return fmt.Errorf("lamports per byte must be positive")
	}
	if !p.RentExemptionMulti.IsPositive() {
		return fmt.Errorf("rent exemption multiplier must be positive")
	}
	if p.MaxCPI == 0 {
		return fmt.Errorf("max CPI depth must be positive")
	}
	// The ceiling sits above the monitor threshold; a ceiling below it would
	// make the threshold meaningless.
	if p.MaxCustodyHardUqor != 0 && p.MaxCustodyHardUqor < p.MaxCustodyUqor {
		return fmt.Errorf("max_custody_hard_uqor %d is below max_custody_uqor %d",
			p.MaxCustodyHardUqor, p.MaxCustodyUqor)
	}
	return nil
}
