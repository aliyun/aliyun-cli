# Help Search, Listing Limits, and Result Guarantees

## 1. 职责

本分册提供统一、确定性的本地 Help 搜索器，以及 AI Mode 产品/API 列表的 20 条上限。它不解析命令行，不决定 Help Provider，不生成错误 JSON。

## 2. 搜索范围

| Help 目标 | 搜索内容 |
| --- | --- |
| Root | Canonical 产品 Code、中英文名称 |
| Product | 当前选中版本的 API Name、`cmd_name` 和中英文描述 |
| API Request | 当前命令风格的参数名、Raw Name、Option、中英文 Help，以及全局 CLI 参数 |
| API Response | 响应字段名、完整字段路径、中英文标题和描述 |

Root 内置命令不纳入产品搜索；未知内置子命令 Recovery 使用父级 Help。

Root 使用 `--cli-search` 时只返回匹配产品，不重复输出与查询无关的内置命令。API Request 搜索只返回当前 `activeParameterSet` 的命中参数；全局参数仍作为同一结果集合参与搜索。

## 3. 匹配与排序

名称匹配执行统一规范化：

- 忽略大小写；
- CamelCase、kebab-case、snake_case 和空格按词边界等价；
- 比较时同时保留完整规范化字符串和词 Token；
- 中文使用原字符串包含匹配。

结果按以下顺序稳定排序：

1. 规范化名称精确匹配；
2. 名称 Token 前缀匹配；
3. 名称包含匹配；
4. 标题或描述包含匹配；
5. 原有稳定顺序：产品 Code、API `cmd_name/name`、参数声明顺序、Response Schema 顺序。

Search 不做编辑距离模糊猜测。`did_you_mean` 由错误模块基于候选列表单独计算。

## 4. 结果数量

- 只要使用 `--cli-search`，返回全部命中结果，不限制 20 条。
- 非 AI Mode 未搜索的列表保持完整。
- AI Mode 未搜索时：Root 内置命令始终完整展示，产品列表最多 20 条；Product 下的 API 列表最多 20 条。
- API Request/Response 无论模式都完整展示，不应用 20 条上限。
- `--cli-all` 只解除 AI Mode 产品/API 列表上限。

文本发生截断时输出：

```text
Showing 20 of 186 APIs.
Use --cli-search <keyword> to narrow the list, or --cli-all to show everything.
```

JSON 发生截断时增加：

```json
{
  "listing": {
    "shown": 20,
    "total": 186,
    "hint": "Use --cli-search <keyword> to narrow the list, or --cli-all to show everything."
  }
}
```

未截断时不输出 `listing`。Root 的 `listing.shown/total` 只统计产品，不把内置命令计入。

## 5. Response 搜索投影

Response 搜索返回全部命中路径，并为每个命中保留从 Schema Root 到字段的最小父级结构。例如搜索 `instance-id` 可命中：

```text
Instances.Instance.InstanceId
```

JSON 仍通过 `outputSchema.schema` 输出裁剪后的合法 Schema 树。多个命中共享父节点时合并父节点，不重复。过滤完成后重新计算可达 Components。

搜索命中数组节点时保留其 `items`；命中数组元素内字段时保留数组和 `items` 父结构。

命中字段位于 Component 内时，Root Schema 保留原 `$ref`，对应 Component 也裁剪为命中字段所需的最小父级结构；仅保留裁剪后节点继续引用的其他 Components。不得因为一个 Ref 命中而返回整个未过滤 Component。

## 6. Recovery 命令预验证

本模块公开与真实 Help 完全相同的搜索入口，供错误模块在构造 Recovery 前调用：

```text
ValidateSearch(target, section, style, version, keyword) -> matched bool
```

- Recovery 搜索命令只有在 `matched=true` 时才能输出。
- 验证必须使用相同 Provider、版本、风格、字段和规范化规则。
- 不能验证或没有结果时，错误模块退回上一层普通 Help。
- 不允许仅用字符串启发式声称某条 Search 命令会有结果。

## 7. 单元测试

- `InstanceId`、`instance-id`、`instance_id` 互相命中；
- 产品/API/参数中英文描述搜索；
- 全局 `header`、`body-file`、`pager` 可在 API Request 中搜索；
- Search 超过 20 条仍全部输出；
- AI 默认列表恰好 20、21 和大量结果；
- `--cli-all` 解除上限；
- Response 多命中最小树、数组父结构和 Ref Components；
- Recovery 验证与真实搜索结果一致。
