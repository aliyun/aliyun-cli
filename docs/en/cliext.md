# CLI extensions

[Documentation index](../README.md) | [简体中文](../zh-CN/cliext.md)

## ossutil

Point `aliyun ossutil` at an existing ossutil v2 executable:

```sh
export ALIBABA_CLOUD_OSSUTIL_EXEC_PATH=/usr/local/bin/ossutil
```

With this variable set, `aliyun ossutil` uses the specified executable without automatic downloads, installation, or remote version checks. The environment manages the binary's version. Missing or non-executable paths fail explicitly without falling back to installation. When the variable is unset or empty, the existing automatic installation and update behavior applies.

## bl

`aliyun bl <command> [args...]` runs the Alibaba Cloud Model Studio CLI (`bl`) as a child process of `aliyun`. The host selects the profile, prepares the child environment, and forwards the remaining arguments. The managed `bl` binary is installed and updated automatically unless `ALIBABA_CLOUD_BL_EXEC_PATH` points to an existing executable.

### Installation

`aliyun bl` runs either the managed `bl` binary that the host installs, or an external executable you provide through `ALIBABA_CLOUD_BL_EXEC_PATH`. When that variable is unset, the host uses the managed binary.

**Managed installation.** The first `aliyun bl` invocation that needs the managed binary downloads `manifest.json` and the matching versioned ZIP from `https://bailian-wiki.oss-cn-hangzhou.aliyuncs.com/release` over HTTPS. The host accepts the manifest only when its identity, version, and per-platform asset names match what it expects, verifies the ZIP against the SHA-256 digest in the manifest, requires the archive to contain exactly one root executable, and then installs it. Because the host prepares the binary before it forwards arguments, even `aliyun bl --help` triggers the installation.

The binary is written to `<config-root>/bl/bin/bl` (`bl.exe` on Windows) with mode `0755` on Linux and macOS, replacing any existing managed binary at that path. The host also records an update-check timestamp in `<config-root>/bl/update-check.json` and serializes concurrent first installations with `<config-root>/bl/.install-lock`.

**Installation directory.** `<config-root>` is the host configuration root: `$HOME/.aliyun` on Linux and macOS, and `%USERPROFILE%\.aliyun` on Windows (the host prefers `%HOMEDRIVE%%HOMEPATH%` and falls back to `%USERPROFILE%`). The root must already exist or be creatable, and the current user must be able to write to it. `--config-path` changes the configuration file the host reads; it does not move the managed `bl` binary.

**Supported platforms.** The managed binary is published for these platform identifiers:

| Host platform | Managed identifier |
| --- | --- |
| macOS, Apple silicon | `darwin-arm64` |
| macOS, Intel | `darwin-x64` |
| Linux, x86-64 | `linux-x64` |
| Windows, x86-64 | `windows-x64` (`amd64` is mapped to `x64`) |

Any other platform fails explicitly before anything is installed:

```text
unsupported bl platform linux/arm64; supported platforms: darwin-arm64, darwin-x64, linux-x64, windows-x64; or set ALIBABA_CLOUD_BL_EXEC_PATH
```

The managed platform check does not apply to `ALIBABA_CLOUD_BL_EXEC_PATH`, so a vetted external executable is how `aliyun bl` runs on a host without a managed release.

**Offline and controlled environments.** A managed installation needs network access to the release endpoint, and there is no offline first install: when no managed binary is installed yet and the download fails, `aliyun bl` fails instead of starting a partial binary. To run without network access, pre-install a vetted `bl` executable and set `ALIBABA_CLOUD_BL_EXEC_PATH` to its absolute path (see External binary and trust boundary below); that path never downloads, installs, or checks the managed version. Neither path requires Node.js or npm: the host starts a standalone `bl` executable and sets `BAILIAN_INSTALL_METHOD=binary` for the child process.

The download is verified only against the SHA-256 digest published in `manifest.json`; the manifest itself is not independently signed, so this check does not authenticate the publisher.

### Installation errors

