package plugin

import (
	"os"
	"runtime"
	"testing"
)

// setTestHomeDir sets the test home directory for cross-platform testing.
// Returns a cleanup function that restores the original environment variables.
func setTestHomeDir(t *testing.T, testHome string) func() {
	originalHome := os.Getenv("HOME")
	originalUserProfile := os.Getenv("USERPROFILE")
	originalHomeDrive := os.Getenv("HOMEDRIVE")
	originalHomePath := os.Getenv("HOMEPATH")

	os.Setenv("HOME", testHome)
	if runtime.GOOS == "windows" {
		os.Setenv("USERPROFILE", testHome)
		// Clear HOMEDRIVE and HOMEPATH to ensure USERPROFILE or HOME is used
		os.Unsetenv("HOMEDRIVE")
		os.Unsetenv("HOMEPATH")
	}

	return func() {
		os.Setenv("HOME", originalHome)
		if runtime.GOOS == "windows" {
			os.Setenv("USERPROFILE", originalUserProfile)
			os.Setenv("HOMEDRIVE", originalHomeDrive)
			os.Setenv("HOMEPATH", originalHomePath)
		}
	}
}
