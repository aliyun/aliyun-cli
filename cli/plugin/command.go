package plugin

import (
	"fmt"
	"sort"
	"text/tabwriter"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

func NewPluginCommand() *cli.Command {
	cmd := &cli.Command{
		Name:                   "plugin",
		Short:                  i18n.T("Manage plugins", "管理插件"),
		Usage:                  "plugin <command> [args]",
		DisablePersistentFlags: true,
		Run: func(ctx *cli.Context, args []string) error {
			return cli.NewErrorWithTip(fmt.Errorf("command missing"), "Use `aliyun plugin --help` for more information.")
		},
	}

	cmd.AddSubCommand(newListCommand())
	cmd.AddSubCommand(newListRemoteCommand())
	cmd.AddSubCommand(newSearchCommand())
	cmd.AddSubCommand(newInstallCommand())
	cmd.AddSubCommand(newInstallAllCommand())
	cmd.AddSubCommand(newUninstallCommand())
	cmd.AddSubCommand(newUpdateCommand())

	return cmd
}

func newListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Short: i18n.T("List installed plugins", "列出已安装的插件"),
		Usage: "list",
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

func newListRemoteCommand() *cli.Command {
	return &cli.Command{
		Name:  "list-remote",
		Short: i18n.T("List available plugins from remote index", "列出远程索引中可用的插件"),
		Usage: "list-remote",
		Run: func(ctx *cli.Context, args []string) error {
			mgr, err := NewManager()
			if err != nil {
				return err
			}

			index, err := mgr.GetIndex()
			if err != nil {
				return fmt.Errorf("failed to fetch remote plugin index: %w", err)
			}

			localManifest, err := mgr.GetLocalManifest()
			if err != nil {
				return err
			}

			return displayRemotePlugins(ctx, index, localManifest)
		},
	}
}

func newSearchCommand() *cli.Command {
	return &cli.Command{
		Name:  "search",
		Short: i18n.T("Search plugin by command name", "根据命令名搜索插件"),
		Usage: "search <command-name>",
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("command name is required")
			}

			commandName := args[0]
			if commandName == "" {
				return fmt.Errorf("command name cannot be empty")
			}

			mgr, err := NewManager()
			if err != nil {
				return err
			}

			pluginName, err := mgr.FindPluginByCommand(commandName)
			if err != nil {
				return err
			}

			return displaySearchResult(ctx, mgr, commandName, pluginName)
		},
	}
}

func newInstallCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "install",
		Short: i18n.T("Install a plugin", "安装插件"),
		Usage: "install --names <plugin_name> [<plugin2> ...] [--version <version>] [--enable-pre]",
		Run: func(ctx *cli.Context, args []string) error {
			names, version, enablePre, err := parseInstallArgs(ctx)
			if err != nil {
				return err
			}

			validatedNames, err := validateInstallArgs(names)
			if err != nil {
				return err
			}

			return executeInstall(ctx, validatedNames, version, enablePre)
		},
	}

	namesFlag := &cli.Flag{
		Name:         "names",
		Short:        i18n.T("Plugin name(s) to install (can specify one or multiple)", "要安装的插件名称（可指定一个或多个）"),
		AssignedMode: cli.AssignedRepeatable,
	}
	cmd.Flags().Add(namesFlag)

	cmd.Flags().Add(&cli.Flag{
		Name:         "version",
		Short:        i18n.T("Specify plugin version", "指定插件版本"),
		AssignedMode: cli.AssignedOnce,
		DefaultValue: "",
	})

	cmd.Flags().Add(&cli.Flag{
		Name:         "enable-pre",
		Short:        i18n.T("Allow installing pre-release versions", "允许安装预发布版本"),
		AssignedMode: cli.AssignedNone,
	})

	return cmd
}

func newInstallAllCommand() *cli.Command {
	return &cli.Command{
		Name:   "install-all",
		Short:  i18n.T("Install all available plugins", "安装所有可用的插件"),
		Usage:  "install-all",
		Hidden: true, // 安装所有可用插件，不推荐使用
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
		Usage: "uninstall --name <plugin_name>",
		Run: func(ctx *cli.Context, args []string) error {
			name := ""
			if v, ok := ctx.Flags().GetValue("name"); ok {
				name = v
			}

			if name == "" {
				return fmt.Errorf("plugin name is required")
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
	})

	return cmd
}

func newUpdateCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "update",
		Short: i18n.T("Update plugin(s)", "更新插件"),
		Usage: "plugin update [--name <plugin_name>] [--enable-pre]",
		Run: func(ctx *cli.Context, args []string) error {
			mgr, err := NewManager()
			if err != nil {
				return err
			}

			name := ""
			if v, ok := ctx.Flags().GetValue("name"); ok {
				name = v
			}

			enablePre := false
			if enablePreFlag := ctx.Flags().Get("enable-pre"); enablePreFlag != nil && enablePreFlag.IsAssigned() {
				enablePre = true
			}

			if name == "" {
				return mgr.UpdateAll(ctx, enablePre)
			}

			return mgr.Upgrade(ctx, name, enablePre)
		},
	}

	cmd.Flags().Add(&cli.Flag{
		Name:         "name",
		Short:        i18n.T("Plugin name to update (optional, update all if not specified)", "要更新的插件名称（可选，不指定则更新所有）"),
		AssignedMode: cli.AssignedOnce,
		Required:     false,
	})

	cmd.Flags().Add(&cli.Flag{
		Name:         "enable-pre",
		Short:        i18n.T("Allow updating to pre-release versions", "允许更新到预发布版本"),
		AssignedMode: cli.AssignedNone,
	})

	return cmd
}

