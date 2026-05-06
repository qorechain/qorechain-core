// Package keeper is the AMM module's keeper. The community-edition build
// has no files in this package — every source file at this path is provided
// by the build overlay in extended (full-tag) builds.
//
// This doc.go ensures the directory exists at compile time so `go vet` and
// `go test` can chdir into the package even when no overlay is active. It
// declares the package and nothing else.
package keeper
