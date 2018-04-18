package cloudapi

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/services/cloudapi"
	"github.com/aliyun/aliyun-cli/i18n"
	"github.com/aliyun/aliyun-cli/cli"
)

func NewExportSwaggerCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "cloudapi_exportSwagger",
		Short: i18n.T("cloudapi: export swagger ", "cloudapi: 导出swagger定义"),
		Usage: "cloudapi_exportSwagger ...",
		Run: func(ctx *cli.Context, args []string) error {
			return runImportSwagger(ctx, args[0])
		},
	}
	return cmd
}

func runExportSwagger(ctx *cli.Context, f string) error {

	// processImport(client, &se, profile.RegionId)
	return nil
}

func processExport(client *cloudapi.Client, se *SwaggerEntity, region string) {
	// TODO, can't export
	return
}