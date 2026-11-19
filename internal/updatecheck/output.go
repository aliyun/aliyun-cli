package updatecheck

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cli/upgrade"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

func outputText(en, zh string) string { return i18n.T(en, zh).Text() }

func writeCheckReport(ctx *cli.Context, r Report) error {
	format, err := upgrade.CheckOutputFormat(ctx)
	if err != nil {
		return err
	}
	if format == cli.HelpOutputJSON {
		encoder := json.NewEncoder(ctx.Stdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(r)
	}
	var out strings.Builder
	status := outputText("Failed", "失败")
	switch r.Status {
	case "complete":
		status = outputText("Complete", "完成")
	case "partial":
		status = outputText("Incomplete; some checks failed", "未完成，部分检查失败")
	}
	fmt.Fprintf(&out, outputText("Update check: %s\n\n", "更新检查：%s\n\n"), status)
	if err := upgrade.WriteCheckSummary(&out, r.CLI); err != nil {
		return err
	}
	fmt.Fprintln(&out, outputText("\nPlugins", "\n插件"))
	switch r.Plugins.Status {
	case "complete":
		if len(r.Plugins.Plugins) == 0 {
			fmt.Fprintln(&out, outputText("  No plugins installed.", "  未安装插件。"))
		}
	case "cached":
		fmt.Fprintln(&out, outputText("  Using verified cache; remote refresh failed.", "  远端刷新失败，使用已验证的缓存结果。"))
	default:
		fmt.Fprintln(&out, outputText("  Plugin update check failed.", "  插件更新检查失败。"))
	}
	for _, p := range r.Plugins.Plugins {
		status := outputText("Unknown", "未知")
		switch p.Status {
		case "update_available":
			status = outputText("Update available", "有可用更新")
		case "up_to_date":
			status = outputText("Up to date", "已是最新版本")
		case "not_in_source":
			status = outputText("Not found in the configured source", "当前插件源未收录")
		case "no_stable_release":
			status = outputText("No stable release", "暂无稳定版本")
		case "unknown_version":
			status = outputText("Cannot compare versions", "无法比较版本")
		case "unsupported_platform":
			status = outputText("Latest release does not support this platform", "最新版本不支持当前平台")
		}
		latest := p.Latest
		if latest == "" {
			latest = "-"
		}
		fmt.Fprintf(&out, "  %s\n", p.Name)
		fmt.Fprintf(&out, outputText("    Current version: %s; latest version: %s\n", "    当前版本：%s；最新版本：%s\n"), p.Current, latest)
		fmt.Fprintf(&out, outputText("    Status: %s\n", "    状态：%s\n"), status)
		if p.InstalledMinCliVersion != "" {
			fmt.Fprintf(&out, outputText("    Installed plugin requires CLI >= %s\n", "    已安装插件要求 CLI >= %s\n"), p.InstalledMinCliVersion)
		}
		if p.MinCliVersion != "" {
			fmt.Fprintf(&out, outputText("    Latest plugin requires CLI >= %s\n", "    最新插件要求 CLI >= %s\n"), p.MinCliVersion)
		}
		if p.RequiresCLIUpgrade {
			fmt.Fprintln(&out, outputText("    Upgrade the CLI first: aliyun upgrade", "    请先升级 CLI：aliyun upgrade"))
		}
		if p.UpdateAvailable {
			fmt.Fprintf(&out, outputText("    Upgrade command: %s\n", "    升级命令：%s\n"), p.Command)
		}
	}
	_, err = io.WriteString(ctx.Stdout(), out.String())
	return err
}
