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

			w := tabwriter.NewWriter(cli.DefaultStdoutWriter(), 8, 0, 1, ' ', 0)
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
	return &cli.Command{
		Name:  "install",
		Short: i18n.T("Install a plugin", "安装插件"),
		Usage: "plugin install <plugin_name> [version]",
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("plugin name required")
			}
			name := args[0]
			version := ""
			if len(args) > 1 {
				version = args[1]
			}

			mgr, err := NewManager()
			if err != nil {
				return err
			}

			return mgr.Install(name, version)
		},
	}
}

func newUninstallCommand() *cli.Command {
	return &cli.Command{
		Name:  "uninstall",
		Short: i18n.T("Uninstall a plugin", "卸载插件"),
		Usage: "plugin uninstall <plugin_name>",
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("plugin name required")
			}
			name := args[0]

			mgr, err := NewManager()
			if err != nil {
				return err
			}

			return mgr.Uninstall(name)
		},
	}
}

func newUpdateCommand() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Short: i18n.T("Update a plugin", "更新插件"),
		Usage: "plugin update <plugin_name>",
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("plugin name required")
			}
			name := args[0]

			mgr, err := NewManager()
			if err != nil {
				return err
			}

			return mgr.Upgrade(name)
		},
	}
}
