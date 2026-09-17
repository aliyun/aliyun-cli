package main

import (
	"path/filepath"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/openapi"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/aimode"
	"github.com/aliyun/aliyun-cli/v3/util"
)

// Resolve AI mode without loading the broken credential configuration or
// executing a command. Respect flag values and the positional terminator.
func startupAIMode(args []string) bool {
	flags := cli.NewFlagSet()
	config.AddFlags(flags)
	openapi.AddFlags(flags)
	forceOn, forceOff := false, false
	configDir := config.GetConfigPath()
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			break
		}
		name, value, inline := strings.Cut(arg, "=")
		if !strings.HasPrefix(name, "-") {
			continue
		}
		flag := flags.Get(strings.TrimPrefix(name, "--"))
		if flag == nil && len(name) == 2 && name[0] == '-' {
			flag = flags.GetByShorthand(rune(name[1]))
		}
		if flag == nil {
			continue
		}
		if flag.AssignedMode != cli.AssignedNone && !inline && i+1 < len(args) {
			i++
			value = args[i]
		}
		switch flag.Name {
		case "cli-ai-mode":
			forceOn = true
		case "no-cli-ai-mode":
			forceOff = true
		case "config-path":
			if value != "" {
				configDir = filepath.Dir(value)
			}
		}
	}
	cfg, _ := aimode.Load(configDir)
	if forceOff {
		return false
	}
	if forceOn {
		return true
	}
	if enabled, ok := aimode.EnvironmentOverride(); ok {
		return enabled
	}
	if util.DetectAgentName() != "" && aimode.AgentAIModeIntegrationEnabled() {
		return true
	}
	return aimode.EnabledForCommand(cfg, false, false)
}
