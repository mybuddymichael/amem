package keyring

import (
	"fmt"
	"os"

	"github.com/zalando/go-keyring"
)

const service = "amem"

// Set stores an encryption key in the OS keychain.
// For global profiles, use account = profile_name.
// For local configs, use account = "local:{absolute_path}".
func Set(account, key string) error {
	return keyring.Set(service, account, key)
}

// Get retrieves an encryption key from the OS keychain.
// Falls back to AMEM_ENCRYPTION_KEY env var if keychain fails.
// For global profiles, use account = profile_name.
// For local configs, use account = "local:{absolute_path}".
func Get(account string) (string, error) {
	key, err := keyring.Get(service, account)
	if err != nil {
		// Fallback to env var
		envKey := os.Getenv("AMEM_ENCRYPTION_KEY")
		if envKey == "" {
			return "", fmt.Errorf("key not found in keychain and AMEM_ENCRYPTION_KEY not set: %w", err)
		}
		return envKey, nil
	}
	return key, nil
}

// Delete removes an encryption key from the OS keychain.
func Delete(account string) error {
	return keyring.Delete(service, account)
}
