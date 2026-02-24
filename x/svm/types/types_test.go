package types

import (
	"bytes"
	"testing"
)

// ---------------------------------------------------------------------------
// TestSVMAccountValidate
// ---------------------------------------------------------------------------

func TestSVMAccountValidate(t *testing.T) {
	t.Run("valid account passes", func(t *testing.T) {
		acc := SVMAccount{
			Address: [32]byte{1},
			DataLen: 4,
			Data:    []byte{0xDE, 0xAD, 0xBE, 0xEF},
			Owner:   [32]byte{2},
		}
		if err := acc.Validate(); err != nil {
			t.Fatalf("expected valid account, got error: %v", err)
		}
	})

	t.Run("zero address fails", func(t *testing.T) {
		acc := SVMAccount{
			Address: [32]byte{},
			DataLen: 0,
			Data:    []byte{},
		}
		if err := acc.Validate(); err == nil {
			t.Fatal("expected error for zero address")
		}
	})

	t.Run("data length mismatch fails", func(t *testing.T) {
		acc := SVMAccount{
			Address: [32]byte{1},
			DataLen: 10,
			Data:    []byte{0x01, 0x02},
		}
		if err := acc.Validate(); err == nil {
			t.Fatal("expected error for data length mismatch")
		}
	})

	t.Run("executable without owner fails", func(t *testing.T) {
		acc := SVMAccount{
			Address:    [32]byte{1},
			DataLen:    0,
			Data:       []byte{},
			Executable: true,
			Owner:      [32]byte{}, // zero
		}
		if err := acc.Validate(); err == nil {
			t.Fatal("expected error for executable account with zero owner")
		}
	})

	t.Run("executable with owner passes", func(t *testing.T) {
		acc := SVMAccount{
			Address:    [32]byte{1},
			DataLen:    0,
			Data:       []byte{},
			Executable: true,
			Owner:      [32]byte{5},
		}
		if err := acc.Validate(); err != nil {
			t.Fatalf("expected valid executable account, got error: %v", err)
		}
	})

	t.Run("marshal and unmarshal round trip", func(t *testing.T) {
		acc := SVMAccount{
			Address:   [32]byte{0xAA},
			Lamports:  1_000_000,
			DataLen:   2,
			Data:      []byte{0x01, 0x02},
			Owner:     [32]byte{0xBB},
			RentEpoch: 42,
		}
		data, err := acc.Marshal()
		if err != nil {
			t.Fatalf("marshal error: %v", err)
		}
		var acc2 SVMAccount
		if err := acc2.Unmarshal(data); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if acc.Address != acc2.Address || acc.Lamports != acc2.Lamports ||
			acc.DataLen != acc2.DataLen || acc.Owner != acc2.Owner ||
			acc.Executable != acc2.Executable || acc.RentEpoch != acc2.RentEpoch ||
			!bytes.Equal(acc.Data, acc2.Data) {
			t.Fatal("round-trip mismatch")
		}
	})
}

// ---------------------------------------------------------------------------
// TestInstructionValidate
// ---------------------------------------------------------------------------

func TestInstructionValidate(t *testing.T) {
	t.Run("valid instruction passes", func(t *testing.T) {
		ix := Instruction{
			ProgramID: [32]byte{1},
			Accounts: []AccountMeta{
				{Address: [32]byte{2}, IsSigner: true, IsWritable: true},
			},
			Data: []byte{0x01},
		}
		if err := ix.Validate(); err != nil {
			t.Fatalf("expected valid instruction, got error: %v", err)
		}
	})

	t.Run("zero program ID fails", func(t *testing.T) {
		ix := Instruction{
			ProgramID: [32]byte{},
			Data:      []byte{0x01},
		}
		if err := ix.Validate(); err == nil {
			t.Fatal("expected error for zero program ID")
		}
	})
}

// ---------------------------------------------------------------------------
// TestAddressMapping
// ---------------------------------------------------------------------------