func parseInstallArgs(ctx *cli.Context) (names []string, version string, enablePre bool, err error) {
	if namesFlag := ctx.Flags().Get("names"); namesFlag != nil && namesFlag.IsAssigned() {
		names = namesFlag.GetValues()
	}

	if v, ok := ctx.Flags().GetValue("version"); ok {
		version = v
	}

	if enablePreFlag := ctx.Flags().Get("enable-pre"); enablePreFlag != nil && enablePreFlag.IsAssigned() {
		enablePre = true
	}

	return names, version, enablePre, nil
}

func validateInstallArgs(names []string) ([]string, error) {
	if len(names) == 0 {
		return nil, fmt.Errorf("--names flag is required")
	}

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

func executeInstall(ctx *cli.Context, names []string, version string, enablePre bool) error {
	mgr, err := NewManager()
	if err != nil {
		return err
	}

	if len(names) == 1 {
		return mgr.Install(ctx, names[0], version, enablePre)
	}

	return mgr.InstallMultiple(ctx, names, version, enablePre)
}

func displayRemotePlugins(ctx *cli.Context, index *Index, localManifest *LocalManifest) error {
	totalPlugins := len(index.Plugins)
	cli.Printf(ctx.Stdout(), "Total plugins available: %d\n\n", totalPlugins)

	if totalPlugins == 0 {
		cli.Printf(ctx.Stdout(), "No plugins available in remote index.\n")
		return nil
	}

	type pluginWithStatus struct {
		plugin      PluginInfo
		isInstalled bool
		localPlugin LocalPlugin
	}

	plugins := make([]pluginWithStatus, 0, totalPlugins)
	var installedCount int

	for _, plugin := range index.Plugins {
		localPlugin, isInstalled := localManifest.Plugins[plugin.Name]
		if isInstalled {
			installedCount++
		}
		plugins = append(plugins, pluginWithStatus{
			plugin:      plugin,
			isInstalled: isInstalled,
			localPlugin: localPlugin,
		})
	}

	sort.Slice(plugins, func(i, j int) bool {
		if plugins[i].isInstalled != plugins[j].isInstalled {
			return plugins[i].isInstalled // installed first
		}
		return plugins[i].plugin.Name < plugins[j].plugin.Name
	})

	w := tabwriter.NewWriter(ctx.Stdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Name\tLatest Version\tPreview\tStatus\tLocal Version\tDescription")
	fmt.Fprintln(w, "----\t--------------\t-------\t------\t-------------\t-----------")

	for _, p := range plugins {
		status := "Not installed"
		localVersion := "-"
		if p.isInstalled {
			status = "Installed"
			localVersion = p.localPlugin.Version
		}

		latestVersion, err := getLatestVersion(&p.plugin, true)
		if err != nil {
			latestVersion = "N/A"
		}

		preview := "No"
		if latestVersion != "N/A" && isPrerelease(latestVersion) {
			preview = "Yes"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			p.plugin.Name,
			latestVersion,
			preview,
			status,
			localVersion,
			p.plugin.Description)
	}
	w.Flush()

	return nil
}

func displaySearchResult(ctx *cli.Context, mgr *Manager, commandName, pluginName string) error {
	cli.Printf(ctx.Stdout(), "Command: %s\n", commandName)
	cli.Printf(ctx.Stdout(), "Plugin: %s\n\n", pluginName)

	localManifest, err := mgr.GetLocalManifest()
	if err != nil {
		return err
	}

	localPlugin, isInstalled := localManifest.Plugins[pluginName]
	if isInstalled {
		cli.Printf(ctx.Stdout(), "Status: Installed\n")
		cli.Printf(ctx.Stdout(), "Local Version: %s\n", localPlugin.Version)
		if localPlugin.Description != "" {
			cli.Printf(ctx.Stdout(), "Description: %s\n", localPlugin.Description)
		}
	} else {
		cli.Printf(ctx.Stdout(), "Status: Not installed\n")

		index, err := mgr.GetIndex()
		if err == nil {
			// Only show remote info if we can successfully fetch the index
			for _, plugin := range index.Plugins {
				if plugin.Name == pluginName {
					latestVersion, err := getLatestVersion(&plugin, true)
					if err == nil {
						cli.Printf(ctx.Stdout(), "Latest Version: %s\n", latestVersion)

						if isPrerelease(latestVersion) {
							cli.Printf(ctx.Stdout(), "Note: This is a pre-release version\n")
						}
					}
					if plugin.Description != "" {
						cli.Printf(ctx.Stdout(), "Description: %s\n", plugin.Description)
					}

					if latestVersion != "" && isPrerelease(latestVersion) {
						cli.Printf(ctx.Stdout(), "\nTo install: aliyun plugin install --names %s --enable-pre\n", pluginName)
					} else {
						cli.Printf(ctx.Stdout(), "\nTo install: aliyun plugin install --names %s\n", pluginName)
					}
					break
				}
			}
		}
	}

	return nil
}
