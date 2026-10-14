# 命令、输出与自动化

[文档索引](../README.md) | [English](../en/usage.md)

## 命令形式

RPC API 可以使用传统大驼峰命名（PascalCase）的操作名，也可以使用短横线命名（kebab-case）的命令名：

```sh
aliyun <product> <Operation> [--Parameter value ...]
aliyun <product> <operation-name> [--parameter value ...]

aliyun ecs DescribeInstances --RegionId cn-hangzhou
aliyun ecs describe-instances --biz-region-id cn-hangzhou
```

两个命令形式调用的是同一个 API，只是操作名和参数名的书写方式不同。例如：

```sh
# 查询地域
aliyun ecs DescribeRegions
aliyun ecs describe-regions

# 查询实例列表
aliyun ecs DescribeInstances --RegionId cn-hangzhou --PageSize 10
aliyun ecs describe-instances --biz-region-id cn-hangzhou --page-size 10

# 查询单个实例详情
aliyun ecs DescribeInstanceAttribute --InstanceId i-1234567890abcdef
aliyun ecs describe-instance-attribute --instance-id i-1234567890abcdef
```

一般来说：

- 大驼峰命令使用 API 原始操作名，例如 `DescribeInstances`，参数名通常也是 API 文档里的原始字段名，例如 `--RegionId`、`--InstanceId`。
- 短横线命令使用元数据生成的命令名，参数名通常也改为短横线形式；具体名称以命令帮助为准。例如 `describe-instances` 使用 `--biz-region-id`，`describe-instance-attribute` 使用 `--instance-id`。

命令名称与参数名称必须采用对应的格式。例如，`aliyun ecs DescribeInstances --biz-region-id cn-hangzhou` 会报未知参数错误；`--RegionId` 与 `DescribeInstances` 配对，`--biz-region-id` 与 `describe-instances` 配对。

产品级帮助同样遵循该规则。未安装产品插件时，`aliyun ecs --help` 默认展示大驼峰命令列表；仅存在于内置元数据中的产品则展示短横线命令列表。可以通过一次性环境变量切换产品帮助；下面的写法只影响当前这次命令，不会修改 Shell 的持久配置：

```sh
# 从传统大驼峰帮助切换到短横线帮助
ALIBABA_CLOUD_BASELINE_PRODUCT_HELP=true aliyun ecs --help

# 返回传统大驼峰帮助
ALIBABA_CLOUD_BASELINE_PRODUCT_HELP=false aliyun ecs --help

# 从已安装插件提供的帮助切换到传统大驼峰帮助
ALIBABA_CLOUD_ORIGINAL_PRODUCT_HELP=true aliyun ecs --help

# 返回已安装插件提供的帮助
ALIBABA_CLOUD_ORIGINAL_PRODUCT_HELP=false aliyun ecs --help
```

帮助风格切换提示只出现在面向用户的产品级帮助中。AI 模式使用结构化帮助描述当前命令风格，API 级帮助只展示当前命令对应的参数，不重复提示另一种风格。

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

## Endpoint 与元数据未收录的 API

产品 JSON Help 通过 `product.endpoints` 返回 endpoint 列表，字段包含 `endpoint` 和地域端点适用的 `regionId`。可本地执行 `aliyun ecs --help --cli-output json --cli-query product.endpoints` 查询，添加 `--endpoint-type vpc` 可选择 VPC endpoint。

CLI 通常从元数据解析 API 版本和 Endpoint。调用内置元数据未收录的 API 时，需要同时指定版本、Endpoint 和 `--force`：

```sh
aliyun newproduct SomeAction \
  --version 2025-01-01 \
  --endpoint newproduct.aliyuncs.com \
  --SomeParameter value \
  --force
```

如果请求里的 API 或参数未包含在本地元数据中，CLI 会返回 `unknown api` 或 `unknown parameter`。`--force` 用于跳过这类本地检查，继续发起调用。

`--force` 只跳过本地 API 和参数元数据校验，不会绕过服务端鉴权或校验。

使用 `--force` 时，通常需要同时明确以下参数：

- `--version`：指定 API 版本号。版本号可从对应产品 API 文档获取，例如 ECS 常见版本为 `2014-05-26`。
- `--endpoint`：指定产品接入地址。请以对应产品 API 文档中的 Endpoint 为准。

