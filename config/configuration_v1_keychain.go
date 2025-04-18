//go:build !darwin || disable_keychain

package config

import (
	"fmt"
	"runtime"
)

func getPasswordFromKeychain(label string) (string, error) {
	return "", fmt.Errorf("platform '%s' doesn't support KeyChain access", runtime.GOOS)
}
