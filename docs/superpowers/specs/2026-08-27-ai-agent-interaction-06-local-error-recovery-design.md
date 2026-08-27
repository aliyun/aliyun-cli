# CLI Local Error JSON and Recovery

## 1. 职责

本分册独占 CLI 本地错误类型、AI Mode JSON Envelope、Recovery 生成，以及非 AI Mode 本地文本错误的 AI Mode Hint。不得根据普通错误文本猜测类型。

## 2. AI Mode Envelope

只有下文明确列出的 CLI 本地错误进入新 Envelope：

```json
{
  "message": "<error-message>",
  "did_you_mean": ["<candidate>"],
  "recovery": {
    "action": "<action>",
    "command": "<command>",
    "hint": "<hint>"
  }
}
```

字段规则：

- `message` 必需，只描述错误；
- `did_you_mean` 可选，只含同层级真实候选；
- `recovery` 必需且只有一个；
- `recovery.action`、`recovery.hint` 必需；
- `recovery.command` 只有安全且经过验证时才输出；
- 空字符串、`null`、空数组和空对象全部省略；
- 删除旧字段：`ok`、`category`、`code`、`details`、`requestId`、`retryable`；
- 内部错误 Code 仍可用于类型分派，但不得输出；
- stdout 为空，stderr 输出一个 JSON 文档；AI Mode 不输出颜色。

用法错误退出码保持 2。本期不设计新的凭证错误 Envelope。

## 3. 类型化错误

已有类型继续使用：未知产品/API/命令/Flag、缺少必填参数等。当前只包含 `Code + Err` 文本的场景必须增加类型化 Cause：

- `InvalidArgumentError`：可选参数、Flag、字段路径和期望类型；
- `InvalidOptionCombinationError`：冲突选项；
- `InvalidHeaderError`：Header 输入和期望格式；
- `InvalidBodyFileError`：文件路径和底层文件错误。

`Error()` 保持现有非 AI 文案；AI Mode 使用 `errors.As` 读取字段。PascalCase 旧执行路径和 kebab Runtime 路径必须适配成相同语义，不能只修一条路径。

## 4. 支持的错误与 Recovery

| 本地错误 | Action | Recovery Command |
| --- | --- | --- |
| 未知产品 | `search_product` | 验证成功后 `aliyun help --cli-search <candidate>`；否则 `aliyun help` |
| 未知 API | `search_api` | 验证成功后 `aliyun help <product> --cli-search <resource-keyword>`；否则 Product Help |
| 未知 CLI 子命令 | `search_command` | 当前父级 `aliyun help ...` |
| 未知参数或 Flag | `search_parameter` | 验证成功后 API Help `--cli-search <parameter>`；否则完整 Request Help |
| 缺少必填参数 | `inspect_request_help` | 完整 `--cli-section request` |
| 参数语法、类型或 JSON 不合法 | `inspect_request_help` | 参数已知时搜索参数，否则完整 Request Help |
| CLI 选项组合冲突 | `fix_option_combination` | 完整 Request Help；Hint 指明删除冲突选项之一 |
| Header 格式不合法 | `inspect_header_usage` | 验证后的 `--cli-search header` |
| Body 文件不可读取 | `fix_body_file` | 验证后的 `--cli-search body-file` |

所有 Help Flag 使用 `--cli-` 前缀，所有 Help 入口使用 `aliyun help ...`。

## 5. Recovery 可靠性

- 保持当前 PascalCase/kebab 风格和显式 API Version。
- API 资源关键词从真实候选生成，例如 `DescribeInstances` → `Instances`、`GetCouponList` → `CouponList`。
- Search Command 必须调用 Search 分册的同一验证接口并确认有结果。
- Meta/Go 插件 Help 不支持新 Flag 时，不生成新 Search Command，只退回其现有普通 Help。
- `did_you_mean` 只给候选，不自动运行候选 API。
- Recovery 不复制用户传入的参数值、Header 值、Body 内容或凭证。

未知 API 示例：

```json
{
  "message": "'DescribeInstnaces' is not a valid API.",
  "did_you_mean": ["DescribeInstances", "describe-instances"],
  "recovery": {
    "action": "search_api",
    "command": "aliyun help ecs --cli-search Instances",
    "hint": "Search APIs related to Instances."
  }
}
```

## 6. 非 AI Mode 本地错误 Hint

相同的类型化宿主本地错误在非 AI Mode 保持原文本，并在 stderr 末尾追加：

```text
For AI agents, run:
  export ALIBABA_CLOUD_CLI_AI_MODE=1

This enables compact Help, structured JSON errors, and actionable recovery guidance.
```

每次错误只追加一次。Structured Error、已经是 JSON 的错误、服务端、网络、插件和未类型化错误不追加。

## 7. 明确排除

- Canonical 参数约束失败：保留现有校验器和 `Error()` 文案，但绕过新的本地 Envelope，走普通错误渲染，不新增 Recovery；
- 本地 Profile/凭证错误：绕过新的本地 Envelope，走普通错误渲染，不新增 Recovery；
- OpenAPI SDK/Tea 服务端错误；
- 网络错误；
- Go 插件进程错误；
- `--cli-query`、Pager、Waiter、表格渲染等调用后处理错误；
- Safety Policy 专用错误；
- Machine Help 自身结构化错误；
- Meta 损坏和未类型化内部错误。

当前 `normalizeAgentError` 必须收窄：服务端、网络和兜底内部错误不再被包装成新的本地 Error Envelope。

## 8. 测试

- 九种支持错误各有 AI JSON 和非 AI 文本断言；
- 空字段省略、删除字段不存在、退出码正确；
- Search 验证成功和回退；
- 风格与显式版本保持；
- 服务端/网络/插件/未类型化错误不被改写；
- Canonical 约束和凭证错误不进入新 Recovery；
- JSON stderr 无 ANSI、无前后杂讯。
