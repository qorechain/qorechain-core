package precompiles

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

func TestAddressesNotZero(t *testing.T) {
	addresses := []struct {
		name string
		addr common.Address
	}{
		{"CrossVMBridge", CrossVMBridgeAddress},
		{"PQCVerify", PQCVerifyAddress},
		{"PQCKeyStatus", PQCKeyStatusAddress},
		{"AIRiskScore", AIRiskScoreAddress},
		{"AIAnomalyCheck", AIAnomalyCheckAddress},
		{"RLConsensusParams", RLConsensusParamsAddress},
	}
	for _, tc := range addresses {
		if tc.addr == (common.Address{}) {
			t.Errorf("%s address is zero", tc.name)
		}
	}
}

func TestAddressesUnique(t *testing.T) {
	seen := make(map[common.Address]string)
	addresses := []struct {
		name string
		addr common.Address
	}{
		{"CrossVMBridge", CrossVMBridgeAddress},
		{"PQCVerify", PQCVerifyAddress},
		{"PQCKeyStatus", PQCKeyStatusAddress},
		{"AIRiskScore", AIRiskScoreAddress},
		{"AIAnomalyCheck", AIAnomalyCheckAddress},
		{"RLConsensusParams", RLConsensusParamsAddress},
	}
	for _, tc := range addresses {
		if prev, ok := seen[tc.addr]; ok {
			t.Errorf("duplicate address %s between %s and %s", tc.addr.Hex(), prev, tc.name)
		}
		seen[tc.addr] = tc.name
	}
}

func TestSelectorsComputed(t *testing.T) {
	zero := [4]byte{}
	selectors := []struct {
		name string
		sel  [4]byte
	}{
		{"PQCVerify", SelectorPQCVerify},
		{"PQCKeyStatus", SelectorPQCKeyStatus},
		{"AIRiskScore", SelectorAIRiskScore},
		{"AIAnomalyCheck", SelectorAIAnomalyCheck},
		{"RLConsensusParams", SelectorRLConsensusParams},
		{"CrossVMCall", SelectorCrossVMCall},
	}
	for _, tc := range selectors {
		if tc.sel == zero {
			t.Errorf("Selector%s is zero", tc.name)
		}
	}
}

func TestSelectorsUnique(t *testing.T) {
	seen := make(map[[4]byte]string)
	selectors := []struct {
		name string
		sel  [4]byte
	}{
		{"PQCVerify", SelectorPQCVerify},
		{"PQCKeyStatus", SelectorPQCKeyStatus},
		{"AIRiskScore", SelectorAIRiskScore},
		{"AIAnomalyCheck", SelectorAIAnomalyCheck},
		{"RLConsensusParams", SelectorRLConsensusParams},
		{"CrossVMCall", SelectorCrossVMCall},
	}
	for _, tc := range selectors {
		if prev, ok := seen[tc.sel]; ok {
			t.Errorf("duplicate selector between %s and %s", prev, tc.name)
		}
		seen[tc.sel] = tc.name
	}
}

func TestPQCVerifyABIRoundTrip(t *testing.T) {
	pubkey := []byte("test-pubkey-data")
	sig := []byte("test-sig-data")
	msg := []byte("hello world")

	// Build input: selector + ABI-encoded args
	bytesType, _ := abi.NewType("bytes", "", nil)
	arguments := abi.Arguments{
		{Type: bytesType}, {Type: bytesType}, {Type: bytesType},
	}
	encoded, err := arguments.Pack(pubkey, sig, msg)
	if err != nil {
		t.Fatal(err)
	}
	fullInput := append(SelectorPQCVerify[:], encoded...)

	gotPub, gotSig, gotMsg, err := DecodePQCVerifyInput(fullInput)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotPub) != string(pubkey) || string(gotSig) != string(sig) || string(gotMsg) != string(msg) {
		t.Error("round-trip mismatch")
	}

	// Test output encoding
	out, err := EncodePQCVerifyOutput(true)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 32 { // bool is padded to 32 bytes
		t.Errorf("expected 32 bytes, got %d", len(out))
	}
}

func TestPQCKeyStatusABIRoundTrip(t *testing.T) {
	addr := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")

	// Build input
	addressType, _ := abi.NewType("address", "", nil)
	arguments := abi.Arguments{{Type: addressType}}
	encoded, err := arguments.Pack(addr)
	if err != nil {
		t.Fatal(err)
	}
	fullInput := append(SelectorPQCKeyStatus[:], encoded...)

	gotAddr, err := DecodePQCKeyStatusInput(fullInput)
	if err != nil {
		t.Fatal(err)
	}
	if gotAddr != addr {
		t.Errorf("expected %s, got %s", addr.Hex(), gotAddr.Hex())
	}

	// Test output encoding
	out, err := EncodePQCKeyStatusOutput(true, 1, []byte("pubkey-data"))
	if err != nil {
		t.Fatal(err)
	}
	if len(out) == 0 {
		t.Error("output is empty")
	}
}

