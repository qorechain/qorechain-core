package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/cosmos/cosmos-sdk/client/flags"
)

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

// TestEVMSendGenerateOnly pins QSR-2026-0267: the EVM module basic registered
// on the root command carries an address codec, so `tx evm send` builds a
// MsgSend instead of panicking on a nil codec.
func TestEVMSendGenerateOnly(t *testing.T) {
	root := NewRootCmd()
	// svrcmd.Execute adds --home in main; mirror it for the test.
	root.PersistentFlags().String(flags.FlagHome, t.TempDir(), "node home")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"tx", "evm", "send",
		"0x7cB61D4117AE31a12E393a1Cfa3BaC666481D02E",
		"0xA2A8B87390F8F2D188242656BFb6852914073D06",
		"10uqor",
		"--generate-only", "--chain-id", "qorechain-test",
		"--keyring-backend", "test",
	})
	stdout := captureStdout(t, func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("tx evm send panicked: %v", r)
			}
		}()
		if err := root.Execute(); err != nil {
			t.Fatalf("tx evm send: %v\n%s", err, out.String())
		}
	})
	all := stdout + out.String()
	if !strings.Contains(all, "MsgSend") {
		t.Fatalf("no MsgSend in the generated tx:\n%s", all)
	}
}

// captureStdout runs f with os.Stdout redirected and returns what it wrote;
// the generate-only path prints there directly.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	os.Stdout = w
	done := make(chan string)
	go func() {
		bz, _ := io.ReadAll(r)
		done <- string(bz)
	}()
	func() {
		defer func() { os.Stdout = orig }()
		f()
	}()
	w.Close()
	return <-done
}