func TestAddressMapping(t *testing.T) {
	t.Run("SVMToCosmosAddress produces 20-byte output", func(t *testing.T) {
		var svmAddr [32]byte
		svmAddr[0] = 0xFF
		svmAddr[31] = 0x01
		cosmosAddr := SVMToCosmosAddress(svmAddr)
		if len(cosmosAddr) != 20 {
			t.Fatalf("expected 20 bytes, got %d", len(cosmosAddr))
		}
	})

	t.Run("EVMToSVMAddress produces 32-byte output", func(t *testing.T) {
		var evmAddr [20]byte
		evmAddr[0] = 0xAA
		svmAddr := EVMToSVMAddress(evmAddr)
		if len(svmAddr) != 32 {
			t.Fatalf("expected 32 bytes, got %d", len(svmAddr))
		}
		// Result should be non-zero
		var zero [32]byte
		if svmAddr == zero {
			t.Fatal("EVMToSVMAddress returned zero address")
		}
	})

	t.Run("SVMToEVMAddress truncates correctly", func(t *testing.T) {
		var svmAddr [32]byte
		for i := 0; i < 32; i++ {
			svmAddr[i] = byte(i)
		}
		evmAddr := SVMToEVMAddress(svmAddr)
		for i := 0; i < 20; i++ {
			if evmAddr[i] != byte(i) {
				t.Fatalf("mismatch at byte %d: expected %d, got %d", i, i, evmAddr[i])
			}
		}
	})

	t.Run("different SVM addresses produce different native addresses", func(t *testing.T) {
		addr1 := SVMToCosmosAddress([32]byte{1})
		addr2 := SVMToCosmosAddress([32]byte{2})
		if bytes.Equal(addr1, addr2) {
			t.Fatal("different SVM addresses should produce different native addresses")
		}
	})
}

// ---------------------------------------------------------------------------
// TestBase58RoundTrip
// ---------------------------------------------------------------------------

func TestBase58RoundTrip(t *testing.T) {
	systemAddrs := map[string][32]byte{
		"SystemProgramAddress": SystemProgramAddress,
		"SPLTokenAddress":      SPLTokenAddress,
		"ATAAddress":           ATAAddress,
		"MemoAddress":          MemoAddress,
		"QorPQCAddress":        QorPQCAddress,
		"QorAIAddress":         QorAIAddress,
	}

	for name, addr := range systemAddrs {
		t.Run(name, func(t *testing.T) {
			encoded := Base58Encode(addr)
			if len(encoded) == 0 {
				t.Fatal("Base58Encode returned empty string")
			}
			decoded, err := Base58Decode(encoded)
			if err != nil {
				t.Fatalf("Base58Decode error: %v", err)
			}
			if decoded != addr {
				t.Fatalf("round-trip mismatch for %s: encoded=%s", name, encoded)
			}
		})
	}

	t.Run("arbitrary address round trip", func(t *testing.T) {
		var addr [32]byte
		for i := 0; i < 32; i++ {
			addr[i] = byte(i * 7)
		}
		encoded := Base58Encode(addr)
		decoded, err := Base58Decode(encoded)
		if err != nil {
			t.Fatalf("Base58Decode error: %v", err)
		}
		if decoded != addr {
			t.Fatal("round-trip mismatch for arbitrary address")
		}
	})

	t.Run("invalid character returns error", func(t *testing.T) {
		_, err := Base58Decode("0OIl") // 0, O, I, l are not in base58 alphabet
		if err == nil {
			t.Fatal("expected error for invalid base58 characters")
		}
	})

	t.Run("empty string returns error", func(t *testing.T) {
		_, err := Base58Decode("")
		if err == nil {
			t.Fatal("expected error for empty string")
		}
	})
}

// ---------------------------------------------------------------------------
// TestSystemAddresses
// ---------------------------------------------------------------------------

func TestSystemAddresses(t *testing.T) {
	addrs := [][32]byte{
		SystemProgramAddress,
		SPLTokenAddress,
		ATAAddress,
		MemoAddress,
		QorPQCAddress,
		QorAIAddress,
	}

	var zeroAddr [32]byte

	t.Run("all non-zero except system program", func(t *testing.T) {
		// SystemProgramAddress is intentionally [32]byte{0} (base58: "1111...1")
		// All other system addresses must be non-zero
		for i, addr := range addrs {
			if i == 0 {
				// SystemProgramAddress is the zero address by definition
				if addr != zeroAddr {
					t.Fatalf("SystemProgramAddress should be the zero address")
				}
				continue
			}
			if addr == zeroAddr {
				t.Fatalf("system address %d is zero", i)
			}
		}
	})

	t.Run("all unique", func(t *testing.T) {
		for i := 0; i < len(addrs); i++ {
			for j := i + 1; j < len(addrs); j++ {
				if addrs[i] == addrs[j] {
					t.Fatalf("system addresses %d and %d are identical", i, j)
				}
			}
		}
	})
}

