package plugin

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func ExecutePlugin(command string, args []string) (bool, error) {
	mgr, err := NewManager()
	if err != nil {
		return false, nil
	}

	manifest, err := mgr.GetLocalManifest()
	if err != nil {
		return false, nil
	}

	var targetPlugin *LocalPlugin
	for _, p := range manifest.Plugins {
		if p.Command == command {
			targetPlugin = &p
			break
		}
	}

	// 尝试匹配 aliyun-cli-<command> 格式的插件
	if targetPlugin == nil {
		altCommand := "aliyun-cli-" + command
		for _, p := range manifest.Plugins {
			if p.Command == altCommand {
				targetPlugin = &p
				break
			}
		}
	}

	if targetPlugin == nil {
		return false, nil
	}

	// pluginManifestPath := filepath.Join(targetPlugin.Path, "manifest.json")
	// TODO: 缓存 bin path 到 LocalManifest，避免每次读取

	binPath := filepath.Join(targetPlugin.Path, targetPlugin.Name) // 默认二进制名
	// Windows 处理
	if _, err := os.Stat(binPath + ".exe"); err == nil {
		binPath += ".exe"
	}

	// fmt.Printf("Executing plugin: %s %v\n", binPath, args)
	cmd := exec.Command(binPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Env = os.Environ()

	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			os.Exit(exitError.ExitCode())
		}
		return true, fmt.Errorf("plugin execution failed: %w", err)
	}

	return true, nil
}
