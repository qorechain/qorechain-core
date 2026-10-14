package cli

import "testing"

// QSR-2026-0550 / 0400: the CLI refuses the key type that locks the account.
func TestCheckRegisterKeyType(t *testing.T) {
	for kt, ok := range map[string]bool{"hybrid": true, "pqc_only": true, "classical_only": false, "x": false} {
		if err := checkRegisterKeyType(kt); (err == nil) != ok {
			t.Errorf("%s: err=%v want ok=%v", kt, err, ok)
		}
	}
}
