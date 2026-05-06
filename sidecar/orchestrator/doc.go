// Package orchestrator manages chain-specific sidecar containers.
// All real implementations live in the extended (full-tag) build via
// the build overlay. This doc.go ensures the directory exists at compile
// time so go vet / go test can chdir into the package.
package orchestrator
