package mock

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	sysmock "github.com/aliyun/aliyun-cli/v3/sysconfig/mock"
)

func TestMockPathCommand(t *testing.T) {
	path := filepath.Join(t.TempDir(), "custom-mocks.json")
	t.Setenv(sysmock.EnvMockPath, path)

	stdout, stderr := executeMockCommand(t, "mock", "path")

	if stdout != path+"\n" {
		t.Fatalf("stdout = %q, want %q", stdout, path+"\n")
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestMockRootCommandUsesDefaultHelp(t *testing.T) {
	cmd := NewMockCommand(t.TempDir)

	if cmd.Run != nil {
		t.Fatal("root mock command Run is set, want nil")
	}
}

func TestMockAddFlagsAreNotPersistent(t *testing.T) {
	cmd := NewMockCommand(t.TempDir)
	add := cmd.GetSubCommand("add")
	if add == nil {
		t.Fatal("add subcommand is nil")
	}

	for _, name := range []string{"name", "cmd", "exit-code", "stdout", "stderr", "times"} {
		flag := add.Flags().Get(name)
		if flag == nil {
			t.Fatalf("%s flag is nil", name)
		}
		if flag.Persistent {
			t.Fatalf("%s flag is persistent, want non-persistent", name)
		}
	}
}

func TestMockImportHelpShowsFlatJSONInputFormat(t *testing.T) {
	stdout, stderr := executeMockCommand(t, "mock", "import", "--help")

	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"JSON",
		`"name": "mock-ecs-all-operation"`,
		`"cmd": "ecs *"`,
		`"exit_code": 0`,
		`"stdout": "ecs 1.0.0\n"`,
		`"stderr": ""`,
		`"times": 10`,
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("help stdout = %q, want contain %q", stdout, want)
		}
	}
}

func TestMockRootHelpShowsAddAndImportCommands(t *testing.T) {
	i18n.SetLanguage("zh")
	defer i18n.SetLanguage("")

	stdout, stderr := executeMockCommand(t, "mock", "--help")

	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "add") || !strings.Contains(stdout, "添加一条 mock 记录") {
		t.Fatalf("root help stdout = %q, want add command", stdout)
	}
	if !strings.Contains(stdout, "import") || !strings.Contains(stdout, "导入并覆盖 JSON mock 记录") {
		t.Fatalf("root help stdout = %q, want import command", stdout)
	}
	if strings.Contains(stdout, "追加 mock 记录") {
		t.Fatalf("root help stdout = %q, should not use append wording", stdout)
	}
}

func TestMockRootHelpShowsEnvironmentSetup(t *testing.T) {
	stdout, stderr := executeMockCommand(t, "mock", "--help")

	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		sysmock.EnvMockEnabled,
		sysmock.EnvMockPath,
		"export ALIBABA_CLOUD_CLI_MOCK=true",
		"aliyun mock path",
		"aliyun ecs DescribeRegions",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("root help stdout = %q, want contain %q", stdout, want)
		}
	}
}

