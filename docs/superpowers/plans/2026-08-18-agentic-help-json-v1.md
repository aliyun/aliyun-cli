# Agentic Help JSON v1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add stable, offline, machine-readable root/product/API help through both `--help=json` and `aliyun help ... --format json` on top of the unified Canonical metadata branch.

**Architecture:** Normalize both public syntaxes into the existing help path with a hidden internal format flag. Project Canonical catalog, version index, API, legacy parameter view, and static command metadata into dedicated Help v1 DTOs; do not serialize internal structs directly. Structured machine-help errors are rendered as JSON on stderr with non-zero exit codes.

**Tech Stack:** Go 1.25, aliyun CLI command framework, `encoding/json`, `io/fs`, Canonical JSON metadata, Go tests with `testing` and `testify`.

---

## File map

- Create `openapi/machine_help_args.go`: recognize and normalize the two public machine-help syntaxes.
- Create `openapi/machine_help_args_test.go`: parser compatibility and normalization tests.
- Create `openapi/machine_help.go`: Help v1 DTOs, root/product/API projection, deterministic JSON rendering, and structured errors.
- Create `openapi/machine_help_test.go`: contract, parameter-tree, version, error, and offline tests.
- Modify `openapi/flags.go`: register the hidden internal help-format flag.
- Modify `openapi/commando.go`: dispatch machine help before text-help/plugin loading.
- Modify `main/main.go`: normalize raw argv before the root parser.
- Modify `main/main_test.go`: command-level coverage for both public syntaxes.
- Modify `canonicalmeta/types.go`: expose the Canonical fields needed by Help v1 without changing legacy projections.
- Modify `canonicalmeta/reader.go`: read the product catalog and version indexes.
- Modify `canonicalmeta/repository.go`: provide query methods used by the Help service.
- Modify `canonicalmeta/canonical_test.go`: validate the new readers and complete API decoding.
- Modify `canonicalmeta/testdata/metadatas/products.json`: deterministic product fixture.
- Modify `canonicalmeta/testdata/canonical/demo/2026-01-01/version.json`: deterministic version-index fixture.
- Modify `meta/product.go`: retain `plugin_default_version`, `versions`, and `distribution` already present in `products.json`.
- Modify `cli/errors.go` and `cli/command.go`: allow a command error to render its own machine-readable payload.
- Modify `cli/command_test.go`: prove existing text errors remain unchanged and JSON errors use the structured path.
- Modify `README.md` and `README-CN.md`: document the public Help v1 entry points.

### Task 1: Normalize the two public command syntaxes

**Files:**
- Create: `openapi/machine_help_args.go`
- Create: `openapi/machine_help_args_test.go`
- Modify: `openapi/flags.go`
- Modify: `main/main.go`

- [ ] **Step 1: Write failing normalization tests**

Cover these exact cases:

```go
func TestNormalizeMachineHelpArgs(t *testing.T) {
    tests := []struct {
        name string
        in   []string
        want []string
    }{
        {"root equals", []string{"--help=json"}, []string{"--help", "--help-format", "json"}},
        {"api equals", []string{"ecs", "describe-instances", "--help=json"}, []string{"ecs", "describe-instances", "--help", "--help-format", "json"}},
        {"help command", []string{"help", "ecs", "DescribeInstances", "--format", "json"}, []string{"ecs", "DescribeInstances", "--help", "--help-format", "json"}},
        {"text unchanged", []string{"help", "ecs"}, []string{"help", "ecs"}},
        {"ordinary format unchanged", []string{"ecs", "Call", "--format", "json"}, []string{"ecs", "Call", "--format", "json"}},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            assert.Equal(t, tt.want, NormalizeMachineHelpArgs(tt.in))
        })
    }
}
```

- [ ] **Step 2: Run the tests and verify RED**

Run: `go test ./openapi -run TestNormalizeMachineHelpArgs -count=1`

Expected: compile failure because `NormalizeMachineHelpArgs` does not exist.

- [ ] **Step 3: Implement minimal normalization and hidden flag**

Add `MachineHelpFormatFlagName = "help-format"`, a hidden persistent `AssignedOnce` flag, and a normalizer that:

