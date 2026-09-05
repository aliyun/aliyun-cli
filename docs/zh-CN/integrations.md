# MCP 代理、OpenTelemetry 与机器可读接口

[文档索引](../README.md) | [English](../en/integrations.md)

## MCP 代理

MCP 代理通过一个本地入口暴露已发现的阿里云 API MCP Server，并负责访问上游服务所需的 OAuth 流程。

使用公开命令路径启动：

```sh
aliyun utils mcp-proxy --region-type CN --port 8088
```

默认配置如下：

| 参数 | 默认值 | 含义 |
| --- | --- | --- |
| `--host` | `127.0.0.1` | 本地监听地址 |
| `--port` | `8088` | 本地监听端口 |
| `--region-type` | `CN` | Endpoint 类型，可选 `CN` 或 `INTL` |
| `--scope` | `/acs/mcp-server` | OAuth scope |
| `--no-browser` | 关闭 | 启用后不自动打开浏览器，改为手工输入授权码 |
| `--oauth-app-name` | 空 | 按名称复用已有 OAuth 应用 |
| `--upstream-url` | 空 | 覆盖配置中的上游 MCP 地址 |

启动后，CLI 会输出已发现 Server 对应的本地 MCP 和 SSE 地址。MCP 客户端应使用输出的地址，不要根据 Server 名称自行拼接路径。

### 访问控制

`--allowed-servers` 和 `--blocked-servers` 接受以逗号分隔的 Server 名称、ID 或路径前缀：

```sh
aliyun utils mcp-proxy \
  --allowed-servers ecs-tools,/mcp/approved \
  --blocked-servers deprecated-server
```

禁止列表优先于允许列表。为了兼容已有配置，两项都省略时会允许所有已发现的 Server。应尽量保留默认的回环监听地址；监听 `0.0.0.0` 或其他非回环地址前，必须配置允许列表，并在主机或网络层增加访问控制。

### 超时与流式连接

本地 Server 的请求头读取超时为 10 秒，完整请求读取超时为 2 分钟，请求正文上限为 64 MiB；超限请求会返回 HTTP 413。空闲 keep-alive 连接超时和等待上游响应头的上限均为 2 分钟。Server 没有全局写入超时，HTTP Client 也没有总请求超时，因此已经建立的 SSE 流不会被这些限制中断。

### 日志与敏感数据

代理日志会保留排障所需的 HTTP 方法、URL origin/path、状态、耗时和载荷字节数。URL user-info、查询参数、fragment、请求正文、OAuth 授权码、OAuth 错误正文和 SSE 消息内容不会写入日志。不要把凭证放在 MCP Server 路径中，因为路径会用于路由诊断并保留在日志里。

## OpenTelemetry Trace Context 传播

阿里云 CLI 可以把 W3C Trace Context 传播到 OpenAPI 请求。该能力只负责向出站请求注入 Header；CLI 本身不会因此创建 span 或向遥测后端导出数据。

| 环境变量 | 行为 |
| --- | --- |
| `ALIBABA_CLOUD_OTEL_TRACEPARENT` | 校验格式后作为 `traceparent` Header 注入 |
| `ALIBABA_CLOUD_OTEL_BAGGAGE` | 不改变值，作为 `baggage` Header 注入 |
| `ALIBABA_CLOUD_OTEL_ENABLED` | `false`、`0` 或 `off` 会关闭传播；其他值只有在 traceparent 或 baggage 存在时才会产生 Header |

示例：

```sh
export ALIBABA_CLOUD_OTEL_TRACEPARENT=00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
export ALIBABA_CLOUD_OTEL_BAGGAGE='workflow.id=deploy-42,environment=staging'

aliyun ecs DescribeRegions
```

`traceparent` 必须使用小写十六进制，并符合以下形式：

```text
00-<32 位十六进制 trace id>-<16 位十六进制 parent id>-<2 位十六进制 flags>
```

无效的 `traceparent` 值会被忽略，并向 stderr 输出警告。环境变量中合法、非空的 `traceparent` 或 `baggage` 值优先于通过 `--header` 指定的同名 Header。兼容的子插件会通过内部编码环境变量接收相同值，用户不应直接设置该内部变量。

