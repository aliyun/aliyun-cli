package util

import (
	"os"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/sysconfig"
	"github.com/stretchr/testify/assert"
)

func collectAgentEnvs() []string {
	envs := make([]string, 0, len(knownAgentEnv)+1)
	envs = append(envs, agentEnvProposal)
	for _, item := range knownAgentEnv {
		envs = append(envs, item.env)
	}
	return envs
}

var allAgentEnvs = collectAgentEnvs()

func snapshotAndUnsetAgentEnvs(t *testing.T) {
	t.Helper()
	saved := make(map[string]string, len(allAgentEnvs))
	for _, k := range allAgentEnvs {
		if v, ok := os.LookupEnv(k); ok {
			saved[k] = v
		}
		_ = os.Unsetenv(k)
	}
	t.Cleanup(func() {
		for _, k := range allAgentEnvs {
			_ = os.Unsetenv(k)
		}
		for k, v := range saved {
			_ = os.Setenv(k, v)
		}
	})
}

func TestDetectAgentName_NoEnv(t *testing.T) {
	snapshotAndUnsetAgentEnvs(t)
	assert.Equal(t, "", DetectAgentName())
	assert.Equal(t, "", GetAgentUserAgentSegment())
}

func TestDetectAgentName_KnownEnvs(t *testing.T) {
	for _, c := range knownAgentEnv {
		t.Run(c.env, func(t *testing.T) {
			snapshotAndUnsetAgentEnvs(t)
			_ = os.Setenv(c.env, "1")
			assert.Equal(t, c.name, DetectAgentName())
			assert.Equal(t, "Agent/"+c.name, GetAgentUserAgentSegment())
		})
	}
}

func TestDetectAgentName_QwenWorkAndQoder(t *testing.T) {
	for _, test := range []struct {
		name string
		envs map[string]string
		want string
	}{
		{
			name: "QwenWork base URL",
			envs: map[string]string{"QWENWORK_BASE_URL": "https://example.com"},
			want: "qwenwork",
		},
		{
			name: "QwenWork execution environment",
			envs: map[string]string{"QODERWORK_EXECUTION_ENV": "desktop"},
			want: "qwenwork",
		},
		{
			name: "Qoder China CLI",
			envs: map[string]string{"QODERCN_CLI": "1"},
			want: "qoder_cn_cli",
		},
		{
			name: "legacy integration product ignored",
			envs: map[string]string{"QODER_WORK_INTEGRATION_PRODUCT": "qwenwork"},
			want: "",
		},
		{
			name: "legacy integration product allows proposal fallback",
			envs: map[string]string{"QODER_WORK_INTEGRATION_PRODUCT": "qwenwork", "AGENT": "1"},
			want: "1",
		},
		{
			name: "QwenWork base URL takes priority over Qoder markers",
			envs: map[string]string{
				"QWENWORK_BASE_URL": "https://example.com",
				"QODERCN_CLI":       "1",
				"QODER_AGENT":       "1",
				"QODER_CLI":         "1",
			},
			want: "qwenwork",
		},
		{
			name: "QwenWork execution environment takes priority over Qoder markers",
			envs: map[string]string{
				"QODERWORK_EXECUTION_ENV": "desktop",
				"QODERCN_CLI":             "1",
				"QODER_AGENT":             "1",
				"QODER_CLI":               "1",
			},
			want: "qwenwork",
		},
		{
			name: "Qoder China CLI takes priority over generic Qoder markers",
			envs: map[string]string{
				"QODERCN_CLI": "1",
				"QODER_AGENT": "1",
				"QODER_CLI":   "1",
			},
			want: "qoder_cn_cli",
		},
		{
			name: "empty QwenWork markers allow Qoder detection",
			envs: map[string]string{
				"QWENWORK_BASE_URL":       "",
				"QODERWORK_EXECUTION_ENV": "",
				"QODERCN_CLI":             "",
				"QODER_CLI":               "1",
			},
			want: "qoder-cli",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			snapshotAndUnsetAgentEnvs(t)
			for env, value := range test.envs {
				t.Setenv(env, value)
			}
			assert.Equal(t, test.want, DetectAgentName())
			wantSegment := ""
			if test.want != "" {
				wantSegment = "Agent/" + test.want
			}
			assert.Equal(t, wantSegment, GetAgentUserAgentSegment())
		})
	}
}

