//go:build proprietary

package keeper

import (
	"testing"

	"github.com/qorechain/qorechain-core/x/rdk/types"
)

func TestDANativeProfileUsesNativeBackend(t *testing.T) {
	cfg := GetPresetProfile(types.ProfileDeFi)
	if cfg.DABackend != types.DANative {
		t.Errorf("DeFi profile should use native DA, got %q", cfg.DABackend)
	}
}

func TestDACelestiaProfileUsesExternalBackend(t *testing.T) {
	cfg := GetPresetProfile(types.ProfileNFT)
	if cfg.DABackend != types.DACelestia {
		t.Errorf("NFT profile should use celestia DA, got %q", cfg.DABackend)
	}
}

func TestDABlobErrorSentinels(t *testing.T) {
	if types.ErrDABlobNotFound == nil {
		t.Error("ErrDABlobNotFound should not be nil")
	}
	if types.ErrDABlobTooLarge == nil {
		t.Error("ErrDABlobTooLarge should not be nil")
	}
	if types.ErrCelestiaDAStubed == nil {
		t.Error("ErrCelestiaDAStubed should not be nil")
	}

	// Verify error messages
	if types.ErrDABlobNotFound.Error() != "rdk: DA blob not found" {
		t.Errorf("unexpected error message: %q", types.ErrDABlobNotFound.Error())
	}
	if types.ErrCelestiaDAStubed.Error() != "rdk: Celestia DA backend is stubbed in v1.3.0" {
		t.Errorf("unexpected error message: %q", types.ErrCelestiaDAStubed.Error())
	}
}
