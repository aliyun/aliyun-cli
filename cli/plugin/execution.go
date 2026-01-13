package plugin

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

// IsPluginInstalled checks if a plugin is installed locally.
// Returns (true, pluginName, nil) if plugin is installed.
// Returns (false, "", nil) if plugin is not installed.
// Returns (false, "", error) if there's an error checking.
func IsPluginInstalled(command string) (bool, string, error) {
	mgr, err := NewManager()
	if err != nil {
		return false, "", err
	}

	pluginName, _, err := mgr.findLocalPlugin(command)
	if err != nil {
		// Check if it's a "plugin not found" error (expected) vs a real error
		var notFoundErr *ErrPluginNotFound
		if errors.As(err, &notFoundErr) {
			return false, "", nil
		}
		return false, "", err
	}

	return true, pluginName, nil
}

// Returns (true, nil) if plugin was found and executed successfully.
// Returns (true, error) if plugin execution failed.
// Returns (false, nil) if plugin was not found (not an error).
// Returns (false, error) if there's an error finding the plugin or resolving the plugin binary path.
// If ctx is nil, uses os.Stdout and os.Stderr.
func ExecutePlugin(command string, args []string, ctx *cli.Context) (bool, error) {
	mgr, err := NewManager()
	if err != nil {
		return false, nil
	}

	_, plugin, err := mgr.findLocalPlugin(command)
	if err != nil {
		var notFoundErr *ErrPluginNotFound
		if errors.As(err, &notFoundErr) {
			return false, nil
		}
		// Real error (e.g., manifest file corrupted)
		return false, err
	}

	binPath, err := resolvePluginBinaryPath(plugin)
	if err != nil {
		return true, fmt.Errorf("failed to resolve plugin binary path: %w", err)
	}

	// Handle plugin-help subcommand: convert to --help
	adjustedArgs := adjustPluginArgs(args)

	var stdout, stderr io.Writer
	if ctx != nil {
		stdout = ctx.Stdout()
		stderr = ctx.Stderr()
	} else {
		stdout = os.Stdout
		stderr = os.Stderr
	}
	// fmt.Println("binPath", binPath)
	// fmt.Println("adjustedArgs", adjustedArgs)

	envs := os.Environ()

	if err := runPluginCommand(binPath, adjustedArgs, stdout, stderr, envs); err != nil {
		return true, err
	}

	return true, nil
}

func adjustPluginArgs(args []string) []string {
	if len(args) <= 1 {
		return args
	}

	// Check if first argument is "plugin-help"
	if args[1] == "plugin-help" {
		// Replace plugin-help with --help and drop the rest of the arguments
		return []string{args[0], "--help"}
	}

	return args
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

func runPluginCommand(binPath string, args []string, stdout io.Writer, stderr io.Writer, envs []string) error {
	if binPath == "" {
		return fmt.Errorf("binary path is empty")
	}

	cmd := exec.Command(binPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = envs

	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			os.Exit(exitError.ExitCode())
		}
		return fmt.Errorf("plugin execution failed: %w", err)
	}

	return nil
}
