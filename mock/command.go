package mock

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	sysmock "github.com/aliyun/aliyun-cli/v3/sysconfig/mock"
)

func NewMockCommand(defaultConfigDir func() string) *cli.Command {
	cmd := &cli.Command{
		Name:  "mock",
		Usage: "mock <add|import|remove|list|clear|path>",
		Short: i18n.T("manage CLI mock records", "管理 CLI mock 记录"),
		Help: func(ctx *cli.Context, args []string) error {
			printRootHelp(ctx)
			return nil
		},
	}

	cmd.AddSubCommand(newAddCommand(defaultConfigDir))
	cmd.AddSubCommand(newImportCommand(defaultConfigDir))
	cmd.AddSubCommand(newRemoveCommand(defaultConfigDir))
	cmd.AddSubCommand(newListCommand(defaultConfigDir))
	cmd.AddSubCommand(newClearCommand(defaultConfigDir))
	cmd.AddSubCommand(newPathCommand(defaultConfigDir))
	return cmd
}

func printRootHelp(ctx *cli.Context) {
	cmd := ctx.Command()
	cmd.PrintHead(ctx)
	cmd.PrintUsage(ctx)
	cmd.PrintSubCommands(ctx)
	cmd.PrintFlags(ctx)
	cli.Printf(ctx.Stdout(), "%s\n", i18n.T(`
Environment:
  Mocking is disabled by default. Set the following environment variables before
  running the command you want to mock:

  export ALIBABA_CLOUD_CLI_MOCK=true
  export ALIBABA_CLOUD_CLI_MOCK_PATH=$(aliyun mock path)

Workflow:
  1. Add or import mock records with aliyun mock add/import.
  2. Enable the environment variables above in the same shell.
  3. Run the target command, for example:
     aliyun ecs DescribeRegions
  4. Disable mocking when finished:
     unset ALIBABA_CLOUD_CLI_MOCK`, `
环境变量:
  mock 默认不生效。运行需要被 mock 的命令前，先在当前 shell 配置:

  export ALIBABA_CLOUD_CLI_MOCK=true
  export ALIBABA_CLOUD_CLI_MOCK_PATH=$(aliyun mock path)

使用流程:
  1. 使用 aliyun mock add/import 添加或导入 mock 记录。
  2. 在同一个 shell 中启用上面的环境变量。
  3. 执行需要被 mock 的目标命令，例如:
     aliyun ecs DescribeRegions
  4. 使用结束后关闭 mock:
     unset ALIBABA_CLOUD_CLI_MOCK`).Text())
	cmd.PrintTail(ctx)
}

func newAddCommand(defaultConfigDir func() string) *cli.Command {
	cmd := &cli.Command{
		Name:  "add",
		Usage: "add --name <name> --cmd <rule> --exit-code <code> --times <count> [--stdout <text>] [--stderr <text>]",
		Short: i18n.T("add one mock record", "添加一条 mock 记录"),
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}

			record, err := recordFromFlags(ctx)
			if err != nil {
				return err
			}
			records := normalizeCommands(ctx, []sysmock.Record{record})

			return sysmock.AppendLenient(sysmock.ResolvePath(defaultConfigDir), records)
		},
	}
	addRecordFlags(cmd)
	return cmd
}

func newImportCommand(defaultConfigDir func() string) *cli.Command {
	cmd := &cli.Command{
		Name:  "import",
		Usage: "import --file <path>",
		Short: i18n.T("import and replace JSON mock records", "导入并覆盖 JSON mock 记录"),
		Help: func(ctx *cli.Context, args []string) error {
			printImportHelp(ctx)
			return nil
		},
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}

			inputPath, err := requiredFlag(ctx, "file")
			if err != nil {
				return err
			}
			data, err := os.ReadFile(inputPath)
			if err != nil {
				return err
			}
			records, err := sysmock.DecodeInput(data)
			if err != nil {
				return err
			}
			records = normalizeCommands(ctx, records)
			return sysmock.Save(sysmock.ResolvePath(defaultConfigDir), records)
		},
	}
	cmd.Flags().Add(&cli.Flag{
		Name:         "file",
		AssignedMode: cli.AssignedOnce,
		Short:        i18n.T("JSON mock input file", "JSON mock 输入文件"),
	})
	return cmd
}

