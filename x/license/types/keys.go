package types

const (
	ModuleName = "license"
	StoreKey   = ModuleName
	RouterKey  = ModuleName
)

var (
	LicensePrefix = []byte("license/grants/")   // license/grants/{grantee}/{feature_id}
	FeaturePrefix = []byte("license/features/")  // license/features/{feature_id} -> list of grantees
	ParamsKey     = []byte("license/params")
)

func LicenseKey(grantee, featureID string) []byte {
	return append(append(LicensePrefix, []byte(grantee+"/")...), []byte(featureID)...)
}

func FeatureKey(featureID string) []byte {
	return append(FeaturePrefix, []byte(featureID)...)
}
