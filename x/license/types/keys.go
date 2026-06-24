package types

const (
	ModuleName = "license"
	StoreKey   = ModuleName
	RouterKey  = ModuleName
)

var (
	LicensePrefix = []byte("license/grants/")   // license/grants/{grantee}/{feature_id}
	FeaturePrefix = []byte("license/features/") // license/features/{feature_id} -> list of grantees
	ParamsKey     = []byte("license/params")
	// AuthorityKey persists the grant authority across restarts. Without it the
	// authority lives only in the keeper struct (set in InitGenesis) and reverts
	// to the constructor default (gov module account) on any node restart.
	AuthorityKey = []byte("license/authority")
)

func LicenseKey(grantee, featureID string) []byte {
	return append(append(LicensePrefix, []byte(grantee+"/")...), []byte(featureID)...)
}

func FeatureKey(featureID string) []byte {
	return append(FeaturePrefix, []byte(featureID)...)
}
