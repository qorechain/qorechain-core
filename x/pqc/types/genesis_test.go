package types

import (
	"testing"
)

func TestDefaultGenesisState(t *testing.T) {
	gs := DefaultGenesisState()

	if gs == nil {
		t.Fatal("DefaultGenesisState() returned nil")
	}

	if len(gs.Algorithms) != 2 {
		t.Errorf("expected 2 default algorithms, got %d", len(gs.Algorithms))
	}

	if gs.Algorithms[0].ID != AlgorithmDilithium5 {
		t.Errorf("first algorithm should be Dilithium-5, got %s", gs.Algorithms[0].Name)
	}
	if gs.Algorithms[1].ID != AlgorithmMLKEM1024 {
		t.Errorf("second algorithm should be ML-KEM-1024, got %s", gs.Algorithms[1].Name)
	}

	if len(gs.Accounts) != 0 {
		t.Errorf("expected 0 accounts, got %d", len(gs.Accounts))
	}

	if len(gs.Migrations) != 0 {
		t.Errorf("expected 0 migrations, got %d", len(gs.Migrations))
	}
}

func TestGenesisState_Validate(t *testing.T) {
	tests := []struct {
		name    string
		gs      GenesisState
		wantErr bool
	}{
		{
			"valid default",
			*DefaultGenesisState(),
			false,
		},
		{
			"invalid security level 0",
			GenesisState{
				Params: Params{
					AllowClassicalFallback: true,
					MinSecurityLevel:       0,
				},
				Algorithms: []AlgorithmInfo{DefaultDilithium5Info()},
			},
			true,
		},
		{
			"invalid security level 6",
			GenesisState{
				Params: Params{
					AllowClassicalFallback: true,
					MinSecurityLevel:       6,
				},
				Algorithms: []AlgorithmInfo{DefaultDilithium5Info()},
			},
			true,
		},
		{
			"duplicate algorithm IDs",
			GenesisState{
				Params: DefaultParams(),
				Algorithms: []AlgorithmInfo{
					DefaultDilithium5Info(),
					DefaultDilithium5Info(),
				},
			},
			true,
		},
		{
			"migration with invalid source",
			GenesisState{
				Params:     DefaultParams(),
				Algorithms: []AlgorithmInfo{DefaultDilithium5Info()},
				Migrations: []MigrationInfo{
					{
						FromAlgorithmID: AlgorithmID(99),
						ToAlgorithmID:   AlgorithmDilithium5,
					},
				},
			},
			true,
		},
		{
			"migration with invalid target",
			GenesisState{
				Params:     DefaultParams(),
				Algorithms: []AlgorithmInfo{DefaultDilithium5Info()},
				Migrations: []MigrationInfo{
					{
						FromAlgorithmID: AlgorithmDilithium5,
						ToAlgorithmID:   AlgorithmID(99),
					},
				},
			},
			true,
		},
		{
			"valid migration",
			GenesisState{
				Params:     DefaultParams(),
				Algorithms: []AlgorithmInfo{DefaultDilithium5Info(), DefaultMLKEM1024Info()},
				Migrations: []MigrationInfo{
					{
						FromAlgorithmID: AlgorithmDilithium5,
						ToAlgorithmID:   AlgorithmMLKEM1024,
						StartHeight:     100,
						EndHeight:       1000100,
					},
				},
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.gs.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("GenesisState.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
