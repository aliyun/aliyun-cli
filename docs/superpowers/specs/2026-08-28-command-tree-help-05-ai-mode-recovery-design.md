# AI Mode、Hint 与本地 Recovery

## 1. Effective AI Mode

保留现有优先级：`--no-cli-ai-mode` > `--cli-ai-mode` > 有效环境变量 > 本地配置；自动 Agent 检测按现有入口接入。宿主 AI Mode 默认：

- Help/本地结构化错误输出 JSON；
- `cli.SetNoColorOverride(true)`，无 ANSI；
- 不显示“使用 `--cli-output json`”提示；
- 若显式进入宿主文本路径，保留 `aliyun configure ai-mode disable`。

AI 默认 JSON 只能在插件 Provider 判断后应用，不能把格式 Flag 注入插件 argv。

## 2. 非 AI Mode Hint

- 宿主纯文本 Help 末尾追加一次进入 AI Mode 的固定 Hint。
- 宿主 JSON Help 用结构化 `aiModeHint`，JSON 前后无文本。
- 类型化 CLI 本地文本错误在 stderr 末尾追加一次 Hint。
- 服务端、网络、插件、未类型化或已经结构化的错误不追加。

## 3. 本地错误 JSON 契约

AI Mode 本地用法错误 stdout 为空、stderr 仅一个 JSON、exit 2：

```json
{
  "message": "required parameter is not assigned: --InstanceId",
  "did_you_mean": ["--InstanceId"],
  "recovery": {
    "action": "inspect_request_help",
    "command": "aliyun help ecs DescribeInstanceAttribute --cli-section request",
    "hint": "Inspect the complete request parameters and provide every required value."
  }
}
```

`message` 必需；`did_you_mean` 可选且只含当前层/风格高可信名称；`recovery` 只含一个最优动作，`action`/`hint` 必需，可靠时才有 `command`。不输出 `ok/category/code/details/requestId/retryable`。空字符串、null、空数组和空对象均省略；构造器应拒绝不完整必需字段。

## 4. 类型白名单

仅对现有明确类型化本地错误归一化：未知产品/API/宿主子命令/Flag，缺少必填参数，参数语法/类型/JSON，Help 选项冲突，Header 格式，Body 文件不可读。不得根据普通 error message 猜类型。

本期明确排除：Canonical 枚举/范围约束、Profile/凭证、OpenAPI 服务端、SDK/Tea、网络、调用后 Query/分页/Waiter/表格、外部插件、Meta 损坏、其他未类型化错误。

## 5. Recovery 映射

| 错误 | 最优 Recovery |
| --- | --- |
| 未知产品 | 已验证 `aliyun --help-search <keyword>`；无命中 `aliyun --help` |
| 未知 API | 已验证 `aliyun <product> --help-search <keyword>`；无命中产品 Help |
| 未知宿主子命令 | 已验证当前层 Search，否则父命令 Help |
| 未知参数/Flag | 已验证 `<action> --help-search <keyword>`，否则 Action Help |
| 缺少必填 | 前置完整 Request Section |
| 已知参数语法/类型/JSON | 真实顶层参数使用 `<action> --<parameter> --help`；否则 Request Search/Section |
| Help 选项冲突 | 不输出 command；Hint 明确列出要删除的冲突选项 |
| Header 格式 | `<action> --header --help` |
| Body 文件不可读 | `<action> --body-file --help` |

普通 Help/Search 使用后置语法，Section 使用前置语法。命令保持当前风格与显式 API Version。Search 必须走 03 分册验证；命令禁止复制用户值、凭证、Header、Body 或文件路径。

## 6. 主要验收

- 所有白名单错误的 JSON 字段精确，无空字段/旧字段/ANSI，stderr 可单独 parse。
- 每个 Search Recovery 实际有命中；无命中降级为普通 Help。
- 选项冲突没有 command；已知参数错误只指向真实 L3 Flag。
- 排除错误保留原渲染，不伪造 Recovery 或 AI Hint。
- 非 AI Hint 恰好一次；AI 文本关闭 Hint 不丢。
