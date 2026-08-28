# Plugin management

[Documentation index](../README.md) | [简体中文](../zh-CN/plugins.md)

Alibaba Cloud CLI can extend product commands through plugins. A product shown as “Plugin available but not installed” in root help can be installed without replacing the main CLI binary.

## Discover plugins

```sh
# Installed plugins
aliyun plugin list

# Plugins available from the configured remote index
aliyun plugin list-remote

# Find a plugin by command/product name
aliyun plugin search <command-name>

# Inspect one installed plugin
aliyun plugin show --name <plugin-name>
```

Root help also indicates when a product plugin is available:

```sh
aliyun --help
```

## Install

Install one plugin from the remote index:

```sh
aliyun plugin install --name <plugin-name>
```

Install multiple plugins:

```sh
aliyun plugin install --names <plugin-one> <plugin-two>
```

Select an exact version or allow a pre-release:

```sh
aliyun plugin install --name <plugin-name> --version <version>
aliyun plugin install --name <plugin-name> --enable-pre
```

Install a local or HTTPS plugin archive:

```sh
aliyun plugin install --package ./plugin-package.tar.gz
aliyun plugin install --package https://example.com/plugin-package.zip
```

Supported archive suffixes are `.zip`, `.tar.gz`, and `.tgz`. When installing from a local file or URL, make sure the plugin comes from a trusted source.

## Update and remove

```sh
# Update one plugin
aliyun plugin update --name <plugin-name>

# Update every installed plugin
aliyun plugin update

# Allow pre-release updates
aliyun plugin update --name <plugin-name> --enable-pre

# Remove one plugin
aliyun plugin uninstall --name <plugin-name>
```

## Plugin source

Show the global plugin settings:

```sh
aliyun configure plugin-settings show
```

Set or clear a custom plugin tree base URL:

```sh
aliyun configure plugin-settings set --source-base https://example.com/plugins
aliyun configure plugin-settings clear
```

For one command, `--source-base <url>` overrides the global setting. The environment variable `ALIBABA_CLOUD_CLI_PLUGIN_SOURCE_BASE` has priority over the saved setting.

## Automatic product-plugin installation

Automatic installation can be saved in a profile:

```sh
aliyun configure set --profile default --auto-plugin-install true
```

Allowing pre-release packages is a separate choice:

```sh
aliyun configure set \
  --profile default \
  --auto-plugin-install true \
  --auto-plugin-install-enable-pre true
```

For deterministic CI environments, explicitly install and pin the required plugins instead of relying on automatic installation.

## Storage and troubleshooting

Plugins are stored under `~/.aliyun/plugins` by default. Set `ALIBABA_CLOUD_CLI_PLUGINS_DIR` to use a different directory.

Useful checks:

```sh
aliyun plugin list
aliyun plugin show --name <plugin-name>
aliyun plugin update --name <plugin-name>
```

If installation fails:

- confirm that the plugin name exists in `plugin list-remote`;
- verify network access to the configured plugin source;
- check that the package supports the current operating system and architecture;
- update the main CLI if the plugin requires a newer minimum CLI version;
- do not manually edit the plugin manifest unless diagnosing a corrupted local installation.

Internal hidden bulk-install commands are intentionally not part of the public workflow or compatibility contract.
