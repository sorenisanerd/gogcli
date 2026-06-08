//go:build !darwin

package secrets

// IsKeychainLockedError returns false on non-macOS platforms.
func IsKeychainLockedError(_ string) bool {
	return false
}

// EnsureKeychainAccess is a no-op on non-macOS platforms.
func EnsureKeychainAccess() error {
	return nil
}