| Message or symptom | What to do |
| --- | --- |
| `unsupported bl platform <os>/<arch>; supported platforms: ...; or set ALIBABA_CLOUD_BL_EXEC_PATH` | The host has no managed release. Use a supported platform or point `ALIBABA_CLOUD_BL_EXEC_PATH` at a vetted binary. |
| `ALIBABA_CLOUD_BL_EXEC_PATH="<path>" does not point to an existing file`, `... is not a regular file`, or `... is not executable` | Use an absolute path to an existing regular file; on Linux and macOS it must have an executable bit. An empty or whitespace-only value counts as unset. |
| `bl install lock exists at <config-root>/bl/.install-lock; retry after the installer finishes; ...` | Another `aliyun bl` process is installing and no binary is installed yet. Wait for it, and remove a stale lock only after confirming no installer is running. |
| `bl download <path>: HTTP <status>`, `invalid bl manifest ...`, or `SHA256 mismatch for <file>` | The release could not be fetched or failed verification, so nothing is installed. Check network access and proxy settings, then retry; do not bypass the digest check. |
| `invalid managed bl path <path>` | A path under `<config-root>/bl` has an unexpected type, for example a file where a directory is expected. Remove or correct that path. |
| The configuration root cannot be created or written | Fix permissions on `$HOME` / `%USERPROFILE%` or on `<config-root>`; the host cannot proceed without a writable configuration root. |

### Managed binary update checks

With the managed binary and no `ALIBABA_CLOUD_BL_EXEC_PATH`, `aliyun bl` decides whether to check for or install an update before it starts the child, as described below. Installation location, platform identifiers, and installation errors are covered in the installation sections above.

| Situation | Behavior |
| --- | --- |
| Installed, less than 24 hours since the last check | The remote check is skipped and the installed version runs. The check time is stored in `<config-root>/bl/update-check.json`. |
| Installed, more than 24 hours since the last check | The remote release is checked again and installed only when it is newer than the local version. |
| Installed, remote version not newer | Nothing is reinstalled and no other version is rolled back to. |
| Installed with `ALIBABA_CLOUD_BL_NO_UPDATE_CHECK=1` | The remote check is skipped, the installed version runs, and `NO_UPDATE_NOTIFIER=1` is added to the child environment. |

`ALIBABA_CLOUD_BL_NO_UPDATE_CHECK` recognizes only the exact value `1`; values such as `0`, `true`, or `yes` have no effect. The variable skips the update check only when a managed binary is already installed, and the first install still happens when none exists (see the installation section above). On Windows the variable name is matched case-insensitively.

When a check or install fails, `aliyun bl` prints a message starting with `Warning:` to standard error and continues with the installed version. A failed attempt also refreshes the 24-hour check time so that later invocations do not hit the remote again. If no usable managed binary exists, `aliyun bl` fails before starting the child instead of continuing silently (see "Installation errors"). Continuing with the installed version is not a rollback to an older release, and the previous release is not guaranteed to behave like the newer one.

Installation is serialized through the `<config-root>/bl/.install-lock` directory. With a concurrent install, an invocation that already has a usable binary prints `Warning:` and continues; an invocation without one fails and must retry after the install finishes. Remove a leftover lock directory manually only after confirming that no installer is running.

To confirm which version actually runs, use the version command the host forwards:

```sh
aliyun bl --version
```

Like every other non-host argument, it is forwarded unchanged to the child, so the output is the version of the child binary, whether that is the managed binary or the file named by `ALIBABA_CLOUD_BL_EXEC_PATH`.

### Profile selection

The host resolves the profile in this order:

1. `--profile <name>` (or `-p <name>`).
2. The `ALIBABACLOUD_PROFILE`, `ALIBABA_CLOUD_PROFILE`, or `ALICLOUD_PROFILE` environment variable, in that order.
3. The `current` profile in the configuration file.

`--config-path` replaces the default `~/.aliyun/config.json`. When `ALIBABA_CLOUD_IGNORE_PROFILE=TRUE` or `ALIBABACLOUD_IGNORE_PROFILE=TRUE` is set, the host skips the configuration file and uses only the explicitly supplied profile flags. Inherited `ALIBABA_CLOUD_ACCESS_KEY_ID`, `ALIBABA_CLOUD_ACCESS_KEY_SECRET`, and `ALIBABA_CLOUD_SECURITY_TOKEN` are never used as the host profile identity.

### Credential delivery to the child

The host keeps the current environment and changes only the variables described in this section. What the child receives depends on the resolved profile:

