package upgrade

import (
	"fmt"
	"io"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

// CheckOutputFormat uses the inherited host output flag and its existing
// validation, without requiring a Help operation or registering another flag.
func CheckOutputFormat(ctx *cli.Context) (cli.HelpOutput, error) {
	value, assigned := ctx.Flags().Get(cli.CLIOutputFlagName).GetValue()
	if !assigned {
		return cli.HelpOutputText, nil
	}
	opts, err := cli.ParseHelpOptions([]string{"--" + cli.CLIOutputFlagName, value})
	return opts.Output, err
}

// WriteCheckSummary renders the CLI part of a read-only update check.
func WriteCheckSummary(w io.Writer, r UpdateResult) error {
	var out strings.Builder
	text := func(en, zh string) string { return i18n.T(en, zh).Text() }
	version := func(v string) string {
		if v == "" {
			return "-"
		}
		return v
	}
	status := text("Check failed", "检查失败")
	switch r.Status {
	case "update_available":
		status = text("Update available", "有可用更新")
	case "up_to_date":
		status = text("Up to date", "已是最新版本")
	case "development_build":
		status = text("Development build; version comparison skipped", "开发版本，跳过版本比较")
	case "no_compatible_release":
		status = text("No compatible upgrade release", "暂无兼容的升级版本")
	}
	fmt.Fprintln(&out, "Alibaba Cloud CLI")
	fmt.Fprintf(&out, text("  Current version: %s\n", "  当前版本：%s\n"), version(r.Current))
	fmt.Fprintf(&out, text("  Latest version: %s\n", "  最新版本：%s\n"), version(r.Latest))
	fmt.Fprintf(&out, text("  Status: %s\n", "  状态：%s\n"), status)
	if r.UpdateAvailable {
		fmt.Fprintf(&out, text("  Upgrade target: %s\n", "  本次可升级至：%s\n"), version(r.Target))
		fmt.Fprintf(&out, text("  Upgrade command: %s\n", "  升级命令：%s\n"), r.Command)
		if r.Target != "" && r.Latest != "" && r.Target != r.Latest {
			fmt.Fprintln(&out, text("  Upgrade again after installing this target to continue toward the latest release.", "  升级到该版本后，再次运行升级命令以继续升级至最新版本。"))
		}
	}
	_, err := io.WriteString(w, out.String())
	return err
}
