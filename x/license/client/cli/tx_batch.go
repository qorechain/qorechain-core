package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/license/types"
)

// Granting licences in one reviewed batch, signed by a person.
//
// WHY THIS EXISTS. The obvious way to issue licences is for the service that takes
// the payment to issue them: verify the money, sign the grant, done. That design
// requires the granting key to live inside that service, and the granting key here is
// not a narrow service credential. It is the chain authority: the same key grants
// every feature, and it also trips and resets the circuit breaker. Putting it in a
// deployment environment means a configuration change, which is a thing people do
// without ceremony, becomes the act that arms irreversible on-chain authority.
//
// So the key stays with the person who holds it, and the batch comes to the key
// rather than the key going to the service. The payment flow keeps running
// unattended and records who is owed a licence; once a day a human looks at the list
// and signs it. Nothing in the network ever needs the key.
//
// WHAT THAT BUYS, precisely: there is no configuration of any service that issues a
// licence. Not a missing environment variable that someone might set, not a feature
// flag, not a disabled code path waiting to be enabled. The capability is absent from
// the network rather than present and switched off.
//
// THE LIST IS UNTRUSTED INPUT. It is produced by another system, so it is data, not
// instruction. Every address is parsed, every feature id is checked against the known
// set, the chain id is matched so a testnet list cannot be signed onto mainnet, and
// the whole batch is printed for review before anything is signed. Confirmation is
// required and cannot be skipped except by an explicit flag, because the human
// reading the list IS the authorisation step. That is the whole design, not a
// courtesy.
//
// ONE TRANSACTION, ALL OR NOTHING. N grants go in one transaction, so either every
// grant in the reviewed list lands or none does. A partially applied batch would
// leave the operator reconciling which of fifteen grants happened, which is exactly
// the state this is meant to avoid.

// blockedFeatures are features that may not be granted yet, whatever the list says
// and whoever signs it.
//
// This is a compile-time block, so lifting it takes a deliberate release rather than
// a flag, a parameter, or an environment variable. That is the point: the reasons
// below are not "not ready yet", they are "granting this today causes a specific,
// measured loss", and a protection against that should not be removable by
// configuration.
var blockedFeatures = map[string]string{
	types.FeatureLightNodeOperator: "the x/lightnode EndBlocker still distributes the " +
		"entire module account balance, and a freshly registered node is eligible " +
		"immediately, so the first grantee to register and delegate is credited the whole " +
		"accumulated pool. Measured 2026-09-13: 3,146,264 QOR on mainnet and 66,515,760,258 " +
		"QOR on testnet, both with zero nodes registered. Remove this entry in the same " +
		"release that ships the reward-pool fix, not before",
}

// batchFile is the document the payment system produces and a person reviews.
type batchFile struct {
	// ChainID names the network these grants are for. Checked against the chain the
	// command is actually talking to, so a list built for one network cannot be
	// signed onto another.
	ChainID     string       `json:"chain_id"`
	GeneratedAt string       `json:"generated_at,omitempty"`
	Grants      []batchGrant `json:"grants"`
}

type batchGrant struct {
	Grantee   string `json:"grantee"`
	FeatureID string `json:"feature_id"`
	ExpiresAt int64  `json:"expires_at,omitempty"`
	Metadata  string `json:"metadata,omitempty"`
	// Reference is the payment system's own identifier, carried through for the
	// operator's reconciliation. It is printed, never sent on chain.
	Reference string `json:"reference,omitempty"`
}