```go
func NormalizeMachineHelpArgs(args []string) []string {
    // Copy input, rewrite only exact --help=<value>, or a leading
    // `help` command containing --format=<value>/--format <value>.
    // Plain text-help argv and ordinary API --format parameters remain untouched.
}
```

Call it in `main.Main` immediately before `rootCmd.Execute(ctx, args)`.

- [ ] **Step 4: Run targeted tests and verify GREEN**

Run: `go test ./openapi ./main -run 'TestNormalizeMachineHelpArgs|TestMainWithNoArgs' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add openapi/machine_help_args.go openapi/machine_help_args_test.go openapi/flags.go main/main.go
git commit -m "feat: normalize machine help arguments"
```

### Task 2: Expose complete Canonical read models

**Files:**
- Modify: `canonicalmeta/types.go`
- Modify: `canonicalmeta/reader.go`
- Modify: `canonicalmeta/repository.go`
- Modify: `canonicalmeta/canonical_test.go`
- Create: `canonicalmeta/testdata/metadatas/products.json`
- Create: `canonicalmeta/testdata/canonical/demo/2026-01-01/version.json`
- Modify: `meta/product.go`

- [ ] **Step 1: Write failing catalog, index, and API decoding tests**

Tests assert:

```go
catalog, err := testFS().GetProducts()
require.NoError(t, err)
assert.Equal(t, "2025-01-01", catalog.Products[0].PluginDefaultVersion)
assert.Equal(t, []string{"2025-01-01", "2026-01-01"}, catalog.Products[0].Versions)

index, err := testFS().GetVersionIndex("demo", "2026-01-01")
require.NoError(t, err)
assert.Equal(t, "describe-regions", index.APIs["DescribeRegions"].CmdName)

api, err := testFS().GetAPI("demo", "2026-01-01", "CreateReport")
require.NoError(t, err)
assert.Equal(t, "create-report", api.CmdName)
assert.Equal(t, "2026-01-01", api.Operation.APIVersion)
assert.Equal(t, []string{"--report-id"}, api.Parameters[0].Options)
```

- [ ] **Step 2: Run the tests and verify RED**

Run: `go test ./canonicalmeta -run 'TestReadProducts|TestReadVersionIndex|TestReadAPICompleteHelpFields' -count=1`

Expected: compile failures for missing catalog/index methods and fields.

- [ ] **Step 3: Add read models and readers**

Add explicit structs for `ProductsIndex`, `ProductEntry`, `VersionIndex`, `VersionAPIEntry`, and `Operation`, plus complete Help-facing fields on `API` and `Parameter`. Add:

```go
func (r *Reader) ReadProducts() (*ProductsIndex, error)
func (r *Reader) ReadVersionIndex(product, version string) (*VersionIndex, error)
func (r *Repository) GetProducts() (*ProductsIndex, error)
func (r *Repository) GetVersionIndex(product, version string) (*VersionIndex, error)
```

Use the existing spec path first and retain the current nested-layout fallback. Extend legacy `meta.Product` only with JSON-backed fields:

```go
PluginDefaultVersion string   `json:"plugin_default_version"`
Versions             []string `json:"versions"`
Distribution         string   `json:"distribution,omitempty"`
```

- [ ] **Step 4: Run canonical and meta tests**

Run: `go test ./canonicalmeta ./meta -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add canonicalmeta meta/product.go
git commit -m "feat: expose canonical help metadata"
```

### Task 3: Implement the stable Help v1 projection

**Files:**
- Create: `openapi/machine_help.go`
- Create: `openapi/machine_help_test.go`

- [ ] **Step 1: Write failing root and product contract tests**

Use an in-memory/fstest Canonical repository and a small static root command. Decode JSON into maps only at the assertion boundary and verify stable DTO fields:

```go
assert.Equal(t, "v1", doc.SchemaVersion)
assert.Equal(t, "root", doc.Kind)
assert.Equal(t, []string{"aliyun"}, doc.Target.Path)
assert.Equal(t, []string{"configure", "plugin"}, commandNames(doc.Commands))

assert.Equal(t, "product", productDoc.Kind)
assert.Equal(t, "2026-01-01", productDoc.Product.LegacyDefaultVersion)
assert.Equal(t, "2025-01-01", productDoc.Product.PluginDefaultVersion)
assert.Equal(t, "2025-01-01", productDoc.Product.SelectedVersion)
assert.Equal(t, "describe-regions", productDoc.APIs[0].CmdName)
```

