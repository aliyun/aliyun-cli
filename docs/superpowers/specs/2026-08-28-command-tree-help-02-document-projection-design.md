# Help Document、投影与渲染

## 1. 共享 Document

Text 与 JSON 不得分别收集对象。Provider 先构建稳定的 Root/Product/Action/Parameter/Section Document；Operation、模式和输出格式只负责投影与渲染。JSON 是字段化对象，不能包裹预渲染文本。

## 2. 默认截断策略

内部策略开关（均不是用户 Flag/环境变量）：

```text
truncateNonAIModeTextHelp = true
truncateNonAIModeJSONHelp = true
showProductActionDescriptionsInDefaultHelp = false
```

AI Mode 默认 JSON 始终截断；非 AI Mode Text/JSON 分别读取上述开关。Default 的 Root/Product/Action 文本预算为 100 个逻辑行，按完整对象选择，不能截断对象内部。参数 Help 和显式 Request/Response Section 永远完整。

`--help-all` 永远完整；Search 不使用默认截断，独立按最多 20 条命中。若 Action 全部必填参数自身超过 100 行，完整保留所有必填参数并停止加入可选参数；Response Query Example 仍保留。

JSON 默认选择与文本摘要相同范围的对象，不按格式化行数二次裁剪。发生截断时使用：

```json
{
  "result": {"shown": 20, "total": 371, "truncated": true},
  "next": {
    "showAll": "aliyun ecs --help-all",
    "search": "aliyun ecs --help-search <keyword>"
  }
}
```

未截断时通常保留 `result.truncated=false`、正确 shown/total，并省略无意义的 `next`。L3 默认/All Parameter Help 是单对象视图，不输出固定的 `1/1/false`；只有 Parameter Search 才输出 `result`。非 AI Mode 显式 JSON 的后续命令继续带 `--cli-output json`；AI Mode 不带冗余格式参数。

## 3. Product Help

- 只读选定版本 `version.json`，不得逐个加载 Action JSON。
- Default：Action 名称与必要状态，不显示描述，按默认策略截断。
- All：全部 Action 和完整描述，不递归拼每个 Action 参数。
- Search：最多 20 条，显示描述解释命中原因。
- JSON 保留 `legacyDefaultVersion`、`pluginDefaultVersion`、`supportedVersions`、`selectedVersion`。
- Action 名称严格跟随已选命令风格。

## 4. Action Help

Default 顺序：用途/Usage → 全部必填业务参数 → 预算内“部分可选参数” → 固定导航。默认不重复全局参数。All 展示全部顶层业务参数、全部公开全局参数、完整说明和示例；Hidden/internal 始终隐藏。

默认用途优先当前语言 `title_*`，缺失时回退 `description_*`，不展开长文案。All 同时展示 Title 和完整 Description；文本中相同内容不重复。Section JSON 忠实保留结构字段。

ROA/RESTful 默认展示 Method 与 PathPattern；RPC 默认不展示重复 HTTP 细节，完整 Operation 留在 All/Section JSON。Deprecated 在文本顶部和 JSON 状态中明确表达。

## 5. Example

- 驼峰只使用 `camel_example`，烤串只使用 `kebab_example`。
- 对应字段缺失则省略，不跨风格回退、不临时合成。
- Default/All/Request Text 只渲染当前风格；Request Section JSON 可保留两个具名字段。

Response Schema 存在 Array 时，Default/All 尾部增加一段导航，只选一个代表性 Array Path：优先分页/List 主数组，否则按声明顺序第一个数组。

```text
响应包含复杂数组结构。可以先查看响应结构，再只提取目标数组：
  aliyun help ecs DescribeInstances --cli-section response
  aliyun ecs DescribeInstances --cli-query 'Instances.Instance'
```

两条命令必须跟随当前 Action 风格和显式版本；不存在 Array 时不输出。

## 6. 文本尾部

只有实际省略对象时输出 `...`、shown/total 和下一步。非 AI 文本 Help 还保留 Show all、Find/Search、One param、JSON 与进入 AI Mode 的 Hint。AI JSON 使用结构化 `next`，不输出 `JSON:` 提示或任何外围文本。

## 7. 主要验收

- Root/Product/Action 的 Text/JSON 使用相同对象顺序和截断结果。
- Default/All/Search 的描述、全局参数和上限严格区分。
- 必填参数安全例外不丢参数。
- Title/Description 去重只影响文本，不破坏 Section JSON。
- camel/kebab 普通 Example 与 Response Query Example 不串风格。