// CmdGrantBatch returns the command that grants a reviewed batch of licences.
func CmdGrantBatch() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "grant-batch [file-or-url]",
		Short: "Grant a reviewed batch of licences in one transaction (authority = --from)",
		Long: `Grant several licences in a single transaction, signed locally.

The batch is a JSON document produced by the payment system:

  {
    "chain_id": "qorechain-vladi",
    "grants": [
      {"grantee": "qor1...", "feature_id": "validator_operator", "reference": "lic-0042"}
    ]
  }

Every entry is validated, grants that already exist on chain are skipped, and the
whole batch is printed for review before anything is signed. Nothing is broadcast
until you confirm.

The granting key never leaves the machine you run this on.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			raw, err := loadBatch(cmd, args[0])
			if err != nil {
				return err
			}
			var batch batchFile
			if err := json.Unmarshal(raw, &batch); err != nil {
				return fmt.Errorf("batch is not valid JSON: %w", err)
			}
			if len(batch.Grants) == 0 {
				return fmt.Errorf("batch contains no grants")
			}

			// A list built for one network must not be signed onto another. This is
			// the single check that stops a testnet rehearsal from becoming a mainnet
			// grant, so it is not skippable.
			if batch.ChainID == "" {
				return fmt.Errorf("batch does not name a chain_id; refusing to guess")
			}
			if batch.ChainID != clientCtx.ChainID {
				return fmt.Errorf("batch is for chain %q but this command is talking to %q",
					batch.ChainID, clientCtx.ChainID)
			}

			// Validate everything before contacting the chain, so a malformed batch
			// fails immediately and completely rather than halfway through.
			seen := map[string]string{}
			for i, g := range batch.Grants {
				where := fmt.Sprintf("grant %d (%s)", i+1, g.Reference)
				if _, err := sdk.AccAddressFromBech32(g.Grantee); err != nil {
					return fmt.Errorf("%s: invalid grantee %q: %w", where, g.Grantee, err)
				}
				if reason, blocked := blockedFeatures[g.FeatureID]; blocked {
					return fmt.Errorf("%s: %q cannot be granted yet.\n\n  %s",
						where, g.FeatureID, reason)
				}
				if !types.IsValidFeatureID(g.FeatureID) {
					return fmt.Errorf("%s: unknown feature id %q", where, g.FeatureID)
				}
				if g.ExpiresAt < 0 {
					return fmt.Errorf("%s: expires_at cannot be negative", where)
				}
				// A batch that names the same pair twice would fail on chain at the
				// second one and take the whole transaction with it.
				key := g.Grantee + "|" + g.FeatureID
				if prev, dup := seen[key]; dup {
					return fmt.Errorf("%s: duplicates %s in the same batch", where, prev)
				}
				seen[key] = where
			}

			// Drop what is already granted. Without this one already-held licence
			// fails the whole transaction, and the operator cannot tell which.
			queryClient := types.NewQueryClient(clientCtx)
			todo := make([]batchGrant, 0, len(batch.Grants))
			skipped := make([]batchGrant, 0)
			for _, g := range batch.Grants {
				res, err := queryClient.Check(cmd.Context(), &types.QueryCheckRequest{
					Grantee: g.Grantee, FeatureId: g.FeatureID,
				})
				if err != nil {
					return fmt.Errorf("could not check existing licence for %s: %w", g.Grantee, err)
				}
				// Any existing grant, suspended or not: GrantLicense refuses a key
				// that already exists, and one refusal fails the whole transaction.
				if res.License != nil {
					skipped = append(skipped, g)
					continue
				}
				todo = append(todo, g)
			}

			printBatch(cmd, clientCtx, batch, todo, skipped)
			if len(todo) == 0 {
				cmd.Println("\nNothing to grant. No transaction built.")
				return nil
			}

			confirmed, _ := cmd.Flags().GetBool("yes")
			if !confirmed {
				ok, err := confirm(cmd, len(todo))
				if err != nil {
					return err
				}
				if !ok {
					cmd.Println("Aborted. Nothing was signed.")
					return nil
				}
			}

			msgs := make([]sdk.Msg, 0, len(todo))
			for _, g := range todo {
				msgs = append(msgs, &types.MsgGrantLicense{
					Authority: clientCtx.GetFromAddress().String(),
					Grantee:   g.Grantee,
					FeatureID: g.FeatureID,
					ExpiresAt: g.ExpiresAt,
					Metadata:  g.Metadata,
				})
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msgs...)
		},
	}
	cmd.Flags().Bool("yes", false,
		"skip the confirmation prompt (for a list you have already reviewed)")
	cmd.Flags().Duration("timeout", 15*time.Second, "timeout when the batch is a URL")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// loadBatch reads the batch from a local file or an http(s) URL.
func loadBatch(cmd *cobra.Command, src string) ([]byte, error) {
	if !strings.HasPrefix(src, "http://") && !strings.HasPrefix(src, "https://") {
		return os.ReadFile(src)
	}
	timeout, _ := cmd.Flags().GetDuration("timeout")
	httpClient := &http.Client{Timeout: timeout}
	resp, err := httpClient.Get(src)
	if err != nil {
		return nil, fmt.Errorf("fetching the batch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching the batch: %s", resp.Status)
	}
	// Bounded so a wrong URL cannot stream something enormous into memory.
	return io.ReadAll(io.LimitReader(resp.Body, 4<<20))
}

// printBatch shows exactly what will be signed. This is the authorisation step, so it
// prints every grant rather than a count.
func printBatch(cmd *cobra.Command, clientCtx client.Context, batch batchFile, todo, skipped []batchGrant) {
	cmd.Printf("\nChain:     %s\n", batch.ChainID)
	cmd.Printf("Authority: %s\n", clientCtx.GetFromAddress().String())
	if batch.GeneratedAt != "" {
		cmd.Printf("Batch:     generated %s\n", batch.GeneratedAt)
	}

	byFeature := map[string]int{}
	for _, g := range todo {
		byFeature[g.FeatureID]++
	}
	features := make([]string, 0, len(byFeature))
	for f := range byFeature {
		features = append(features, f)
	}
	sort.Strings(features)

	cmd.Printf("\nTo grant (%d):\n", len(todo))
	for _, g := range todo {
		expiry := "perpetual"
		if g.ExpiresAt > 0 {
			expiry = fmt.Sprintf("expires at height %d", g.ExpiresAt)
		}
		ref := ""
		if g.Reference != "" {
			ref = "  [" + g.Reference + "]"
		}
		cmd.Printf("  %-45s %-22s %s%s\n", g.Grantee, g.FeatureID, expiry, ref)
	}
	for _, f := range features {
		cmd.Printf("    %s: %d\n", f, byFeature[f])
	}

	if len(skipped) > 0 {
		cmd.Printf("\nAlready granted, skipped (%d):\n", len(skipped))
		for _, g := range skipped {
			cmd.Printf("  %-45s %s\n", g.Grantee, g.FeatureID)
		}
	}
}

// confirm requires the operator to type the number of grants, not just "y". Typing
// the count means reading the list, which is the point of showing it.
func confirm(cmd *cobra.Command, n int) (bool, error) {
	cmd.Printf("\nSigning as the chain authority. Type %d to confirm, anything else to abort: ", n)
	var answer string
	if _, err := fmt.Fscanln(cmd.InOrStdin(), &answer); err != nil {
		return false, nil
	}
	return strings.TrimSpace(answer) == fmt.Sprint(n), nil
}
