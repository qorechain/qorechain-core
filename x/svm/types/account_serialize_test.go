package types

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestSerializeAccountsForBPF_Empty(t *testing.T) {
	var programID [32]byte
	for i := range programID {
		programID[i] = byte(i)
	}

	buf := SerializeAccountsForBPF(nil, nil, nil, programID)

	// 8 (num_accounts=0) + 8 (ix_data_len=0) + 32 (program_id) = 48
	if len(buf) != 48 {
		t.Fatalf("expected buffer size 48, got %d", len(buf))
	}

	numAccounts := binary.LittleEndian.Uint64(buf[0:8])
	if numAccounts != 0 {
		t.Fatalf("expected 0 accounts, got %d", numAccounts)
	}

	ixLen := binary.LittleEndian.Uint64(buf[8:16])
	if ixLen != 0 {
		t.Fatalf("expected 0 instruction data length, got %d", ixLen)
	}

	var gotPID [32]byte
	copy(gotPID[:], buf[16:48])
	if gotPID != programID {
		t.Fatalf("program ID mismatch")
	}
}

func TestSerializeAccountsForBPF_Roundtrip(t *testing.T) {
	var addr, owner, programID [32]byte
	for i := range addr {
		addr[i] = byte(i + 1)
	}
	for i := range owner {
		owner[i] = byte(i + 100)
	}
	for i := range programID {
		programID[i] = byte(i + 200)
	}

	data := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	ixData := []byte{0x01, 0x02, 0x03}

	accounts := []SVMAccount{
		{
			Address:    addr,
			Lamports:   1_000_000,
			DataLen:    uint64(len(data)),
			Data:       data,
			Owner:      owner,
			Executable: true,
			RentEpoch:  42,
		},
	}

	metas := []AccountMeta{
		{
			Address:    addr,
			IsSigner:   true,
			IsWritable: true,
		},
	}

	buf := SerializeAccountsForBPF(accounts, metas, ixData, programID)

	gotAccounts, gotIxData, gotPID, err := DeserializeAccountsFromBPF(buf)
	if err != nil {
		t.Fatalf("deserialization error: %v", err)
	}

	if len(gotAccounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(gotAccounts))
	}

	a := gotAccounts[0]
	if a.Address != addr {
		t.Errorf("address mismatch")
	}
	if a.Owner != owner {
		t.Errorf("owner mismatch")
	}
	if a.Lamports != 1_000_000 {
		t.Errorf("lamports mismatch: got %d", a.Lamports)
	}
	if a.DataLen != uint64(len(data)) {
		t.Errorf("data_len mismatch: got %d", a.DataLen)
	}
	if !bytes.Equal(a.Data, data) {
		t.Errorf("data mismatch")
	}
	if !a.Executable {
		t.Errorf("expected executable = true")
	}
	if a.RentEpoch != 42 {
		t.Errorf("rent_epoch mismatch: got %d", a.RentEpoch)
	}

	if !bytes.Equal(gotIxData, ixData) {
		t.Errorf("instruction data mismatch")
	}
	if gotPID != programID {
		t.Errorf("program ID mismatch")
	}
}

func TestSerializeAccountsForBPF_Alignment(t *testing.T) {
	// Test various data sizes to ensure 8-byte alignment of the full buffer.
	for dataSize := 0; dataSize <= 17; dataSize++ {
		var addr, owner, programID [32]byte
		addr[0] = byte(dataSize)

		data := make([]byte, dataSize)
		for i := range data {
			data[i] = byte(i)
		}

		accounts := []SVMAccount{
			{
				Address: addr,
				DataLen: uint64(dataSize),
				Data:    data,
				Owner:   owner,
			},
		}

		metas := []AccountMeta{
			{Address: addr},
		}

		buf := SerializeAccountsForBPF(accounts, metas, nil, programID)

		// The full buffer should parse without error, which validates
		// that alignment is handled correctly internally.
		got, _, _, err := DeserializeAccountsFromBPF(buf)
		if err != nil {
			t.Errorf("dataSize=%d: roundtrip error: %v", dataSize, err)
			continue
		}
		if len(got) != 1 {
			t.Errorf("dataSize=%d: expected 1 account, got %d", dataSize, len(got))
			continue
		}
		if !bytes.Equal(got[0].Data, data) {
			t.Errorf("dataSize=%d: data mismatch", dataSize)
		}
	}
}

