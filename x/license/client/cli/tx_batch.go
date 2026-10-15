package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/license/types"
	lightnodetypes "github.com/qorechain/qorechain-core/x/lightnode/types"
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

// blockedFeatures are features that may not be granted at all, whatever the list
// says and whoever signs it. Keyed by feature id, valued by the reason shown to the
// operator.
//
// This is a compile-time block, so lifting it takes a deliberate release rather than
// a flag, a parameter, or an environment variable. That is the point: an entry here
// is not "not ready yet", it is "granting this causes a specific, measured loss", and
// a protection against that should not be removable by configuration.
//
// Empty since the lightnode reward-pool fix. Until then lightnode_operator sat here,
// because the x/lightnode EndBlocker paid the module account's entire balance
// (3,146,264 QOR on mainnet, 66,515,760,258 QOR on testnet, measured 2026-09-13 with
// zero nodes registered) to whichever grantee registered first. That feature is now
// gated on the state of the chain instead: see chainGatedFeatures.
var blockedFeatures = map[string]string{}

// chainGatedFeatures are features whose safety depends on the chain this command is
// talking to, decided once per batch by asking it. The decision is made against the
// NETWORK rather than against the binary in the operator's hands: the release that
// carries the fix is the same binary before and after the upgrade height, and the
// validators keep running the previous release until that height. A compile-time
// lift would have let a grant through onto a chain whose EndBlocker still pays the
// whole balance out.
//
// lightnode_operator: the reward-pool accounting is established by the coordinated
// upgrade handler at the plan height, and the chain reports it through the
// reward_pool query. Before that height the query either does not exist (previous
// release) or answers initialized=false (this release, upgrade not yet executed),
// and both refuse the grant.
var chainGatedFeatures = map[string]func(ctx context.Context, clientCtx client.Context) (reason string, blocked bool){
	types.FeatureLightNodeOperator: func(ctx context.Context, clientCtx client.Context) (string, bool) {
		res, err := lightnodetypes.NewQueryClient(clientCtx).RewardPool(ctx,
			&lightnodetypes.QueryRewardPoolRequest{})
		return lightNodeGrantGate(res, err)
	},
}

// lightNodeGrantGate decides from the reward-pool answer whether a light-node
// operator licence may be granted on the chain that gave it. Kept apart from the
// query so the three answers a chain can give are testable without a network.
func lightNodeGrantGate(res *lightnodetypes.QueryRewardPoolResponse, err error) (string, bool) {
	if err != nil {
		return "the chain does not answer the light-node reward_pool query (" + err.Error() +
			"), so it runs a release whose EndBlocker pays the whole module balance to the " +
			"first node that registers; the licence cannot be granted there yet", true
	}
	if res == nil || !res.Initialized {
		return "the chain carries the reward-pool code but has not executed the upgrade " +
			"that establishes the accounting; until that height the validators may still " +
			"run the release that pays the whole module balance to the first node that " +
			"registers", true
	}
	return "", false
}

