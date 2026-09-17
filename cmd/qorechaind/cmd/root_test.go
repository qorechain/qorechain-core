package cmd

import "testing"

// THE STARTUP PATH. NewRootCmd runs depinject, which builds the interface
// registry and validates the signing context: every message with a signer
// field, every custom signer, every module registration. Nothing else in the
// test suite exercises that path, which is how a binary that panicked on its
// first line reached the release build host with 44 green packages behind it
// (a custom signer registered for a message whose proto already declares a
// signer field). This test is the line that stops it: if NewRootCmd panics,
// no binary can start, and the suite must say so before anything is built.
func TestRootCommandConstructs(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("NewRootCmd panicked; the binary would not start: %v", r)
		}
	}()
	root := NewRootCmd()
	if root == nil {
		t.Fatal("NewRootCmd returned nil")
	}
	for _, name := range []string{"start", "genesis", "query", "tx", "keys"} {
		if c, _, err := root.Find([]string{name}); err != nil || c == nil || c.Name() != name {
			t.Fatalf("root command lacks %q", name)
		}
	}
}
