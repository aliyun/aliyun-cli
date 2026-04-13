package plugin

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
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
	cmd.AddSubCommand(newShowCommand())
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

			names := make([]string, 0, len(manifest.Plugins))
			for name := range manifest.Plugins {
				names = append(names, name)
			}
			sort.Strings(names)

			w := tabwriter.NewWriter(ctx.Stdout(), 20, 0, 3, ' ', 0)
			fmt.Fprintln(w, "Name\tVersion\tDescription")
			fmt.Fprintln(w, "----\t-------\t-----------")

			for _, name := range names {
				p := manifest.Plugins[name]
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
			return runSearch(ctx, args)
		},
	}
}

func runSearch(ctx *cli.Context, args []string) error {
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

	results, err := mgr.SearchPluginsByCommandPrefix(commandName)
	if err != nil {
		return err
	}

	if len(results) == 0 {
		return fmt.Errorf("no plugin found for command prefix: %s", commandName)
	}

	return displaySearchResults(ctx, mgr, commandName, results)
}

func newShowCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "show",
		Short: i18n.T("Show details of an installed plugin", "显示已安装插件的详细信息"),
		Usage: "show --name <plugin_name>",
		Run: func(ctx *cli.Context, args []string) error {
			name := ""
			if v, ok := ctx.Flags().GetValue("name"); ok {
				name = v
			}
			if strings.TrimSpace(name) == "" {
				return fmt.Errorf("plugin name is required (use --name)")
			}

			mgr, err := NewManager()
			if err != nil {
				return err
			}

			actualName, p, err := mgr.findLocalPlugin(name)
			if err != nil {
				return err
			}

			return displayInstalledPluginDetails(ctx, actualName, p)
		},
	}

	cmd.Flags().Add(&cli.Flag{
		Name:         "name",
		Short:        i18n.T("Installed plugin name", "已安装的插件名称"),
		AssignedMode: cli.AssignedOnce,
	})

	return cmd
}

func displayInstalledPluginDetails(ctx *cli.Context, canonicalName string, p *LocalPlugin) error {
	out := ctx.Stdout()
	fmt.Fprintf(out, "Name:\t%s\n", canonicalName)
	fmt.Fprintf(out, "Version:\t%s\n", p.Version)
	if pc := strings.TrimSpace(p.ProductCode); pc != "" {
		fmt.Fprintf(out, "Product code:\t%s\n", pc)
	}
	// fmt.Fprintf(out, "Path:\t%s\n", p.Path)
	if p.Command != "" {
		fmt.Fprintf(out, "Command:\t%s\n", p.Command)
	}
	if p.ShortDescription != "" {
		fmt.Fprintf(out, "Short description:\t%s\n", p.ShortDescription)
	}
	if p.Description != "" {
		fmt.Fprintf(out, "Description:\t%s\n", p.Description)
	}
	if av, err := readPluginAPIVersionsFromDir(p.Path); err == nil && apiVersionsPresent(av) {
		writePluginAPIVersionsSection(out, av)
	}
	if p.Inner {
		fmt.Fprintf(out, "Inner:\t%v\n", p.Inner)
	}
	return nil
}

