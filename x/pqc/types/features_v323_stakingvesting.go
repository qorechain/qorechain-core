package types

// v3.2.3 staking and vesting area. The release integrator appends
// V323StakingVestingFeatures to ReleaseV3_2_3ProtocolFeatures.
const (
	// StakeAuthorization allow and deny lists compare validator addresses by
	// their decoded bytes, not by their string form, so an all-uppercase
	// bech32 address no longer slips past a deny list (QSR-2026-0500). The
	// switch lives in the patched SDK (deps/patches/cosmos-sdk/0003).
	FeatureStakeAuthzCanonicalValidator = "stake-authz-canonical-validator"

	// The three vesting-creation messages are refused unless the sender
	// creates the account for itself, or the sender is governance or holds a
	// circuit-breaker permission over that message (QSR-2026-0402). The switch
	// lives in the patched SDK (deps/patches/cosmos-sdk/0004).
	FeatureVestingCreateSelfOnly = "vesting-create-self-only"
)

// V323StakingVestingFeatures are the switches of the v3.2.3 staking and
// vesting area.
var V323StakingVestingFeatures = []string{
	FeatureStakeAuthzCanonicalValidator,
	FeatureVestingCreateSelfOnly,
}
