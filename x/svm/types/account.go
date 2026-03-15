package types

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	errorsmod "cosmossdk.io/errors"
)

// SVMAccount represents an account in the SVM runtime, mirroring the
// account model used by SVM-compatible virtual machines.
type SVMAccount struct {
	Address    [32]byte `json:"address"`
	Lamports   uint64   `json:"lamports"`
	DataLen    uint64   `json:"data_len"`
	Data       []byte   `json:"data"`
	Owner      [32]byte `json:"owner"`
	Executable bool     `json:"executable"`
	RentEpoch  uint64   `json:"rent_epoch"`
}

// Validate checks the SVMAccount for internal consistency.
func (a *SVMAccount) Validate() error {
	var zeroAddr [32]byte
	if a.Address == zeroAddr {
		return errorsmod.Wrap(ErrInvalidAddress, "SVM account address cannot be zero")
	}
	if uint64(len(a.Data)) != a.DataLen {
		return errorsmod.Wrapf(ErrInvalidAccountOwner, "data length mismatch: declared %d, actual %d", a.DataLen, len(a.Data))
	}
	if a.Executable && a.Owner == zeroAddr {
		return errorsmod.Wrap(ErrInvalidAccountOwner, "executable account must have non-zero owner")
	}
	return nil
}

// Marshal serializes the SVM account to bytes.
// TODO: replace JSON encoding with binary (protobuf or manual) before production use.
func (a *SVMAccount) Marshal() ([]byte, error) {
	return json.Marshal(a)
}

// Unmarshal deserializes the SVMAccount from JSON bytes.
func (a *SVMAccount) Unmarshal(data []byte) error {
	return json.Unmarshal(data, a)
}

// boolByte returns 1 if v is true, 0 otherwise.
func boolByte(v bool) byte {
	if v {
		return 1
	}
	return 0
}

// alignPadding returns the number of padding bytes needed to align dataLen to 8 bytes.
func alignPadding(dataLen uint64) int {
	return int((8 - (dataLen % 8)) % 8)
}

// SerializeAccountsForBPF serializes accounts, instruction data, and program ID
// into the Solana-compatible binary format expected by BPF programs.
//
// Layout:
//
//	num_accounts (u64 LE)
//	per account: is_signer(u8) is_writable(u8) executable(u8) padding(5) key(32) owner(32) lamports(u64) data_len(u64) data(var) padding(align8) rent_epoch(u64)
//	instruction_data_len (u64 LE)
//	instruction_data (var)
//	program_id (32)
func SerializeAccountsForBPF(
	accounts []SVMAccount,
	metas []AccountMeta,
	instructionData []byte,
	programID [32]byte,
) []byte {
	// Pre-calculate total size.
	// Header: 8 bytes for num_accounts
	// Per account: 1+1+1+5 + 32+32 + 8+8 + data_len + padding + 8 = 96 + data_len + padding
	// Footer: 8 + len(instructionData) + 32
	size := 8 // num_accounts
	for _, acct := range accounts {
		size += 8 + 32 + 32 + 8 + 8 + int(acct.DataLen) + alignPadding(acct.DataLen) + 8
	}
	size += 8 + len(instructionData) + 32

	buf := make([]byte, size)
	offset := 0

	// num_accounts
	binary.LittleEndian.PutUint64(buf[offset:], uint64(len(accounts)))
	offset += 8

	for i, acct := range accounts {
		var isSigner, isWritable bool
		if i < len(metas) {
			isSigner = metas[i].IsSigner
			isWritable = metas[i].IsWritable
		}

		buf[offset] = boolByte(isSigner)
		buf[offset+1] = boolByte(isWritable)
		buf[offset+2] = boolByte(acct.Executable)
		// offset+3..offset+7 are padding (already zero)
		offset += 8

		copy(buf[offset:], acct.Address[:])
		offset += 32

		copy(buf[offset:], acct.Owner[:])
		offset += 32

		binary.LittleEndian.PutUint64(buf[offset:], acct.Lamports)
		offset += 8

		binary.LittleEndian.PutUint64(buf[offset:], acct.DataLen)
		offset += 8

		copy(buf[offset:], acct.Data)
		offset += int(acct.DataLen)

		pad := alignPadding(acct.DataLen)
		offset += pad // already zeroed

		binary.LittleEndian.PutUint64(buf[offset:], acct.RentEpoch)
		offset += 8
	}

	// instruction data
	binary.LittleEndian.PutUint64(buf[offset:], uint64(len(instructionData)))
	offset += 8

	copy(buf[offset:], instructionData)
	offset += len(instructionData)

	// program ID
	copy(buf[offset:], programID[:])

	return buf
}

