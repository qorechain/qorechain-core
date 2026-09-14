package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// testAddr is generated at init time with the correct qor bech32 prefix.
var testAddr string

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("qor", "qorpub")
	config.SetBech32PrefixForValidator("qorvaloper", "qorvaloperpub")
	config.SetBech32PrefixForConsensusNode("qorvalcons", "qorvalconspub")

	// Generate a valid bech32 address from 20 zero bytes
	addr := sdk.AccAddress(make([]byte, 20))
	testAddr = addr.String()
}

func TestMsgRegisterPQCKey_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     MsgRegisterPQCKey
		wantErr bool
	}{
		{
			"valid hybrid",
			MsgRegisterPQCKey{
				Sender:          testAddr,
				DilithiumPubkey: make([]byte, 2592),
				KeyType:         KeyTypeHybrid,
			},
			false,
		},
		{
			"valid pqc_only",
			MsgRegisterPQCKey{
				Sender:          testAddr,
				DilithiumPubkey: make([]byte, 2592),
				KeyType:         KeyTypePQCOnly,
			},
			false,
		},
		{
			"valid classical_only",
			MsgRegisterPQCKey{
				Sender:  testAddr,
				KeyType: KeyTypeClassicalOnly,
			},
			false,
		},
		{
			"invalid sender",
			MsgRegisterPQCKey{
				Sender:          "invalid",
				DilithiumPubkey: make([]byte, 2592),
				KeyType:         KeyTypeHybrid,
			},
			true,
		},
		{
			"missing pubkey for hybrid",
			MsgRegisterPQCKey{
				Sender:  testAddr,
				KeyType: KeyTypeHybrid,
			},
			true,
		},
		{
			"invalid key type",
			MsgRegisterPQCKey{
				Sender:          testAddr,
				DilithiumPubkey: make([]byte, 2592),
				KeyType:         "invalid",
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("MsgRegisterPQCKey.ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMsgRegisterPQCKeyV2_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     MsgRegisterPQCKeyV2
		wantErr bool
	}{
		{
			"valid dilithium5",
			MsgRegisterPQCKeyV2{
				Sender:      testAddr,
				PublicKey:   make([]byte, 2592),
				AlgorithmID: AlgorithmDilithium5,
				KeyType:     KeyTypeHybrid,
			},
			false,
		},
		{
			// This case used to assert that a correctly sized ML-KEM key is a VALID
			// authentication key, which is how the defect survived: the test table
			// encoded it as intended behaviour. ML-KEM is a key-encapsulation
			// mechanism with no signing operation, so an account registering it can
			// never produce the hybrid signature the chain requires and cannot
			// rotate out, because rotation verifies against the key on file.
			"mlkem1024 rejected: a KEM cannot be an authentication key",
			MsgRegisterPQCKeyV2{
				Sender:      testAddr,
				PublicKey:   make([]byte, 1568),
				AlgorithmID: AlgorithmMLKEM1024,
				KeyType:     KeyTypePQCOnly,
			},
			true,
		},
		{
			"unspecified algorithm",
			MsgRegisterPQCKeyV2{
				Sender:      testAddr,
				PublicKey:   make([]byte, 2592),
				AlgorithmID: AlgorithmUnspecified,
				KeyType:     KeyTypeHybrid,
			},
			true,
		},
		{
			"invalid sender",
			MsgRegisterPQCKeyV2{
				Sender:      "bad",
				PublicKey:   make([]byte, 2592),
				AlgorithmID: AlgorithmDilithium5,
				KeyType:     KeyTypeHybrid,
			},
			true,
		},
		{
			"missing pubkey for pqc_only",
			MsgRegisterPQCKeyV2{
				Sender:      testAddr,
				AlgorithmID: AlgorithmDilithium5,
				KeyType:     KeyTypePQCOnly,
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("MsgRegisterPQCKeyV2.ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMsgMigratePQCKey_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     MsgMigratePQCKey
		wantErr bool
	}{
		{
			"valid migration",
			MsgMigratePQCKey{
				Sender:         testAddr,
				OldPublicKey:   make([]byte, 32),
				NewPublicKey:   make([]byte, 2592),
				NewAlgorithmID: AlgorithmDilithium5,
				OldSignature:   make([]byte, 64),
				NewSignature:   make([]byte, 64),
			},
			false,
		},
		{
			// A destination key of the wrong size would be stored and could never
			// verify: the account is bricked at the next transaction.
			"new pubkey of the wrong size for the target algorithm",
			MsgMigratePQCKey{
				Sender:         testAddr,
				OldPublicKey:   make([]byte, 32),
				NewPublicKey:   make([]byte, 32),
				NewAlgorithmID: AlgorithmDilithium5,
				OldSignature:   make([]byte, 64),
				NewSignature:   make([]byte, 64),
			},
			true,
		},
		{
			// ML-KEM encapsulates; it cannot sign, so it cannot authenticate.
			"migration to a key-encapsulation algorithm",
			MsgMigratePQCKey{
				Sender:         testAddr,
				OldPublicKey:   make([]byte, 32),
				NewPublicKey:   make([]byte, 1568),
				NewAlgorithmID: AlgorithmMLKEM1024,
				OldSignature:   make([]byte, 64),
				NewSignature:   make([]byte, 64),
			},
			true,
		},
		{
			"missing old pubkey",
			MsgMigratePQCKey{
				Sender:         testAddr,
				NewPublicKey:   make([]byte, 32),
				NewAlgorithmID: AlgorithmDilithium5,
				OldSignature:   make([]byte, 64),
				NewSignature:   make([]byte, 64),
			},
			true,
		},
		{
			"missing new pubkey",
			MsgMigratePQCKey{
				Sender:         testAddr,
				OldPublicKey:   make([]byte, 32),
				NewAlgorithmID: AlgorithmDilithium5,
				OldSignature:   make([]byte, 64),
				NewSignature:   make([]byte, 64),
			},
			true,
		},
		{
			"unspecified new algorithm",
			MsgMigratePQCKey{
				Sender:         testAddr,
				OldPublicKey:   make([]byte, 32),
				NewPublicKey:   make([]byte, 32),
				NewAlgorithmID: AlgorithmUnspecified,
				OldSignature:   make([]byte, 64),
				NewSignature:   make([]byte, 64),
			},
			true,
		},
		{
			"missing old signature",
			MsgMigratePQCKey{
				Sender:         testAddr,
				OldPublicKey:   make([]byte, 32),
				NewPublicKey:   make([]byte, 32),
				NewAlgorithmID: AlgorithmDilithium5,
				NewSignature:   make([]byte, 64),
			},
			true,
		},
		{
			"missing new signature",
			MsgMigratePQCKey{
				Sender:         testAddr,
				OldPublicKey:   make([]byte, 32),
				NewPublicKey:   make([]byte, 32),
				NewAlgorithmID: AlgorithmDilithium5,
				OldSignature:   make([]byte, 64),
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("MsgMigratePQCKey.ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMsgAddAlgorithm_ValidateBasic(t *testing.T) {
	validAlgo := AlgorithmInfo{
		ID:             AlgorithmID(3),
		Name:           "falcon512",
		Category:       CategorySignature,
		NISTLevel:      1,
		PublicKeySize:  897,
		PrivateKeySize: 1281,
		SignatureSize:  690,
	}

	tests := []struct {
		name    string
		msg     MsgAddAlgorithm
		wantErr bool
	}{
		{
			"valid",
			MsgAddAlgorithm{Authority: testAddr, Algorithm: validAlgo},
			false,
		},
		{
			"invalid authority",
			MsgAddAlgorithm{Authority: "bad", Algorithm: validAlgo},
			true,
		},
		{
			"invalid algorithm",
			MsgAddAlgorithm{Authority: testAddr, Algorithm: AlgorithmInfo{ID: AlgorithmUnspecified}},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("MsgAddAlgorithm.ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMsgDeprecateAlgorithm_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     MsgDeprecateAlgorithm
		wantErr bool
	}{
		{
			"valid",
			MsgDeprecateAlgorithm{
				Authority:        testAddr,
				AlgorithmID:      AlgorithmDilithium5,
				MigrationBlocks:  1000000,
				ReplacementAlgID: AlgorithmID(3),
			},
			false,
		},
		{
			"same algorithm as replacement",
			MsgDeprecateAlgorithm{
				Authority:        testAddr,
				AlgorithmID:      AlgorithmDilithium5,
				MigrationBlocks:  1000000,
				ReplacementAlgID: AlgorithmDilithium5,
			},
			true,
		},
		{
			"zero migration blocks",
			MsgDeprecateAlgorithm{
				Authority:        testAddr,
				AlgorithmID:      AlgorithmDilithium5,
				MigrationBlocks:  0,
				ReplacementAlgID: AlgorithmID(3),
			},
			true,
		},
		{
			"unspecified algorithm",
			MsgDeprecateAlgorithm{
				Authority:        testAddr,
				AlgorithmID:      AlgorithmUnspecified,
				MigrationBlocks:  1000000,
				ReplacementAlgID: AlgorithmID(3),
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("MsgDeprecateAlgorithm.ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMsgDisableAlgorithm_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     MsgDisableAlgorithm
		wantErr bool
	}{
		{
			"valid",
			MsgDisableAlgorithm{
				Authority:   testAddr,
				AlgorithmID: AlgorithmDilithium5,
				Reason:      "vulnerability discovered",
			},
			false,
		},
		{
			"missing reason",
			MsgDisableAlgorithm{
				Authority:   testAddr,
				AlgorithmID: AlgorithmDilithium5,
				Reason:      "",
			},
			true,
		},
		{
			"unspecified algorithm",
			MsgDisableAlgorithm{
				Authority:   testAddr,
				AlgorithmID: AlgorithmUnspecified,
				Reason:      "test",
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("MsgDisableAlgorithm.ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
