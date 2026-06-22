package cli

import (
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/qorechain/qorechain-core/x/svm/types"
)

// GetTxCmd returns the transaction commands for the SVM module.
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "SVM module transaction commands",
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		GetCmdDeployProgram(),
		GetCmdExecuteProgram(),
		GetCmdCreateAccount(),
	)

	return cmd
}

// GetCmdDeployProgram returns the command to deploy a BPF program to the SVM runtime.
func GetCmdDeployProgram() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy-program [bytecode-file]",
		Short: "Deploy a BPF ELF program to the SVM runtime",
		Long: `Deploy a BPF ELF program to the SVM runtime.

The bytecode-file argument should be a path to a compiled BPF ELF binary.
The program will be assigned a deterministic address derived from the deployer
and bytecode hash.`,
		Example: `  # Deploy a compiled BPF program
  qorechaind tx svm deploy-program ./target/deploy/my_program.so --from mykey

  # Deploy with explicit gas
  qorechaind tx svm deploy-program ./my_program.so --from mykey --gas 500000`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			bytecode, err := os.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("failed to read bytecode file: %w", err)
			}

			if len(bytecode) == 0 {
				return fmt.Errorf("bytecode file is empty")
			}

			msg := &types.MsgDeployProgram{
				Sender:   clientCtx.GetFromAddress().String(),
				Bytecode: bytecode,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// GetCmdExecuteProgram returns the command to execute an instruction on a deployed SVM program.
func GetCmdExecuteProgram() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "execute [program-id-base58] [data-hex]",
		Short: "Execute an instruction on a deployed SVM program",
		Long: `Execute an instruction on a deployed SVM program.

The program-id-base58 argument is the base58-encoded 32-byte program address.
The data-hex argument is the hex-encoded instruction data to pass to the program.

Input accounts are supplied with repeated --accounts flags in the form
<base58-address>:<modifiers>, where modifiers is any combination of:
  s  - account is a signer
  w  - account is writable
A bare address (or empty modifiers) denotes a read-only, non-signer account.`,
		Example: `  # Execute with no input accounts
  qorechaind tx svm execute <program-id-base58> <instruction-data-hex> --from mykey

  # Execute passing a writable signer and a read-only account
  qorechaind tx svm execute <prog> <data-hex> \
    --accounts <addrA>:sw --accounts <addrB> --from mykey`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			programID, err := types.Base58Decode(args[0])
			if err != nil {
				return fmt.Errorf("invalid base58 program ID: %w", err)
			}

			data, err := hex.DecodeString(args[1])
			if err != nil {
				return fmt.Errorf("invalid hex-encoded instruction data: %w", err)
			}

			accountSpecs, err := cmd.Flags().GetStringArray(flagAccounts)
			if err != nil {
				return err
			}
			accounts, err := parseAccountMetas(accountSpecs)
			if err != nil {
				return err
			}

			msg := &types.MsgExecuteProgram{
				Sender:    clientCtx.GetFromAddress().String(),
				ProgramID: types.Bytes32(programID),
				Accounts:  accounts,
				Data:      data,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().StringArray(flagAccounts, nil, "input account as <base58-address>:<modifiers> (modifiers: s=signer, w=writable); repeatable")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// flagAccounts is the repeatable flag for SVM instruction input accounts.
const flagAccounts = "accounts"

// parseAccountMetas converts CLI account specs (<base58>:<modifiers>) into the
// SvmAccountMeta list expected by MsgExecuteProgram.
func parseAccountMetas(specs []string) ([]types.SvmAccountMeta, error) {
	if len(specs) == 0 {
		return nil, nil
	}
	metas := make([]types.SvmAccountMeta, 0, len(specs))
	for _, spec := range specs {
		addrStr, mods, _ := strings.Cut(spec, ":")
		addr, err := types.Base58Decode(addrStr)
		if err != nil {
			return nil, fmt.Errorf("invalid base58 account address %q: %w", addrStr, err)
		}
		meta := types.SvmAccountMeta{Address: types.Bytes32(addr)}
		for _, m := range mods {
			switch m {
			case 's', 'S':
				meta.IsSigner = true
			case 'w', 'W':
				meta.IsWritable = true
			default:
				return nil, fmt.Errorf("invalid account modifier %q in %q (allowed: s, w)", string(m), spec)
			}
		}
		metas = append(metas, meta)
	}
	return metas, nil
}

// GetCmdCreateAccount returns the command to create a new SVM data account.
func GetCmdCreateAccount() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-account [owner-base58] [space] [lamports]",
		Short: "Create a new SVM data account with allocated space",
		Long: `Create a new SVM data account owned by the specified program.

The owner-base58 argument is the base58-encoded 32-byte owner program address.
The space argument is the number of bytes to allocate for the account data.
The lamports argument is the number of lamports to fund the account with.`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			owner, err := types.Base58Decode(args[0])
			if err != nil {
				return fmt.Errorf("invalid base58 owner address: %w", err)
			}

			space, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid space value: %w", err)
			}

			lamports, err := strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid lamports value: %w", err)
			}

			msg := &types.MsgCreateAccount{
				Sender:   clientCtx.GetFromAddress().String(),
				Owner:    types.Bytes32(owner),
				Space:    space,
				Lamports: lamports,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