| Resolved profile | Child environment |
| --- | --- |
| No profile credentials: mode, access key ID, access key secret, and STS token are all empty | Passed through unchanged, including any `ALIBABA_CLOUD_*` credential variables exported in the shell. |
| `AK` mode | The access key ID and secret must both be set, otherwise `aliyun bl` fails before starting the child. The host removes any inherited `ALIBABA_CLOUD_ACCESS_KEY_ID`, `ALIBABA_CLOUD_ACCESS_KEY_SECRET`, and `ALIBABA_CLOUD_SECURITY_TOKEN`, then injects the profile pair. |
| `StsToken` mode | The access key pair and the STS token must all be set. The host removes the inherited values and injects all three. |
| `Anonymous` mode | The host removes the inherited access key ID, secret, and security token and injects nothing. Other variables are preserved. |
| Other modes resolved through the profile provider, such as OAuth, RAM role, CloudSSO, or external process profiles | The host resolves credentials through the provider. If resolution fails or returns an incomplete access key pair, `aliyun bl` fails before starting the child and does not fall back to inherited credentials. |

Only `ALIBABA_CLOUD_ACCESS_KEY_ID`, `ALIBABA_CLOUD_ACCESS_KEY_SECRET`, and `ALIBABA_CLOUD_SECURITY_TOKEN` are replaced or removed. All other entries keep their order and value, except for the runtime variables below.

The host also prepares runtime variables for the child: `PATH` starts with `<config>/bl/bin`; `BAILIAN_INSTALL_METHOD`, `BAILIAN_SHARE_DIR`, `BAILIAN_BIN_DIR`, and `ALIBABA_CLOUD_BL_COMPAT_MODE` are set to the values used by `aliyun bl`; and `ALIBABA_CLOUD_BL_NO_UPDATE_CHECK=1` skips updates of the managed binary and adds `NO_UPDATE_NOTIFIER=1` to the child environment.

See [Configuration and credentials](./configuration.md) for how each profile mode resolves credentials.

### Host and child argument ownership

`aliyun bl` forwards raw arguments. The host consumes the following flags, and their short forms where defined, without passing them to `bl`:

`--profile`, `--config-path`, `--sts-token`, `--sts-region`, `--sts-endpoint`, `--ram-role-name`, `--ram-role-arn`, `--role-session-name`, `--external-id`, `--source-profile`, `--private-key`, `--key-pair-name`, `--process-command`, `--oidc-provider-arn`, `--oidc-token-file`, `--sso-session`, `--cloud-sso-sign-in-url`, `--cloud-sso-access-config`, `--cloud-sso-account-id`, `--external-account-type`, `--bearer-token`, `--bearer-token-header-key`.

Every other argument is forwarded unchanged, including `--mode`, `--access-key-id`, `--access-key-secret`, `--security-token`, `--region`, `--config`, `--endpoint`, `-e`, `--output`, and unknown flags. Similar names can have different owners: `--sts-token` configures the host profile while `--security-token` belongs to `bl`; `--config-path` belongs to the host while `--config` belongs to `bl`.

`--` is a hard boundary. The host stops interpreting arguments at `--` and forwards everything from that point, including `--` itself, to `bl`. Use it when the child needs a value that matches a host flag name:

```sh
aliyun bl chat --profile prod -- --profile child-value
```

The host does not start the child when a consumed flag is missing its value, for example `aliyun bl --profile`.

### External binary and trust boundary

```sh
export ALIBABA_CLOUD_BL_EXEC_PATH=/opt/bailian/bin/bl
aliyun bl --version
```

With `ALIBABA_CLOUD_BL_EXEC_PATH` set, `aliyun bl` runs that executable without downloading, installing, or updating the managed binary. The value must name an existing regular file and, on non-Windows platforms, an executable file; otherwise `aliyun bl` fails explicitly. An empty or whitespace-only value is treated as unset.

The managed binary and the `ALIBABA_CLOUD_BL_EXEC_PATH` executable both run as your user in the current working directory with the inherited standard input, output, and error streams. They receive the prepared environment described above, including resolved credentials. Treat that executable as part of the trusted execution boundary and point the variable only at binaries you trust; this page is not a security endorsement of third-party binaries.

### Example with placeholder credentials

The examples use fake values only.

```sh
# A selected profile without credentials: exported credentials reach the child unchanged.
export ALIBABA_CLOUD_ACCESS_KEY_ID=<fake-ak>
export ALIBABA_CLOUD_ACCESS_KEY_SECRET=<fake-sk>
aliyun bl chat

# AK profile: the host injects the profile pair and drops the inherited STS token.
export ALIBABA_CLOUD_SECURITY_TOKEN=<fake-stale-token>
aliyun bl --profile ak-profile chat

# StsToken profile: the host injects the key pair and token from the profile.
aliyun bl --profile sts-profile chat
```
