# Aliyun CLI Agent 交互优化：共享契约与并行工作流

## 1. 目标

本设计基于 `unify-meta-test` 提交 `e08928a9`，为未安装产品插件时由宿主 CLI 读取内置 Canonical Meta 的路径增加：

- Request / Response Help Section；
- Help 搜索、AI Mode 列表上限和完整列表开关；
- Array Response 的 Schema 与 `--cli-query` 引导；
- 非 AI Mode 进入 AI Mode 的 Hint；
- 仅面向明确 CLI 本地错误的精简 JSON 和 Recovery。

设计拆成多个正交分册，以便后续多个 Agent 并行开发。各分册必须遵守本文的共享契约，不得自行修改用户可见语法。

## 2. 共享命令契约

```text
aliyun help [<product> [<api>]]
  [--cli-section request|response]
  [--cli-search <keyword>]
  [--cli-all]
  [--format json]
```

| 规则 | 结论 |
| --- | --- |
| 默认 Section | `request`；`aliyun help <product> <api>` 保持原行为 |
| Response Help | `aliyun help <product> <api> --cli-section response` |
| 新增 Flag 命名 | 所有 CLI 自身新增 Flag 使用 `--cli-` 前缀 |
| JSON 格式 | `--format json` 与 Section/Search 正交，可组合 |
| 旧入口 | 现有 `--help` 和 `--help=json` 继续兼容；引导文案统一推荐 `aliyun help ...` |
| Request/Response 大小 | AI Mode 和非 AI Mode 都默认完整展示，不截断，不要求 `--cli-all` |
| 搜索结果 | 全部返回；超过 20 条也不截断 |
| AI Mode 默认上限 | 只限制未搜索的产品列表和 API 列表，最多 20 条 |
| `--cli-all` | 只用于解除 AI Mode 产品/API 列表上限 |
| 命令风格 | Help 和示例保持用户当前输入的 PascalCase 或 kebab-case 风格 |
| 数据来源 | 只读本地 Canonical，不调用 OpenAPI 或其他网络服务 |

## 3. Help 所有权边界

文本 Help 和 Machine JSON Help 的 Provider 规则不同。`--format json` 是 Unify 已有的宿主 Canonical 能力，本期不改变它的 Provider 路由，只扩展其内容。

| 场景 | 文本 Help | `--format json` |
| --- | --- | --- |
| 未安装插件 | 宿主 CLI + 内置 Canonical，实现全部新能力 | 宿主稳定 Machine Help + 内置 Canonical |
| 已安装 Meta 插件 | 保持插件 Meta 的现有 Runtime Help | 仍由宿主稳定 Machine Help 读取内置 Canonical |
| 已安装 Go 插件 | 插件二进制，宿主只转发 | 仍由宿主稳定 Machine Help 读取内置 Canonical |
| 产品存在插件但未安装 | 宿主 CLI + 内置 Canonical | 宿主稳定 Machine Help + 内置 Canonical |

新 Recovery 命令不得假设已安装插件支持 `--cli-search` 或 `--cli-section`。如果实际 Help Provider 不支持新 Flag，只能退回该 Provider 已有的普通 Help。

## 4. 正交分册

| 分册 | 独占职责 | 主要输出 | 依赖 |
| --- | --- | --- | --- |
| [01 Canonical Response Contract](2026-08-27-ai-agent-interaction-01-canonical-response-design.md) | 读取 `responses/components`、成功响应选择、Ref 闭包 | `ResponseSchemaDocument` | Canonical 生产侧最终字段 |
| [02 Help Command Surface](2026-08-27-ai-agent-interaction-02-help-command-surface-design.md) | Help 路由、Flag、Section、文本/JSON 外壳 | 稳定命令面与渲染入口 | 01 的接口 |
| [03 Search and Listing](2026-08-27-ai-agent-interaction-03-search-listing-design.md) | 搜索、排序、20 条上限、`--cli-all` | 统一 `HelpSearchResult` | 02 的目标上下文 |
| [04 Response Query Guidance](2026-08-27-ai-agent-interaction-04-response-query-guidance-design.md) | Array Path 选择与查询示例 | `ResponseQueryExample` | 01 的 Schema 文档 |
| [05 AI Mode Help Hints](2026-08-27-ai-agent-interaction-05-ai-mode-help-hints-design.md) | 环境变量、有效模式、文本/JSON Help Hint | `aiModeHint` | 02 的扩展点 |
| [06 Local Error Recovery](2026-08-27-ai-agent-interaction-06-local-error-recovery-design.md) | 本地错误类型、AI JSON、非 AI 错误 Hint | 新本地错误 Envelope | 03 的搜索验证接口 |
| [07 Integration and Acceptance](2026-08-27-ai-agent-interaction-07-integration-acceptance-design.md) | 合并顺序、性能、降级、验收矩阵 | 集成完成标准 | 01—06 |

## 5. 并行开发约束

- 01、03、04、05、06 可以基于本文定义的接口并行开发。
- 02 独占 Help Flag 解析、Provider 路由和最终文档组装；其他工作流通过新文件中的纯函数或模型接入，避免同时重写 `openapi/machine_help.go` 和 `openapi/commando_help.go`。
- 06 独占 `AgentError` Envelope 和本地错误分类；05 不自行判断错误类型。
- 07 只做跨模块接线与集成测试，不重新实现各分册算法。
- Canonical 生产侧由其他开发者完成。本仓库以 Fixture 和约定字段并行开发，不等待全量 Meta 更新。

## 6. 明确不做

- 不修改已安装插件的文本 Help 内容；Machine JSON 继续走宿主 Canonical。
- 不内联展开 `$ref`，不访问网络补 Schema。
- 不为 OpenAPI 服务端错误、网络错误、Go 插件错误或未类型化内部错误编造 Recovery。
- 不新增 Canonical 参数约束失败的 Recovery。
- 不新增本地 Profile / 凭证错误的 Recovery。
- 不改变 API 调用、请求序列化或响应反序列化行为。