可以重复使用 `--header X-foo=bar` 添加自定义 HTTP Header。`--secure` 强制使用 HTTPS。`--skip-secure-verify` 会关闭证书校验，不建议使用。

## 调用前验证

对写操作可以先使用 `--cli-dry-run`：

```sh
aliyun ecs run-instances ... --cli-dry-run
```

具体支持情况取决于 API 元数据。CLI dry-run 验证不能替代权限和资源影响审查。

## 筛选和格式化输出

### 表格输出

OpenAPI 默认返回 JSON。例如，`DescribeInstances` 的响应中可能包含以下集合：

```json
{
  "PageNumber": 1,
  "TotalCount": 2,
  "PageSize": 10,
  "RequestId": "2B76ECBD-A296-407E-BE17-7E668A609DDA",
  "Instances": {
    "Instance": [
      {
        "InstanceId": "i-12345678912345678123",
        "Status": "Stopped"
      },
      {
        "InstanceId": "i-abcdefghijklmnopqrst",
        "Status": "Running"
      }
    ]
  }
}
```

使用 `--output` 可以选择集合，并把指定字段渲染为表格：

```sh
aliyun ecs DescribeInstances \
  --output cols=InstanceId,Status rows=Instances.Instance[]
```

输出结果如下：

```text
InstanceId                    | Status
----------                    | ------
i-12345678912345678123        | Stopped
i-abcdefghijklmnopqrst        | Running
```

`--output` 支持以下字段：

- `cols`：必填。以逗号分隔，每个字段都相对于当前行求值，字段名同时作为表头。
- `rows`：可选。使用 JMESPath 表达式选择作为表格行的数据数组。对于嵌套的 API 响应，通常需要指定。
- `num=true`：可选。增加从零开始的 `Num` 序号列。

例如：

```sh
aliyun ecs DescribeInstances \
  --output cols=InstanceId,Status rows=Instances.Instance[] num=true
```

`rows` 表达式必须得到数组。缺少 `cols`、JMESPath 表达式无效或行结果不是数组时，命令会返回错误，不会输出不完整的表格。

### JMESPath 查询

使用 `--cli-query` 筛选或重组 JSON 输出：

```sh
aliyun ecs describe-instances \
  --cli-query 'Instances.Instance[].{Id:InstanceId,Status:Status}'
```

支持时可以使用 `--quiet` 隐藏普通输出。

## 分页

对支持分页的 API，`--pager` 会重复调用 API，并合并每次响应中的结果集合：

```sh
aliyun ecs describe-instances --pager
```

没有附加字段时，CLI 使用常见的 `PageNumber`、`PageSize`、`TotalCount` 和 `NextToken` 响应字段，并尝试自动识别嵌套的结果数组。无法识别集合时，需要显式指定对应的 JMESPath：

```sh
aliyun ecs DescribeInstances \
  --pager path='Instances.Instance[]'
```

API 使用非标准分页字段时，可以覆盖字段映射：

```sh
aliyun <product> <operation> \
  --pager \
  path='Data.Items[]' \
  PageNumber='PageInfo.PageNumber' \
  PageSize='PageInfo.PageSize' \
  TotalCount='PageInfo.TotalCount'
```

基于 Token 分页的 API 也可以指定 Token 字段：

```sh
aliyun <product> <operation> \
  --pager path='Data.Items[]' NextToken='Data.NextToken'
```

分页字段含义如下：

| 字段 | 用途 |
| --- | --- |
| `path` | 选择并合并结果数组的 JMESPath 表达式 |
| `PageNumber` | 页码请求字段和响应表达式 |
| `PageSize` | 每页数量请求字段和响应表达式 |
| `TotalCount` | 总记录数响应表达式 |
| `NextToken` | 下一页 Token 请求字段和响应表达式 |

最终 JSON 会在 `path` 对应的位置包含合并后的集合；每一页的 RequestId、页码等元数据不会保留。分页可能产生大量请求，并在内存中保存全部合并结果，处理超大结果集时应谨慎使用。`--pager` 不能与 `--waiter` 或 `--cli-dry-run` 同时使用。

