//go:build !proprietary

package keeper

// QDRWTallyHandler is a stub in the public build.
// When QDRW is not available, standard governance tally behavior is used.
type QDRWTallyHandler struct{}
