package types

import (
	"bytes"
	"testing"
)

// The bug this replaces: the signed bytes did not name the destination key, so
// a captured migration could be replayed with a different new_public_key and
// both signatures still verified. Every input that changes what the chain will
// store must change the bytes.
func TestMigrationSignBytesBindEveryDecidingField(t *testing.T) {
	base := func() []byte {
		return MigrationSignBytes("qorechain-vladi", "qor1account", 1, 3, 4200, []byte("old-key"), []byte("new-key"))
	}
	variants := map[string][]byte{
		"chain id":    MigrationSignBytes("qorechain-diana", "qor1account", 1, 3, 4200, []byte("old-key"), []byte("new-key")),
		"account":     MigrationSignBytes("qorechain-vladi", "qor1other", 1, 3, 4200, []byte("old-key"), []byte("new-key")),
		"from algo":   MigrationSignBytes("qorechain-vladi", "qor1account", 2, 3, 4200, []byte("old-key"), []byte("new-key")),
		"to algo":     MigrationSignBytes("qorechain-vladi", "qor1account", 1, 4, 4200, []byte("old-key"), []byte("new-key")),
		"height":      MigrationSignBytes("qorechain-vladi", "qor1account", 1, 3, 4201, []byte("old-key"), []byte("new-key")),
		"old key":     MigrationSignBytes("qorechain-vladi", "qor1account", 1, 3, 4200, []byte("old-kez"), []byte("new-key")),
		"new key":     MigrationSignBytes("qorechain-vladi", "qor1account", 1, 3, 4200, []byte("old-key"), []byte("attacker")),
		"key lengths": MigrationSignBytes("qorechain-vladi", "qor1account", 1, 3, 4200, []byte("old-ke"), []byte("ynew-key")),
	}
	for name, v := range variants {
		if bytes.Equal(v, base()) {
			t.Fatalf("changing the %s left the sign bytes unchanged", name)
		}
	}
	if !bytes.HasPrefix(base(), []byte(MigrationSignBytesDomain)) {
		t.Fatal("sign bytes must start with the domain tag")
	}
	if bytes.Equal(base(), MigrationSignBytes("qorechain-vladi", "qor1account", 1, 3, 4200, []byte("old-key"), []byte("new-key"))) == false {
		t.Fatal("the encoding must be deterministic")
	}
}
