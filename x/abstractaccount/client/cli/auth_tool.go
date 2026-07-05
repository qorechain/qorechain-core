package cli

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"github.com/spf13/cobra"

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

// CmdAuthKeygen generates an ed25519 authenticator keypair and prints the 32-byte
// seed (hex, keep secret) and the public key (base64, for register-authenticator).
func CmdAuthKeygen() *cobra.Command {
	return &cobra.Command{
		Use:   "auth-keygen",
		Short: "Generate an ed25519 authenticator keypair (prints seed-hex + pubkey-base64)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			pub, priv, err := ed25519.GenerateKey(nil)
			if err != nil {
				return err
			}
			cmd.Printf("seed_hex: %s\n", hex.EncodeToString(priv.Seed()))
			cmd.Printf("pubkey_b64: %s\n", base64.StdEncoding.EncodeToString(pub))
			return nil
		},
	}
}

func edFromSeedHex(seedHex string) (ed25519.PrivateKey, ed25519.PublicKey, error) {
	seed, err := hex.DecodeString(seedHex)
	if err != nil || len(seed) != ed25519.SeedSize {
		return nil, nil, fmt.Errorf("seed-hex must be %d bytes of hex", ed25519.SeedSize)
	}
	priv := ed25519.NewKeyFromSeed(seed)
	return priv, priv.Public().(ed25519.PublicKey), nil
}

// CmdAuthSignCosmos signs the Native-lane authenticator sign-bytes for a
// MsgExecuteCosmos and prints the signature (base64) + pubkey (base64). The
// amount is normalized identically to the execute-cosmos handler so the bytes match.
func CmdAuthSignCosmos() *cobra.Command {
	return &cobra.Command{
		Use:   "auth-sign-cosmos [seed-hex] [chain-id] [account] [to] [amount] [nonce]",
		Short: "Produce the authenticator signature for execute-cosmos",
		Args:  cobra.ExactArgs(6),
		RunE: func(cmd *cobra.Command, args []string) error {
			priv, pub, err := edFromSeedHex(args[0])
			if err != nil {
				return err
			}
			amount, err := sdk.ParseCoinsNormalized(args[4])
			if err != nil {
				return fmt.Errorf("amount: %w", err)
			}
			var nonce uint64
			if _, err := fmt.Sscanf(args[5], "%d", &nonce); err != nil {
				return fmt.Errorf("nonce: %w", err)
			}
			sb := types.CosmosAuthSignBytes(args[1], args[2], pub, args[3], amount.String(), nonce)
			cmd.Printf("pubkey_b64: %s\n", base64.StdEncoding.EncodeToString(pub))
			cmd.Printf("signature_b64: %s\n", base64.StdEncoding.EncodeToString(ed25519.Sign(priv, sb)))
			return nil
		},
	}
}

// CmdAuthSignEVM signs the EVM-lane authenticator sign-bytes for a MsgExecuteEVM.
func CmdAuthSignEVM() *cobra.Command {
	return &cobra.Command{
		Use:   "auth-sign-evm [seed-hex] [chain-id] [account] [to-0xhex] [value-wei] [data-hex] [nonce]",
		Short: "Produce the authenticator signature for execute-evm",
		Args:  cobra.ExactArgs(7),
		RunE: func(cmd *cobra.Command, args []string) error {
			priv, pub, err := edFromSeedHex(args[0])
			if err != nil {
				return err
			}
			data, err := hex.DecodeString(trim0x(args[5]))
			if err != nil {
				return fmt.Errorf("data must be hex: %w", err)
			}
			var nonce uint64
			if _, err := fmt.Sscanf(args[6], "%d", &nonce); err != nil {
				return fmt.Errorf("nonce: %w", err)
			}
			sb := types.EVMAuthSignBytes(args[1], args[2], pub, args[3], args[4], data, nonce)
			cmd.Printf("pubkey_b64: %s\n", base64.StdEncoding.EncodeToString(pub))
			cmd.Printf("signature_b64: %s\n", base64.StdEncoding.EncodeToString(ed25519.Sign(priv, sb)))
			return nil
		},
	}
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
