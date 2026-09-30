package types

import multilayertypes "github.com/qorechain/qorechain-core/x/multilayer/types"

// ValidateRollupID checks a new rollup id against the id grammar shared with
// x/multilayer layer ids (a rollup registers its layer under the same id):
// ^[a-z0-9][a-z0-9._-]{2,63}$. See multilayertypes.IDGrammarError for why.
// Clients (CLI, SDK, dashboard) should apply it before paying a fee.
func ValidateRollupID(id string) error {
	if msg := multilayertypes.IDGrammarError(id); msg != "" {
		return ErrInvalidRollupID.Wrapf("rollup_id %q %s", id, msg)
	}
	return nil
}