- [ ] **Step 2: Run root/product tests and verify RED**

Run: `go test ./openapi -run 'TestMachineHelpRoot|TestMachineHelpProduct' -count=1`

Expected: compile failure because the Help v1 service and DTOs do not exist.

- [ ] **Step 3: Implement root and product DTO projection**

Define explicit JSON structs headed by:

```go
type machineHelpEnvelope struct {
    SchemaVersion string            `json:"schemaVersion"`
    Kind          string            `json:"kind"`
    Target        machineHelpTarget `json:"target"`
}
```

Sort command names, products, versions, and API index entries before marshaling. Product help selects explicit version first, otherwise `plugin_default_version`, then legacy `version`, then the first sorted supported version.

- [ ] **Step 4: Run root/product tests and verify GREEN**

Run: `go test ./openapi -run 'TestMachineHelpRoot|TestMachineHelpProduct' -count=1`

Expected: PASS.

- [ ] **Step 5: Write failing API and parameter-tree tests**

Assert both command styles resolve the same Canonical identity at the same selected version while retaining their active projection:

```go
assert.Equal(t, "CreateReport", camelDoc.API.Name)
assert.Equal(t, "CreateReport", kebabDoc.API.Name)
assert.Equal(t, "camel", camelDoc.ActiveParameterSet)
assert.Equal(t, "kebab", kebabDoc.ActiveParameterSet)
assert.Equal(t, "--body", camelDoc.ParameterSets.Camel[0].Option)
assert.Equal(t, "--report-id", kebabDoc.ParameterSets.Kebab[0].Options[0])
assert.Nil(t, kebabDoc.OutputSchema)
assert.Nil(t, kebabDoc.Pagination)
assert.Nil(t, kebabDoc.Risk)
assert.Nil(t, kebabDoc.Recovery)
```

Add recursive fixtures/assertions for repeat-list object fields, array element shapes, map values, body compatibility parameters, descriptions, and both examples.

- [ ] **Step 6: Run API tests and verify RED**

Run: `go test ./openapi -run 'TestMachineHelpAPI|TestMachineHelpNestedParameters' -count=1`

Expected: failures because API projection is not implemented.

- [ ] **Step 7: Implement API resolution and recursive projection**

Resolve PascalCase by API index key and kebab-case by `cmd_name`. Build both parameter sets for every API:

```go
type machineHelpParameterSets struct {
    Camel []machineHelpParameter `json:"camel"`
    Kebab []machineHelpParameter `json:"kebab"`
}
```

Use `LegacyTopLevelParameters()` recursively for camel and Canonical `Parameters`/`Fields`/`Element`/`Value` recursively for kebab. Emit both examples and explicit null fields without `omitempty`.

- [ ] **Step 8: Run all machine-help tests and verify GREEN**

