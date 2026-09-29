package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// Fixed inputs for the golden vectors. The signatures were produced by the
// sign-byte functions before the seed moved off argv, so a match proves the
// signed bytes did not change.
const (
	goldenSeed    = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"
	goldenAccount = "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a"
	goldenPub     = "A6EHv/POEL4dcN0Y50vAmWfk1jCbpQ1fHdyGZBJVMbg="
	goldenCosmos  = "GbX7gHBCm0ORLOnbVsEuvj87B99HR0Dkl1gNJb9fNwgErO6+t0XLDj46yzCafrcP69OqpsWEf3llUvpCX9SwCg=="
	goldenEVM     = "oI0L+2CGn/Ud6FvKfWLuaMuCdKmEAaPB+vAWVroo0RAgLlu5gm69dHM5mnsINKY2TRPGT7F6DHwLnF2FB2qsAg=="
)

func runCmd(t *testing.T, cmd *cobra.Command, stdin string, args ...string) (string, string, error) {
	t.Helper()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetArgs(args)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	err := cmd.Execute()
	return out.String(), errOut.String(), err
}

func writeFileMode(t *testing.T, name, content string, mode os.FileMode) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p, mode); err != nil {
		t.Fatal(err)
	}
	return p
}

var cosmosArgs = []string{"qorechain-diana", goldenAccount, goldenAccount, "100uqor", "3"}
var evmArgs = []string{"qorechain-diana", goldenAccount, "0x000000000000000000000000000000000000dEaD", "1000", "0xa9059cbb", "4"}

func TestAuthSignGoldenFromSeedFile(t *testing.T) {
	seedFile := writeFileMode(t, "seed", goldenSeed+"\n", 0o600)

	out, _, err := runCmd(t, CmdAuthSignCosmos(), "", append(cosmosArgs, "--seed-file", seedFile)...)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "pubkey_b64: "+goldenPub) || !strings.Contains(out, "signature_b64: "+goldenCosmos) {
		t.Fatalf("cosmos output does not match the golden vector:\n%s", out)
	}

	out, _, err = runCmd(t, CmdAuthSignEVM(), "", append(evmArgs, "--seed-file", seedFile)...)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "signature_b64: "+goldenEVM) {
		t.Fatalf("evm output does not match the golden vector:\n%s", out)
	}
	if strings.Contains(out, goldenSeed) {
		t.Fatal("seed printed")
	}
}

func TestAuthSignSeedFromStdinAndEnv(t *testing.T) {
	out, _, err := runCmd(t, CmdAuthSignCosmos(), "  0x"+goldenSeed+"\r\n", append(cosmosArgs, "--seed-file", "-")...)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "signature_b64: "+goldenCosmos) {
		t.Fatalf("stdin seed: output does not match the golden vector:\n%s", out)
	}

	t.Setenv(envAuthSeed, goldenSeed)
	out, _, err = runCmd(t, CmdAuthSignCosmos(), "", cosmosArgs...)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "signature_b64: "+goldenCosmos) {
		t.Fatalf("env seed: output does not match the golden vector:\n%s", out)
	}
}

func TestAuthSignRefusesSeedOnArgv(t *testing.T) {
	t.Setenv(envAuthSeed, goldenSeed)
	_, _, err := runCmd(t, CmdAuthSignCosmos(), "", append([]string{goldenSeed}, cosmosArgs...)...)
	if err == nil {
		t.Fatal("old layout with the seed as first argument was accepted")
	}
	if strings.Contains(err.Error(), goldenSeed) {
		t.Fatal("error echoes the seed")
	}
	if _, _, err := runCmd(t, CmdAuthSignEVM(), "", append([]string{goldenSeed}, evmArgs...)...); err == nil {
		t.Fatal("old evm layout with the seed as first argument was accepted")
	}
}

func TestAuthSignNoSeedSource(t *testing.T) {
	os.Unsetenv(envAuthSeed)
	if _, _, err := runCmd(t, CmdAuthSignCosmos(), "", cosmosArgs...); err == nil {
		t.Fatal("signed with no seed source")
	}
}

func TestAuthSignRefusesReadableSeedFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX modes do not apply")
	}
	loose := writeFileMode(t, "seed", goldenSeed, 0o644)
	if _, _, err := runCmd(t, CmdAuthSignCosmos(), "", append(cosmosArgs, "--seed-file", loose)...); err == nil {
		t.Fatal("world-readable seed file accepted")
	}
	out, _, err := runCmd(t, CmdAuthSignCosmos(), "", append(cosmosArgs, "--seed-file", loose, "--insecure-seed-file-perms")...)
	if err != nil || !strings.Contains(out, goldenCosmos) {
		t.Fatalf("override: %v\n%s", err, out)
	}
}

func TestAuthKeygenWritesPrivateFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.seed")
	out, errOut, err := runCmd(t, CmdAuthKeygen(), "", "--out", path)
	if err != nil {
		t.Fatal(err)
	}
	bz, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	seed := strings.TrimSpace(string(bz))
	if len(seed) != 64 {
		t.Fatalf("seed file holds %q", seed)
	}
	if strings.Contains(out, seed) || strings.Contains(errOut, seed) {
		t.Fatal("keygen printed the seed without --print-seed")
	}
	if !strings.Contains(out, "pubkey_b64: ") || !strings.Contains(out, "seed_file: "+path) {
		t.Fatalf("unexpected output:\n%s", out)
	}
	if runtime.GOOS != "windows" {
		fi, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if fi.Mode().Perm() != 0o600 {
			t.Fatalf("seed file mode %04o, want 0600", fi.Mode().Perm())
		}
	}
	if _, _, err := runCmd(t, CmdAuthKeygen(), "", "--out", path); err == nil {
		t.Fatal("keygen overwrote an existing seed file")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(after, bz) {
		t.Fatal("existing seed file changed")
	}

	// The written file signs, and yields the pubkey keygen printed.
	signOut, _, err := runCmd(t, CmdAuthSignCosmos(), "", append(cosmosArgs, "--seed-file", path)...)
	if err != nil {
		t.Fatal(err)
	}
	pubLine := strings.SplitN(out, "\n", 2)[0]
	if !strings.Contains(signOut, pubLine) {
		t.Fatalf("signer pubkey differs from keygen's: %q vs\n%s", pubLine, signOut)
	}
}

func TestAuthKeygenDefaultFileAndPrintSeed(t *testing.T) {
	t.Chdir(t.TempDir())
	out, _, err := runCmd(t, CmdAuthKeygen(), "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "seed_file: authenticator-") {
		t.Fatalf("no default seed file:\n%s", out)
	}
	matches, _ := filepath.Glob("authenticator-*.seed")
	if len(matches) != 1 {
		t.Fatalf("default seed file not created: %v", matches)
	}

	out, errOut, err := runCmd(t, CmdAuthKeygen(), "", "--print-seed")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(errOut, "warning") {
		t.Fatal("--print-seed warning not on stderr")
	}
	if strings.Contains(out, "warning") {
		t.Fatal("--print-seed warning polluted stdout")
	}
	if !strings.Contains(out, "seed_hex: ") || strings.Contains(out, "seed_file: ") {
		t.Fatalf("--print-seed alone should print the seed and write no file:\n%s", out)
	}
}
