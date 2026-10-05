//go:build !linux

package main

import "errors"

var errNoSecureRead = errors.New("secure terminal read not available on this platform")

// getPass: no secure echo-off read is provided on this platform, so callers
// fall back to a plain stdin read (key may be visible). The recommended way
// to supply a key here is an environment variable.
func getPass(prompt string) ([]byte, error) {
	return nil, errNoSecureRead
}
