package types

import (
	"strings"
	"testing"

	multilayertypes "github.com/qorechain/qorechain-core/x/multilayer/types"
)

func TestValidateRollupID(t *testing.T) {
	good := []string{
		"abc", "a1b", "0rollup", "my-rollup", "my_rollup", "my.rollup", "r--",
		"victim", strings.Repeat("a", 64), "9.9.9",
	}
	bad := []string{
		"", "ab", strings.Repeat("a", 65),
		"victim/x", "Victim", "ROLL", "-abc", ".abc", "_abc",
		"has space", "tab\tid", "nul\x00id", "new\nline", "é-rollup", "rollup✓",
		"a/b", "a\\b", "a:b", "a+b", "a%2fb",
	}
	for _, id := range good {
		if err := ValidateRollupID(id); err != nil {
			t.Errorf("%q refused: %v", id, err)
		}
		if err := multilayertypes.ValidateLayerID(id); err != nil {
			t.Errorf("layer %q refused: %v", id, err)
		}
	}
	for _, id := range bad {
		if err := ValidateRollupID(id); err == nil {
			t.Errorf("%q accepted", id)
		} else if !ErrInvalidRollupID.Is(err) {
			t.Errorf("%q: wrong error class: %v", id, err)
		}
		if err := multilayertypes.ValidateLayerID(id); err == nil {
			t.Errorf("layer %q accepted", id)
		}
	}
}
