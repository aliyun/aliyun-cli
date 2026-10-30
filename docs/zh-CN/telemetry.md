# Aliyun CLI 遥测数据采集说明

[文档索引](../README.md) | [English](../en/telemetry.md)

## 数据采集说明

阿里云 CLI 遥测默认开启。CLI 会在命令执行结束后采集结构化的使用信息，并通过后台进程经 HTTPS 发送至阿里云，用于分析功能使用情况、兼容性、执行结果和性能，改进产品稳定性与易用性。

采集范围包括普通命令、认证配置和插件管理等操作，也包括没有发起云端 API 请求的本地执行结果。`aliyun configure telemetry` 及其管理子命令自身不采集。原命令不等待遥测网络上传结果；关闭遥测不影响正常命令功能。

## 采集哪些信息

以下列出遥测事件中的字段及用途。部分字段仅在相应命令或运行环境中提供；命令、环境和插件属性存放在事件的 `attributes` 对象中。

| 类别 | 字段 | 说明与用途 |
| --- | --- | --- |
| 事件信息 | `schemaVersion`、`eventId`、`eventType`、`timestamp` | 数据格式版本、每条事件随机生成的 ID、事件类别和命令完成时间，用于区分事件、统计与排查问题。 |
| 客户端信息 | `clientName`、`clientVersion` | 客户端名称和 CLI 版本，用于分析版本使用情况与兼容性。 |
| 安装实例标识 | `instanceId` | 在本机随机生成，用于统计同一安装实例的使用情况，不基于账号或硬件信息生成；可通过 `purge` 重置。 |
| 命令信息 | `commandType`、`command`、`parameterNames` | 命令实现类型、命令及子命令名称、命名参数的名称，用于分析功能和参数使用情况；不记录业务参数值。 |
| 执行结果 | `status`、`durationMs`、`localErrorCode`、`apiErrorCode`、`httpStatusCode` | 执行结果、命令执行耗时、结构化错误码和 HTTP 状态码，用于分析稳定性和性能；不记录错误文案。 |
| 系统环境 | `os`、`arch`、`goVersion`、`linuxDistroName`、`linuxDistroId`、`linuxDistroVersion` | 操作系统、CPU 架构、Go 运行时版本及适用时的 Linux 发行版信息，用于分析兼容性。 |
| 终端环境 | `shellType`、`stdinIsTTY`、`locale` | Shell 类型、标准输入是否连接终端、语言及编码设置，用于分析交互式与自动化使用场景。 |
| 环境变量名称 | `environmentVariableNames` | 只记录符合格式限制的 `ALIBABA_CLOUD_` 前缀环境变量名称，用于分析配置方式；不记录这些变量的值。 |
| 配置与集成 | `authMode`、`authModeSource`、`aiModeEnabled`、`agentName` | 认证模式、配置模式的选择来源、是否开启 AI 模式，以及识别到的已知 Agent 类型；不包含认证凭据。 |
| 插件信息 | `plugin`、`pipeline` | 已安装插件的名称和版本，以及适用时的内部插件分类标记，用于分析插件使用情况。 |
| 插件管理 | `operation`、`pluginNames`、`fromVersion`、`toVersion`、`pluginType`、`inner`、`sourceType` | 插件管理操作、涉及的插件名称和版本、插件类型与属性、安装来源类别；不记录安装包路径、下载 URL 或源码地址。 |

命令信息记录命令和参数名称，不采集业务参数值。认证模式、语言设置、插件版本等信息按上表列出的字段记录。

## 不采集哪些信息

上述结构化遥测事件不采集以下内容：

- AccessKey ID、AccessKey Secret、SecurityToken、Bearer Token、Cookie、密码等凭据。
- 阿里云账号 ID、UID、员工账号、邮箱、用户名、主机名、MAC 地址或硬件设备标识。
- 业务参数值、完整命令行、`ALIBABA_CLOUD_` 环境变量的值或完整环境变量转储。
- 本地文件路径、文件内容、SQL、模板、请求正文、服务响应正文或命令输出。
- 错误文案、错误摘要和调用堆栈。
- 会话 ID、终端 ID、父进程 ID、OpenAPI 或网关 RequestId、遥测接口响应 RequestId。
- 当前 Profile 配置的云地域、客户端 IP 字段。

这里的范围是 CLI 构造并上传的遥测事件，不涵盖网络接入层日志或您主动调用云服务时发送的业务请求。CLI 的普通认证配置仍可能保存凭据，详见[配置与凭证](./configuration.md)。

## 如何管理遥测

遥测设置作用于同一用户配置目录中的所有 Profile，统一控制普通命令和内部插件的遥测。

| 操作 | 命令 |
| --- | --- |
| 查看基础配置 | `aliyun configure telemetry show` |
| 关闭遥测 | `aliyun configure telemetry disable` |
| 重新开启遥测 | `aliyun configure telemetry enable` |

`show` 的输出示例：

```json
{
  "enabled": true,
  "environment_disabled": false,
  "config_file": "/Users/example/.aliyun/telemetry.json"
}
```

`enabled` 表示本地开关，`environment_disabled` 表示是否被环境变量禁用，`config_file` 是配置文件路径。只有 `enabled` 为 `true` 且 `environment_disabled` 为 `false` 时，开关才允许采集。`show` 不会生成安装 ID，也不读取 ID 或扫描缓存。

也可以通过环境变量禁用遥测，适合脚本、CI 或无需写入配置的环境。

macOS / Linux Shell：

```sh
export ALIBABA_CLOUD_CLI_TELEMETRY_DISABLE=1
```

PowerShell：

```powershell
$env:ALIBABA_CLOUD_CLI_TELEMETRY_DISABLE = "1"
```

环境变量的值为 `1`、`true` 或 `yes` 时禁用遥测，不区分大小写。设为 `0`、`false`、`no` 或移除该变量，只撤销环境变量禁用，不会覆盖本地关闭设置。执行 `enable` 后，如果环境变量仍要求禁用，遥测仍保持关闭。

需要重新开启时，先移除环境变量，再执行 `aliyun configure telemetry enable`：

```sh
unset ALIBABA_CLOUD_CLI_TELEMETRY_DISABLE
aliyun configure telemetry enable
```

```powershell
Remove-Item Env:ALIBABA_CLOUD_CLI_TELEMETRY_DISABLE -ErrorAction SilentlyContinue
aliyun configure telemetry enable
```

关闭设置对后续采集生效，不会撤回已上传的数据或中止已经启动的上传进程。

## 本地数据与清理

遥测使用以下本地文件。`~` 表示用户主目录；Windows 上对应用户目录下的 `.aliyun` 文件夹。

| 路径 | 用途 |
| --- | --- |
| `~/.aliyun/telemetry.json` | 保存是否开启遥测的设置。 |
| `~/.aliyun/telemetry-instance-id` | 保存随机生成的安装实例标识。 |
| `~/.aliyun/telemetry-cache/` | 保存交给后台上传进程处理的临时事件文件。 |

临时事件文件用于本地进程之间交接数据，不是完整命令日志。上传进程读取后会尝试删除该文件，再发送请求；上传失败不进行历史事件重试。

清理本地缓存并重置安装实例标识：

```sh
aliyun configure telemetry purge
```

`purge` 保留当前遥测开关设置，不删除已上传的数据，也不取消已经开始的采集或上传。如果遥测仍开启，后续符合采集条件的命令会生成新的随机标识。

如果希望关闭遥测并清理已有本地数据，依次执行：

```sh
aliyun configure telemetry disable
aliyun configure telemetry purge
```
