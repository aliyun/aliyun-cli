package telemetry

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtensionCommandUsesOnlyConfirmedPaths(t *testing.T) {
	previousPaths, previousExtensions := extensionCommandPaths, cliExtensions
	extensionCommandPaths = map[string][]string{
		"test-extension": {"resource list", "resource get", "config show"},
	}
	RegisterCLIExtensions(map[string]bool{"test-extension": true, "unknown-extension": true, "spark-submit": true})
	t.Cleanup(func() { extensionCommandPaths, cliExtensions = previousPaths, previousExtensions })
	for _, test := range []struct {
		args []string
		want string
	}{
		{[]string{"test-extension", "resource", "list", "--tag", "private-sentinel-one", "private-sentinel-two"}, "test-extension resource list"},
		{[]string{"test-extension", "resource", "get", "private-sentinel"}, "test-extension resource get"},
		{[]string{"test-extension", "Resource", "List"}, "test-extension Resource List"},
		{[]string{"test-extension", "resource", "private-sentinel"}, "test-extension resource"},
		{[]string{"test-extension", "private-sentinel", "resource", "list"}, "test-extension"},
		{[]string{"test-extension", "--profile", "private-sentinel", "resource", "list"}, "test-extension"},
		{[]string{"test-extension", "resource", "--", "list", "private-sentinel"}, "test-extension resource"},
		{[]string{"test-extension", "resource", "get", "private-sentinel", "config", "show"}, "test-extension resource get"},
		{[]string{"test-extension", "-p", "private-sentinel", "config", "show"}, "test-extension"},
		{[]string{"unknown-extension", "resource", "list", "private-sentinel"}, "unknown-extension"},
		{[]string{"spark-submit", "resource", "list", "oss://private-sentinel/main.jar"}, "spark-submit"},
	} {
		t.Run(strings.Join(test.args[:2], " ")+"/"+test.want, func(t *testing.T) {
			event, err := BuildEvent(BuildInput{
				Scope:    Scope{Active: true, Pipeline: PipelinePublic, CommandToken: test.args[0]},
				RootArgs: test.args, InstanceID: "instance-id",
			})
			require.NoError(t, err)
			require.Equal(t, test.want, event.Command)
			require.Equal(t, CommandTypeCLIExt, event.CommandType)
			for _, payload := range []any{event, newUploadEvent(event)} {
				data, err := json.Marshal(payload)
				require.NoError(t, err)
				require.NotContains(t, string(data), "private-sentinel")
			}
		})
	}
	require.Equal(t, []string{"test-extension"}, extensionCommandParts(nil, "test-extension"))
	require.Equal(t, []string{"test-extension"}, extensionCommandParts([]string{"another"}, "test-extension"))
}

func TestConfirmedExtensionPathsAreSafeAndUnique(t *testing.T) {
	require.NotEmpty(t, extensionCommandPaths)
	require.NotContains(t, extensionCommandPaths, "spark-submit")
	require.NotEmpty(t, extensionCommandPaths["mseutil"])
	for root, paths := range extensionCommandPaths {
		require.True(t, safeTelemetryIdentifier(root, false), root)
		seen := make(map[string]bool)
		for _, path := range paths {
			require.Equal(t, strings.ToLower(path), path)
			require.False(t, seen[path], root+" "+path)
			seen[path] = true
			for _, token := range strings.Fields(path) {
				require.True(t, safeTelemetryIdentifier(token, false), root+" "+path)
			}
			args := append([]string{root}, strings.Fields(path)...)
			args = append(args, "private-sentinel", "--tag", "private-sentinel-one", "private-sentinel-two")
			require.Equal(t, root+" "+path, normalizeCommand(extensionCommandParts(args, root)))
		}
	}
}

