# Response Array Query Guidance

## 1. 职责

本分册只负责从选中的 Response Schema 选出一个代表性 Array Path，并生成与当前命令风格一致的 Schema/Query 示例。它不输出完整 Response Schema，不实现 JMESPath 执行。

现有 `--cli-query` 使用 JMESPath，本设计直接生成兼容路径，例如 `Instances.Instance`。

## 2. Array Path 收集

- 从成功响应 Body Schema 开始深度优先遍历。
- 遍历 Inline Object、Array `items` 和本地 Component Ref。
- 输出字段路径使用响应 JSON 的真实字段名，以 `.` 连接。
- Ref 仅用于遍历，不在 Schema 输出中被内联。
- 循环 Ref 使用 `visited` 防护。
- `additionalProperties` Map 不臆造运行时 Key，不生成虚假固定路径。

## 3. 主 Array 选择

只选择一个示例，排序规则固定：

1. 已有分页配置明确指向且能解析为 Array 的 Collection Path；
2. 与 `NextToken`、`TotalCount`、`PageNumber`、`PageSize` 等分页字段同层的结果 Array；
3. 去掉 API 名开头的常见动作 Token（如 `Get`、`List`、`Describe`、`Query`、`Search`）后，将剩余资源 Token 与 Array Path Token 做忽略大小写和单复数的完整匹配；例如 `DescribeInstances` 优先 `Instances.Instance`；
4. 常见主结果名称，如 `Items`、`Records`、`Results`、`List`；
5. Schema 声明顺序中的第一个 Array Path。

不得为了找“更好”的路径加载其他 API 或调用网络。

Response 使用 `--cli-search` 时：

- 如果过滤结果中包含 Array，按上述规则从过滤结果选择；
- 如果过滤结果只包含标量字段，不输出 Query Example。

## 4. 文本 Help

默认 Request Help 末尾追加：

```text
Response query example (Instances.Instance):
This response contains a complex array. Inspect its structure with the response section, then use --cli-query to return only that array:
  aliyun help ecs DescribeInstances --cli-section response
  aliyun ecs DescribeInstances --cli-query 'Instances.Instance'
```

Response Help 已经展示 Schema，因此末尾只显示查询动作：

```text
Query this array directly:
  aliyun ecs DescribeInstances --cli-query 'Instances.Instance'
```

kebab Help 必须生成：

```text
aliyun help ecs describe-instances --cli-section response
aliyun ecs describe-instances --cli-query 'Instances.Instance'
```

显式 `--api-version` 在 Schema Help 命令和 API Query 命令中都要保留。

## 5. JSON Help

有 Array 时增加：

```json
{
  "responseQueryExample": {
    "path": "Instances.Instance",
    "schemaCommand": "aliyun help ecs DescribeInstances --cli-section response",
    "queryCommand": "aliyun ecs DescribeInstances --cli-query 'Instances.Instance'"
  }
}
```

没有 Array、无法生成有效 JMESPath 或过滤结果只有标量时，整个字段省略。

## 6. 测试

- 单数组、多数组和无数组；
- 分页字段同层选择；
- API 资源名选择；
- Inline、Ref、自引用和缺失 Ref；
- PascalCase/kebab 与显式版本；
- Request 显示两条命令，Response 只显示 Query；
- Response Search 只基于过滤结果选 Array；
- Map 不生成虚假 Key Path。
