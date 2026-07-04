// Package measure is the payload-measurement harness (OpenSpec change
// payload-measurement-harness).
//
// The actual harness lives in build-tagged files (`//go:build measure`) so it
// never runs in the default `go test ./...` suite — only via `make measure`.
// This file carries no build tag and no imports so that, without the tag, the
// package still builds (as an empty package with no test files) and neither
// `go build ./...` nor `go test ./...` errors with "build constraints exclude
// all Go files". With `-tags measure`, the harness files compile and run.
//
// See internal/arch/arch_test.go, where `measure` is classified `exempt`: like
// testutil it drives the full production router over HTTP.
package measure
