# Aliyun CLI Command Tree 与 Help：共享契约

## 1. 目标与基线

本设计基于 `origin/unify-meta-test@395a2a01a3168679f0c3f43fa2dede4d6559e48f`，Canonical Meta 固定为 gitlink `405d3a27746425db3f2c21f2764a9a1b26af0563`。

本轮只改宿主 CLI 在**目标产品未安装插件**时的 Command Tree、Help、AI Mode 本地错误引导。已安装产品插件、Extension CLI 和 OpenAPI 调用链保持原有所有权和行为。

## 2. 不可破坏的边界

1. 目标包含产品时，先判断本地插件，再构建 Canonical Help、选择 AI 默认格式或追加宿主 Hint。
2. 已安装插件时，原始参数交给插件；宿主不改 Help、JSON、颜色、截断、错误或执行。
3. 未安装插件时，驼峰与烤串共用同一份 Canonical、Help Document、选择与渲染逻辑；只改变命令和参数名称。
4. 默认无插件风格为驼峰；`ALIBABA_CLOUD_BASELINE_PRODUCT_HELP=true` 选择烤串风。
5. 根 Help 必须离线，只读内置 Canonical 与本地命令注册，不访问远程 Plugin Index。
6. 本轮不改变 API 请求序列化、调用、业务响应的 `--output/-o`、插件安装格式或 Canonical 文件结构。

## 3. 最终公开语法

```text
aliyun [<product> [<action> [--<parameter>]]] --help
aliyun [<product> [<action> [--<parameter>]]] --help-all
aliyun [<product> [<action> [--<parameter>]]] --help-search <query>

aliyun ... <help-operation> --cli-output json

aliyun help <product> <action> [--cli-section request|response]
aliyun help <product> <action> --cli-section request|response --help-search <query>
```

最终公开语法不保留 `--cli-all`、`--cli-search`、`--help=json`、`--help-json` 和 `help ... --format json`。`--cli-output json` 只改变宿主响应格式；单独出现时不能进入 Help。

`--help`、`--help-all`、`--help-search` 三者互斥，重复或组合使用均返回本地选项冲突。`--cli-output json` 与任一 Help 操作正交。

## 4. 统一 Help Target

所有宿主入口先解析为一个内部目标，再选择 Provider 和渲染器：

```text
HelpTarget
├── level: root | product | action | parameter | utility
├── product / action / parameter
├── commandStyle: camel | kebab
├── section: request | response
├── operation: default | all | search
├── searchQuery
├── output: text | json
└── provider: plugin | host
```

L0～L2 同时兼容 `aliyun help ...`，但所有普通 Help、Search、JSON `next` 和 Recovery 统一生成后置命令。只有 Request/Response Section 使用前置 `aliyun help ... --cli-section ...`。L3 只使用后置参数 Help。

## 5. 模式与输出

| 场景 | 默认行为 |
| --- | --- |
| 非 AI Mode | 沿用原默认输出；宿主文本 Help/本地可识别文本错误保留进入 AI Mode 的 Hint |
| AI Mode | 宿主默认等价于 `--cli-output json`，关闭颜色和 ANSI；JSON 前后不得追加文本 |
| 已安装插件 | 无论是否 AI Mode，插件完全接管 |
| Extension CLI | 保留独立命令树和输出契约 |

非 AI 文本 Help 的固定 Hint：

```text
For AI agents, run:
  export ALIBABA_CLOUD_CLI_AI_MODE=1

This enables compact Help, structured JSON errors, and actionable recovery guidance.
```

## 6. 正交分册

| 分册 | 独占职责 |
| --- | --- |
| [01 Help Surface and Provider Routing](2026-08-28-command-tree-help-01-surface-routing-design.md) | 全局 Flag、Help Target、插件优先、风格、根命令树与 Utils |
| [02 Help Document and Projection](2026-08-28-command-tree-help-02-document-projection-design.md) | Root/Product/Action 文档、Text/JSON 投影、截断、Title/Example |
| [03 Search](2026-08-28-command-tree-help-03-search-design.md) | 候选集、分词、排名、20 条上限、Recovery 搜索验证 |
| [04 Parameter and Sections](2026-08-28-command-tree-help-04-parameter-sections-design.md) | L3 参数 Help、Request/Response Section、Response Ref/Components |
| [05 AI Mode and Local Recovery](2026-08-28-command-tree-help-05-ai-mode-recovery-design.md) | 默认 JSON/无色、本地错误 Envelope、Recovery 策略、非 AI Hint |
| [06 Integration Acceptance](2026-08-28-command-tree-help-06-integration-acceptance-design.md) | 合并边界、回归矩阵、构建、包和端到端验收 |

## 7. 明确不做

- 不修改已安装插件和 Extension CLI 的 Help/输出。
- 不为 OpenAPI 服务端、网络、凭证/Profile、Canonical 参数约束、插件或未类型化内部错误新增 Recovery。
- 不在 CLI 中递归内联 Response `$ref`。
- 不把 `--cli-output` 与业务结果 `--output/-o` 合并。
- 不为嵌套 Request 字段虚构可执行 Flag。
