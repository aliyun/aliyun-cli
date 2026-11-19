# CLI 扩展

[文档索引](../README.md) | [English](../en/cliext.md)

## ossutil

让 `aliyun ossutil` 使用已有的 ossutil v2 可执行文件：

```sh
export ALIBABA_CLOUD_OSSUTIL_EXEC_PATH=/usr/local/bin/ossutil
```

设置该变量后，`aliyun ossutil` 直接使用指定的可执行文件，跳过自动下载、安装和远端版本检查，工具版本由运行环境管理。路径不存在或不可执行时会明确报错，不回退到自动安装。未设置该变量或值为空时，保持原有自动安装和更新行为。

## bl

`aliyun bl <command> [args...]` 以子进程方式运行阿里云百炼 CLI（`bl`）。宿主负责选择 profile、准备子进程环境并转发其余参数。托管的 `bl` 二进制默认自动安装和更新；设置 `ALIBABA_CLOUD_BL_EXEC_PATH` 可改用已有可执行文件。

### 安装

`aliyun bl` 使用两种可执行文件之一：宿主自行安装的托管 `bl`，或通过 `ALIBABA_CLOUD_BL_EXEC_PATH` 提供的外部可执行文件。未设置该变量时使用托管二进制。

**托管安装。** 首次需要托管二进制时，宿主通过 HTTPS 从 `https://bailian-wiki.oss-cn-hangzhou.aliyuncs.com/release` 下载 `manifest.json` 和对应版本的 ZIP。宿主只接受身份、版本和各平台资产名都符合预期的 manifest，按 manifest 中的 SHA-256 摘要校验 ZIP，并要求压缩包内只有一个根可执行文件，然后才安装。由于宿主在转发参数之前准备二进制，`aliyun bl --help` 同样会触发安装。

二进制写入 `<config-root>/bl/bin/bl`（Windows 为 `bl.exe`），在 Linux 和 macOS 上权限为 `0755`，并替换该路径下已有的托管二进制。宿主还会在 `<config-root>/bl/update-check.json` 记录更新检查时间，并用 `<config-root>/bl/.install-lock` 串行化并发的首次安装。

**安装目录。** `<config-root>` 是宿主配置根目录：Linux 和 macOS 为 `$HOME/.aliyun`，Windows 为 `%USERPROFILE%\.aliyun`（优先 `%HOMEDRIVE%%HOMEPATH%`，为空时回退到 `%USERPROFILE%`）。该目录必须已存在或可被创建，且当前用户可写。`--config-path` 只改变宿主读取的配置文件位置，不会移动托管 `bl` 二进制。

**支持平台。** 托管二进制按以下平台标识发布：

| 宿主平台 | 托管标识 |
| --- | --- |
| macOS（Apple 芯片） | `darwin-arm64` |
| macOS（Intel） | `darwin-x64` |
| Linux x86-64 | `linux-x64` |
| Windows x86-64 | `windows-x64`（`amd64` 会映射为 `x64`） |

其他平台会在安装前显式失败：

```text
unsupported bl platform linux/arm64; supported platforms: darwin-arm64, darwin-x64, linux-x64, windows-x64; or set ALIBABA_CLOUD_BL_EXEC_PATH
```

`ALIBABA_CLOUD_BL_EXEC_PATH` 不受托管平台检查约束，因此在没有托管版本的主机上，应使用经过审核的外部可执行文件。

**离线与受控环境。** 托管安装需要访问发布端点，没有离线首装：在尚未安装托管二进制时，如果下载失败，`aliyun bl` 会直接报错，不会用不完整的二进制启动。需要在无网络环境运行时，请自行预置经过审核的 `bl` 可执行文件，并把 `ALIBABA_CLOUD_BL_EXEC_PATH` 设置为其绝对路径（参见下文“外部二进制与可信边界”）；该路径不会下载、安装或检查托管版本。两条路径都不依赖 Node.js 或 npm：宿主直接运行独立的 `bl` 可执行文件，并为子进程设置 `BAILIAN_INSTALL_METHOD=binary`。

下载仅按 `manifest.json` 中发布的 SHA-256 摘要校验；manifest 本身没有独立签名，该校验不证明发布者身份。

### 安装错误

