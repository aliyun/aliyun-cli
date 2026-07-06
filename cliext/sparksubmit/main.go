package sparksubmit

import (
	"text/tabwriter"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

func NewSparkSubmitCommand() *cli.Command {
	cmd := &cli.Command{
		Name: "spark-submit",
		Short: i18n.T(
			"EMR Serverless Spark submit CLI (spark-submit / spark-sql)",
			"EMR Serverless Spark 任务提交工具（spark-submit / spark-sql）"),
		Usage: "spark-submit [spark-submit options] [app args...]",
		Sample: "aliyun spark-submit --name SparkPi --queue dev_queue \\\n" +
			"  --class org.apache.spark.examples.SparkPi \\\n" +
			"  oss://bucket/path/spark-examples.jar 10000\n" +
			" \n" +
			"  aliyun spark-submit --status jr-xxxxxxxx",
		Hidden: true,
	}
	cmd.Run = func(ctx *cli.Context, args []string) error {
		if ctx.IsHelp() {
			return cmd.Help(ctx, args)
		}
		c := NewContext(ctx)
		err := c.Run(args)
		if exitErr, ok := err.(*ExitError); ok {
			cli.Exit(exitErr.ExitCode())
			return nil
		}
		return err
	}
	cmd.EnableUnknownFlag = true
	cmd.KeepArgs = true
	cmd.SkipDefaultHelp = true
	cmd.Help = func(ctx *cli.Context, _ []string) error {
		cmd.PrintHead(ctx)
		cmd.PrintUsage(ctx)
		cmd.PrintSample(ctx)
		printSparkSubmitNotes(ctx)
		printSparkSubmitEnvVars(ctx)
		cmd.PrintTail(ctx)
		return nil
	}
	return cmd
}

func printSparkSubmitNotes(ctx *cli.Context) {
	cli.Printf(ctx.Stdout(), "\n%s\n", i18n.T(
		"Notes:",
		"说明：").Text())
	notes := []string{
		i18n.T("Requires Java 8+ (JRE or JDK) on PATH. Credentials and region come from `aliyun configure`.",
			"需要 PATH 中有 Java 8+（JRE 或 JDK 均可）；凭据与地域来自 `aliyun configure`。").Text(),
		i18n.T("Set ALIBABA_CLOUD_EMR_SERVERLESS_SPARK_WORKSPACE_ID or workspaceId in ~/.aliyun/emr-serverless-spark-tool/conf/connection.properties.",
			"通过环境变量 ALIBABA_CLOUD_EMR_SERVERLESS_SPARK_WORKSPACE_ID 或 connection.properties 中的 workspaceId 指定工作空间。").Text(),
		i18n.T("For spark-sql, place spark-sql in the same bin/ directory and set ALIBABA_CLOUD_EMR_SPARK_SUBMIT_EXEC_PATH to that script.",
			"运行 spark-sql 时，可将 ALIBABA_CLOUD_EMR_SPARK_SUBMIT_EXEC_PATH 指向工具包 bin/spark-sql。").Text(),
	}
	for _, n := range notes {
		cli.Printf(ctx.Stdout(), "  %s\n", n)
	}
}

func printSparkSubmitEnvVars(ctx *cli.Context) {
	cli.Printf(ctx.Stdout(), "\n%s\n", i18n.T("Environment:", "环境变量：").Text())
	w := tabwriter.NewWriter(ctx.Stdout(), 8, 0, 1, ' ', 0)
	rows := []struct{ k, v string }{
		{EnvNoUpdateCheck + "=1", i18n.T("Skip the daily update check.", "跳过每日版本更新检查。").Text()},
		{EnvWorkspaceID, i18n.T("Default EMR Serverless Spark workspace ID.", "默认工作空间 ID。").Text()},
		{EnvEndpoint, i18n.T("Override EMR Serverless Spark API endpoint.", "覆盖 EMR Serverless Spark API endpoint。").Text()},
	}
	for _, r := range rows {
		cli.Printf(w, "  %s\t%s\n", r.k, r.v)
	}
	w.Flush()
}
