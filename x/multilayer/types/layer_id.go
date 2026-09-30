package types

// Layer and rollup ids are store-key components joined with '/', and some
// keeper loops walk every key under "<id>/". An id that contains '/' (or
// sorts as another id's extension) lets one layer's anchors, batches or blobs
// be read as another's (QSR-2026-0262). From v3.2.2 (FeatureRDKIntegrity) a
// new id must match
//
//	^[a-z0-9][a-z0-9._-]{2,63}$
//
// lowercase letters, digits, '.', '_' and '-', 3 to 64 bytes, starting with a
// letter or digit. The same grammar also keeps ids usable as REST path
// segments. It is checked in the keeper at creation, not in ValidateBasic:
// ValidateBasic has no block height, and tightening it would refuse
// transactions that historical blocks accepted.
const (
	IDMinLen = 3
	IDMaxLen = 64
)

// IDGrammarError reports why id does not match the id grammar, or "" if it does.
func IDGrammarError(id string) string {
	if len(id) < IDMinLen || len(id) > IDMaxLen {
		return "must be 3 to 64 characters"
	}
	for i := 0; i < len(id); i++ {
		c := id[i]
		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9':
		case (c == '.' || c == '_' || c == '-') && i > 0:
		default:
			if i == 0 {
				return "must start with a lowercase letter or a digit"
			}
			return "may contain only lowercase letters, digits, '.', '_' and '-'"
		}
	}
	return ""
}

// ValidateLayerID checks a new layer id against the id grammar.
func ValidateLayerID(id string) error {
	if msg := IDGrammarError(id); msg != "" {
		return ErrInvalidLayerID.Wrapf("layer_id %q %s", id, msg)
	}
	return nil
}
