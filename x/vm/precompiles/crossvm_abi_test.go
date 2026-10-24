package precompiles

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// The published ABI of the CrossVM bridge (contracts/abi/IQoreCrossVM.json) is
// the one the code speaks: its selector is the one the precompile accepts, an
// input packed from the JSON decodes with DecodeCrossVMCallInput, and the
// output EncodeCrossVMCallOutput produces unpacks with the JSON's outputs. If
// either side changes alone, this fails.
func TestCrossVMBridgePublishedABIMatchesCode(t *testing.T) {
	raw, err := os.ReadFile("../../../contracts/abi/IQoreCrossVM.json")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := abi.JSON(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Methods) != 1 {
		t.Fatalf("the published ABI has %d methods, want 1", len(parsed.Methods))
	}
	m, ok := parsed.Methods["executeCrossVMCall"]
	if !ok {
		t.Fatal("executeCrossVMCall is missing from the published ABI")
	}
	if m.Sig != "executeCrossVMCall(uint8,string,bytes)" {
		t.Fatalf("published signature %q", m.Sig)
	}
	if !bytes.Equal(m.ID, SelectorCrossVMCall[:]) {
		t.Fatalf("published selector %x, the precompile accepts %x", m.ID, SelectorCrossVMCall)
	}
	// The two-argument form some clients used is a different selector, so the
	// precompile can refuse it instead of misreading its arguments.
	if two := computeSelector("executeCrossVMCall(string,bytes)"); bytes.Equal(two[:], SelectorCrossVMCall[:]) {
		t.Fatal("the two-argument selector collides with the canonical one")
	}

	target := "qor1" + strings.Repeat("q", 58)
	payload := []byte(`{"transfer":{"recipient":"qor1x","amount":"1"}}`)
	input, err := parsed.Pack("executeCrossVMCall", uint8(1), target, payload)
	if err != nil {
		t.Fatal(err)
	}
	vmType, gotTarget, gotPayload, err := DecodeCrossVMCallInput(input)
	if err != nil {
		t.Fatal(err)
	}
	if vmType != 1 || gotTarget != target || !bytes.Equal(gotPayload, payload) {
		t.Fatalf("decoded (%d, %q, %q)", vmType, gotTarget, gotPayload)
	}

	answer := []byte(`{"ok":true}`)
	out, err := EncodeCrossVMCallOutput(answer)
	if err != nil {
		t.Fatal(err)
	}
	vals, err := parsed.Unpack("executeCrossVMCall", out)
	if err != nil {
		t.Fatal(err)
	}
	if len(vals) != 1 || !bytes.Equal(vals[0].([]byte), answer) {
		t.Fatalf("unpacked %v", vals)
	}
}
