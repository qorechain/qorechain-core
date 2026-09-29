package cli

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/abstractaccount/types"
)

// These are CLIENT-SIDE helpers for the linked-wallet authenticator flow. They
// never touch the chain; they let a relayer/QoreX/dashboard produce exactly the
// bytes the chain verifies (via the module's own sign-byte functions), so a test
// harness or a wallet can drive execute-cosmos / execute-evm. They belong under
// the query group only because they do not broadcast.
//
// The authenticator seed never travels on the command line: argv is readable by
// every local user through ps and /proc/<pid>/cmdline and ends up in shell
// history and CI logs. keygen writes it to a 0600 file; the signers read it from
// that file, from stdin, from a no-echo prompt or from QORE_AUTH_SEED. None of
// this defends against malware running as the same user, which can read the
// file or the process memory; use a hardware or keyring signer for that.

const (
	flagOut               = "out"
	flagPrintSeed         = "print-seed"
	flagSeedFile          = "seed-file"
	flagInsecureSeedPerms = "insecure-seed-file-perms"

	// envAuthSeed is read when no --seed-file is given and stdin is not a
	// terminal. The environment is private to the user and root, but child
	// processes inherit it.
	envAuthSeed = "QORE_AUTH_SEED"
)

// CmdAuthKeygen generates an ed25519 authenticator keypair, writes the 32-byte
// seed (hex) to a new 0600 file and prints the public key (base64, for
// register-authenticator) and the file path.
func CmdAuthKeygen() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth-keygen",
		Short: "Generate an ed25519 authenticator keypair (seed to a 0600 file, pubkey to stdout)",
		Long: `Generate an ed25519 authenticator keypair. Only ed25519 authenticators are
produced; secp256k1 and Ethereum-address authenticators come from their own
wallets.

The seed is written as hex to a new file created with mode 0600 (--out, default
./authenticator-<pubkey prefix>.seed); an existing file is never overwritten.
Only the public key and the file path are printed. --print-seed also prints the
seed to stdout, for scripted use where the output is captured securely.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			pub, priv, err := ed25519.GenerateKey(nil)
			if err != nil {
				return err
			}
			defer wipe(priv)
			seedHex := []byte(hex.EncodeToString(priv.Seed()))
			defer wipe(seedHex)

			out, _ := cmd.Flags().GetString(flagOut)
			printSeed, _ := cmd.Flags().GetBool(flagPrintSeed)
			if out == "" && !printSeed {
				out = fmt.Sprintf("authenticator-%s.seed", hex.EncodeToString(pub[:4]))
			}
			if out != "" {
				line := make([]byte, len(seedHex)+1)
				copy(line, seedHex)
				line[len(seedHex)] = '\n'
				err := writeSeedFile(out, line)
				wipe(line)
				if err != nil {
					return err
				}
			}
			cmd.Printf("pubkey_b64: %s\n", base64.StdEncoding.EncodeToString(pub))
			if out != "" {
				cmd.Printf("seed_file: %s\n", out)
			}
			if printSeed {
				cmd.PrintErrln("warning: printing the authenticator seed; anyone who sees this output can act for the account within the authenticator's permissions")
				cmd.Printf("seed_hex: %s\n", seedHex)
			}
			return nil
		},
	}
	cmd.Flags().String(flagOut, "", "file to create (mode 0600) holding the seed hex; refuses to overwrite")
	cmd.Flags().Bool(flagPrintSeed, false, "also print the seed hex to stdout (keep that output secret)")
	return cmd
}

// writeSeedFile creates path exclusively with mode 0600, writes bz and syncs it.
func writeSeedFile(path string, bz []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create seed file: %w", err)
	}
	// The umask can only narrow the mode, but set it explicitly anyway.
	if runtime.GOOS != "windows" {
		if err := f.Chmod(0o600); err != nil {
			f.Close()
			return err
		}
	}
	if _, err := f.Write(bz); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// addSeedFlags registers the seed-source flags shared by the signers.
func addSeedFlags(cmd *cobra.Command) {
	cmd.Flags().String(flagSeedFile, "", "file holding the seed hex (\"-\" reads it from stdin); without it the seed comes from "+envAuthSeed+" or a no-echo prompt")
	cmd.Flags().Bool(flagInsecureSeedPerms, false, "accept a seed file readable by group or others")
}

// readSeed resolves the authenticator seed: --seed-file (or stdin for "-"),
// else QORE_AUTH_SEED, else a no-echo prompt on a terminal. It never reads the
// seed from the arguments.
func readSeed(cmd *cobra.Command) ([]byte, error) {
	path, _ := cmd.Flags().GetString(flagSeedFile)
	switch {
	case path == "-":
		return readSeedFrom(cmd.InOrStdin())
	case path != "":
		insecure, _ := cmd.Flags().GetBool(flagInsecureSeedPerms)
		return readSeedFile(path, insecure)
	}
	if v, ok := os.LookupEnv(envAuthSeed); ok {
		return []byte(v), nil
	}
	if f, ok := cmd.InOrStdin().(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		cmd.PrintErr("authenticator seed (hex): ")
		bz, err := term.ReadPassword(int(f.Fd()))
		cmd.PrintErrln()
		return bz, err
	}
	return nil, fmt.Errorf("no authenticator seed: pass --%s <file>, --%s - with the seed on stdin, or set %s", flagSeedFile, flagSeedFile, envAuthSeed)
}

func readSeedFrom(r io.Reader) ([]byte, error) {
	// A seed is 64 hex characters; read a little more to allow 0x and newlines,
	// and never buffer an unbounded stream.
	return io.ReadAll(io.LimitReader(r, 256))
}

// readSeedFile opens path (following symlinks) and checks the mode of the file
// actually opened, so the check cannot race a rename. Group- or world-readable
// files are refused unless insecure is set. POSIX modes do not apply on
// Windows, where the check is skipped.
func readSeedFile(path string, insecure bool) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open seed file: %w", err)
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("seed file %s is not a regular file", path)
	}
	if runtime.GOOS != "windows" && fi.Mode().Perm()&0o077 != 0 && !insecure {
		return nil, fmt.Errorf("seed file %s has mode %04o, readable by others; chmod 600 it or pass --%s", path, fi.Mode().Perm(), flagInsecureSeedPerms)
	}
	return readSeedFrom(f)
}

// edFromSeed decodes a hex seed (surrounding whitespace and a 0x prefix are
// ignored) into an ed25519 key pair and wipes the input.
func edFromSeed(raw []byte) (ed25519.PrivateKey, ed25519.PublicKey, error) {
	defer wipe(raw)
	s := bytes.TrimSpace(raw)
	if len(s) >= 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		s = s[2:]
	}
	seed := make([]byte, hex.DecodedLen(len(s)))
	defer wipe(seed)
	n, err := hex.Decode(seed, s)
	if err != nil || n != ed25519.SeedSize {
		return nil, nil, fmt.Errorf("the authenticator seed must be %d bytes of hex", ed25519.SeedSize)
	}
	priv := ed25519.NewKeyFromSeed(seed[:n])
	return priv, priv.Public().(ed25519.PublicKey), nil
}

// wipe zeroes key material once it is no longer needed (best effort).
func wipe(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// seedArgsRemoved rejects the old argument layout, whose first argument was
// the seed, without echoing any of it.
func seedArgsRemoved(want int) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		if len(args) == want+1 {
			return fmt.Errorf("the seed is no longer taken as an argument (it leaks through ps and shell history); drop it and pass --%s instead", flagSeedFile)
		}
		if len(args) != want {
			return fmt.Errorf("accepts %d arg(s), received %d", want, len(args))
		}
		return nil
	}
}

// CmdAuthSignCosmos signs the Native-lane authenticator sign-bytes for a
// MsgExecuteCosmos and prints the signature (base64) + pubkey (base64). The
// amount is normalized identically to the execute-cosmos handler so the bytes match.
func CmdAuthSignCosmos() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth-sign-cosmos [chain-id] [account] [to] [amount] [nonce]",
		Short: "Produce the authenticator signature for execute-cosmos (seed via --seed-file, stdin, prompt or QORE_AUTH_SEED)",
		Args:  seedArgsRemoved(5),
		RunE: func(cmd *cobra.Command, args []string) error {
			amount, err := sdk.ParseCoinsNormalized(args[3])
			if err != nil {
				return fmt.Errorf("amount: %w", err)
			}
			var nonce uint64
			if _, err := fmt.Sscanf(args[4], "%d", &nonce); err != nil {
				return fmt.Errorf("nonce: %w", err)
			}
			raw, err := readSeed(cmd)
			if err != nil {
				return err
			}
			priv, pub, err := edFromSeed(raw)
			if err != nil {
				return err
			}
			defer wipe(priv)
			sb := types.CosmosAuthSignBytes(args[0], args[1], pub, args[2], amount.String(), nonce)
			cmd.Printf("pubkey_b64: %s\n", base64.StdEncoding.EncodeToString(pub))
			cmd.Printf("signature_b64: %s\n", base64.StdEncoding.EncodeToString(ed25519.Sign(priv, sb)))
			return nil
		},
	}
	addSeedFlags(cmd)
	return cmd
}

// CmdAuthSignEVM signs the EVM-lane authenticator sign-bytes for a MsgExecuteEVM.
func CmdAuthSignEVM() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth-sign-evm [chain-id] [account] [to-0xhex] [value-wei] [data-hex] [nonce]",
		Short: "Produce the authenticator signature for execute-evm (seed via --seed-file, stdin, prompt or QORE_AUTH_SEED)",
		Args:  seedArgsRemoved(6),
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := hex.DecodeString(trim0x(args[4]))
			if err != nil {
				return fmt.Errorf("data must be hex: %w", err)
			}
			var nonce uint64
			if _, err := fmt.Sscanf(args[5], "%d", &nonce); err != nil {
				return fmt.Errorf("nonce: %w", err)
			}
			raw, err := readSeed(cmd)
			if err != nil {
				return err
			}
			priv, pub, err := edFromSeed(raw)
			if err != nil {
				return err
			}
			defer wipe(priv)
			sb := types.EVMAuthSignBytes(args[0], args[1], pub, args[2], args[3], data, nonce)
			cmd.Printf("pubkey_b64: %s\n", base64.StdEncoding.EncodeToString(pub))
			cmd.Printf("signature_b64: %s\n", base64.StdEncoding.EncodeToString(ed25519.Sign(priv, sb)))
			return nil
		},
	}
	addSeedFlags(cmd)
	return cmd
}

func trim0x(s string) string {
	if len(s) >= 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		return s[2:]
	}
	return s
}

// CmdQueryPermissionSchema prints the on-chain authenticator permission taxonomy.
func CmdQueryPermissionSchema() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "permission-schema",
		Short: "Query the canonical authenticator permission taxonomy",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).PermissionSchema(cmd.Context(), &types.QueryPermissionSchemaRequest{})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
