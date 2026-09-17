package util

import (
	"os"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/sysconfig"
)

// 设计原则：
//   - 仅输出严格 sanitize 后的 **固定枚举**，不携带工作区名、路径等可变字段；
//   - 仅依赖业界已记录在案的 **Agent 自身** 环境变量是否存在；

const (
	agentEnvProposal = "AGENT"

	agentSegmentPrefix = "Agent/"

	agentProposalName = "1"
)

var knownAgentEnv = []struct {
	env  string
	name string
}{
	{"CURSOR_AGENT", "cursor"},
	{"CLAUDECODE", "claude-code"},
	{"CLAUDE_CODE", "claude-code"},
	{"GEMINI_CLI", "gemini-cli"},
	{"AUGMENT_AGENT", "augment"},
	{"OPENCODE", "opencode"},
	{"OPENCODE_CLIENT", "opencode"},
	{"CLINE_ACTIVE", "cline"},
	{"CODEX_SHELL", "codex"},
	{"CODEX_SANDBOX", "codex"},
	{"QODER_AGENT", "qoder"},
	{"QODER_CLI", "qoder-cli"},
	// Qoder CLI's China distribution uses a different shell identification variable.
	{"QODERCN_CLI", "qoder-cli"},
	{"WORKBUDDY_APP_NAME", "workbuddy"},
	{"TRAE_BRAND_NAME", "trae"},
	{"HERMES_AGENT", "hermes"},
}

// Agent 标识（小写固定枚举；AGENT 变量仅作兜底，固定为 1）；
func DetectAgentName() string {
	for _, item := range knownAgentEnv {
		if os.Getenv(item.env) != "" {
			return item.name
		}
	}
	if strings.TrimSpace(os.Getenv(agentEnvProposal)) != "" {
		return agentProposalName
	}
	return ""
}

func GetAgentUserAgentSegment() string {
	name := DetectAgentName()
	if name == "" {
		return ""
	}
	return agentSegmentPrefix + name
}

func MergeAgentSegmentIntoPluginEnvs(envs map[string]string) {
	if envs == nil {
		return
	}
	seg := GetAgentUserAgentSegment()
	if seg == "" {
		return
	}
	base := strings.TrimSpace(envs[sysconfig.EnvUserAgent])
	if base == "" {
		base = strings.TrimSpace(os.Getenv(sysconfig.EnvUserAgent))
	}
	if base == "" {
		envs[sysconfig.EnvUserAgent] = seg
		return
	}
	envs[sysconfig.EnvUserAgent] = base + " " + seg
}
