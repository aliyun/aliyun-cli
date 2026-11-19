package plugin

import (
	"fmt"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

// CheckLocalPluginCliVersion records advisory compatibility facts. Actual
// archive/schema/invocation failures remain errors; a minimum alone is not a gate.
func CheckLocalPluginCliVersion(ctx *cli.Context, name string, local *LocalPlugin) error {
	if local == nil {
		return fmt.Errorf("plugin is nil")
	}
	if name == "" {
		name = local.Name
	}
	minimum := strings.TrimSpace(local.MinCliVersion)
	if minimum == "" || isDevVersion(cli.GetVersion()) {
		return nil
	}
	if !validVersion(minimum) {
		return fmt.Errorf("plugin %s has an invalid minimum CLI version", name)
	}
	if compareVersion(cli.GetVersion(), minimum) < 0 {
		ctx.AddCompatibilityNotice(cli.UpdateNotice{Component: name, Current: cli.GetVersion(), Latest: minimum, Severity: "compatibility", Command: "aliyun upgrade", Message: fmt.Sprintf("Plugin %s requires CLI >= %s; current CLI is %s. Run aliyun upgrade.", name, minimum, cli.GetVersion())})
	}
	return nil
}

func CheckPluginCliVersion(ctx *cli.Context, command string) error {
	m, err := NewManager()
	if err != nil {
		return err
	}
	name, p, err := m.findLocalPlugin(command)
	if err != nil {
		return err
	}
	return CheckLocalPluginCliVersion(ctx, name, p)
}