// setAside splits a validated batch into what may be granted and what this binary
// refuses, with the reason per refused feature. EXCLUDED, NOT FATAL: see the
// comment at the call site.
func setAside(ctx context.Context, clientCtx client.Context, grants []batchGrant) (allowed, blocked []batchGrant, reasons map[string]string) {
	allowed = make([]batchGrant, 0, len(grants))
	blocked = make([]batchGrant, 0)
	reasons = map[string]string{}
	// Each gate asks the chain once per batch, whatever the number of grants.
	decided := map[string]bool{}
	for _, g := range grants {
		if r, isBlocked := blockedFeatures[g.FeatureID]; isBlocked {
			reasons[g.FeatureID] = r
			blocked = append(blocked, g)
			continue
		}
		if gate, gated := chainGatedFeatures[g.FeatureID]; gated {
			if !decided[g.FeatureID] {
				decided[g.FeatureID] = true
				if r, isBlocked := gate(ctx, clientCtx); isBlocked {
					reasons[g.FeatureID] = r
				}
			}
			if _, isBlocked := reasons[g.FeatureID]; isBlocked {
				blocked = append(blocked, g)
				continue
			}
		}
		allowed = append(allowed, g)
	}
	return allowed, blocked, reasons
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

			// Set aside what the binary currently refuses to grant. EXCLUDED, NOT
			// FATAL, and the difference was a real design error on the first cut.
			//
			// Failing the batch meant that once a blocked feature was queued, every
			// batch containing one would be refused, taking the grants that CAN pass
			// down with it. The block would then be silently blocking the wrong thing.
			//
			// Skipping is safe here in a way it is not elsewhere: each grant is
			// independent, so removing one does not change what the others mean. That
			// is the test for whether skipping is acceptable, and it passes. The
			// refusal itself is untouched, because the excluded entries never reach the
			// transaction.
			//
			// Keeping this decision in the binary rather than asking the producer of
			// the batch to pre-filter also keeps the list of blocked features in ONE
			// place. Two copies would drift, and the copy that matters is this one.
			allowed, blocked, reasons := setAside(cmd.Context(), clientCtx, batch.Grants)

			// Drop what is already granted. Without this one already-held licence
			// fails the whole transaction, and the operator cannot tell which.
			queryClient := types.NewQueryClient(clientCtx)
			todo := make([]batchGrant, 0, len(allowed))
			skipped := make([]batchGrant, 0)
			for _, g := range allowed {
				res, err := queryClient.Check(cmd.Context(), &types.QueryCheckRequest{
					Grantee: g.Grantee, FeatureId: g.FeatureID,
				})
				held, err := licenceHeld(res, err)
				if err != nil {
					return fmt.Errorf("could not check existing licence for %s: %w", g.Grantee, err)
				}
				// Any existing grant, suspended or not: GrantLicense refuses a key
				// that already exists, and one refusal fails the whole transaction.
				if held {
					skipped = append(skipped, g)
					continue
				}
				todo = append(todo, g)
			}

			printBatch(cmd, clientCtx, batch, todo, skipped, blocked, reasons)
			if len(todo) == 0 {
				fmt.Fprintln(cmd.ErrOrStderr(), "\nNothing to grant. No transaction built.")
				return nil
			}

			// With --generate-only nothing is signed or broadcast here: the output is
			// an unsigned document that still has to be cosigned, which is where the
			// real authorisation happens. Prompting would only block the pipeline.
			generateOnly, _ := cmd.Flags().GetBool(flags.FlagGenerateOnly)
			confirmed, _ := cmd.Flags().GetBool(flags.FlagSkipConfirmation)
			if !confirmed && !generateOnly {
				ok, err := confirm(cmd, len(todo))
				if err != nil {
					return err
				}
				if !ok {
					fmt.Fprintln(cmd.ErrOrStderr(), "Aborted. Nothing was signed.")
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
	cmd.Flags().Duration("timeout", 15*time.Second, "timeout when the batch is a URL")
	// The review prompt is skipped by the SDK's own --yes/-y, which
	// AddTxFlagsToCmd defines: defining a second "yes" here made cobra panic at
	// startup ("flag redefined"), which took the whole binary down, not just
	// this command. One flag, one meaning: a list already reviewed.
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
func printBatch(cmd *cobra.Command, clientCtx client.Context, batch batchFile, todo, skipped, blocked []batchGrant, reasons map[string]string) {
	// Deliberately stderr. The mainnet flow is
	//   grant-batch ... --generate-only > tx.json
	//   tx pqc cosign tx.json ...
	// because the chain requires a hybrid post-quantum signature, and a review table
	// written to stdout would corrupt the document being redirected.
	out := cmd.ErrOrStderr()
	fmt.Fprintf(out, "\nChain:     %s\n", batch.ChainID)
	fmt.Fprintf(out, "Authority: %s\n", clientCtx.GetFromAddress().String())
	if batch.GeneratedAt != "" {
		fmt.Fprintf(out, "Batch:     generated %s\n", batch.GeneratedAt)
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

	fmt.Fprintf(out, "\nTo grant (%d):\n", len(todo))
	for _, g := range todo {
		expiry := "perpetual"
		if g.ExpiresAt > 0 {
			expiry = fmt.Sprintf("expires at height %d", g.ExpiresAt)
		}
		ref := ""
		if g.Reference != "" {
			ref = "  [" + g.Reference + "]"
		}
		fmt.Fprintf(out, "  %-45s %-22s %s%s\n", g.Grantee, g.FeatureID, expiry, ref)
	}
	for _, f := range features {
		fmt.Fprintf(out, "    %s: %d\n", f, byFeature[f])
	}

	if len(blocked) > 0 {
		fmt.Fprintf(out, "\nNOT granted, refused (%d):\n", len(blocked))
		for _, g := range blocked {
			fmt.Fprintf(out, "  %-45s %s\n", g.Grantee, g.FeatureID)
		}
		shown := map[string]bool{}
		for _, g := range blocked {
			if r, ok := reasons[g.FeatureID]; ok && !shown[g.FeatureID] {
				shown[g.FeatureID] = true
				fmt.Fprintf(out, "\n  %s: %s\n", g.FeatureID, r)
			}
		}
		fmt.Fprintf(out, "\n  These stay queued. They are not lost, and they need no\n"+
			"  re-entry: they grant themselves once the refusal lifts, whether by a\n"+
			"  release or by the chain reaching the state the grant waits for.\n")
	}

	if len(skipped) > 0 {
		fmt.Fprintf(out, "\nAlready granted, skipped (%d):\n", len(skipped))
		for _, g := range skipped {
			fmt.Fprintf(out, "  %-45s %s\n", g.Grantee, g.FeatureID)
		}
	}
}

// confirm requires the operator to type the number of grants, not just "y". Typing
// the count means reading the list, which is the point of showing it.
func confirm(cmd *cobra.Command, n int) (bool, error) {
	fmt.Fprintf(cmd.ErrOrStderr(),
		"\nSigning as the chain authority. Type %d to confirm, anything else to abort: ", n)
	var answer string
	if _, err := fmt.Fscanln(cmd.InOrStdin(), &answer); err != nil {
		return false, nil
	}
	return strings.TrimSpace(answer) == fmt.Sprint(n), nil
}

// licenceHeld reads the answer to a Check query. The query server answers
// NotFound for a grantee without the licence, which is the ordinary case for a
// new grant and must not fail the batch. Any other error is real.
func licenceHeld(res *types.QueryCheckResponse, err error) (bool, error) {
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return false, nil
		}
		return false, err
	}
	return res != nil && res.License != nil, nil
}
