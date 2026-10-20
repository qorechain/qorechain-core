//go:build !full

package app

// FullBuild reports whether this binary carries the full overlay (PQC
// verification, the full keepers and the patched dependencies). This is the
// community build, which does not. The full build supplies its own value.
const FullBuild = false
