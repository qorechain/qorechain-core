package types

import (
	"encoding/binary"
	"testing"
)

func TestNodeFaultRecoveryHandler(t *testing.T) {
	if err := NodeFaultRecoveryHandler("other"); err != nil {
		t.Fatalf("an ordinary panic was taken: %v", err)
	}
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("NodeFault was not re-raised")
		} else if _, ok := r.(NodeFault); !ok {
			t.Fatalf("re-raised %T", r)
		}
	}()
	_ = NodeFaultRecoveryHandler(NodeFault{Reason: "x"})
}

func TestNativeAllocation(t *testing.T) {
	create := binary.LittleEndian.AppendUint32(nil, 0)
	create = binary.LittleEndian.AppendUint64(create, 1)
	create = binary.LittleEndian.AppendUint64(create, 100_000)
	if got := NativeAllocation(SystemProgramAddress, create); got != 100_000 {
		t.Fatalf("CreateAccount space %d", got)
	}
	big := binary.LittleEndian.AppendUint32(nil, 8)
	big = binary.LittleEndian.AppendUint64(big, 1<<40)
	if got := NativeAllocation(SystemProgramAddress, big); got != MaxNativeAllocation {
		t.Fatalf("Allocate capped at %d", got)
	}
	if NativeAllocation(SPLTokenAddress, create) != 0 || NativeAllocation(SystemProgramAddress, []byte{0}) != 0 {
		t.Fatal("non-System or short data charged")
	}
}

func TestNativeProgramIDs(t *testing.T) {
	for _, a := range [][32]byte{SystemProgramAddress, SPLTokenAddress, ATAAddress, MemoAddress, QorPQCAddress,
		QorAIAddress, BPFLoaderAddress, mustBase58Decode("SysvarC1ock11111111111111111111111111111111"),
		mustBase58Decode("metaqbxxUerdq28cj1RbAWkYQm3ybzjb6a8bt518x1s")} {
		if !IsNativeProgramID(a) {
			t.Errorf("%s not a native id", Base58Encode(a))
		}
	}
	var wallet [32]byte
	wallet[0] = 1
	if IsNativeProgramID(wallet) {
		t.Error("a wallet address is a native id")
	}
}

func TestGenesisPQCKeyLength(t *testing.T) {
	gs := DefaultGenesis()
	var a [32]byte
	a[0] = 9
	gs.PQCKeys = []PQCKeyEntry{{Address: a, Key: make([]byte, SVMPQCPublicKeyLen)}}
	if err := gs.Validate(); err != nil {
		t.Fatalf("a %d-byte key refused: %v", SVMPQCPublicKeyLen, err)
	}
	for _, n := range []int{0, 1, SVMPQCPublicKeyLen - 1, SVMPQCPublicKeyLen + 1} {
		gs.PQCKeys[0].Key = make([]byte, n)
		if gs.Validate() == nil {
			t.Errorf("a %d-byte key validated", n)
		}
	}
}

func TestParamsHardCeilingValidation(t *testing.T) {
	p := DefaultParams()
	p.MaxCustodyUqor, p.MaxCustodyHardUqor = 10, 9
	if p.Validate() == nil {
		t.Fatal("ceiling below the threshold validated")
	}
	p.MaxCustodyHardUqor = 0
	if err := p.Validate(); err != nil {
		t.Fatalf("no ceiling refused: %v", err)
	}
	p.MaxCustodyHardUqor = 1000
	if err := p.Validate(); err != nil {
		t.Fatalf("ceiling above the threshold refused: %v", err)
	}
}

func TestAllRuntimeFlags(t *testing.T) {
	if AllRuntimeFlags != 1<<27-1 {
		t.Fatalf("AllRuntimeFlags %#x does not name every flag 1<<0..1<<26", uint64(AllRuntimeFlags))
	}
	if err := CheckRuntimeFlags(AllRuntimeFlags); err != nil {
		t.Fatal(err)
	}
	if err := CheckRuntimeFlags(AllRuntimeFlags &^ RuntimeFlagRegistrySharedCharge); err == nil {
		t.Fatal("a library without 1<<26 was accepted")
	}
}
