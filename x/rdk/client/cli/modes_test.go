package cli

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudflare/circl/kem/mlkem/mlkem1024"

	"github.com/qorechain/qorechain-core/x/rdk/privpost"
)

// decrypt-private-data opens an envelope offline with a member's key file
// (hex seed), and refuses another chain id.
func TestDecryptPrivateDataCommand(t *testing.T) {
	seed := bytes.Repeat([]byte{5}, mlkem1024.KeySeedSize)
	pk, _ := mlkem1024.NewKeyFromSeed(seed)
	ek, _ := pk.MarshalBinary()
	env, err := privpost.Seal(rand.Reader, "qorechain-diana", "priv-roll", 2,
		[]privpost.Recipient{{Member: "qor1member", EncapsulationKey: ek}}, []byte("hello members"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	keyFile := filepath.Join(dir, "member.kem")
	if err := os.WriteFile(keyFile, []byte(hex.EncodeToString(seed)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	envFile := filepath.Join(dir, "post.bin")
	_ = os.WriteFile(envFile, env, 0o600)

	run := func(chain string) (string, error) {
		cmd := cmdDecryptPrivateData()
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetArgs([]string{"priv-roll", "@" + envFile, "--member", "qor1member", "--kem-key-file", keyFile, "--chain-id", chain})
		err := cmd.Execute()
		return out.String(), err
	}
	got, err := run("qorechain-diana")
	if err != nil || got != "hello members" {
		t.Fatalf("decrypt: %q %v", got, err)
	}
	if _, err := run("qorechain-vladi"); err == nil {
		t.Fatal("opened with another chain id")
	}
}