## 等待目标状态

`--waiter` 会重复调用同一个 API，直到响应中的 JMESPath 表达式等于目标值。典型场景是在创建 ECS 实例后等待实例进入 `Running` 状态：

```sh
aliyun ecs DescribeInstances \
  --InstanceIds '["i-12345678912345678123"]' \
  --waiter expr='Instances.Instance[0].Status' to=Running
```

表达式匹配后，CLI 停止轮询并输出最后一次 API 响应。支持以下字段：

- `expr`：必填。用于查询每次响应的 JMESPath 表达式。
- `to`：必填。期望值，与表达式结果按文本比较。
- `timeout`：可选。默认 180 秒；为了兼容不同命令形式，应使用 1 到 600 秒。
- `interval`：可选。默认 5 秒；为了兼容不同命令形式，应使用 2 到 10 秒。

覆盖轮询时间的完整示例：

```sh
aliyun ecs DescribeInstances \
  --InstanceIds '["i-12345678912345678123"]' \
  --waiter \
  expr='Instances.Instance[0].Status' \
  to=Running \
  interval=5 \
  timeout=300
```

API 调用或 JMESPath 求值失败时，Waiter 会立即停止。到达超时时间时，错误信息会包含表达式、期望值和最后一次查询结果。`--waiter` 不能与 `--pager` 或 `--cli-dry-run` 同时使用。

## 安全策略与非交互使用

有潜在破坏性的操作可能受到 CLI 安全策略和人工确认控制。管理安全策略：

```sh
aliyun configure safety-policy --help
```

`--yes` 可以在非交互场景跳过确认提示，但不能绕过 deny 策略。在自动化中使用前，请检查命令和资源范围。

安全策略默认关闭，`confirm` 只是 CLI 本地执行前确认，不代表云端授权或审批。使用 `show`/`list` 查看当前生效策略；如果设置了 `ALIBABA_CLOUD_SAFETY_POLICY_ENABLED` 或 `ALIBABA_CLOUD_SAFETY_POLICY_RULES`，展示结果包含环境变量覆盖后的配置。策略文件或环境规则无法完整解析时，CLI 会拒绝继续执行。

### 命令匹配规则

对于内置 API，一条不含通配符的规则可以同时匹配大驼峰（PascalCase）和短横线（kebab-case）命令名称。例如，以下规则可同时禁止执行 `ecs DeleteInstance` 和 `ecs delete-instance`。两种名称的对应关系须包含在所选版本的本地 API 元数据中：

```sh
aliyun configure safety-policy add --pattern 'ecs:DeleteInstance' --action deny
aliyun configure safety-policy enable
```

当所选 API 版本的元数据明确将 `DeleteInstance` 映射到 `delete-instance` 时，上述规则会拒绝两种写法。将规则写为 `ecs:delete-instance`，效果相同。不含通配符的 `confirm` 和 `allow` 规则也采用相同的匹配方式。

匹配方式按规则类型区分：

- **不含通配符的内置 API 规则**：对于所选版本中可识别的 API，同时匹配其 PascalCase 和 kebab-case 命令名称。两种名称的对应关系以本地 API 元数据为准。
- **包含 `*` 的规则**：仅匹配用户输入的原始命令，不区分大小写。例如，`ecs:Delete*` 通过共同前缀匹配两种写法，但 `ecs:DeleteI*` 不匹配 `delete-instance`。产品名含通配符时同样不会自动匹配另一种名称。例如，`*:DeleteInstance` 不匹配 `ecs:delete-instance`。
- **插件命令（包括通过元数据插件提供的命令）和 REST 请求**：按输入的命令名称或路径匹配，不区分大小写。API 未知，或本地元数据缺失、无效、无法唯一确定 API 时，也采用此方式；如需限制多种命令名称，请分别配置规则。

两种命令名称的对应关系仅在本次调用选定的 API 版本内有效，不使用其他版本的对应关系。大驼峰命令通过 `--version` 选择版本；短横线命令通过 `--api-version` 或对应产品的 API 版本环境变量选择版本。匹配范围以该版本的本地 API 元数据为准。AI 模式同样适用。

