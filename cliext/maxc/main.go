package maxc

import (
	"text/tabwriter"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

// NewMaxcCommand wires the `aliyun maxc` cliext entrypoint. The parent
// renders aliyun-style help on `--help` / `-h` / bare `help`; everything
// else (including `--help` attached to a maxc subcommand) is forwarded to
// the child process verbatim so the child's argparse can answer.
func NewMaxcCommand() *cli.Command {
	cmd := &cli.Command{
		Name: "maxc",
		Short: i18n.T(
			"MaxCompute CLI for AI agents — structured envelope output, query/job/meta/data tools.",
			"MaxCompute CLI 工具层（供 AI agent 调用） — 结构化输出，覆盖 SQL/作业/元数据/数据采样。"),
		Usage: "maxc <command> [args...] [options...]",
		Sample: "aliyun maxc auth whoami\n" +
			"  aliyun maxc query \"select 1\"\n" +
			"  aliyun maxc meta list-tables",
		Hidden:            false,
		EnableUnknownFlag: true,
		KeepArgs:          true,
		Run: func(ctx *cli.Context, args []string) error {
			c := NewContext(ctx)
			return c.Run(args)
		},
	}
	cmd.Help = func(ctx *cli.Context, _ []string) error {
		cmd.PrintHead(ctx)
		cmd.PrintUsage(ctx)
		cmd.PrintSample(ctx)
		printMaxcCommandGroups(ctx)
		printMaxcEnvVars(ctx)
		cmd.PrintTail(ctx)
		return nil
	}
	return cmd
}

// maxc command groups are owned by the Python child (cli.py); we mirror
// them here so `aliyun maxc --help` shows the same surface without
// shelling out. Keep in sync with src/maxc_cli/cli.py:build_parser.
var maxcCommandGroups = []struct{ name, desc string }{
	{"query", i18n.T("Execute SQL queries (sync/async), estimate cost, explain plans.", "执行 SQL（同步/异步）、估算成本、查看执行计划。").Text()},
	{"job", i18n.T("Manage jobs: status, wait, fetch results, cancel, diagnose.", "作业管理：状态、等待、取数、取消、诊断。").Text()},
	{"meta", i18n.T("Browse metadata: tables, columns, partitions, search.", "元数据：表/列/分区/检索。").Text()},
	{"data", i18n.T("Sample rows and profile column statistics.", "数据抽样和列分布画像。").Text()},
	{"auth", i18n.T("Identity, project access, and permission checks.", "身份、项目访问、权限校验。").Text()},
	{"session", i18n.T("Set the active project/region for this user.", "切换默认 project/region。").Text()},
	{"cache", i18n.T("Inspect or clear the local metadata cache.", "查看或清理本地元数据缓存。").Text()},
	{"agent", i18n.T("Agent-oriented batch operations.", "面向 agent 的批量操作。").Text()},
}

func printMaxcCommandGroups(ctx *cli.Context) {
	cli.Printf(ctx.Stdout(), "\nCommands:\n")
	w := tabwriter.NewWriter(ctx.Stdout(), 8, 0, 1, ' ', 0)
	for _, g := range maxcCommandGroups {
		cli.Printf(w, "  %s\t%s\n", g.name, g.desc)
	}
	w.Flush()
}

func printMaxcEnvVars(ctx *cli.Context) {
	cli.Printf(ctx.Stdout(), "\n%s\n", i18n.T("Environment:", "环境变量：").Text())
	w := tabwriter.NewWriter(ctx.Stdout(), 8, 0, 1, ' ', 0)
	rows := []struct{ k, v string }{
		{"ALIBABA_CLOUD_MAXC_NO_UPDATE_CHECK=1", i18n.T("Skip the daily update check.", "跳过每日的版本更新检查。").Text()},
		{"ALIBABA_CLOUD_MAXC_DOWNLOAD_BASE_URL", i18n.T("Override the OSS download base URL.", "覆盖 OSS 下载源地址（私有镜像/本地 mock）。").Text()},
		{"ALIBABA_CLOUD_MAXC_EXEC_PATH", i18n.T("Use a local maxc binary; bypass install.", "直接使用本地 maxc 可执行文件，跳过下载/安装。").Text()},
	}
	for _, r := range rows {
		cli.Printf(w, "  %s\t%s\n", r.k, r.v)
	}
	w.Flush()
}