// ---------------------------------------------------------------------------
// TestKVStoreKeys
// ---------------------------------------------------------------------------

func TestKVStoreKeys(t *testing.T) {
	t.Run("AccountKey produces correct prefix", func(t *testing.T) {
		var addr [32]byte
		addr[0] = 0xAA
		key := AccountKey(addr)
		if len(key) != 33 {
			t.Fatalf("expected 33 bytes (1 prefix + 32 addr), got %d", len(key))
		}
		if key[0] != AccountKeyPrefix[0] {
			t.Fatalf("expected prefix 0x01, got 0x%02x", key[0])
		}
		if key[1] != 0xAA {
			t.Fatalf("expected first addr byte 0xAA, got 0x%02x", key[1])
		}
	})

	t.Run("ProgramKey produces correct prefix", func(t *testing.T) {
		var addr [32]byte
		addr[0] = 0xBB
		key := ProgramKey(addr)
		if len(key) != 33 {
			t.Fatalf("expected 33 bytes (1 prefix + 32 addr), got %d", len(key))
		}
		if key[0] != ProgramKeyPrefix[0] {
			t.Fatalf("expected prefix 0x02, got 0x%02x", key[0])
		}
	})

	t.Run("keys are unique for different addresses", func(t *testing.T) {
		addr1 := [32]byte{1}
		addr2 := [32]byte{2}
		key1 := AccountKey(addr1)
		key2 := AccountKey(addr2)
		if bytes.Equal(key1, key2) {
			t.Fatal("different addresses should produce different keys")
		}
	})

	t.Run("AccountKey and ProgramKey have different prefixes", func(t *testing.T) {
		addr := [32]byte{1}
		accKey := AccountKey(addr)
		progKey := ProgramKey(addr)
		if bytes.Equal(accKey, progKey) {
			t.Fatal("AccountKey and ProgramKey should differ for same address")
		}
	})

	t.Run("AddrMapKey produces correct prefix", func(t *testing.T) {
		cosmosAddr := make([]byte, 20)
		cosmosAddr[0] = 0xCC
		key := AddrMapKey(cosmosAddr)
		if len(key) != 21 {
			t.Fatalf("expected 21 bytes (1 prefix + 20 addr), got %d", len(key))
		}
		if key[0] != AddrMapKeyPrefix[0] {
			t.Fatalf("expected prefix 0x03, got 0x%02x", key[0])
		}
	})
}

// ---------------------------------------------------------------------------
// TestRecentBlockhashKey
// ---------------------------------------------------------------------------

func TestRecentBlockhashKey(t *testing.T) {
	t.Run("produces correct 9-byte key", func(t *testing.T) {
		key := RecentBlockhashKey(12345)
		if len(key) != 9 {
			t.Fatalf("expected 9 bytes (1 prefix + 8 height), got %d", len(key))
		}
		if key[0] != RecentBlockhashPrefix[0] {
			t.Fatalf("expected prefix 0x07, got 0x%02x", key[0])
		}
	})

	t.Run("different heights produce different keys", func(t *testing.T) {
		key1 := RecentBlockhashKey(100)
		key2 := RecentBlockhashKey(200)
		if bytes.Equal(key1, key2) {
			t.Fatal("different heights should produce different keys")
		}
	})

	t.Run("height encoding is big-endian", func(t *testing.T) {
		key := RecentBlockhashKey(1)
		// Big-endian encoding of 1 in 8 bytes: 0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x01
		if key[8] != 0x01 {
			t.Fatalf("expected last byte to be 0x01 for height=1, got 0x%02x", key[8])
		}
		for i := 1; i < 8; i++ {
			if key[i] != 0x00 {
				t.Fatalf("expected zero byte at position %d, got 0x%02x", i, key[i])
			}
		}
	})
}