| 报错或现象 | 处理方式 |
| --- | --- |
| `unsupported bl platform <os>/<arch>; supported platforms: ...; or set ALIBABA_CLOUD_BL_EXEC_PATH` | 当前主机没有托管版本。改用受支持平台，或把 `ALIBABA_CLOUD_BL_EXEC_PATH` 指向经过审核的二进制。 |
| `ALIBABA_CLOUD_BL_EXEC_PATH="<path>" does not point to an existing file`、`... is not a regular file` 或 `... is not executable` | 使用指向已存在普通文件的绝对路径；在 Linux 和 macOS 上必须具备可执行位。空值或纯空白视为未设置。 |
| `bl install lock exists at <config-root>/bl/.install-lock; retry after the installer finishes; ...` | 另一个 `aliyun bl` 进程正在安装，且当前还没有已安装的二进制。等待其结束；确认没有安装进程后再清理遗留锁。 |
| `bl download <path>: HTTP <status>`、`invalid bl manifest ...` 或 `SHA256 mismatch for <file>` | 发布包无法获取或校验失败，宿主不会安装。检查网络和代理后重试；不要绕过摘要校验。 |
| `invalid managed bl path <path>` | `<config-root>/bl` 下某个路径类型异常，例如本应是目录却是文件。删除或修正该路径。 |
| 配置根目录无法创建或写入 | 修正 `$HOME` / `%USERPROFILE%` 或 `<config-root>` 的权限；没有可写的配置根目录时宿主无法继续。 |

### 托管二进制的更新检查

使用托管二进制且未设置 `ALIBABA_CLOUD_BL_EXEC_PATH` 时，`aliyun bl` 在启动子进程前按下表决定是否检查或安装更新；安装位置、平台标识和安装错误见上文“安装”“安装错误”两节。

| 场景 | 行为 |
| --- | --- |
| 已安装，距上次检查不足 24 小时 | 跳过远端检查，直接运行已安装版本。检查时间记录在 `<config-root>/bl/update-check.json`。 |
| 已安装，距上次检查超过 24 小时 | 重新检查远端，只有远端版本高于本地版本时才安装。 |
| 已安装，远端版本不高于本地版本 | 不重装，也不回退到其他版本。 |
| 已安装并设置 `ALIBABA_CLOUD_BL_NO_UPDATE_CHECK=1` | 跳过远端检查，直接运行已安装版本，并向子进程环境加入 `NO_UPDATE_NOTIFIER=1`。 |

`ALIBABA_CLOUD_BL_NO_UPDATE_CHECK` 只识别精确值 `1`，`0`、`true`、`yes` 等取值都不生效；该变量只在已有托管二进制时跳过更新检查，尚未安装时仍会执行首次安装（见上文“安装”）。Windows 上变量名按不区分大小写匹配。

检查或安装失败时，`aliyun bl` 在标准错误输出打印以 `Warning:` 开头的说明，然后继续运行已安装的版本；失败同样会刷新 24 小时检查时间，避免每次调用都重复访问远端。若此时没有可用的托管二进制，`aliyun bl` 会在启动子进程前报错退出（见上文“安装错误”）。这种“失败时沿用已安装版本”的行为不是回退到某个更旧的版本，也不保证新旧版本行为一致。

安装过程用 `<config-root>/bl/.install-lock` 目录互斥。并发安装时，已有可用二进制的一方打印 `Warning:` 后继续运行；没有可用二进制的一方报错，需等待安装结束后重试。只有确认没有安装进程在运行时，才应手动删除残留的锁目录。

要确认实际执行的版本，直接运行由宿主转发的版本命令：

```sh
aliyun bl --version
```

该参数与其余非宿主参数一样原样转发给子进程，输出的是子进程（托管二进制或 `ALIBABA_CLOUD_BL_EXEC_PATH` 指定的文件）的版本。

### profile 选择

宿主按以下顺序确定 profile：

1. `--profile <name>`（或 `-p <name>`）。
2. 环境变量 `ALIBABACLOUD_PROFILE`、`ALIBABA_CLOUD_PROFILE`、`ALICLOUD_PROFILE`，按此顺序。
3. 配置文件中的 `current` profile。

`--config-path` 用于替换默认的 `~/.aliyun/config.json`。设置 `ALIBABA_CLOUD_IGNORE_PROFILE=TRUE` 或 `ALIBABACLOUD_IGNORE_PROFILE=TRUE` 时，宿主跳过配置文件，只使用显式传入的 profile 参数。继承的 `ALIBABA_CLOUD_ACCESS_KEY_ID`、`ALIBABA_CLOUD_ACCESS_KEY_SECRET` 和 `ALIBABA_CLOUD_SECURITY_TOKEN` 不会作为宿主 profile 的身份来源。

### 子进程凭据传递

宿主保留当前环境，只改动本节列出的变量。子进程最终拿到什么取决于解析出的 profile：