func addRecordFlags(cmd *cli.Command) {
	for _, flag := range []struct {
		name  string
		short *i18n.Text
	}{
		{"name", i18n.T("mock record name", "mock 记录名称")},
		{"cmd", i18n.T("command match rule", "命令匹配规则")},
		{"exit-code", i18n.T("mock command exit code", "模拟命令退出码")},
		{"stdout", i18n.T("mock command stdout", "模拟命令标准输出")},
		{"stderr", i18n.T("mock command stderr", "模拟命令标准错误")},
		{"times", i18n.T("match count; 0 means unlimited", "匹配次数；0 表示不限次数")},
	} {
		cmd.Flags().Add(&cli.Flag{
			Name:         flag.name,
			AssignedMode: cli.AssignedOnce,
			Short:        flag.short,
		})
	}
}

func recordFromFlags(ctx *cli.Context) (sysmock.Record, error) {
	if missing := missingRequiredFlags(ctx, []string{"name", "cmd", "exit-code", "stdout", "stderr", "times"}); len(missing) > 0 {
		return sysmock.Record{}, fmt.Errorf("%s", addMissingFlagsMessage(missing))
	}

	name, err := requiredFlag(ctx, "name")
	if err != nil {
		return sysmock.Record{}, err
	}
	cmd, err := requiredFlag(ctx, "cmd")
	if err != nil {
		return sysmock.Record{}, err
	}
	exitCodeText, err := requiredFlag(ctx, "exit-code")
	if err != nil {
		return sysmock.Record{}, err
	}
	stdout, err := requiredFlag(ctx, "stdout")
	if err != nil {
		return sysmock.Record{}, err
	}
	stderr, err := requiredFlag(ctx, "stderr")
	if err != nil {
		return sysmock.Record{}, err
	}
	timesText, err := requiredFlag(ctx, "times")
	if err != nil {
		return sysmock.Record{}, err
	}

	exitCode, err := strconv.Atoi(exitCodeText)
	if err != nil {
		return sysmock.Record{}, fmt.Errorf("invalid --exit-code %q", exitCodeText)
	}
	times, err := strconv.Atoi(timesText)
	if err != nil {
		return sysmock.Record{}, fmt.Errorf("invalid --times %q", timesText)
	}
	record := sysmock.Record{
		Name:     name,
		Cmd:      cmd,
		ExitCode: exitCode,
		Stdout:   stdout,
		Stderr:   stderr,
		Times:    times,
	}
	if err := sysmock.ValidateRecord(record); err != nil {
		return sysmock.Record{}, err
	}
	return record, nil
}

func normalizeCommands(ctx *cli.Context, records []sysmock.Record) []sysmock.Record {
	for i := range records {
		if normalized, ok := stripAliyunPrefix(records[i].Cmd); ok {
			cli.Errorf(ctx.Stderr(), "WARNING: removed leading aliyun from mock command %q; stored as %q\n", records[i].Cmd, normalized)
			records[i].Cmd = normalized
		}
	}
	return records
}

func stripAliyunPrefix(cmd string) (string, bool) {
	const prefix = "aliyun "
	if !strings.HasPrefix(cmd, prefix) {
		return cmd, false
	}
	return strings.TrimLeft(cmd[len(prefix):], " \t"), true
}

func missingRequiredFlags(ctx *cli.Context, names []string) []string {
	missing := make([]string, 0)
	for _, name := range names {
		flag := ctx.Flags().Get(name)
		if flag == nil || !flag.IsAssigned() {
			missing = append(missing, "--"+name)
			continue
		}
		value, _ := flag.GetValue()
		if value == "" {
			missing = append(missing, "--"+name)
		}
	}
	return missing
}

func addMissingFlagsMessage(missing []string) string {
	return fmt.Sprintf(`missing required flags: %s

Example:
  aliyun mock add --name mock-ecs-version --cmd 'ecs *' --exit-code 0 --stdout 'ecs 1.0.0\n' --stderr 'mock stderr\n' --times 10`, strings.Join(missing, ", "))
}

func requiredFlag(ctx *cli.Context, name string) (string, error) {
	flag := ctx.Flags().Get(name)
	if flag == nil || !flag.IsAssigned() {
		return "", fmt.Errorf("missing --%s", name)
	}
	value, _ := flag.GetValue()
	if value == "" {
		return "", fmt.Errorf("missing --%s", name)
	}
	return value, nil
}

