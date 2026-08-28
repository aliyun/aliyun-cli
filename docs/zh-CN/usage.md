# 命令、输出与自动化

[文档索引](../README.md) | [English](../en/usage.md)

## 命令形式

RPC API 可以使用传统大驼峰命名（PascalCase）的操作名；在 Canonical metadata 可用时，也可以使用短横线命名（kebab-case）的命令名：

```sh
aliyun <product> <Operation> [--Parameter value ...]
aliyun <product> <operation-name> [--parameter value ...]

aliyun ecs DescribeInstances --RegionId cn-hangzhou
aliyun ecs describe-instances --region-id cn-hangzhou
```

常规 API 调用应优先使用上述产品和操作名形式。对于只能通过原始 HTTP 方法和路径访问的 API，CLI 仍保留以下 RESTful 兼容形式，但不建议作为一般用法：

```sh
aliyun cs GET /clusters
aliyun cs POST /clusters --body "$(cat input.json)"
aliyun cs DELETE /clusters/<cluster-id>
```

每个 API 只支持一种 API 风格。不确定时请查看对应 OpenAPI 文档或 CLI 本地帮助。

## 查找命令和参数

人类可读帮助：

```sh
aliyun --help
aliyun ecs DescribeInstances --help
aliyun ecs describe-instances --help
```

## Endpoint 与 metadata 未收录的 API

CLI 通常从 metadata 解析 API 版本和 Endpoint。调用内置 metadata 未收录的 API 时，需要同时指定版本、Endpoint 和 `--force`：

```sh
aliyun newproduct SomeAction \
  --version 2025-01-01 \
  --endpoint newproduct.aliyuncs.com \
  --SomeParameter value \
  --force
```

`--force` 只跳过本地 API 和参数 metadata 校验，不会绕过服务端鉴权或校验。

可以重复使用 `--header X-foo=bar` 添加自定义 HTTP Header。`--secure` 强制使用 HTTPS。`--skip-secure-verify` 会关闭证书校验，不建议使用。

## 调用前验证

对写操作可以先使用 `--cli-dry-run`：

```sh
aliyun ecs run-instances ... --cli-dry-run
```

具体支持情况取决于 API metadata。CLI dry-run 验证不能替代权限和资源影响审查。

## 筛选和格式化输出

### 表格输出

`--output` 接受列名和可选的 JMESPath 行选择表达式：

```sh
aliyun ecs DescribeInstances \
  --output cols=InstanceId,Status rows=Instances.Instance[]
```

- `cols`：以逗号分隔的表格列字段。
- `rows`：可选，用 JMESPath 表达式选择行数据集合。

### JMESPath 查询

使用 `--cli-query` 筛选或重组 JSON 输出：

```sh
aliyun ecs describe-instances \
  --cli-query 'Instances.Instance[].{Id:InstanceId,Status:Status}'
```

支持时可以使用 `--quiet` 隐藏普通输出。

## 分页

对支持分页的 API，`--pager` 会请求后续页面并合并结果：

```sh
aliyun ecs describe-instances --pager
```

不同产品的分页字段不同，请查看具体 API 帮助。

## 等待目标状态

`--waiter` 会轮询 API，直到 JMESPath 表达式达到目标值：

```sh
aliyun ecs DescribeInstances \
  --InstanceIds '["i-example"]' \
  --waiter expr='Instances.Instance[0].Status' to=Running interval=5 timeout=300
```

- `expr`：从 JSON 响应选择被轮询字段。
- `to`：期望值。
- `interval`：可选，轮询间隔秒数。
- `timeout`：可选，总超时秒数。

## 安全策略与非交互使用

有潜在破坏性的操作可能受到 CLI 安全策略和人工确认控制。管理安全策略：

```sh
aliyun configure safety-policy --help
```

`--yes` 可以在非交互场景跳过确认提示，但不能绕过 deny 策略。在自动化中使用前，请检查命令和资源范围。

CLI 支持 `--language en` 和 `--language zh`。需要稳定脚本时，应优先解析命令的 JSON 输出，避免解析本地化的人类可读文本。

## 面向 Agent 的优化与 AI mode

AI mode 可以全局管理，也可以针对单次命令控制：

```sh
aliyun configure ai-mode --help
aliyun ecs describe-instances --cli-ai-mode
```

检测到受支持的 Agent 环境时，进程内 OpenAPI 命令会自动启用面向 Agent 的交互和执行优化。目前包括更严格的 metadata 参数校验和更结构化的错误输出，但具体优化行为可能随版本调整，不属于稳定兼容协议。

通过自动探测启用后，请求会增加以下通用 UA 标记：

```text
AlibabaCloud-AIMode/enabled
```

该标记不包含检测到的 Agent 名称。需要对单次命令关闭自动 Agent 模式时，使用 `--no-cli-ai-mode`：

```sh
aliyun ecs describe-instances --no-cli-ai-mode
```

## 参数边界情况

参数值以 `-` 开头时，使用 `--name=value` 形式，避免被解析成另一个 Flag：

```sh
aliyun ecs SomeOperation --PortRange=-1/-1
```

下一步：[管理产品插件](./plugins.md)。