func TestAIRiskScoreABIRoundTrip(t *testing.T) {
	txData := []byte("test-tx-data")

	bytesType, _ := abi.NewType("bytes", "", nil)
	arguments := abi.Arguments{{Type: bytesType}}
	encoded, err := arguments.Pack(txData)
	if err != nil {
		t.Fatal(err)
	}
	fullInput := append(SelectorAIRiskScore[:], encoded...)

	gotData, err := DecodeAIRiskScoreInput(fullInput)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotData) != string(txData) {
		t.Error("round-trip mismatch")
	}

	// Test output
	score := big.NewInt(5000)
	level := uint8(2)
	out, err := EncodeAIRiskScoreOutput(score, level)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 64 { // uint256 + uint8 padded
		t.Errorf("expected 64 bytes, got %d", len(out))
	}
}

func TestAIAnomalyCheckABIRoundTrip(t *testing.T) {
	addr := common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	amount := big.NewInt(1000000)

	addressType, _ := abi.NewType("address", "", nil)
	uint256Type, _ := abi.NewType("uint256", "", nil)
	arguments := abi.Arguments{
		{Type: addressType},
		{Type: uint256Type},
	}
	encoded, err := arguments.Pack(addr, amount)
	if err != nil {
		t.Fatal(err)
	}
	fullInput := append(SelectorAIAnomalyCheck[:], encoded...)

	gotAddr, gotAmount, err := DecodeAIAnomalyCheckInput(fullInput)
	if err != nil {
		t.Fatal(err)
	}
	if gotAddr != addr {
		t.Errorf("address mismatch: expected %s, got %s", addr.Hex(), gotAddr.Hex())
	}
	if gotAmount.Cmp(amount) != 0 {
		t.Errorf("amount mismatch: expected %s, got %s", amount.String(), gotAmount.String())
	}

	// Test output
	out, err := EncodeAIAnomalyCheckOutput(big.NewInt(7500), true)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 64 { // uint256 + bool padded
		t.Errorf("expected 64 bytes, got %d", len(out))
	}
}

func TestCrossVMCallABIRoundTrip(t *testing.T) {
	targetVM := uint8(1) // CosmWasm
	targetContract := "qor14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9sr4k5yd"
	payload := []byte(`{"execute":{"amount":"100"}}`)

	uint8Type, _ := abi.NewType("uint8", "", nil)
	stringType, _ := abi.NewType("string", "", nil)
	bytesType, _ := abi.NewType("bytes", "", nil)
	arguments := abi.Arguments{
		{Type: uint8Type},
		{Type: stringType},
		{Type: bytesType},
	}
	encoded, err := arguments.Pack(targetVM, targetContract, payload)
	if err != nil {
		t.Fatal(err)
	}
	fullInput := append(SelectorCrossVMCall[:], encoded...)

	gotVM, gotContract, gotPayload, err := DecodeCrossVMCallInput(fullInput)
	if err != nil {
		t.Fatal(err)
	}
	if gotVM != targetVM {
		t.Errorf("targetVM mismatch: expected %d, got %d", targetVM, gotVM)
	}
	if gotContract != targetContract {
		t.Errorf("targetContract mismatch")
	}
	if string(gotPayload) != string(payload) {
		t.Errorf("payload mismatch")
	}
}

func TestRLConsensusParamsABIOutput(t *testing.T) {
	out, err := EncodeRLConsensusParamsOutput(
		big.NewInt(5000), // blockTime ms
		big.NewInt(100),  // baseGasPrice
		big.NewInt(100),  // validatorSetSize
		big.NewInt(0),    // epoch
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 128 { // 4 x uint256
		t.Errorf("expected 128 bytes, got %d", len(out))
	}
}

func TestDecodeShortInput(t *testing.T) {
	short := []byte{0x01, 0x02}

	_, _, _, err := DecodePQCVerifyInput(short)
	if err == nil {
		t.Error("expected error for short input")
	}

	_, err = DecodePQCKeyStatusInput(short)
	if err == nil {
		t.Error("expected error for short input")
	}

	_, err = DecodeAIRiskScoreInput(short)
	if err == nil {
		t.Error("expected error for short input")
	}

	_, _, err = DecodeAIAnomalyCheckInput(short)
	if err == nil {
		t.Error("expected error for short input")
	}

	_, _, _, err = DecodeCrossVMCallInput(short)
	if err == nil {
		t.Error("expected error for short input")
	}
}