func TestDeserializeModifiedAccounts(t *testing.T) {
	var addr1, addr2 [32]byte
	for i := range addr1 {
		addr1[i] = byte(i + 1)
	}
	for i := range addr2 {
		addr2[i] = byte(i + 50)
	}

	data1 := []byte{0xAA, 0xBB}
	data2 := []byte{0xCC, 0xDD, 0xEE}

	// Build buffer manually.
	buf := make([]byte, 4+32+8+8+len(data1)+32+8+8+len(data2))
	offset := 0

	binary.LittleEndian.PutUint32(buf[offset:], 2)
	offset += 4

	copy(buf[offset:], addr1[:])
	offset += 32
	binary.LittleEndian.PutUint64(buf[offset:], 500)
	offset += 8
	binary.LittleEndian.PutUint64(buf[offset:], uint64(len(data1)))
	offset += 8
	copy(buf[offset:], data1)
	offset += len(data1)

	copy(buf[offset:], addr2[:])
	offset += 32
	binary.LittleEndian.PutUint64(buf[offset:], 999)
	offset += 8
	binary.LittleEndian.PutUint64(buf[offset:], uint64(len(data2)))
	offset += 8
	copy(buf[offset:], data2)

	accounts, err := DeserializeModifiedAccounts(buf)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(accounts) != 2 {
		t.Fatalf("expected 2 accounts, got %d", len(accounts))
	}

	if accounts[0].Address != addr1 {
		t.Errorf("account 0 address mismatch")
	}
	if accounts[0].Lamports != 500 {
		t.Errorf("account 0 lamports: got %d", accounts[0].Lamports)
	}
	if !bytes.Equal(accounts[0].Data, data1) {
		t.Errorf("account 0 data mismatch")
	}

	if accounts[1].Address != addr2 {
		t.Errorf("account 1 address mismatch")
	}
	if accounts[1].Lamports != 999 {
		t.Errorf("account 1 lamports: got %d", accounts[1].Lamports)
	}
	if !bytes.Equal(accounts[1].Data, data2) {
		t.Errorf("account 1 data mismatch")
	}
}

func TestDeserializeModifiedAccounts_Empty(t *testing.T) {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, 0)

	accounts, err := DeserializeModifiedAccounts(buf)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(accounts) != 0 {
		t.Fatalf("expected 0 accounts, got %d", len(accounts))
	}
}

func TestSerializeAccountsForBPF_MultipleAccounts(t *testing.T) {
	var programID [32]byte
	programID[0] = 0xFF

	accounts := make([]SVMAccount, 3)
	metas := make([]AccountMeta, 3)
	ixData := []byte{0x10, 0x20, 0x30, 0x40, 0x50}

	dataSizes := []int{0, 7, 13} // varying sizes to test alignment
	for i := 0; i < 3; i++ {
		for j := range accounts[i].Address {
			accounts[i].Address[j] = byte(i*32 + j)
		}
		for j := range accounts[i].Owner {
			accounts[i].Owner[j] = byte(i*32 + j + 128)
		}
		accounts[i].Lamports = uint64(i+1) * 1000
		accounts[i].DataLen = uint64(dataSizes[i])
		accounts[i].Data = make([]byte, dataSizes[i])
		for j := 0; j < dataSizes[i]; j++ {
			accounts[i].Data[j] = byte(j + i*10)
		}
		accounts[i].Executable = i == 1
		accounts[i].RentEpoch = uint64(i * 10)

		metas[i].Address = accounts[i].Address
		metas[i].IsSigner = i == 0
		metas[i].IsWritable = i != 2
	}

	buf := SerializeAccountsForBPF(accounts, metas, ixData, programID)

	gotAccounts, gotIxData, gotPID, err := DeserializeAccountsFromBPF(buf)
	if err != nil {
		t.Fatalf("deserialization error: %v", err)
	}

	if len(gotAccounts) != 3 {
		t.Fatalf("expected 3 accounts, got %d", len(gotAccounts))
	}

	for i := 0; i < 3; i++ {
		if gotAccounts[i].Address != accounts[i].Address {
			t.Errorf("account %d: address mismatch", i)
		}
		if gotAccounts[i].Owner != accounts[i].Owner {
			t.Errorf("account %d: owner mismatch", i)
		}
		if gotAccounts[i].Lamports != accounts[i].Lamports {
			t.Errorf("account %d: lamports mismatch", i)
		}
		if gotAccounts[i].DataLen != accounts[i].DataLen {
			t.Errorf("account %d: data_len mismatch", i)
		}
		if !bytes.Equal(gotAccounts[i].Data, accounts[i].Data) {
			t.Errorf("account %d: data mismatch", i)
		}
		if gotAccounts[i].Executable != accounts[i].Executable {
			t.Errorf("account %d: executable mismatch", i)
		}
		if gotAccounts[i].RentEpoch != accounts[i].RentEpoch {
			t.Errorf("account %d: rent_epoch mismatch", i)
		}
	}

	if !bytes.Equal(gotIxData, ixData) {
		t.Errorf("instruction data mismatch")
	}
	if gotPID != programID {
		t.Errorf("program ID mismatch")
	}
}

func TestDeserializeAccountsFromBPF_TooShort(t *testing.T) {
	// Empty buffer
	_, _, _, err := DeserializeAccountsFromBPF(nil)
	if err == nil {
		t.Error("expected error for nil buffer")
	}

	// Buffer says 1 account but has no account data
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, 1)
	_, _, _, err = DeserializeAccountsFromBPF(buf)
	if err == nil {
		t.Error("expected error for truncated account header")
	}

	// Buffer with 3 bytes (too short for count)
	_, _, _, err = DeserializeAccountsFromBPF([]byte{0x01, 0x02, 0x03})
	if err == nil {
		t.Error("expected error for 3-byte buffer")
	}
}
