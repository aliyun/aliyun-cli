# Alibaba Cloud CLI telemetry data collection

[Documentation index](../README.md) | [简体中文](../zh-CN/telemetry.md)

## About data collection

Alibaba Cloud CLI telemetry is enabled by default. After command execution, the CLI collects structured usage information and sends it to Alibaba Cloud over HTTPS from a background process. This information helps analyze feature usage, compatibility, execution results, and performance to improve product reliability and usability.

Collection covers operations such as ordinary commands, credential configuration, and plugin management, including local execution results that do not involve a cloud API request. `aliyun configure telemetry` and its management subcommands do not collect telemetry about themselves. The original command does not wait for the telemetry network upload; disabling telemetry does not affect normal CLI functionality.

## What information is collected

The following table lists the event fields and their purposes. Some fields are present only for the relevant command or environment. Command, environment, and plugin properties are contained in the event's `attributes` object.

| Category | Fields | Description and purpose |
| --- | --- | --- |
| Event information | `schemaVersion`, `eventId`, `eventType`, `timestamp` | Data format version, a randomly generated ID for each event, event category, and command completion time, used to distinguish events, produce statistics, and investigate issues. |
| Client information | `clientName`, `clientVersion` | Client name and CLI version, used to analyze version usage and compatibility. |
| Installation identifier | `instanceId` | Generated randomly on your machine to measure usage of the same installation. It is not derived from an account or hardware information and can be reset with `purge`. |
| Command information | `commandType`, `command`, `parameterNames` | Command implementation type, command and subcommand names, and named parameter names, used to analyze feature and parameter usage. Business parameter values are not recorded. |
| Execution results | `status`, `durationMs`, `localErrorCode`, `apiErrorCode`, `httpStatusCode` | Execution outcome, command execution duration, structured error codes, and HTTP status codes, used to analyze reliability and performance. Error message text is not recorded. |
| System environment | `os`, `arch`, `goVersion`, `linuxDistroName`, `linuxDistroId`, `linuxDistroVersion` | Operating system, CPU architecture, Go runtime version, and Linux distribution information where applicable, used to analyze compatibility. |
| Terminal environment | `shellType`, `stdinIsTTY`, `locale` | Shell type, whether standard input is connected to a terminal, and language and encoding settings, used to understand interactive and automated usage. |
| Environment variable names | `environmentVariableNames` | Only names with the `ALIBABA_CLOUD_` prefix that meet format restrictions, used to understand configuration methods. The values of these variables are not recorded. |
| Configuration and integrations | `authMode`, `authModeSource`, `aiModeEnabled`, `agentName` | Authentication mode, how the mode was selected during configuration, whether AI mode is enabled, and a recognized agent type. These fields do not contain credentials. |
| Plugin information | `plugin`, `pipeline` | Installed plugin name and version, plus an internal-plugin classification marker where applicable, used to analyze plugin usage. |
| Plugin management | `operation`, `pluginNames`, `fromVersion`, `toVersion`, `pluginType`, `inner`, `sourceType` | Plugin management operation, relevant plugin names and versions, plugin type and attributes, and installation source category. Package paths, download URLs, and source repository addresses are not recorded. |

Command information records command and parameter names, without business parameter values. Authentication mode, locale, plugin version, and other information listed above are recorded in their corresponding fields.

## What information is not collected

The structured telemetry events described above do not collect:

- Credentials such as AccessKey IDs, AccessKey secrets, SecurityTokens, bearer tokens, cookies, or passwords.
- Alibaba Cloud account IDs, UIDs, employee accounts, email addresses, usernames, hostnames, MAC addresses, or hardware identifiers.
- Business parameter values, complete command lines, values of `ALIBABA_CLOUD_` environment variables, or full environment dumps.
- Local file paths, file contents, SQL, templates, request bodies, service response bodies, or command output.
- Error message text, error summaries, or stack traces.
- Session IDs, terminal IDs, parent process IDs, OpenAPI or gateway request IDs, or request IDs returned by the telemetry endpoint.
- The cloud region configured in the current profile or a client IP address field.

This scope applies to the telemetry events constructed and uploaded by the CLI. It does not cover network access logs or business requests you explicitly send to cloud services. Ordinary CLI credential configuration may still store credentials; see [configuration and credentials](./configuration.md).

## Manage telemetry

Telemetry settings apply across profiles in the same user's configuration directory and control telemetry for both ordinary commands and internal plugins.

| Action | Command |
| --- | --- |
| View basic configuration | `aliyun configure telemetry show` |
| Disable telemetry | `aliyun configure telemetry disable` |
| Enable telemetry again | `aliyun configure telemetry enable` |

Example output from `show`:

```json
{
  "enabled": true,
  "environment_disabled": false,
  "config_file": "/Users/example/.aliyun/telemetry.json"
}
```

`enabled` is the local setting, `environment_disabled` indicates whether an environment variable disables telemetry, and `config_file` is the configuration file path. The settings allow collection only when `enabled` is `true` and `environment_disabled` is `false`. `show` does not create or read an installation ID and does not scan the cache.

You can also disable telemetry with an environment variable, which is useful for scripts, CI, or environments where you do not want to write configuration.

macOS / Linux shell:

```sh
export ALIBABA_CLOUD_CLI_TELEMETRY_DISABLE=1
```

PowerShell:

```powershell
$env:ALIBABA_CLOUD_CLI_TELEMETRY_DISABLE = "1"
```

The values `1`, `true`, and `yes` disable telemetry, case-insensitively. Setting the variable to `0`, `false`, or `no`, or removing it, only removes the environment override; it does not override a disabled local setting. If the environment variable still disables telemetry, running `enable` will not make collection active.

To enable telemetry again, first remove the environment variable, then run `aliyun configure telemetry enable`:

```sh
unset ALIBABA_CLOUD_CLI_TELEMETRY_DISABLE
aliyun configure telemetry enable
```

```powershell
Remove-Item Env:ALIBABA_CLOUD_CLI_TELEMETRY_DISABLE -ErrorAction SilentlyContinue
aliyun configure telemetry enable
```

Disabling telemetry applies to subsequent collection. It does not recall data already uploaded or stop an upload process that has already started.

## Local data and cleanup

Telemetry uses the following local files. `~` denotes the user's home directory; on Windows, these files are in the `.aliyun` folder under the user directory.

| Path | Purpose |
| --- | --- |
| `~/.aliyun/telemetry.json` | Stores whether telemetry is enabled. |
| `~/.aliyun/telemetry-instance-id` | Stores the randomly generated installation identifier. |
| `~/.aliyun/telemetry-cache/` | Stores temporary event files for the background uploader. |

Temporary event files pass data between local processes; they are not complete command logs. The uploader reads a file and attempts to delete it before sending the request. Failed uploads are not retried as a backlog of historical events.

To clear the local cache and reset the installation identifier:

```sh
aliyun configure telemetry purge
```

`purge` preserves your telemetry setting. It does not delete uploaded data or cancel collection or uploads already in progress. If telemetry remains enabled, a later eligible command generates a new random identifier.

To disable telemetry and clear existing local data, run:

```sh
aliyun configure telemetry disable
aliyun configure telemetry purge
```