func printImportHelp(ctx *cli.Context) {
	cmd := ctx.Command()
	cmd.PrintHead(ctx)
	cmd.PrintUsage(ctx)
	cmd.PrintFlags(ctx)
	cli.Printf(ctx.Stdout(), "%s\n", i18n.T(`
JSON input format:
  The file can contain one flat mock record object or an array of mock records.
  Import replaces all existing mock records after the file is validated.

Fields:
  name      required record name
  cmd       required command match rule; "*" and "?" wildcards are supported
  exit_code required mocked command exit code
  stdout    required mocked command stdout
  stderr    required mocked command stderr
  times     required match count; 0 means unlimited

Example:
[
  {
    "name": "mock-ecs-all-operation",
    "cmd": "ecs *",
    "exit_code": 0,
    "stdout": "ecs 1.0.0\n",
    "stderr": "",
    "times": 10
  }
]`, `
JSON 输入格式:
  文件可以包含单个扁平 mock 记录对象，也可以包含 mock 记录数组。
  import 会先校验文件，校验通过后覆盖所有现有 mock 记录。

字段:
  name      必填的记录名称
  cmd       必填的命令匹配规则，支持 "*" 和 "?" 通配符
  exit_code 必填的模拟命令退出码
  stdout    必填的模拟命令标准输出
  stderr    必填的模拟命令标准错误
  times     必填的匹配次数；0 表示不限次数

示例:
[
  {
    "name": "mock-ecs-all-operation",
    "cmd": "ecs *",
    "exit_code": 0,
    "stdout": "ecs 1.0.0\n",
    "stderr": "",
    "times": 10
  }
]`).Text())
	cmd.PrintTail(ctx)
}

func newRemoveCommand(defaultConfigDir func() string) *cli.Command {
	cmd := &cli.Command{
		Name:  "remove",
		Usage: "remove --name <name> | --index <zero-based-index>",
		Short: i18n.T("remove one mock record by name or index", "按名称或索引删除一条 mock 记录"),
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}

			nameFlag := ctx.Flags().Get("name")
			indexFlag := ctx.Flags().Get("index")
			hasName := nameFlag != nil && nameFlag.IsAssigned()
			hasIndex := indexFlag != nil && indexFlag.IsAssigned()
			if hasName == hasIndex {
				return fmt.Errorf("specify exactly one of --name <name> or --index <zero-based-index>")
			}

			path := sysmock.ResolvePath(defaultConfigDir)
			if hasName {
				name, _ := nameFlag.GetValue()
				if name == "" {
					return fmt.Errorf("missing --name <name>")
				}
				return sysmock.RemoveByName(path, name)
			}

			value, _ := indexFlag.GetValue()
			index, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("invalid --index %q", value)
			}
			return sysmock.RemoveByIndex(path, index)
		},
	}
	cmd.Flags().Add(&cli.Flag{
		Name:         "name",
		AssignedMode: cli.AssignedOnce,
		Short:        i18n.T("mock record name to remove", "要删除的 mock 记录名称"),
	})
	cmd.Flags().Add(&cli.Flag{
		Name:         "index",
		AssignedMode: cli.AssignedOnce,
		Short:        i18n.T("zero-based mock record index to remove", "要删除的 mock 记录索引，从 0 开始"),
	})
	return cmd
}

func newListCommand(defaultConfigDir func() string) *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "list",
		Short: i18n.T("list mock records", "列出 mock 记录"),
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			records := sysmock.LoadLenient(sysmock.ResolvePath(defaultConfigDir))
			data, err := json.MarshalIndent(records, "", "  ")
			if err != nil {
				return err
			}
			cli.Printf(ctx.Stdout(), "%s\n", data)
			return nil
		},
	}
}

func newClearCommand(defaultConfigDir func() string) *cli.Command {
	return &cli.Command{
		Name:  "clear",
		Usage: "clear",
		Short: i18n.T("clear mock records", "清除 mock 记录"),
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			return sysmock.Clear(sysmock.ResolvePath(defaultConfigDir))
		},
	}
}

func newPathCommand(defaultConfigDir func() string) *cli.Command {
	return &cli.Command{
		Name:  "path",
		Usage: "path",
		Short: i18n.T("print mock records path", "打印 mock 记录路径"),
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			cli.Printf(ctx.Stdout(), "%s\n", sysmock.ResolvePath(defaultConfigDir))
			return nil
		},
	}
}
