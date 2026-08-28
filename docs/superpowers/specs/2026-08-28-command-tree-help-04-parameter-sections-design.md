# L3 参数 Help 与 Request/Response Section

## 1. L3 参数 Help

标准入口：

```shell
aliyun ecs DescribeInstances --InstanceIds --help
aliyun ecs describe-instances --instance-ids --help
```

当一个真实、未赋值的顶层参数 Flag/alias 紧邻 Help 操作时进入 L3，不执行 API、不校验其他必填参数，也不要求给目标参数赋值。若参数已有值，Help 仍以 Action 为目标；多个未赋值参数无法唯一定位时返回本地歧义错误。

参数 Target 支持业务参数和公开全局参数；Hidden/internal 不可见。只能使用当前风格真实 Flag/alias，不能把嵌套字段路径伪造成可执行 Flag。

## 2. 参数完整文档

Default、All、JSON 均返回完整单参数对象，不应用 100 行限制。内容包括：

- 当前风格 Flag/alias 与 OpenAPI `raw_name`；
- `type`、`location`、element/value type；
- `required`、`param_style`/serialization；
- 当前语言 `help_*`；
- example、enum、pattern、minimum、maximum 等存在的约束；
- `fields`、`element`、`value` 及其全部有限子层级。

Canonical Request 已展开，不读取 Components、不处理 Request `$ref`。Text 使用插件交集一致的单参数渲染；JSON 返回结构化树。

`--help-all` 在 L3 合法但与 `--help` 内容相同。`--help-search` 只搜索当前参数及嵌套字段，零命中 exit 0，最多 20 条。

## 3. L2 Action/Section 边界

| 命令 | 语义 | 截断 |
| --- | --- | --- |
| `<action> --help` | 人类可读 Action 摘要 | 默认策略 |
| `<action> --help-all` | 完整人类可读 Action Help | 否 |
| `aliyun help <product> <action>` | 兼容入口，等价完整 Request Section | 否 |
| `... --cli-section request` | 完整 Request 元数据 | 否 |
| `... --cli-section response` | 完整 Response 元数据 | 否 |

All 与 Section 不是别名。Section 与 Search/`--cli-output json` 可组合；与 All 组合返回本地选项冲突。省略 Section 的前置 Action Help 默认 request。

## 4. Response 数据

Response Section 展示当前 Action 完整 `responses` 以及其 `$ref` 可达的 `components/schemas`：

- 保留原始 `$ref`，不递归内联；
- 仅附带可达 Components，循环引用通过 visited set 终止；
- `title`/`description` 使用 Canonical 规范字段 `title_en/title_zh/description_en/description_zh`；
- 缺少 Schema 时返回明确的结构化空结果，不伪造字段；
- Search 只返回命中结构和必要的可达 Components，遵循统一排名和 20 条上限。

Request/Response Text 和 JSON 均完整。JSON 不是文本包装。

## 5. Parser 与执行隔离

Help 参数扫描在业务参数校验与 API dispatch 前完成，但在产品插件检查后才由宿主消费。解析器应返回明确的 HelpTarget/Option，而不是通过删除 argv 让后续代码猜目标。

参数 Help 不读取参数值，也不得把用户值、Body/Header、本地路径写入 Help/Recovery。

## 6. 主要验收

- 驼峰/烤串顶层参数均能独立 Help；赋值参数不会误判。
- 参数 Help 跳过 Action 其他 required 校验并完整展示嵌套树。
- 多个未赋值参数报歧义，未知参数给当前风格 did-you-mean。
- Request/Response Section 不截断，Response 循环 Ref 不递归爆栈。
- 已安装插件时相同 argv 完全由插件处理。