func TestLocallyVerifiedExtensionEventExamples(t *testing.T) {
	previousExtensions := cliExtensions
	extensions := make(map[string]bool)
	for root := range extensionCommandPaths {
		extensions[root] = true
	}
	RegisterCLIExtensions(extensions)
	t.Cleanup(func() { cliExtensions = previousExtensions })
	for _, test := range []struct {
		args []string
		want string
	}{
		{[]string{"bl", "text", "chat", "--model", "private-sentinel", "private-sentinel-prompt"}, "bl text chat"},
		{[]string{"dgate", "acl", "data", "apply", "private-sentinel"}, "dgate acl data apply"},
		{[]string{"kmscli", "secret", "private-sentinel"}, "kmscli secret"},
		{[]string{"kmscli", "secret", "getsecret", "private-sentinel"}, "kmscli secret"},
		{[]string{"lindorm", "auth", "list", "private-sentinel"}, "lindorm auth list"},
		{[]string{"ecctl", "ack", "cluster", "list", "private-sentinel"}, "ecctl ack cluster list"},
		{[]string{"cms2", "alert", "rule", "create", "--name", "private-sentinel"}, "cms2 alert rule create"},
		{[]string{"ossutil", "api", "get-object", "oss://private-sentinel/path"}, "ossutil api get-object"},
		{[]string{"oss", "api", "get-object", "oss://private-sentinel/path"}, "oss"},
		{[]string{"oss", "ls", "oss://private-sentinel/path"}, "oss ls"},
		{[]string{"otsutil", "get", "private-sentinel"}, "otsutil get"},
		{[]string{"agentbay", "apikey", "concurrency", "set", "private-sentinel"}, "agentbay apikey concurrency set"},
		{[]string{"codeup-cli", "backup", "private-sentinel"}, "codeup-cli backup"},
		{[]string{"saectl", "create", "service", "clusterip", "private-sentinel"}, "saectl create service clusterip"},
		{[]string{"appmanager", "agent", "deploy", "private-sentinel"}, "appmanager agent deploy"},
		{[]string{"computenest-cli", "build-image", "private-sentinel"}, "computenest-cli build-image"},
		{[]string{"esa-cli", "deployments", "list", "private-sentinel"}, "esa-cli deployments list"},
		{[]string{"flow-cli", "step", "get", "private-sentinel"}, "flow-cli step get"},
		{[]string{"maxc", "query", "run", "SELECT private-sentinel"}, "maxc query run"},
		{[]string{"mseutil", "nacos", "get", "private-sentinel"}, "mseutil nacos get"},
		{[]string{"mseutil", "net", "dns", "private-sentinel.example.com"}, "mseutil net dns"},
		{[]string{"mseutil", "zookeeper", "get", "/private-sentinel/path"}, "mseutil zookeeper get"},
		{[]string{"mseutil", "net", "private-sentinel", "dns"}, "mseutil net"},
		{[]string{"mseutil", "--server", "private-sentinel", "nacos", "get"}, "mseutil"},
		{[]string{"iact3", "test", "/private-sentinel/template.yaml"}, "iact3 test"},
		{[]string{"rostran", "rules", "list", "private-sentinel"}, "rostran rules list"},
		{[]string{"acrutil", "diagnosis", "private-sentinel.example.com"}, "acrutil diagnosis"},
		{[]string{"acrutil", "skill", "install", "/private-sentinel/path"}, "acrutil skill install"},
		{[]string{"mseutil", "private-sentinel", "list"}, "mseutil"},
	} {
		t.Run(test.want, func(t *testing.T) {
			event, err := BuildEvent(BuildInput{
				Scope:    Scope{Active: true, Pipeline: PipelinePublic, CommandToken: test.args[0]},
				RootArgs: test.args, InstanceID: "instance-id",
			})
			require.NoError(t, err)
			require.Equal(t, test.want, event.Command)
			for _, payload := range []any{event, newUploadEvent(event)} {
				data, err := json.Marshal(payload)
				require.NoError(t, err)
				require.NotContains(t, string(data), "private-sentinel")
			}
		})
	}
}
