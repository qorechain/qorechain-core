package cli

import (
	"encoding/hex"
	"fmt"
	"os"
	"strconv"

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
		DisableFlagParsing:         true,
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
The data-hex argument is the hex-encoded instruction data to pass to the program.`,
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

			msg := &types.MsgExecuteProgram{
				Sender:    clientCtx.GetFromAddress().String(),
				ProgramID: programID,
				Data:      data,
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
				Owner:    owner,
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
