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
- 短横线命令使用 metadata 生成的命令名，参数名通常也改为短横线形式；具体名称以命令帮助为准。例如 `describe-instances` 使用 `--biz-region-id`，`describe-instance-attribute` 使用 `--instance-id`。

子命令的大小写决定由哪个引擎处理调用：全小写的子命令进入插件/runtime 引擎，包含大写字母的子命令走内置 OpenAPI 链路。不要在同一条命令里混用两种写法——`aliyun ecs DescribeInstances --biz-region-id cn-hangzhou` 会报未知参数错误；`--RegionId` 与 `DescribeInstances` 配对，`--biz-region-id` 与 `describe-instances` 配对。

产品级帮助同样遵循该规则。未安装产品插件时，`aliyun ecs --help` 默认展示大驼峰命令列表；仅存在于内置 metadata 中的产品则展示短横线命令列表。可以通过一次性环境变量切换产品帮助；下面的写法只影响当前这次命令，不会修改 Shell 的持久配置：

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

## Endpoint 与 metadata 未收录的 API

CLI 通常从 metadata 解析 API 版本和 Endpoint。调用内置 metadata 未收录的 API 时，需要同时指定版本、Endpoint 和 `--force`：

```sh
aliyun newproduct SomeAction \
  --version 2025-01-01 \
  --endpoint newproduct.aliyuncs.com \
  --SomeParameter value \
  --force
```

如果请求里的 API 或参数未包含在本地 metadata 中，CLI 会返回 `unknown api` 或 `unknown parameter`。`--force` 用于跳过这类本地检查，继续发起调用。

`--force` 只跳过本地 API 和参数 metadata 校验，不会绕过服务端鉴权或校验。

使用 `--force` 时，通常需要同时明确以下参数：

- `--version`：指定 API 版本号。版本号可从对应产品 API 文档获取，例如 ECS 常见版本为 `2014-05-26`。
- `--endpoint`：指定产品接入地址。请以对应产品 API 文档中的 Endpoint 为准。

可以重复使用 `--header X-foo=bar` 添加自定义 HTTP Header。`--secure` 强制使用 HTTPS。`--skip-secure-verify` 会关闭证书校验，不建议使用。

## 调用前验证

对写操作可以先使用 `--cli-dry-run`：

```sh
aliyun ecs run-instances ... --cli-dry-run
```

具体支持情况取决于 API metadata。CLI dry-run 验证不能替代权限和资源影响审查。

## 预估 API 调用费用

使用 `--estimate-cost` 可以在不调用目标 API 的情况下请求费用预估：

```sh
aliyun ecs RunInstances --version 2014-05-26 --RegionId cn-hangzhou ... --estimate-cost
```

使用以下工具命令可查询支持费用预估的 API。产品代码过滤不区分大小写，API 版本过滤需精确匹配：

```sh
aliyun utils list-supported-pricing-apis
aliyun utils list-supported-pricing-apis --product Ecs --api-version 2014-05-26
```

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

`rows` 表达式必须得到数组。缺少 `cols`、JMESPath 表达式无效或行结果不是数组时，命令会返回错误，不会静默输出不完整的表格。

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

- `expr`：必填。每次响应上求值的 JMESPath 表达式。
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

API 调用或 JMESPath 求值失败时，Waiter 会立即停止。到达超时时间时，错误信息会包含表达式、期望值和最后一次观测值。`--waiter` 不能与 `--pager` 或 `--cli-dry-run` 同时使用。

## 安全策略与非交互使用

有潜在破坏性的操作可能受到 CLI 安全策略和人工确认控制。管理安全策略：

```sh
aliyun configure safety-policy --help
```

`--yes` 可以在非交互场景跳过确认提示，但不能绕过 deny 策略。在自动化中使用前，请检查命令和资源范围。

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

需要对单次命令关闭自动 Agent 模式时，使用 `--no-cli-ai-mode`：

```sh
aliyun ecs describe-instances --no-cli-ai-mode
```

JSON Help 协议、Agent 错误 envelope、退出状态、Trace Context 传播和 MCP 代理安全行为见 [MCP 代理、OpenTelemetry 与机器可读接口](./integrations.md)。

## 大驼峰命令的参数边界情况

这条规则仅适用于传统大驼峰命名（PascalCase）命令。参数值以 `-` 开头时，使用 `--name=value` 形式，避免该值被解析成另一个 Flag：

```sh
aliyun ecs SomeOperation --PortRange=-1/-1
```

下一步：[管理产品插件](./plugins.md)。
