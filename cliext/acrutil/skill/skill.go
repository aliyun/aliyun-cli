package skill

import (
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cliext/acrutil/binmgr"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

// Wiring for the acr-skill binary. All install/update/exec behavior lives in
// the binmgr package; this file is just the parent CLI's subcommand glue.
//
// acr-skill-cli's -u/-p flags are for ACR Registry password auth (not AK/SK);
// the acrutil wrapper does not inject credentials. Provide them via:
//  1. command-line -u/-p flags, or
//  2. REGISTRY_USERNAME / REGISTRY_PASSWORD env vars.
var skillConfig = binmgr.Config{
	Name:             "acr-skill",
	BaseURL:          "https://acr-public-asset.oss-cn-hangzhou.aliyuncs.com/acr-skill/",
	EnvCompatMode:    "ALIBABA_CLOUD_ACR_SKILL_COMPAT_MODE",
	EnvCompatModeVal: "aliyun acrutil skill",
	EnvUserAgent:     "ALIBABA_CLOUD_ACR_SKILL",
	PlatformPaths: map[string]struct{}{
		"linux-amd64":  {},
		"linux-arm64":  {},
		"darwin-arm64": {},
	},
	StripFlags: skillStripFlags(),
}

// skillStripFlags builds the set of parent-CLI flags to strip for acr-skill.
// acr-skill owns --quiet/-q, so it is NOT stripped (passed through).
// --mode and --yes are parent-only for this subcommand, so they ARE stripped.
func skillStripFlags() map[string]bool {
	flags := binmgr.BaseStripFlags()
	flags["mode"] = true
	flags["yes"] = true
	return flags
}

// NewSkillCommand creates the `acrutil skill` subcommand.
func NewSkillCommand() *cli.Command {
	return &cli.Command{
		Name:   "skill",
		Short:  i18n.T("ACR Skill Management", "ACR Skill管理"),
		Usage:  "acrutil skill <command> [args...]",
		Hidden: false,
		Run: func(ctx *cli.Context, args []string) error {
			return binmgr.New(skillConfig, ctx).Run(args)
		},
		EnableUnknownFlag: true,
		KeepArgs:          true,
		SkipDefaultHelp:   true,
	}
}
