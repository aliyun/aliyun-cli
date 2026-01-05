package plugin

import (
	"fmt"
	"text/tabwriter"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

func NewPluginCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "plugin",
		Short: i18n.T("Manage plugins", "管理插件"),
		Usage: "plugin <command> [args]",
		Run: func(ctx *cli.Context, args []string) error {
			return cli.NewErrorWithTip(fmt.Errorf("command missing"), "Use `aliyun plugin --help` for more information.")
		},
	}

	cmd.AddSubCommand(newListCommand())
	cmd.AddSubCommand(newInstallCommand())
	cmd.AddSubCommand(newInstallAllCommand())
	cmd.AddSubCommand(newUninstallCommand())
	cmd.AddSubCommand(newUpdateCommand())
	// cmd.AddSubCommand(newSearchCommand())

	return cmd
}

func newListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Short: i18n.T("List installed plugins", "列出已安装的插件"),
		Run: func(ctx *cli.Context, args []string) error {
			mgr, err := NewManager()
			if err != nil {
				return err
			}

			manifest, err := mgr.GetLocalManifest()
			if err != nil {
				return err
			}

			w := tabwriter.NewWriter(ctx.Stdout(), 20, 0, 3, ' ', 0)
			fmt.Fprintln(w, "Name\tVersion\tDescription")
			fmt.Fprintln(w, "----\t-------\t-----------")

			for _, p := range manifest.Plugins {
				fmt.Fprintf(w, "%s\t%s\t%s\n", p.Name, p.Version, p.Description)
			}
			w.Flush()
			return nil
		},
	}
}

func newInstallCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "install",
		Short: i18n.T("Install a plugin", "安装插件"),
		Usage: "plugin install --name <plugin_name> [--version <version>] OR plugin install --names <plugin1> <plugin2> ... [--version <version>]",
		Run: func(ctx *cli.Context, args []string) error {
			name, names, version, err := parseInstallArgs(ctx)
			if err != nil {
				return err
			}

			validatedNames, err := validateInstallArgs(name, names)
			if err != nil {
				return err
			}

			return executeInstall(ctx, name, validatedNames, version)
		},
	}

	nameFlag := &cli.Flag{
		Name:         "name",
		Short:        i18n.T("Plugin name to install", "要安装的插件名称"),
		AssignedMode: cli.AssignedOnce,
		Required:     false,
		ExcludeWith:  []string{"names"},
	}
	cmd.Flags().Add(nameFlag)

	namesFlag := &cli.Flag{
		Name:         "names",
		Short:        i18n.T("Plugin names to install (can specify multiple)", "要安装的插件名称（可指定多个）"),
		AssignedMode: cli.AssignedRepeatable,
		Required:     false,
		ExcludeWith:  []string{"name"},
	}
	cmd.Flags().Add(namesFlag)

	cmd.Flags().Add(&cli.Flag{
		Name:         "version",
		Short:        i18n.T("Specify plugin version", "指定插件版本"),
		AssignedMode: cli.AssignedOnce,
		DefaultValue: "",
	})

	return cmd
}

func newInstallAllCommand() *cli.Command {
	return &cli.Command{
		Name:  "install-all",
		Short: i18n.T("Install all available plugins", "安装所有可用的插件"),
		Usage: "plugin install-all",
		Run: func(ctx *cli.Context, args []string) error {
			mgr, err := NewManager()
			if err != nil {
				return err
			}

			return mgr.InstallAll(ctx)
		},
	}
}

func newUninstallCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "uninstall",
		Short: i18n.T("Uninstall a plugin", "卸载插件"),
		Usage: "plugin uninstall --name <plugin_name>",
		Run: func(ctx *cli.Context, args []string) error {
			name := ""
			if v, ok := ctx.Flags().GetValue("name"); ok {
				name = v
			}

			mgr, err := NewManager()
			if err != nil {
				return err
			}

			return mgr.Uninstall(ctx, name)
		},
	}

	cmd.Flags().Add(&cli.Flag{
		Name:         "name",
		Short:        i18n.T("Plugin name to uninstall", "要卸载的插件名称"),
		AssignedMode: cli.AssignedOnce,
		Required:     true,
	})

	return cmd
}

func newUpdateCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "update",
		Short: i18n.T("Update plugin(s)", "更新插件"),
		Usage: "plugin update [--name <plugin_name>]",
		Run: func(ctx *cli.Context, args []string) error {
			mgr, err := NewManager()
			if err != nil {
				return err
			}

			name := ""
			if v, ok := ctx.Flags().GetValue("name"); ok {
				name = v
			}

			if name == "" {
				return mgr.UpdateAll(ctx)
			}

			return mgr.Upgrade(ctx, name)
		},
	}

	cmd.Flags().Add(&cli.Flag{
		Name:         "name",
		Short:        i18n.T("Plugin name to update (optional, update all if not specified)", "要更新的插件名称（可选，不指定则更新所有）"),
		AssignedMode: cli.AssignedOnce,
		Required:     false,
	})

	return cmd
}

func parseInstallArgs(ctx *cli.Context) (name string, names []string, version string, err error) {
	if v, ok := ctx.Flags().GetValue("name"); ok {
		name = v
	}

	if namesFlag := ctx.Flags().Get("names"); namesFlag != nil && namesFlag.IsAssigned() {
		names = namesFlag.GetValues()
	}

	if v, ok := ctx.Flags().GetValue("version"); ok {
		version = v
	}

	return name, names, version, nil
}

func validateInstallArgs(name string, names []string) ([]string, error) {
	if name == "" && len(names) == 0 {
		return nil, fmt.Errorf("either --name or --names flag is required")
	}

	if name != "" && len(names) > 0 {
		return nil, fmt.Errorf("flags --name and --names are mutually exclusive")
	}

	if len(names) > 0 {
		validNames := []string{}
		for _, n := range names {
			if n != "" {
				validNames = append(validNames, n)
			}
		}
		if len(validNames) == 0 {
			return nil, fmt.Errorf("--names requires at least one plugin name")
		}
		return validNames, nil
	}

	return names, nil
}

func executeInstall(ctx *cli.Context, name string, names []string, version string) error {
	mgr, err := NewManager()
	if err != nil {
		return err
	}

	if name != "" {
		return mgr.Install(ctx, name, version)
	}

	return installMultiplePlugins(ctx, mgr, names, version)
}

func installMultiplePlugins(ctx *cli.Context, mgr *Manager, pluginNames []string, version string) error {
	var installed, failed int

	for _, pluginName := range pluginNames {
		cli.Printf(ctx.Stdout(), "Installing %s...\n", pluginName)
		if err := mgr.Install(ctx, pluginName, version); err != nil {
			cli.Printf(ctx.Stderr(), "Failed to install %s: %v\n", pluginName, err)
			failed++
			continue
		}
		installed++
	}

	if installed > 0 {
		cli.Printf(ctx.Stdout(), "Installed: %d\n", installed)
	}
	if failed > 0 {
		cli.Printf(ctx.Stdout(), "Failed: %d\n", failed)
		return fmt.Errorf("%d plugin(s) failed to install", failed)
	}

	return nil
}
