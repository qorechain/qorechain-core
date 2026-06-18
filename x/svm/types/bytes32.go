package types

import "fmt"

// Bytes32 is a fixed 32-byte value usable as a gogoproto customtype. It keeps
// SVM addresses / program IDs as fixed arrays on the wire while satisfying the
// gogoproto custom-type marshaling interface.
type Bytes32 [32]byte

// Marshal returns the raw 32 bytes.
func (b Bytes32) Marshal() ([]byte, error) {
	return b[:], nil
}

// MarshalTo writes the 32 bytes into data and returns the count.
func (b Bytes32) MarshalTo(data []byte) (int, error) {
	return copy(data, b[:]), nil
}

// Unmarshal copies up to 32 bytes from data.
func (b *Bytes32) Unmarshal(data []byte) error {
	if len(data) > 32 {
		return fmt.Errorf("Bytes32: too many bytes: %d", len(data))
	}
	var z Bytes32
	*b = z
	copy(b[:], data)
	return nil
}

// Size is always 32.
func (b Bytes32) Size() int {
	return 32
}
