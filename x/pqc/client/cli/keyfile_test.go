//go:build full

package cli

import (
	bridgetypes "github.com/qorechain/qorechain-core/x/bridge/types"

	"bytes"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	qpqc "github.com/qorechain/qorechain-pqc/go"

	"github.com/qorechain/qorechain-core/x/pqc/types"
)

// QSR-2026-0563: writing a key never silently replaces an existing one;
// --force keeps the old key as a backup.
func TestStoreNewPQCKeyNeverOverwrites(t *testing.T) {
	home := t.TempDir()
	cmd := &cobra.Command{}
	cmd.SetErr(&bytes.Buffer{})
	if err := storeNewPQCKey(cmd, home, "k", []byte{1}, false); err != nil {
		t.Fatal(err)
	}
	if err := storeNewPQCKey(cmd, home, "k", []byte{2}, false); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("second write without --force: %v", err)
	}
	if got, _ := os.ReadFile(pqcKeyPath(home, "k")); string(got) != hex.EncodeToString([]byte{1}) {
		t.Fatalf("the original key was changed: %s", got)
	}
	if err := storeNewPQCKey(cmd, home, "k", []byte{2}, true); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(pqcKeyPath(home, "k")); string(got) != hex.EncodeToString([]byte{2}) {
		t.Fatalf("--force did not write the new key: %s", got)
	}
	backups, _ := filepath.Glob(pqcKeyPath(home, "k") + ".bak-*")
	if len(backups) != 1 {
		t.Fatalf("got backups %v, want one", backups)
	}
	if got, _ := os.ReadFile(backups[0]); string(got) != hex.EncodeToString([]byte{1}) {
		t.Fatalf("backup holds %s, want the original key", got)
	}
}

func TestGenKeyRefusesExistingName(t *testing.T) {
	fx := newCLIFixture(t)
	before, _ := os.ReadFile(pqcKeyPath(fx.home, "pq"))
	if _, err := fx.run(getCmdGenPQCKey(), "", "pq"); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("gen-key over an existing key: %v", err)
	}
	after, _ := os.ReadFile(pqcKeyPath(fx.home, "pq"))
	if !bytes.Equal(before, after) {
		t.Fatal("gen-key replaced an existing key")
	}
}

func TestRecoverKeyRefusesExistingName(t *testing.T) {
	fx := newCLIFixture(t)
	mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	if _, err := fx.run(getCmdRecoverPQCKey(), mnemonic, "pq", fx.from.String()); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("recover-key over an existing key: %v", err)
	}
	if _, err := fx.run(getCmdRecoverPQCKey(), mnemonic, "pq", fx.from.String(), "--force"); err != nil {
		t.Fatalf("recover-key --force: %v", err)
	}
}

// The proof of possession made by the CLI verifies against the registered key
// over the bytes the chain checks, and a mismatched key file is refused.
func TestProofOfPossession(t *testing.T) {
	fx := newCLIFixture(t)
	sender := fx.from.String()
	sig, err := proofOfPossession(fx.home, "pq", testChainID, types.AlgorithmDilithium5, sender, fx.pqcPub)
	if err != nil {
		t.Fatal(err)
	}
	bz := types.ProofOfPossessionSignBytes(testChainID, uint32(types.AlgorithmDilithium5), sender, fx.pqcPub)
	if !qpqc.MLDSA87.Verify(fx.pqcPub, bz, sig) {
		t.Fatal("the proof does not verify")
	}
	otherPub, _, err := qpqc.MLDSA87.Keygen()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := proofOfPossession(fx.home, "pq", testChainID, types.AlgorithmDilithium5, sender, otherPub); err == nil {
		t.Fatal("a key file that does not match the registered key was accepted")
	}
}

func TestRotateKeyRefusesBridgeTarget(t *testing.T) {
	fx := newCLIFixture(t)
	_, err := fx.run(getCmdRotatePQCKey(), "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about",
		"--from", fx.keyName, "--old-derivation", "adapter", "--new-derivation", "bridge", "--chain-id", testChainID, "--offline")
	if err == nil || !strings.Contains(err.Error(), "deprecated bridge") {
		t.Fatalf("got %v", err)
	}
}

// The bridge validator registration signs its proof of possession through
// SignProofOfPossession, over the bytes the bridge module checks.
func TestBridgeValidatorProofOfPossession(t *testing.T) {
	fx := newCLIFixture(t)
	validator := fx.from.String()
	sig, err := SignProofOfPossession(fx.home, "pq", testChainID, validator, fx.pqcPub)
	if err != nil {
		t.Fatal(err)
	}
	if !qpqc.MLDSA87.Verify(fx.pqcPub, bridgetypes.ValidatorKeyPoPSignBytes(testChainID, validator, fx.pqcPub), sig) {
		t.Fatal("the bridge proof does not verify over the bridge sign bytes")
	}
}