// DeserializeAccountsFromBPF parses accounts back from the BPF input buffer.
// Returns the accounts, instruction data, program ID, and any error.
func DeserializeAccountsFromBPF(buf []byte) ([]SVMAccount, []byte, [32]byte, error) {
	var zeroPID [32]byte

	if len(buf) < 8 {
		return nil, nil, zeroPID, fmt.Errorf("buffer too short for account count: need 8, got %d", len(buf))
	}

	numAccounts := binary.LittleEndian.Uint64(buf[0:8])
	offset := 8

	accounts := make([]SVMAccount, numAccounts)

	for i := uint64(0); i < numAccounts; i++ {
		// Minimum per-account bytes before data: 8 + 32 + 32 + 8 + 8 = 88
		if offset+88 > len(buf) {
			return nil, nil, zeroPID, fmt.Errorf("buffer too short at account %d header: need %d, got %d", i, offset+88, len(buf))
		}

		// is_signer(1) is_writable(1) executable(1) padding(5)
		accounts[i].Executable = buf[offset+2] != 0
		offset += 8

		copy(accounts[i].Address[:], buf[offset:offset+32])
		offset += 32

		copy(accounts[i].Owner[:], buf[offset:offset+32])
		offset += 32

		accounts[i].Lamports = binary.LittleEndian.Uint64(buf[offset:])
		offset += 8

		dataLen := binary.LittleEndian.Uint64(buf[offset:])
		accounts[i].DataLen = dataLen
		offset += 8

		pad := alignPadding(dataLen)
		needed := int(dataLen) + pad + 8 // data + padding + rent_epoch
		if offset+needed > len(buf) {
			return nil, nil, zeroPID, fmt.Errorf("buffer too short at account %d data: need %d more, got %d", i, needed, len(buf)-offset)
		}

		accounts[i].Data = make([]byte, dataLen)
		copy(accounts[i].Data, buf[offset:offset+int(dataLen)])
		offset += int(dataLen) + pad

		accounts[i].RentEpoch = binary.LittleEndian.Uint64(buf[offset:])
		offset += 8
	}

	// instruction data
	if offset+8 > len(buf) {
		return nil, nil, zeroPID, fmt.Errorf("buffer too short for instruction data length at offset %d", offset)
	}
	ixLen := binary.LittleEndian.Uint64(buf[offset:])
	offset += 8

	if offset+int(ixLen)+32 > len(buf) {
		return nil, nil, zeroPID, fmt.Errorf("buffer too short for instruction data + program ID")
	}

	ixData := make([]byte, ixLen)
	copy(ixData, buf[offset:offset+int(ixLen)])
	offset += int(ixLen)

	var programID [32]byte
	copy(programID[:], buf[offset:offset+32])

	return accounts, ixData, programID, nil
}

// DeserializeModifiedAccounts parses modified accounts from the result buffer
// returned by the Rust execution engine.
//
// Layout: count(u32 LE), then per account: address(32) lamports(u64) data_len(u64) data(var)
func DeserializeModifiedAccounts(buf []byte) ([]SVMAccount, error) {
	if len(buf) < 4 {
		return nil, fmt.Errorf("buffer too short for modified accounts count: need 4, got %d", len(buf))
	}

	count := binary.LittleEndian.Uint32(buf[0:4])
	offset := 4

	accounts := make([]SVMAccount, count)

	for i := uint32(0); i < count; i++ {
		// address(32) + lamports(8) + data_len(8) = 48
		if offset+48 > len(buf) {
			return nil, fmt.Errorf("buffer too short at modified account %d header", i)
		}

		copy(accounts[i].Address[:], buf[offset:offset+32])
		offset += 32

		accounts[i].Lamports = binary.LittleEndian.Uint64(buf[offset:])
		offset += 8

		dataLen := binary.LittleEndian.Uint64(buf[offset:])
		accounts[i].DataLen = dataLen
		offset += 8

		if offset+int(dataLen) > len(buf) {
			return nil, fmt.Errorf("buffer too short at modified account %d data", i)
		}

		accounts[i].Data = make([]byte, dataLen)
		copy(accounts[i].Data, buf[offset:offset+int(dataLen)])
		offset += int(dataLen)
	}

	return accounts, nil
}