func readPluginAPIVersionsFromDir(pluginDir string) (*PluginAPIVersions, error) {
	path := filepath.Join(pluginDir, "manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var wrapper struct {
		APIVersions *PluginAPIVersions `json:"apiVersions"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, err
	}
	return wrapper.APIVersions, nil
}

func apiVersionsPresent(v *PluginAPIVersions) bool {
	if v == nil {
		return false
	}
	return v.Default != "" || len(v.Supported) > 0
}

func writePluginAPIVersionsSection(out io.Writer, v *PluginAPIVersions) {
	if v.Default != "" {
		fmt.Fprintf(out, "API default:\t%s\n", v.Default)
	}
	if len(v.Supported) > 0 {
		fmt.Fprintf(out, "API supported:\t%s\n", strings.Join(v.Supported, ", "))
	}
}

func newInstallCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "install",
		Short: i18n.T("Install a plugin (from remote index or local archive)", "安装插件（远程索引或本地包）"),
		Usage: "install --names <plugin_name> [<plugin2> ...] [--version <version>] [--enable-pre] | install --source <path-to-archive> [--version <version>]",
		Run: func(ctx *cli.Context, args []string) error {
			names, source, version, enablePre, err := parseInstallArgs(ctx)
			if err != nil {
				return err
			}

			validatedNames, err := validateInstallArgs(names, source)
			if err != nil {
				return err
			}

			return executeInstall(ctx, validatedNames, source, version, enablePre)
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
		Short:        i18n.T("Specify plugin version (remote) or must match manifest when using --source", "指定插件版本（远程）；与 --source 连用时须与 manifest 一致"),
		AssignedMode: cli.AssignedOnce,
		DefaultValue: "",
	})

	cmd.Flags().Add(&cli.Flag{
		Name:         "enable-pre",
		Short:        i18n.T("Allow installing pre-release versions", "允许安装预发布版本"),
		AssignedMode: cli.AssignedNone,
	})

	cmd.Flags().Add(&cli.Flag{
		Name:         "source",
		Short:        i18n.T("Install from a local plugin archive (.zip, .tar.gz, .tgz)", "从本地插件包安装（.zip、.tar.gz、.tgz）"),
		AssignedMode: cli.AssignedOnce,
		DefaultValue: "",
	})

	return cmd
}

func newInstallAllCommand() *cli.Command {
	cmd := &cli.Command{
		Name:   "install-all",
		Short:  i18n.T("Install all available plugins", "安装所有可用的插件"),
		Usage:  "install-all [--enable-pre]",
		Hidden: true, // 不推荐使用
		Run: func(ctx *cli.Context, args []string) error {
			mgr, err := NewManager()
			if err != nil {
				return err
			}

			enablePre := false
			if enablePreFlag := ctx.Flags().Get("enable-pre"); enablePreFlag != nil && enablePreFlag.IsAssigned() {
				enablePre = true
			}

			return mgr.InstallAll(ctx, enablePre)
		},
	}

	cmd.Flags().Add(&cli.Flag{
		Name:         "enable-pre",
		Short:        i18n.T("Allow installing pre-release versions", "允许安装预发布版本"),
		AssignedMode: cli.AssignedNone,
	})

	return cmd
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

func parseInstallArgs(ctx *cli.Context) (names []string, source string, version string, enablePre bool, err error) {
	if namesFlag := ctx.Flags().Get("names"); namesFlag != nil && namesFlag.IsAssigned() {
		names = namesFlag.GetValues()
	}

	if v, ok := ctx.Flags().GetValue("source"); ok {
		source = v
	}

	if v, ok := ctx.Flags().GetValue("version"); ok {
		version = v
	}

	if enablePreFlag := ctx.Flags().Get("enable-pre"); enablePreFlag != nil && enablePreFlag.IsAssigned() {
		enablePre = true
	}

	return names, source, version, enablePre, nil
}

func validateInstallArgs(names []string, source string) ([]string, error) {
	if strings.TrimSpace(source) != "" {
		if len(names) > 0 {
			return nil, fmt.Errorf("--names cannot be used together with --source")
		}
		return nil, nil
	}

	if len(names) == 0 {
		return nil, fmt.Errorf("either --names or --source is required")
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

func executeInstall(ctx *cli.Context, names []string, source, version string, enablePre bool) error {
	mgr, err := NewManager()
	if err != nil {
		return err
	}

	if strings.TrimSpace(source) != "" {
		return mgr.InstallFromLocalFile(ctx, source, version)
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

func displaySearchResults(ctx *cli.Context, mgr *Manager, prefix string, results map[string][]string) error {
	keys := make([]string, 0, len(results))
	for k := range results {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	index, err := mgr.GetIndex()
	if err != nil {
		// If remote index fails, we can't show latest version/description for uninstalled plugins
		// but we can still show installed status
		cli.Printf(ctx.Stderr(), "Warning: Failed to fetch remote index: %v\n", err)
		index = &Index{Plugins: []PluginInfo{}}
	}

	localManifest, err := mgr.GetLocalManifest()
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(ctx.Stdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Plugin\tLatest Version\tPreview\tStatus\tLocal Version\tDescription")
	fmt.Fprintln(w, "------\t--------------\t-------\t------\t-------------\t-----------")

	for _, pluginName := range keys {
		status := "Not installed"
		localVersion := "-"

		if localPlugin, ok := localManifest.Plugins[pluginName]; ok {
			status = "Installed"
			localVersion = localPlugin.Version
		}

		var targetPlugin *PluginInfo
		for _, p := range index.Plugins {
			if p.Name == pluginName {
				targetPlugin = &p
				break
			}
		}

		latestVersion := "N/A"
		preview := "No"
		description := ""

		if targetPlugin != nil {
			description = targetPlugin.Description
			if v, err := getLatestVersion(targetPlugin, true); err == nil {
				latestVersion = v
				if isPrerelease(v) {
					preview = "Yes"
				}
			}
		} else if localPlugin, ok := localManifest.Plugins[pluginName]; ok {
			// Fallback to local description if not in remote index
			description = localPlugin.Description
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			pluginName,
			latestVersion,
			preview,
			status,
			localVersion,
			description)
	}
	w.Flush()
	return nil
}
