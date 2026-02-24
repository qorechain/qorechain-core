package types

import "fmt"

const (
	DefaultMaxProgramSize     uint64  = 10 * 1024 * 1024 // 10MB
	DefaultMaxAccountDataSize uint64  = 10 * 1024 * 1024 // 10MB
	DefaultComputeBudgetMax   uint64  = 1_400_000
	DefaultLamportsPerByte    uint64  = 3480
	DefaultRentExemptionMulti float64 = 2.0 // 2 years of rent
	DefaultEnabled            bool    = true
	DefaultSVMSlotOffset      int64   = 0
	DefaultSigScheme          uint8   = 0 // Ed25519
	DefaultMaxCPI             uint8   = 4
)

// Params defines the configurable parameters for the SVM runtime module.
type Params struct {
	MaxProgramSize     uint64  `json:"max_program_size"`
	MaxAccountDataSize uint64  `json:"max_account_data_size"`
	ComputeBudgetMax   uint64  `json:"compute_budget_max"`
	LamportsPerByte    uint64  `json:"lamports_per_byte"`
	RentExemptionMulti float64 `json:"rent_exemption_multi"`
	Enabled            bool    `json:"enabled"`
	SVMSlotOffset      int64   `json:"svm_slot_offset"`
	DefaultSigScheme   uint8   `json:"default_sig_scheme"`
	MaxCPI             uint8   `json:"max_cpi"`
}

// DefaultParams returns a default set of SVM parameters.
func DefaultParams() Params {
	return Params{
		MaxProgramSize:     DefaultMaxProgramSize,
		MaxAccountDataSize: DefaultMaxAccountDataSize,
		ComputeBudgetMax:   DefaultComputeBudgetMax,
		LamportsPerByte:    DefaultLamportsPerByte,
		RentExemptionMulti: DefaultRentExemptionMulti,
		Enabled:            DefaultEnabled,
		SVMSlotOffset:      DefaultSVMSlotOffset,
		DefaultSigScheme:   DefaultSigScheme,
		MaxCPI:             DefaultMaxCPI,
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
	if p.RentExemptionMulti <= 0 {
		return fmt.Errorf("rent exemption multiplier must be positive")
	}
	if p.MaxCPI == 0 {
		return fmt.Errorf("max CPI depth must be positive")
	}
	return nil
}
