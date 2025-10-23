package keyring

import (
	"os"
	"testing"
)

func TestSetAndGet(t *testing.T) {
	account := "test-account"
	key := "test-key-123"

	// Clean up before and after
	defer func() { _ = Delete(account) }()
	_ = Delete(account)

	// Set key
	if err := Set(account, key); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Get key
	got, err := Get(account)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if got != key {
		t.Errorf("Get() = %q, want %q", got, key)
	}
}

func TestGetFallbackToEnv(t *testing.T) {
	account := "nonexistent-account"
	envKey := "env-key-456"

	// Ensure account doesn't exist
	_ = Delete(account)

	// Set env var
	_ = os.Setenv("AMEM_ENCRYPTION_KEY", envKey)
	defer func() { _ = os.Unsetenv("AMEM_ENCRYPTION_KEY") }()

	// Get should fallback to env
	got, err := Get(account)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if got != envKey {
		t.Errorf("Get() = %q, want %q", got, envKey)
	}
}

func TestGetNoKeyNoEnv(t *testing.T) {
	account := "nonexistent-account-2"

	// Ensure account doesn't exist
	_ = Delete(account)

	// Ensure env var is not set
	_ = os.Unsetenv("AMEM_ENCRYPTION_KEY")

	// Get should fail
	_, err := Get(account)
	if err == nil {
		t.Error("Get() should have failed, but didn't")
	}
}

func TestDelete(t *testing.T) {
	account := "test-delete-account"
	key := "test-key-789"

	// Set key
	if err := Set(account, key); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Delete key
	if err := Delete(account); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Get should fail (without env fallback)
	_ = os.Unsetenv("AMEM_ENCRYPTION_KEY")
	_, err := Get(account)
	if err == nil {
		t.Error("Get() after Delete() should have failed, but didn't")
	}
}
