# Alibaba Cloud CLI automatic update checks overview

[Documentation index](../README.md) | [简体中文](../zh-CN/update-check.md)

The CLI checks for CLI and installed plugin updates in the background during ordinary commands.
Checks do not block the current command. Install updates manually.

## Automatic checks and reminders

- Automatic checks run at most once per 24 hours. Check failures are silent. Without valid cached results, the current command shows no reminder.
- Successful commands, including `aliyun version`, CI, scripts and agent calls, retain their output and exit status.
- Plain text failures reported by the CLI may include the current version, upgrade version and upgrade command after the original error on stderr. The exit code stays unchanged.
- Supported AI errors include reminders in `update_notices`, retaining one JSON document. Errors printed directly by a plugin before it exits receive no appended reminder.
- Each component and target version is reminded at most once per 72 hours. Plugin reminders only concern the plugin involved in the call.

## Check and upgrade

```sh
aliyun upgrade --check                    # Check the CLI and all installed plugins
aliyun upgrade --check --cli-output json  # Output indented JSON
aliyun upgrade                           # Upgrade the CLI
aliyun plugin update --name <plugin-name> # Upgrade a plugin
```

Explicit checks default to a text summary and install nothing.
Completed checks exit 0, including when updates exist.
Incomplete checks exit nonzero and retain the results obtained.

When a Homebrew or Linuxbrew installation is detected, `aliyun upgrade` displays a message
and automatically runs `brew update`, followed by `brew upgrade aliyun-cli`.
Checks use the official release source; the version actually installed depends on what brew provides.

Update checks show plugin CLI version requirements. Installation or upgrade from
the plugin source is rejected if the CLI is too old, preserving the existing plugin. For execution
of an installed plugin, the version requirement remains advisory.

## Preferences

```sh
aliyun configure update-notifier show     # Show settings
aliyun configure update-notifier disable  # Disable
aliyun configure update-notifier enable   # Enable
export ALIBABA_CLOUD_CLI_NO_UPDATE_NOTIFIER=1  # Disable through the environment
```

Settings apply to all profiles. The environment disable override takes precedence.
Removing the variable does not override a saved disabled setting.
