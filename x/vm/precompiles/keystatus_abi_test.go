package precompiles

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// The four-value pqcKeyStatus output still decodes as the three-value one,
// and carries the possession flag for a caller that reads it.
func TestKeyStatusOutputWithPossessionKeepsABI(t *testing.T) {
	pub := bytes.Repeat([]byte{0xab}, 2592)
	v1, err := EncodePQCKeyStatusOutput(true, 1, pub)
	if err != nil {
		t.Fatal(err)
	}
	v2, err := EncodePQCKeyStatusOutputWithPossession(true, 1, pub, true)
	if err != nil {
		t.Fatal(err)
	}
	boolT, _ := abi.NewType("bool", "", nil)
	u8T, _ := abi.NewType("uint8", "", nil)
	bytesT, _ := abi.NewType("bytes", "", nil)
	three := abi.Arguments{{Type: boolT}, {Type: u8T}, {Type: bytesT}}
	for name, out := range map[string][]byte{"v1": v1, "v2": v2} {
		vals, err := three.Unpack(out)
		if err != nil {
			t.Fatalf("%s: three-value decode: %v", name, err)
		}
		if vals[0].(bool) != true || vals[1].(uint8) != 1 || !bytes.Equal(vals[2].([]byte), pub) {
			t.Fatalf("%s: three-value decode read %v %v %d bytes", name, vals[0], vals[1], len(vals[2].([]byte)))
		}
	}
	four := abi.Arguments{{Type: boolT}, {Type: u8T}, {Type: bytesT}, {Type: boolT}}
	vals, err := four.Unpack(v2)
	if err != nil || vals[3].(bool) != true {
		t.Fatalf("four-value decode: %v %v", vals, err)
	}
}
