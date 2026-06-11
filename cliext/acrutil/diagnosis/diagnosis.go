package diagnosis

import (
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cliext/acrutil/binmgr"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

var diagnosisConfig = binmgr.Config{
	Name:             "acr-diagnosis",
	BaseURL:          "https://acr-public-asset.oss-cn-hangzhou.aliyuncs.com/cr-diagnosis/",
	EnvCompatMode:    "ALIBABA_CLOUD_CR_DIAG_COMPAT_MODE",
	EnvCompatModeVal: "aliyun acrutil diagnosis",
	EnvUserAgent:     "ALIBABA_CLOUD_CR_DIAG",
	PlatformPaths: map[string]struct{}{
		"linux-amd64":  {},
		"linux-arm64":  {},
		"darwin-amd64": {},
		"darwin-arm64": {},
	},
	StripFlags: diagnosisStripFlags(),
}

// diagnosisStripFlags builds the set of parent-CLI flags to strip for
// cr-diagnosis. cr-diagnosis owns --mode, --yes, and --quiet, so those are
// NOT stripped (passed through to the child).
func diagnosisStripFlags() map[string]bool {
	return binmgr.BaseStripFlags()
}

// NewDiagnosisCommand creates the `acrutil diagnosis` subcommand.
func NewDiagnosisCommand() *cli.Command {
	return &cli.Command{
		Name:   "diagnosis",
		Short:  i18n.T("ACR Instance Diagnosis", "ACR 实例诊断"),
		Usage:  "acrutil diagnosis [domain] [options...]",
		Hidden: false,
		Run: func(ctx *cli.Context, args []string) error {
			return binmgr.New(diagnosisConfig, ctx).Run(args)
		},
		EnableUnknownFlag: true,
		KeepArgs:          true,
		SkipDefaultHelp:   true,
	}
}