| 解析出的 profile | 子进程环境 |
| --- | --- |
| 无 profile 凭据：mode、access key ID、access key secret、STS token 均为空 | 原样透传，包括 shell 中导出的任何 `ALIBABA_CLOUD_*` 凭据变量。 |
| `AK` 模式 | access key ID 与 secret 必须同时配置，否则 `aliyun bl` 在启动子进程前报错。宿主先删除继承的 `ALIBABA_CLOUD_ACCESS_KEY_ID`、`ALIBABA_CLOUD_ACCESS_KEY_SECRET`、`ALIBABA_CLOUD_SECURITY_TOKEN`，再注入 profile 中的这一对凭据。 |
| `StsToken` 模式 | access key ID、secret 与 STS token 必须全部配置。宿主删除继承值并注入全部三项。 |
| `Anonymous` 模式 | 宿主删除继承的 access key ID、secret 与 security token，不注入任何凭据，其他变量保留。 |
| 其他通过凭据提供方解析的模式，例如 OAuth、RAM 角色、CloudSSO 或外部进程 profile | 宿主通过凭据提供方解析。解析失败或返回不完整的 access key 对时，`aliyun bl` 在启动子进程前报错，不回退到继承凭据。 |

只有 `ALIBABA_CLOUD_ACCESS_KEY_ID`、`ALIBABA_CLOUD_ACCESS_KEY_SECRET`、`ALIBABA_CLOUD_SECURITY_TOKEN` 会被替换或删除。其余条目保持原有顺序和值，下面列出的运行时变量除外。

宿主还会为子进程准备运行时变量：`PATH` 以 `<config>/bl/bin` 开头；`BAILIAN_INSTALL_METHOD`、`BAILIAN_SHARE_DIR`、`BAILIAN_BIN_DIR`、`ALIBABA_CLOUD_BL_COMPAT_MODE` 设置为 `aliyun bl` 使用的值；`ALIBABA_CLOUD_BL_NO_UPDATE_CHECK=1` 会跳过托管二进制的更新，并向子进程环境加入 `NO_UPDATE_NOTIFIER=1`。

各 profile 模式的凭据解析细节见[配置与凭证](./configuration.md)。

### 宿主参数与子进程参数归属

`aliyun bl` 直接转发原始参数。以下参数（以及已定义的短参数）由宿主消费，不会传给 `bl`：

`--profile`、`--config-path`、`--sts-token`、`--sts-region`、`--sts-endpoint`、`--ram-role-name`、`--ram-role-arn`、`--role-session-name`、`--external-id`、`--source-profile`、`--private-key`、`--key-pair-name`、`--process-command`、`--oidc-provider-arn`、`--oidc-token-file`、`--sso-session`、`--cloud-sso-sign-in-url`、`--cloud-sso-access-config`、`--cloud-sso-account-id`、`--external-account-type`、`--bearer-token`、`--bearer-token-header-key`。

其余参数原样转发给子进程，包括 `--mode`、`--access-key-id`、`--access-key-secret`、`--security-token`、`--region`、`--config`、`--endpoint`、`-e`、`--output` 以及未知参数。名称相近的参数归属可能不同：`--sts-token` 配置宿主 profile，而 `--security-token` 属于 `bl`；`--config-path` 属于宿主，而 `--config` 属于 `bl`。

`--` 是硬边界。宿主遇到 `--` 后停止解析，把从该位置开始（含 `--` 本身）的内容全部转发给 `bl`。当子命令需要用与宿主参数同名的值时使用：

```sh
aliyun bl chat --profile prod -- --profile child-value
```

宿主参数缺少取值时不会启动子进程，例如 `aliyun bl --profile`。

### 外部二进制与可信边界

```sh
export ALIBABA_CLOUD_BL_EXEC_PATH=/opt/bailian/bin/bl
aliyun bl --version
```

设置 `ALIBABA_CLOUD_BL_EXEC_PATH` 后，`aliyun bl` 直接运行该可执行文件，不下载、安装或更新托管二进制。该值必须指向已存在的普通文件，非 Windows 平台还要求可执行，否则 `aliyun bl` 明确报错。空值或仅含空白字符按未设置处理。

托管二进制和 `ALIBABA_CLOUD_BL_EXEC_PATH` 指定的可执行文件都以当前用户身份、在当前工作目录运行，并继承标准输入、输出和错误流。它们会收到上述准备好的环境，其中包含解析后的凭据。请把该可执行文件视为可信执行边界的一部分，只指向你信任的二进制；本页不构成对第三方二进制的安全背书。

### 使用占位凭据的示例

以下示例只使用假值。

```sh
# 所选 profile 无凭据：导出的凭据原样传给子进程。
export ALIBABA_CLOUD_ACCESS_KEY_ID=<fake-ak>
export ALIBABA_CLOUD_ACCESS_KEY_SECRET=<fake-sk>
aliyun bl chat

# AK profile：宿主注入 profile 中的 AK/SK，并移除继承的 STS token。
export ALIBABA_CLOUD_SECURITY_TOKEN=<fake-stale-token>
aliyun bl --profile ak-profile chat

# StsToken profile：宿主注入 profile 中的 AK/SK 与 token。
aliyun bl --profile sts-profile chat
```
