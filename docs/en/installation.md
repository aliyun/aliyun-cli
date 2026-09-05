# Installation and upgrade

[Documentation index](../README.md) | [简体中文](../zh-CN/installation.md)

Alibaba Cloud CLI supports macOS, Linux, and Windows. After installation, the executable name is `aliyun` (`aliyun.exe` on Windows).

## Package downloads

Download and extract the package for your platform, then place the executable in a directory on `PATH`.

| Platform | Package |
| --- | --- |
| macOS | [GUI installer](https://aliyuncli.alicdn.com/aliyun-cli-latest.pkg) |
| macOS | [Universal archive](https://aliyuncli.alicdn.com/aliyun-cli-macosx-latest-universal.tgz) |
| Linux AMD64 | [Archive](https://aliyuncli.alicdn.com/aliyun-cli-linux-latest-amd64.tgz) |
| Linux ARM64 | [Archive](https://aliyuncli.alicdn.com/aliyun-cli-linux-latest-arm64.tgz) |
| Windows x64 | [ZIP archive](https://aliyuncli.alicdn.com/aliyun-cli-windows-latest-amd64.zip) |

All published versions are available from [GitHub Releases](https://github.com/aliyun/aliyun-cli/releases).

On macOS or Linux, a typical manual installation is:

```sh
tar -xzf <downloaded-archive>.tgz
chmod +x aliyun
sudo mv aliyun /usr/local/bin/aliyun
aliyun version
```

On Windows, extract `aliyun.exe` and add its directory to the user or system `Path` environment variable, then open a new terminal.

## Homebrew

```sh
brew install aliyun-cli
aliyun version
```

Upgrade a Homebrew installation with:

```sh
brew update
brew upgrade aliyun-cli
```

## Installer script

The installer script supports macOS and Linux:

```sh
/bin/bash -c "$(curl -fsSL https://aliyuncli.alicdn.com/install.sh)"
```

In environments with strict software supply-chain controls, download and review the script before running it.

## Upgrade an existing CLI

The CLI includes an upgrade command:

```sh
aliyun upgrade
```

If the CLI was installed through a package manager, prefer that package manager's upgrade command. You can always replace a standalone binary with a newer package from the Releases page.

## Build from source

Requirements:

- Git with submodule support
- The Go version declared in [`go.mod`](../../go.mod)
- Platform tools required by the selected Make target

Clone submodules and build the release-style binary with packed metadata:

```sh
git clone --recurse-submodules https://github.com/aliyun/aliyun-cli.git
cd aliyun-cli
make build
./out/aliyun version
```

For a development build that reads metadata from the checked-out submodule:

```sh
go build -o out/aliyun ./main
```

## Troubleshooting

- `command not found`: ensure the executable directory is on `PATH`, then open a new shell.
- Permission denied on macOS/Linux: run `chmod +x aliyun`.
- Wrong CPU architecture: download the archive matching AMD64/x86_64 or ARM64.
- Old version still runs: use `command -v aliyun` (or `where aliyun` on Windows) to find duplicate installations.

Next: [configure credentials](./configuration.md).
