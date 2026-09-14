//go:build !full

package app

// wireDependencyGates connects behaviour switches in patched upstream modules
// to the protocol-feature record. The community build compiles against the
// unpatched modules, which have no switches, so there is nothing to wire.
func (app *QoreChainApp) wireDependencyGates() {}
