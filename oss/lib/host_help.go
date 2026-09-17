package lib

import (
	"fmt"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

// Render the embedded command's help without inheriting duplicate or unrelated
// OpenAPI flags. Legacy ossutil help remains available to standalone callers.
func printHostOSSHelp(ctx *cli.Context, _ []string) error {
	command := ctx.Command()
	command.PrintHead(ctx)
	if command.Name == "oss" {
		fmt.Fprintln(ctx.Stdout(), "\nUsage:\n  aliyun oss [command] [args...] [options...]")
		command.PrintSubCommands(ctx)
	} else {
		fmt.Fprintf(ctx.Stdout(), "\nUsage:\n  aliyun oss %s\n", command.Usage)
		fmt.Fprintln(ctx.Stdout(), strings.ReplaceAll(command.Long.Text(), "ossutil ", "aliyun oss "))
	}
	fmt.Fprintln(ctx.Stdout(), "\nFlags:")
	for _, flag := range command.Flags().Flags() {
		if flag.Hidden {
			continue
		}
		name := "--" + flag.Name
		if flag.Shorthand != 0 {
			name += ", -" + string(flag.Shorthand)
		}
		fmt.Fprintf(ctx.Stdout(), "  %-30s %s\n", name, flag.Short.Text())
	}
	fmt.Fprintln(ctx.Stdout(), hostOSSWorkflowHelp())
	return nil
}

func hostOSSWorkflowHelp() string {
	return i18n.T(`
Machine workflows:
  # Bounded JSON listing; continue with the returned next_cursor and identical list options.
  aliyun oss ls oss://bucket/prefix/ --cli-output json --limited-num 100
  aliyun oss ls oss://bucket/prefix/ --cli-output json --limited-num 100 --cli-cursor '<next_cursor>'
  # JSONL emits item records followed by a summary; stop when complete=true.
  aliyun oss ls oss://bucket/prefix/ --cli-output jsonl
  # Validate locally, or inspect a read-only plan; neither executes the transfer.
  aliyun oss cp ./file.txt oss://bucket/file.txt --cli-validate
  aliyun oss cp ./file.txt oss://bucket/file.txt --cli-plan
  # After authorization, execute and record failed items in a new JSONL file.
  aliyun oss cp ./src/ oss://bucket/prefix/ -r -f --cli-non-interactive --cli-failure-report ./failed.jsonl
  aliyun oss rm oss://bucket/file.txt --cli-plan
  # AI errors do not authorize writes or confirmation.
  aliyun oss ls oss://bucket/ --cli-ai-mode

JSON/JSONL success output is supported by ls; cat always emits raw bytes.
AI-mode stderr may include progress text before the final JSON error envelope.
Parse the final envelope; do not assume every stderr line is JSON.
Inspect failed items and operation state before retrying writes.`, `
机器工作流示例：
  # 有界 JSON 列表：使用返回的 next_cursor 续页，保持其他列表参数不变。
  aliyun oss ls oss://bucket/prefix/ --cli-output json --limited-num 100
  aliyun oss ls oss://bucket/prefix/ --cli-output json --limited-num 100 --cli-cursor '<next_cursor>'
  # JSONL 逐条输出，最后一条为 summary；complete=true 时结束分页。
  aliyun oss ls oss://bucket/prefix/ --cli-output jsonl
  # 本地校验或只读计划，均不执行传输。
  aliyun oss cp ./file.txt oss://bucket/file.txt --cli-validate
  aliyun oss cp ./file.txt oss://bucket/file.txt --cli-plan
  # 获得授权后执行，并将失败项写入新的 JSONL 文件。
  aliyun oss cp ./src/ oss://bucket/prefix/ -r -f --cli-non-interactive --cli-failure-report ./failed.jsonl
  aliyun oss rm oss://bucket/file.txt --cli-plan
  # AI 错误输出不代表已授权写操作或确认。
  aliyun oss ls oss://bucket/ --cli-ai-mode

JSON/JSONL 成功输出适用于 ls；cat 始终输出原始字节。
AI 模式 stderr 可能在末尾 JSON 错误 envelope 前包含进度文本。
应解析末尾 envelope，不能假定 stderr 每行都是 JSON。
重试写操作前，先检查失败清单与实际操作状态。`).Text()
}