Run: `go test ./openapi -run TestMachineHelp -count=1`

Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add openapi/machine_help.go openapi/machine_help_test.go
git commit -m "feat: add help json v1 projection"
```

### Task 4: Wire machine help and structured JSON errors

**Files:**
- Modify: `cli/errors.go`
- Modify: `cli/command.go`
- Modify: `cli/command_test.go`
- Modify: `openapi/commando.go`
- Modify: `openapi/machine_help.go`
- Modify: `openapi/machine_help_test.go`
- Modify: `main/main_test.go`

- [ ] **Step 1: Write failing structured-error tests**

Define the desired seam in tests:

```go
type structuredTestError struct{}
func (structuredTestError) Error() string { return "fallback" }
func (structuredTestError) RenderError(w io.Writer) error {
    _, err := io.WriteString(w, `{"schemaVersion":"v1","error":{"code":"TEST"}}`+"\n")
    return err
}
func (structuredTestError) ExitCode() int { return 2 }
```

Verify `processError` writes only the JSON payload for this interface, while the existing `errors.New("test")` output remains byte-for-byte unchanged.

- [ ] **Step 2: Run CLI test and verify RED**

Run: `go test ./cli -run 'TestProcessStructuredError|TestProcessError' -count=1`

Expected: structured error receives the old `ERROR:` prefix until the new interface is handled.

- [ ] **Step 3: Implement the structured error seam**

Add:

```go
type StructuredError interface {
    error
    RenderError(io.Writer) error
    ExitCode() int
}
```

At the top of `Command.processError`, render a `StructuredError` directly to stderr, call `Exit` with its code, and return. Leave the existing path untouched.

- [ ] **Step 4: Write failing wiring and end-to-end tests**

Tests must prove:

- machine help is handled before `Commando.loadPlugins`, so no remote catalog call is attempted;
- `aliyun --help=json`, root help command syntax, product help, PascalCase API help, and kebab API help produce JSON;
- unknown product/API and invalid format produce JSON on stderr with non-zero exit;
- plain text help snapshots remain unchanged;
- no API execution hook, credential load, or plugin install hook is invoked.

- [ ] **Step 5: Run wiring tests and verify RED**

Run: `go test ./openapi ./main -run 'TestMachineHelp|TestMainMachineHelp' -count=1`

Expected: failures because `Commando.help` still enters text help.

- [ ] **Step 6: Wire the machine-help renderer**

At the start of `Commando.help`, before `loadPlugins`, read the hidden format flag and call the Help v1 service. Convert lookup/validation failures to a structured machine-help error containing `schemaVersion`, stable code, message, target, and sorted suggestions.

- [ ] **Step 7: Run targeted package tests and verify GREEN**

Run: `go test ./cli ./canonicalmeta ./meta ./openapi ./main -count=1`

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add cli openapi main
git commit -m "feat: wire machine readable help"
```

### Task 5: Documentation and full verification

**Files:**
- Modify: `README.md`
- Modify: `README-CN.md`

- [ ] **Step 1: Document the public interface**

Add concise English and Chinese sections showing root, product, PascalCase API, and kebab API examples. State that `schemaVersion` is `v1`, data is local/offline, and currently unavailable output/pagination/risk/recovery fields are null.

- [ ] **Step 2: Format and run static checks**

Run:

```bash
gofmt -w cli/errors.go cli/command.go canonicalmeta/types.go canonicalmeta/reader.go canonicalmeta/repository.go canonicalmeta/canonical_test.go meta/product.go openapi/flags.go openapi/commando.go openapi/machine_help_args.go openapi/machine_help_args_test.go openapi/machine_help.go openapi/machine_help_test.go main/main.go main/main_test.go
git diff --check
```

Expected: no output from `git diff --check`.

- [ ] **Step 3: Run all tests**

Run:

```bash
go test ./...
(cd aliyun-openapi-runtime && go test ./...)
```

Expected: both suites PASS with zero failures.

- [ ] **Step 4: Build and smoke-test the binary**

Run:

```bash
go build -o /tmp/aliyun-help-json-v1 ./main
/tmp/aliyun-help-json-v1 --help=json
/tmp/aliyun-help-json-v1 help ecs --format json
/tmp/aliyun-help-json-v1 help ecs DescribeInstances --format json
/tmp/aliyun-help-json-v1 ecs describe-instances --help=json
```

Expected: each command emits valid JSON with `schemaVersion: "v1"`; API invocations are not executed.

- [ ] **Step 5: Commit documentation**

```bash
git add README.md README-CN.md
git commit -m "docs: document help json v1"
```

- [ ] **Step 6: Verify branch diff and history**

Run:

```bash
git status --short
git log --oneline origin/unify-meta-test..HEAD
git diff --stat origin/unify-meta-test...HEAD
```

Expected: clean worktree and only Help v1/design changes.

- [ ] **Step 7: Push and create the pull request**

Push `codex/agentic-help-json-v1` to the `fork` remote, then create a GitHub PR with base repository `aliyun/aliyun-cli`, base branch `unify-meta-test`, and Aone work item `85538412` in the description. Verify the PR URL, base/head branches, and CI state.