应把 baggage 视为会发送给目标服务的数据，不要放入 AccessKey、Token、授权码、密码或其他机密信息。

## 机器可读 Help

在 Help 操作中使用 `--cli-output json` 显式请求 JSON：

```sh
aliyun --help --cli-output json
aliyun ecs --help --cli-output json
aliyun ecs DescribeInstances --help --cli-output json
aliyun ecs DescribeInstances --cli-section request --help --cli-output json
aliyun ecs DescribeInstances --cli-section response --help --cli-output json
aliyun ecs --help-search instance --cli-output json
```

`--cli-output json` 只选择 Help 格式，不是 API 执行结果格式参数。OpenAPI 成功响应默认已经是 JSON；需要格式化 API 响应时使用 `--output` 或 `--cli-query`。

机器 Help 会向 stdout 写入一个 JSON 文档，并以换行结尾。通用字段如下：

| 字段 | 含义 |
| --- | --- |
| `schemaVersion` | 机器 Help 协议版本，当前为 `v1` |
| `helpLevel` | `root`、`utility`、`product`、`api` 或 `parameter` |
| `name` | 当前 Help 目标 |
| `description` | 当前输出语言对应的说明 |
| `target` | 解析后的命令路径、命令风格及适用时的 API 版本 |
| `commands`、`parameters`、`flags` | 当前层级可用的条目 |
| `result.shown`、`result.total`、`result.truncated` | 当前投影及截断信息 |
| `next` | 搜索、完整输出或切换 Help section 的建议命令 |

`--help-all` 单独使用时可以取消正常结果数量上限。`--help-search <文本>` 用于筛选当前 Help 层级，并默认返回全部匹配项。搜索时同时传入 `--help-all` 仍然兼容，但不会改变结果集。建议使用尽量具体的搜索词以控制机器输出体积。调用方应根据 `schemaVersion` 和 `helpLevel` 分支处理，忽略未知字段，并优先使用 `next`，不要自行拼接后续命令。

机器 Help 请求错误会写入 stderr，并以状态码 2 退出，采用带版本号的 envelope：

```json
{
  "schemaVersion": "v1",
  "error": {
    "code": "INVALID_CLI_OUTPUT",
    "message": "--cli-output only supports json",
    "target": ["aliyun"],
    "suggestions": []
  }
}
```

## Agent 模式错误

可以全局启用 Agent 行为，也可以只对单次命令启用：

```sh
export ALIBABA_CLOUD_CLI_AI_MODE=1
# 或
aliyun ecs describe-instances --cli-ai-mode
```

目前支持的本地用法错误、查询错误、传输错误、OAuth 错误和服务端错误会以单个紧凑 JSON 对象写入 stderr；成功结果仍写入 stdout。没有值的可选字段会被省略：

```json
{
  "message": "unknown flag --instnace-type",
  "did_you_mean": ["--instance-type"],
  "recovery": {
    "action": "search_parameter",
    "command": "aliyun ecs describe-instances --help-search instance-type",
    "hint": "Search request parameters related to instance-type."
  }
}
```

远端服务错误还可能包含 `error_code`、`status_code` 和 `request_id`。`did_you_mean` 和 `recovery.command` 也是可选字段；每个结构化 Agent 错误都会包含 `message`、`recovery.action` 和 `recovery.hint`。

Agent 错误对象是一套独立的紧凑接口，目前没有 `schemaVersion`；机器 Help 的 `v1` 协议不适用于它。并非所有错误都已经结构化，因此调用方还必须兼容 stderr 中的人类可读错误。

| 退出状态 | 含义 |
| --- | --- |
| `0` | 命令或 Help 成功 |
| `1` | 一般执行失败 |
| `2` | 用法错误、结构化 Agent 错误或机器 Help 请求错误 |
| `3` | 带 CLI 恢复提示的失败 |

需要在不调用 API 的情况下检查请求时，`--cli-dry-run-json` 会输出结构化请求详情；`--cli-dry-run` 是人类可读形式。

下一步：[命令、输出与自动化](./usage.md)。