多条规则匹配时，按拒绝（`deny`）、确认（`confirm`）、允许（`allow`）的优先级选择规则。`forbid` 是 `confirm` 的别名。规则操作相同时，优先选择精确匹配已识别 API 的规则；匹配优先级也相同时，选择配置顺序靠前的规则。未匹配任何规则的命令允许执行。

**兼容性变化**：旧版本仅执行首条匹配规则。当前版本中，即使 `allow` 配置在前或匹配范围更小，也不能覆盖匹配的 `deny` 或 `confirm`，插件和 REST 规则同样适用。升级前请检查重叠规则；如需允许特定命令执行，应调整限制规则的匹配范围或移除冲突规则，仅调整配置顺序不会改变结果。例如，先配置 `ecs:*=allow`，再配置 `ecs:DeleteInstance=deny`，当所选版本的本地元数据可识别该 API 的两种名称时，两种命令均会被拒绝。

REST 方法与路径采用 `product:METHOD/path`（如 `cs:DELETE/clusters`），`cs:DELETE/*` 匹配路径以 `/` 开头的 DELETE 命令；插件子命令层级以 `:` 分隔（如 `fc:function:create`）。这两类命令不会自动匹配 API 的其他名称。云端授权仍由服务端与 RAM 权限控制。

内置扩展（如 `saectl`、`ossutil`、`acrutil`、`spark-submit`）也在调用扩展前检查策略，早于扩展的安装、凭据解析和执行。命令标识以 `:` 连接位置参数，例如 `saectl:restore`、`acrutil:diagnosis:restore`；`*restore*` 可匹配这些写法。CLI 全局配置参数不参与匹配；扩展未知选项后的非选项参数会保留，因此需要覆盖额外参数时应使用通配符。不带扩展参数、单独的 `help`/`--help`/`-h` 或 `version`/`--version` 不触发检查；其他调用即使包含 `--help` 也会检查，避免把选项值误当成帮助请求而绕过策略。

## Agent 集成与 AI 模式

AI 模式可以全局管理，也可以针对单次命令控制：

```sh
aliyun configure ai-mode --help
aliyun ecs describe-instances --cli-ai-mode
```

检测到受支持的 Agent 环境时，由 CLI 直接执行的 OpenAPI 命令会自动启用面向 Agent 的交互和执行优化。目前包括更严格的元数据参数校验和更结构化的错误输出，但具体优化行为可能随版本调整，不属于稳定兼容协议。

通过自动探测启用后，请求会增加以下 User-Agent 标记：

```text
AlibabaCloud-AIMode/enabled
```

需要对单次命令关闭自动 Agent 模式时，使用 `--no-cli-ai-mode`：

```sh
aliyun ecs describe-instances --no-cli-ai-mode
```

需要在当前环境中持续关闭 Agent 自动探测带来的 AI 模式（包括 AI Help 文本）时，显式设置：

```sh
export ALIBABA_CLOUD_CLI_AGENT_INTEGRATION=disabled
```

`ALIBABA_CLOUD_CLI_AGENT_INTEGRATION` 控制自动 Agent 集成：未设置时启用默认集成，`disabled` 全部关闭，`all` 全部开启，逗号分隔的列表（如 `ai-mode`）仅开启指定集成。此开关不会启用安全策略。检测到 Agent 时还会添加固定的 `Agent/<name>` User-Agent 标识，该标识独立于 AI 模式集成开关。设置为 `disabled` 不会覆盖 `configure ai-mode enable`；显式 `--cli-ai-mode` 仍可开启单次命令的 AI 模式。

JSON 帮助格式、Agent 错误格式、退出状态、Trace Context 传播和 MCP 代理安全行为见 [MCP 代理、OpenTelemetry 与机器可读接口](./integrations.md)。

## 大驼峰命令的参数边界情况

这条规则仅适用于传统大驼峰命名（PascalCase）命令。参数值以 `-` 开头时，使用 `--name=value` 形式，避免该值被解析成另一个参数：

```sh
aliyun ecs SomeOperation --PortRange=-1/-1
```

## 内置 OSS

内置 OSS 的 JSON/JSONL 列表、只读计划、确认和失败报告见 [OSS 自动化指南](./oss.md)。

下一步：[管理产品插件](./plugins.md)。
