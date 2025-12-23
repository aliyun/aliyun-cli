package plugin

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Returns (true, nil) if plugin was found and executed successfully.
// Returns (false, nil) if plugin was not found (not an error).
// Returns (true, error) if plugin execution failed.
func ExecutePlugin(command string, args []string) (bool, error) {
	mgr, err := NewManager()
	if err != nil {
		return false, nil
	}

	_, plugin, err := mgr.findLocalPlugin(command)
	if err != nil {
		// Plugin not found, not an error
		return false, nil
	}

	binPath, err := resolvePluginBinaryPath(plugin)
	if err != nil {
		return true, fmt.Errorf("failed to resolve plugin binary path: %w", err)
	}

	if err := runPluginCommand(binPath, args); err != nil {
		return true, err
	}

	return true, nil
}

func resolvePluginBinaryPath(plugin *LocalPlugin) (string, error) {
	if plugin == nil {
		return "", fmt.Errorf("plugin is nil")
	}

	binPath := filepath.Join(plugin.Path, plugin.Name)

	// Windows handling: check for .exe extension
	if _, err := os.Stat(binPath + ".exe"); err == nil {
		binPath += ".exe"
	}

	if _, err := os.Stat(binPath); err != nil {
		return "", fmt.Errorf("plugin binary not found at %s: %w", binPath, err)
	}

	return binPath, nil
}

func runPluginCommand(binPath string, args []string) error {
	if binPath == "" {
		return fmt.Errorf("binary path is empty")
	}

	cmd := exec.Command(binPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			os.Exit(exitError.ExitCode())
		}
		return fmt.Errorf("plugin execution failed: %w", err)
	}

	return nil
}
