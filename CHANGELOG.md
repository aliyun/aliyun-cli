# CHANGE LOG

Release notes are also published at https://github.com/aliyun/aliyun-cli/releases. This file summarizes the major changes in each version.

### v3.5.1

- Added a shared `schemaVersion: "v1"` to AI-mode OpenAPI and OSS error envelopes, and structured initial configuration-loading failures in AI mode. Preserved non-AI output, existing exit statuses, and the separately versioned OSS extension.

- Added opt-in built-in OSS JSON/JSONL listings with bounded pages and resumable cursors, structured errors with recovery guidance, and non-interactive confirmation handling. Added local validation and read-only plans for single-file uploads and single-object deletion, plus exclusive JSONL failure reports for `cp`/`sync`. See the [OSS automation guide](docs/en/oss.md) ([简体中文](docs/zh-CN/oss.md)) for supported operations and contract limits.
- Fixed the built-in OSS bridge to preserve option order and `--`, propagate parser and service errors, pass profile region/timeouts/retry budgets, and refresh host credentials at request time without appending resolved credentials to argv. Retry loops stop on permanent errors and ambiguous write failures, with jittered backoff for eligible retries. `aliyun oss update` now directs users to update the host CLI through its installation method.
- Added queryable product endpoints to machine-readable Help and case-insensitive product routing for in-process OpenAPI commands. Added `ALIBABA_CLOUD_CLI_AGENT_INTEGRATION` controls for automatic AI mode and expanded Agent detection and User-Agent identification.
- Hardened safety-policy parsing and environment overrides to fail closed on invalid configuration. Documented command-spelling matching, startup errors, and exit statuses; API aliases and PascalCase/kebab-case spellings remain separate policy matches.
- Fixed dry-run redaction for JSON, form bodies, and embedded JSON values based on body content, and limited legacy FNF request-signing compatibility to the intended products.
- Hardened plugin archive extraction, package replacement, and download URL diagnostics. Newly installed metadata plugins can now execute in the current process.
- Fixed installation to prefer native macOS architecture packages and reject incomplete release downloads. Prerelease versions no longer update the stable latest pointer; expanded required CI checks to cover CLI extensions and cross-platform behavior.

### v3.5.0

- Introduced a unified OpenAPI runtime backed by canonical, bundled metadata, with lazy loading and consistent behavior across built-in and plugin-provided APIs.
- Added lowercase kebab-case commands and parameters alongside the existing PascalCase syntax, while preserving legacy command compatibility and improving command, API, and parameter validation.
- Added versioned machine-readable Help with `--cli-output json`, including hierarchical request/response sections, Help search, response schemas, query examples, and navigation hints for automation.
- Added agent-aware AI mode with structured error output, spelling suggestions, actionable recovery guidance, and safer fail-closed handling for invalid safety-policy files.
- Improved OpenAPI request construction for RPC, ROA, form data, paths, arrays, maps, nested composite parameters, direct bodies, API versions, and endpoint resolution.
- **Breaking change:** Aligned the product catalog and API versions with the public OpenAPI portal by removing the retired `acm`, `aigen`, `Ddi`, `scdn`, `scsp`, `viapi-regen`, and `WebPlus` products, and replacing the unavailable legacy API versions used by PascalCase commands for `Domain` (`2016-05-11`) and `Qualitycheck` (`2016-08-01`) with `2018-01-29` and `2019-01-15`, respectively. The retired products and legacy API versions are no longer available; existing scripts must migrate to supported products and API versions.
- Expanded plugin metadata and installation support, enforced minimum CLI versions, and hardened plugin names and installation paths against directory traversal.
- Hardened the MCP proxy by redacting sensitive logs and bounding request, connection, and upstream-header resource usage without terminating established SSE streams; also added finite CloudSSO HTTP timeouts and safer response handling.
- Added comprehensive English and Simplified Chinese documentation for configuration, usage, plugins, MCP, OpenTelemetry, and machine-facing interfaces.
- Upgraded the minimum Go version to 1.25.14, refreshed security-sensitive dependencies, and strengthened CI and release validation for the runtime and packed metadata.

### v3.0.0 GA Version

- refactoring cli package design, support composite flag with fields
- refactoring openapi package design, make it more extensible
- support `--quiet` flag
- support `--dryrun` flag
- support `aliyun oss --profile xxx`

### 0.81

- support `--output`
- support `--waiter`
- use `go -ldflags` to enable single Version in Makefile

### 0.80

- support auto completion for zsh/bash
- fix bugs for RepeatList parameter
- refactoring RamRoleArn and EcsRamRole authenticate flow
- oss command can support RamRoleArn and EcsRamRole authenticate mode
- oss command can support --profile and other configure flags

### 0.70

- integrate `ossutil` toolset with aliyun-cli
- optimize `--help` command messages
- config flags (such as ak, profile, sts) can used with openapi call
- support `configure delete`
- fix bug with restful force call

### 0.61

- support --all-pages flags to merge pager APIs

### 0.60

- support suggestions
- optimized error and help message
- integrate more completion of metadata
- fix some caller bugs

### 0.50

- support i18n `aliyun-openapi-meta`
- full support `configure [get|set|list]` command
- optimize help
- support `--quiet` flag

### 0.33

- fix bug for error processing when rpc/restful call
- auto add Content-Type header for restful call

### 0.32

- auto migrate legacy settings

### 0.31

- fix bug of check parameters, skip Action, Region parameters
- support `aliyun configure list` command

### 0.30

- integrate with 64 products meta
- implemented help command for product and api
- support fully certificated mode AK|StsToken|RamRoleArn|EcsRamRole|RsaKeyPair,

### 0.16

- support --content-type flag to set Header
- support --body-file flat to use file as body input

### 0.15

- support ecs-ram-role
- fix cross platform build problem
- test after configure

### 0.12

- fix bug for configure
- ignore case of ProductName

### 0.11

- Support simple ROA call

### 0.1

- Refactoring with golang
- Basic configure
- Auto endpoint locator
- 2018.1.11