func TestMockAddByFlagsListAndClear(t *testing.T) {
	mockPath := filepath.Join(t.TempDir(), "mocks.json")
	t.Setenv(sysmock.EnvMockPath, mockPath)

	stdout, stderr := executeMockCommand(t, "mock", "add",
		"--name", "one",
		"--cmd", "ecs DescribeInstances",
		"--exit-code", "0",
		"--stdout", "ok",
		"--stderr", "warn",
		"--times", "10",
	)
	if stdout != "" || stderr != "" {
		t.Fatalf("add stdout = %q stderr = %q, want both empty", stdout, stderr)
	}

	stdout, stderr = executeMockCommand(t, "mock", "list")
	if stderr != "" {
		t.Fatalf("list stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, `"name": "one"`) ||
		!strings.Contains(stdout, `"cmd": "ecs DescribeInstances"`) ||
		!strings.Contains(stdout, `"exit_code": 0`) ||
		!strings.Contains(stdout, `"stdout": "ok"`) ||
		!strings.Contains(stdout, `"stderr": "warn"`) ||
		!strings.Contains(stdout, `"times": 10`) {
		t.Fatalf("list stdout = %q, want added record", stdout)
	}

	stdout, stderr = executeMockCommand(t, "mock", "clear")
	if stdout != "" || stderr != "" {
		t.Fatalf("clear stdout = %q stderr = %q, want both empty", stdout, stderr)
	}

	stdout, stderr = executeMockCommand(t, "mock", "list")
	if stderr != "" {
		t.Fatalf("list after clear stderr = %q, want empty", stderr)
	}
	if stdout != "[]\n" {
		t.Fatalf("list after clear stdout = %q, want []\\n", stdout)
	}
}

func TestMockAddWarnsAndStripsAliyunCommandPrefix(t *testing.T) {
	mockPath := filepath.Join(t.TempDir(), "mocks.json")
	t.Setenv(sysmock.EnvMockPath, mockPath)

	stdout, stderr := executeMockCommand(t, "mock", "add",
		"--name", "prefixed",
		"--cmd", "aliyun ecs describe-regions --accept-language zh-*",
		"--exit-code", "0",
		"--stdout", "ok",
		"--stderr", "warn",
		"--times", "0",
	)
	if stdout != "" {
		t.Fatalf("add stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "WARNING:") || !strings.Contains(stderr, "removed leading aliyun") {
		t.Fatalf("add stderr = %q, want aliyun prefix warning", stderr)
	}

	stdout, stderr = executeMockCommand(t, "mock", "list")
	if stderr != "" {
		t.Fatalf("list stderr = %q, want empty", stderr)
	}
	if strings.Contains(stdout, `"cmd": "aliyun ecs`) {
		t.Fatalf("list stdout = %q, should not keep aliyun prefix", stdout)
	}
	if !strings.Contains(stdout, `"cmd": "ecs describe-regions --accept-language zh-*"`) {
		t.Fatalf("list stdout = %q, want normalized cmd", stdout)
	}
}

func TestMockImportReplacesExistingRecords(t *testing.T) {
	mockPath := filepath.Join(t.TempDir(), "mocks.json")
	t.Setenv(sysmock.EnvMockPath, mockPath)
	writeFile(t, mockPath, `[{"name":"old","cmd":"ecs old","exit_code":0,"stdout":"old","stderr":"","times":0}]`)
	inputPath := filepath.Join(t.TempDir(), "input.json")
	writeFile(t, inputPath, `[
  {"name":"new","cmd":"ecs new","exit_code":3,"stdout":"new out","stderr":"new err","times":2}
]`)

	stdout, stderr := executeMockCommand(t, "mock", "import", "--file", inputPath)
	if stdout != "" || stderr != "" {
		t.Fatalf("import stdout = %q stderr = %q, want both empty", stdout, stderr)
	}

	stdout, stderr = executeMockCommand(t, "mock", "list")
	if stderr != "" {
		t.Fatalf("list stderr = %q, want empty", stderr)
	}
	if strings.Contains(stdout, `"name": "old"`) {
		t.Fatalf("list stdout = %q, want old record replaced", stdout)
	}
	if !strings.Contains(stdout, `"name": "new"`) || !strings.Contains(stdout, `"exit_code": 3`) {
		t.Fatalf("list stdout = %q, want imported record", stdout)
	}
}

func TestMockImportWarnsAndStripsAliyunCommandPrefix(t *testing.T) {
	mockPath := filepath.Join(t.TempDir(), "mocks.json")
	t.Setenv(sysmock.EnvMockPath, mockPath)
	inputPath := filepath.Join(t.TempDir(), "input.json")
	writeFile(t, inputPath, `[
  {"name":"prefixed","cmd":"aliyun ecs describe-regions --accept-language zh-*","exit_code":0,"stdout":"ok","stderr":"warn","times":0}
]`)

	stdout, stderr := executeMockCommand(t, "mock", "import", "--file", inputPath)
	if stdout != "" {
		t.Fatalf("import stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "WARNING:") || !strings.Contains(stderr, "removed leading aliyun") {
		t.Fatalf("import stderr = %q, want aliyun prefix warning", stderr)
	}

	stdout, stderr = executeMockCommand(t, "mock", "list")
	if stderr != "" {
		t.Fatalf("list stderr = %q, want empty", stderr)
	}
	if strings.Contains(stdout, `"cmd": "aliyun ecs`) {
		t.Fatalf("list stdout = %q, should not keep aliyun prefix", stdout)
	}
	if !strings.Contains(stdout, `"cmd": "ecs describe-regions --accept-language zh-*"`) {
		t.Fatalf("list stdout = %q, want normalized cmd", stdout)
	}
}

func TestMockRemoveByNameRemovesFirstMatchingRecord(t *testing.T) {
	mockPath := filepath.Join(t.TempDir(), "mocks.json")
	t.Setenv(sysmock.EnvMockPath, mockPath)
	writeFile(t, mockPath, `[
  {"name":"first","cmd":"ecs first","exit_code":0,"stdout":"1","stderr":"","times":0},
  {"name":"second","cmd":"ecs second","exit_code":0,"stdout":"2","stderr":"","times":0},
  {"name":"first","cmd":"ecs first again","exit_code":0,"stdout":"3","stderr":"","times":0}
]`)

	stdout, stderr := executeMockCommand(t, "mock", "remove", "--name", "first")
	if stdout != "" || stderr != "" {
		t.Fatalf("remove stdout = %q stderr = %q, want both empty", stdout, stderr)
	}

	stdout, stderr = executeMockCommand(t, "mock", "list")
	if stderr != "" {
		t.Fatalf("list stderr = %q, want empty", stderr)
	}
	if strings.Contains(stdout, `"cmd": "ecs first"`) {
		t.Fatalf("list stdout = %q, want first matching record removed", stdout)
	}
	if !strings.Contains(stdout, `"cmd": "ecs second"`) || !strings.Contains(stdout, `"cmd": "ecs first again"`) {
		t.Fatalf("list stdout = %q, want other records retained", stdout)
	}
}

func TestMockRemoveByIndexUsesZeroBasedArrayIndex(t *testing.T) {
	mockPath := filepath.Join(t.TempDir(), "mocks.json")
	t.Setenv(sysmock.EnvMockPath, mockPath)
	writeFile(t, mockPath, `[
  {"name":"zero","cmd":"ecs zero","exit_code":0,"stdout":"0","stderr":"","times":0},
  {"name":"one","cmd":"ecs one","exit_code":0,"stdout":"1","stderr":"","times":0},
  {"name":"two","cmd":"ecs two","exit_code":0,"stdout":"2","stderr":"","times":0}
]`)

	stdout, stderr := executeMockCommand(t, "mock", "remove", "--index", "1")
	if stdout != "" || stderr != "" {
		t.Fatalf("remove stdout = %q stderr = %q, want both empty", stdout, stderr)
	}

	stdout, stderr = executeMockCommand(t, "mock", "list")
	if stderr != "" {
		t.Fatalf("list stderr = %q, want empty", stderr)
	}
	if strings.Contains(stdout, `"cmd": "ecs one"`) {
		t.Fatalf("list stdout = %q, want index 1 removed", stdout)
	}
	if !strings.Contains(stdout, `"cmd": "ecs zero"`) || !strings.Contains(stdout, `"cmd": "ecs two"`) {
		t.Fatalf("list stdout = %q, want other records retained", stdout)
	}
}

func TestMockRemoveRequiresExactlyOneSelector(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "missing selector", args: []string{"mock", "remove"}},
		{name: "both selectors", args: []string{"mock", "remove", "--name", "one", "--index", "0"}},
		{name: "invalid index", args: []string{"mock", "remove", "--index", "bad"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr := executeMockCommand(t, tt.args...)
			if stdout != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
			if !strings.Contains(stderr, "ERROR:") {
				t.Fatalf("stderr = %q, want command error", stderr)
			}
		})
	}
}

func TestMockImportAcceptsArrayInputAndPreservesOrder(t *testing.T) {
	mockPath := filepath.Join(t.TempDir(), "mocks.json")
	t.Setenv(sysmock.EnvMockPath, mockPath)
	inputPath := filepath.Join(t.TempDir(), "input.json")
	writeFile(t, inputPath, `[
  {"name":"first","cmd":"ecs first","exit_code":0,"stdout":"1","stderr":"","times":0},
  {"name":"second","cmd":"ecs second","exit_code":0,"stdout":"2","stderr":"","times":0}
]`)

	executeMockCommand(t, "mock", "import", "--file", inputPath)

	stdout, _ := executeMockCommand(t, "mock", "list")
	first := strings.Index(stdout, `"name": "first"`)
	second := strings.Index(stdout, `"name": "second"`)
	if first < 0 || second < 0 {
		t.Fatalf("list stdout = %q, want both records", stdout)
	}
	if first > second {
		t.Fatalf("list stdout = %q, want first before second", stdout)
	}
}

func TestMockAddRecoversMalformedExistingFile(t *testing.T) {
	mockPath := filepath.Join(t.TempDir(), "mocks.json")
	t.Setenv(sysmock.EnvMockPath, mockPath)
	writeFile(t, mockPath, `{bad`)

	stdout, stderr := executeMockCommand(t, "mock", "add",
		"--name", "recovered",
		"--cmd", "ecs recovered",
		"--exit-code", "0",
		"--stdout", "ok",
		"--stderr", "warn",
		"--times", "0",
	)
	if stdout != "" || stderr != "" {
		t.Fatalf("add stdout = %q stderr = %q, want both empty", stdout, stderr)
	}

	stdout, stderr = executeMockCommand(t, "mock", "list")
	if stderr != "" {
		t.Fatalf("list stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, `"name": "recovered"`) {
		t.Fatalf("list stdout = %q, want recovered record", stdout)
	}
}

func TestMockListTreatsMalformedFileAsEmpty(t *testing.T) {
	mockPath := filepath.Join(t.TempDir(), "mocks.json")
	t.Setenv(sysmock.EnvMockPath, mockPath)
	writeFile(t, mockPath, `{bad`)

	stdout, stderr := executeMockCommand(t, "mock", "list")

	if stderr != "" {
		t.Fatalf("list stderr = %q, want empty", stderr)
	}
	if stdout != "[]\n" {
		t.Fatalf("list stdout = %q, want []\\n", stdout)
	}
}

func TestMockImportWithoutFileReturnsError(t *testing.T) {
	cmd := NewMockCommand(t.TempDir)
	importCmd := cmd.GetSubCommand("import")
	if importCmd == nil {
		t.Fatal("import subcommand is nil")
	}

	var stdout, stderr bytes.Buffer
	ctx := cli.NewCommandContext(&stdout, &stderr)
	ctx.EnterCommand(cmd)
	ctx.EnterCommand(importCmd)

	err := importCmd.Run(ctx, nil)
	if err == nil {
		t.Fatal("import without --file returned nil error")
	}
	if !strings.Contains(err.Error(), "--file") {
		t.Fatalf("error = %q, want mention --file", err.Error())
	}
}

func TestMockAddRequiresAllFieldsAndShowsExample(t *testing.T) {
	stdout, stderr := executeMockCommand(t, "mock", "add")

	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	for _, want := range []string{
		"ERROR:",
		"missing required flags:",
		"--name",
		"--cmd",
		"--exit-code",
		"--stdout",
		"--stderr",
		"--times",
		"Example:",
		"aliyun mock add --name mock-ecs-version",
	} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("stderr = %q, want contain %q", stderr, want)
		}
	}
}

func TestMockImportWithEmptyFileValueReturnsCommandError(t *testing.T) {
	cmd := NewMockCommand(t.TempDir)
	importCmd := cmd.GetSubCommand("import")
	if importCmd == nil {
		t.Fatal("import subcommand is nil")
	}

	var stdout, stderr bytes.Buffer
	ctx := cli.NewCommandContext(&stdout, &stderr)
	ctx.EnterCommand(cmd)
	ctx.EnterCommand(importCmd)
	fileFlag := ctx.Flags().Get("file")
	if fileFlag == nil {
		t.Fatal("file flag is nil")
	}
	fileFlag.SetAssigned(true)
	fileFlag.SetValue("")

	err := importCmd.Run(ctx, nil)
	if err == nil {
		t.Fatal("import with empty --file value returned nil error")
	}
	if !strings.Contains(err.Error(), "--file") {
		t.Fatalf("error = %q, want mention --file", err.Error())
	}
}

func TestMockImportWithUnreadableFileReturnsError(t *testing.T) {
	cmd := NewMockCommand(t.TempDir)
	importCmd := cmd.GetSubCommand("import")
	if importCmd == nil {
		t.Fatal("import subcommand is nil")
	}

	var stdout, stderr bytes.Buffer
	ctx := cli.NewCommandContext(&stdout, &stderr)
	ctx.EnterCommand(cmd)
	ctx.EnterCommand(importCmd)
	fileFlag := ctx.Flags().Get("file")
	fileFlag.SetAssigned(true)
	fileFlag.SetValue(filepath.Join(t.TempDir(), "missing.json"))

	err := importCmd.Run(ctx, nil)
	if err == nil {
		t.Fatal("import with missing file returned nil error")
	}
}

func TestMockImportWithInvalidJSONReturnsError(t *testing.T) {
	cmd := NewMockCommand(t.TempDir)
	importCmd := cmd.GetSubCommand("import")
	if importCmd == nil {
		t.Fatal("import subcommand is nil")
	}
	inputPath := filepath.Join(t.TempDir(), "input.json")
	writeFile(t, inputPath, `{bad`)

	var stdout, stderr bytes.Buffer
	ctx := cli.NewCommandContext(&stdout, &stderr)
	ctx.EnterCommand(cmd)
	ctx.EnterCommand(importCmd)
	fileFlag := ctx.Flags().Get("file")
	fileFlag.SetAssigned(true)
	fileFlag.SetValue(inputPath)

	err := importCmd.Run(ctx, nil)
	if err == nil {
		t.Fatal("import with invalid JSON returned nil error")
	}
}

func TestMockSubcommandsRejectPositionalArgs(t *testing.T) {
	cmd := NewMockCommand(t.TempDir)
	for _, name := range []string{"add", "import", "remove", "list", "clear", "path"} {
		t.Run(name, func(t *testing.T) {
			sub := cmd.GetSubCommand(name)
			if sub == nil {
				t.Fatalf("%s subcommand is nil", name)
			}
			var stdout, stderr bytes.Buffer
			ctx := cli.NewCommandContext(&stdout, &stderr)
			ctx.EnterCommand(cmd)
			ctx.EnterCommand(sub)

			err := sub.Run(ctx, []string{"extra"})
			if err == nil {
				t.Fatalf("%s accepted positional args", name)
			}
		})
	}
}

func executeMockCommand(t *testing.T, args ...string) (string, string) {
	t.Helper()

	cli.DisableExitCode()
	defer cli.EnableExitCode()

	var stdout, stderr bytes.Buffer
	cmd := &cli.Command{Name: "aliyun"}
	cmd.AddSubCommand(NewMockCommand(t.TempDir))
	ctx := cli.NewCommandContext(&stdout, &stderr)
	ctx.EnterCommand(cmd)
	cmd.Execute(ctx, args)
	return stdout.String(), stderr.String()
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}