func TestDetectAgentName_EmptyValueIgnored(t *testing.T) {
	snapshotAndUnsetAgentEnvs(t)
	_ = os.Setenv("CURSOR_AGENT", "")
	assert.Equal(t, "", DetectAgentName())
}

func TestDetectAgentName_SpecificEnvWinsOverProposal(t *testing.T) {
	snapshotAndUnsetAgentEnvs(t)
	_ = os.Setenv(agentEnvProposal, "goose")
	_ = os.Setenv("CURSOR_AGENT", "1")
	assert.Equal(t, "cursor", DetectAgentName(),
		"专有变量应优先于 AGENT 提案变量（避免类似 OpenCode 同时设两者时丢失具体名）")
}

func TestDetectAgentName_ProposalUsedWhenNoSpecific(t *testing.T) {
	snapshotAndUnsetAgentEnvs(t)
	_ = os.Setenv(agentEnvProposal, "Goose")
	assert.Equal(t, agentProposalName, DetectAgentName())
	assert.Equal(t, "Agent/1", GetAgentUserAgentSegment())
}

func TestDetectAgentName_ProposalIgnoresValue(t *testing.T) {
	snapshotAndUnsetAgentEnvs(t)
	_ = os.Setenv(agentEnvProposal, "  My-Agent.v1_x  ")
	assert.Equal(t, agentProposalName, DetectAgentName(),
		"AGENT 兜底不读取变量值，固定为 1")
}

func TestDetectAgentName_ProposalEmptyIgnored(t *testing.T) {
	snapshotAndUnsetAgentEnvs(t)
	_ = os.Setenv(agentEnvProposal, "   ")
	assert.Equal(t, "", DetectAgentName())
}

func TestDetectAgentName_PriorityOrder(t *testing.T) {
	snapshotAndUnsetAgentEnvs(t)
	_ = os.Setenv("CLAUDECODE", "1")
	_ = os.Setenv("CURSOR_AGENT", "1")
	assert.Equal(t, "cursor", DetectAgentName(),
		"CURSOR_AGENT 在表中靠前，应优先返回")
}

func TestMergeAgentSegmentIntoPluginEnvs_NoAgent(t *testing.T) {
	snapshotAndUnsetAgentEnvs(t)
	envs := map[string]string{}
	MergeAgentSegmentIntoPluginEnvs(envs)
	_, ok := envs[sysconfig.EnvUserAgent]
	assert.False(t, ok, "未检测到 agent 时不应写入任何 env")
}

func TestMergeAgentSegmentIntoPluginEnvs_NilMap(t *testing.T) {
	snapshotAndUnsetAgentEnvs(t)
	_ = os.Setenv("CURSOR_AGENT", "1")
	assert.NotPanics(t, func() {
		MergeAgentSegmentIntoPluginEnvs(nil)
	})
}

func TestMergeAgentSegmentIntoPluginEnvs_FreshEnv(t *testing.T) {
	snapshotAndUnsetAgentEnvs(t)
	_ = os.Unsetenv(sysconfig.EnvUserAgent)
	_ = os.Setenv("CURSOR_AGENT", "1")
	envs := map[string]string{}
	MergeAgentSegmentIntoPluginEnvs(envs)
	assert.Equal(t, "Agent/cursor", envs[sysconfig.EnvUserAgent])
}

func TestMergeAgentSegmentIntoPluginEnvs_PreservesParentEnv(t *testing.T) {
	snapshotAndUnsetAgentEnvs(t)
	_ = os.Setenv("CURSOR_AGENT", "1")
	t.Setenv(sysconfig.EnvUserAgent, "skill/foo")
	envs := map[string]string{}
	MergeAgentSegmentIntoPluginEnvs(envs)
	assert.Equal(t, "skill/foo Agent/cursor", envs[sysconfig.EnvUserAgent],
		"已 export 的 ALIBABA_CLOUD_USER_AGENT 必须保留并在末尾追加 Agent 段")
}

func TestMergeAgentSegmentIntoPluginEnvs_RuntimeEnvWinsOverParent(t *testing.T) {
	snapshotAndUnsetAgentEnvs(t)
	_ = os.Setenv("CURSOR_AGENT", "1")
	t.Setenv(sysconfig.EnvUserAgent, "from-parent")
	envs := map[string]string{
		sysconfig.EnvUserAgent: "from-runtime",
	}
	MergeAgentSegmentIntoPluginEnvs(envs)
	assert.Equal(t, "from-runtime Agent/cursor", envs[sysconfig.EnvUserAgent],
		"envs 中已有值时优先使用 envs 中的值")
}
