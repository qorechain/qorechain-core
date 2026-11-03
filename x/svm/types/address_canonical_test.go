package types

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCanonicalSVMToCosmos(t *testing.T) {
	native := []byte("canonical-native-acc")
	canonical := CosmosToSVMAddress(native)

	got, err := CanonicalSVMToCosmos(canonical)
	if err != nil || !bytes.Equal(got, native) {
		t.Fatalf("canonical address: got %x, %v; want %x", got, err, native)
	}

	// Any non-zero byte in the tail makes the address non-canonical: its first
	// 20 bytes are someone else's account.
	for i := 20; i < 32; i++ {
		alias := canonical
		alias[i] = 1
		if got, err := CanonicalSVMToCosmos(alias); err == nil {
			t.Fatalf("tail byte %d: non-canonical address mapped to %x", i, got)
		}
	}
	// The truncating function still truncates (it is kept for replay).
	alias := canonical
	alias[31] = 0xAA
	if !bytes.Equal(SVMToCosmosAddress(alias), native) {
		t.Fatal("SVMToCosmosAddress changed; blocks before the v3.2.7 height depend on it")
	}
}

func TestCanonicalCosmosToSVM(t *testing.T) {
	native := []byte("canonical-native-acc")
	got, err := CanonicalCosmosToSVM(native)
	if err != nil || got != CosmosToSVMAddress(native) {
		t.Fatalf("20-byte account: got %x, %v", got, err)
	}
	for _, n := range []int{0, 19, 21, 32} {
		if _, err := CanonicalCosmosToSVM(make([]byte, n)); err == nil {
			t.Fatalf("a %d-byte native address got an SVM address", n)
		}
	}
	// What the checked form refuses: a 32-byte account would be cut to the
	// 20-byte account that shares its prefix.
	wide := append(append([]byte{}, native...), bytes.Repeat([]byte{7}, 12)...)
	if CosmosToSVMAddress(wide) != CosmosToSVMAddress(native) {
		t.Fatal("CosmosToSVMAddress changed; blocks before the v3.2.7 height depend on it")
	}
}

// lossyUseAllowlist is every non-test call of the truncating conversions. Each
// one is either a replay branch below a v3.2.5/v3.2.7 switch or runs only after
// a canonical check. A new call must be added here on purpose, after reading
// testnet/handoff-v327/svm-address-audit.md; prefer CanonicalSVMToCosmos.
var lossyUseAllowlist = map[string]int{
	// core
	"x/svm/types/address.go": 3, // the definitions and CanonicalSVMToCosmos
	// proprietary overlay (scanned when QORE_HANDLER_ROOTS names it)
	"x/svm/keeper/balance.go":    3, // GetNativeLamports and credit/debit, behind svm-native-read-canonical
	"x/svm/keeper/accounts.go":   3, // SetAccount legacy branch, DeleteAccount
	"x/svm/keeper/msg_server.go": 1, // MsgRegisterSVMPQCKey replay branch
}

func countLossyUses(t *testing.T, root string) map[string]int {
	t.Helper()
	out := map[string]int{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "vendor", "deps", "build":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		bz, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		n := 0
		for _, line := range strings.Split(string(bz), "\n") {
			code := line
			if i := strings.Index(code, "//"); i >= 0 {
				code = code[:i]
			}
			n += strings.Count(code, "SVMToCosmosAddress(") + strings.Count(code, "SVMToEVMAddress(")
		}
		if n > 0 {
			rel, _ := filepath.Rel(root, p)
			out[filepath.ToSlash(rel)] = n
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// TestNoNewLossyAddressConversions keeps the truncating conversions from
// spreading: a 32-byte SVM address must not become a 20-byte account by
// keeping its prefix. It scans this repository and, when QORE_HANDLER_ROOTS
// names the proprietary repository (make test sets it), that one too.
func TestNoNewLossyAddressConversions(t *testing.T) {
	roots := []string{filepath.Join("..", "..", "..")}
	if r := os.Getenv("QORE_HANDLER_ROOTS"); r != "" {
		roots = append(roots, r)
	}
	for _, root := range roots {
		for file, n := range countLossyUses(t, root) {
			if max, ok := lossyUseAllowlist[file]; !ok || n > max {
				t.Errorf("%s: %d call(s) of a truncating SVM address conversion (allowed %d); "+
					"use CanonicalSVMToCosmos / CanonicalCosmosToSVM", filepath.Join(root, file), n, max)
			}
		}
	}
}
