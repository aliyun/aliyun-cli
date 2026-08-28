# Help Surface、Provider 路由与根 Command Tree

## 1. 入口解析

解析顺序固定为：识别 Help 操作与目标层级 → 目标含产品时检查已安装插件 → 未安装插件才选择 Canonical、命令风格、投影和渲染器。

`--help`、`--help-all`、`--help-search` 都能独立进入 Help。参数级目标只在“一个真实、未赋值的参数 Flag/alias 后出现 Help 操作”时成立；参数已有值时仍是 Action Help。

非法组合统一返回类型化本地错误：Help 操作冲突、重复 Help 操作、空 Search、无效 `--cli-output`。不得用优先级静默消解。

## 2. 插件优先

产品级和 Action/参数级入口必须使用同一套本地 manifest 匹配能力（插件名、short name、alias）。一旦命中已安装插件：

- 原始 argv 转发给插件；
- 不进入 Machine Help、Canonical Runtime Help、宿主 Search/All/Section；
- 不应用 AI Mode 默认 JSON/无色、宿主截断和 Hint；
- `ALIBABA_CLOUD_ORIGINAL_PRODUCT_HELP` 不得覆盖插件接管。

插件未安装时不依赖远程索引判断宿主 Help 是否可用；Canonical 中存在的产品直接走宿主。

## 3. 命令风格

- L1 无插件默认使用驼峰；`ALIBABA_CLOUD_BASELINE_PRODUCT_HELP=true` 使用烤串。
- L2/L3 由实际 Action token 决定风格。
- 驼峰默认版本为 `products.json.version`；烤串默认版本为 `plugin_default_version`；显式 `--api-version`（以及兼容 `--version`）优先。
- 所有正文、示例、`next`、Hint 和 Recovery 使用同一个风格结果。

无插件烤串文本 Help 与插件版采用“交集一致”：共同出现的章节/参数/示例在名称、类型、必填、顺序、描述、结构提示和主要排版上一致。优先复用 runtime/plugin 渲染模块；无法直接复用时用 Golden 对照测试固定单对象契约。

## 4. 根 Command Tree

`aliyun`、`aliyun --help`、`aliyun help` 解析为同一个 L0 Target。根 Help 不展示旧 Usage 段，顺序固定：

1. CLI 名称与版本；
2. Quick Start；
3. Core Commands；
4. Global Flags；
5. Extension CLIs；
6. Cloud Product CLIs；
7. Search/Show all/JSON 导航。

根产品列表只读内置 Canonical `products.json`；Core、Utils、Extension 只读本地注册信息。禁止读取远程 Plugin Index，也不展示远程插件安装状态。

### 4.1 分组

| 分组 | 默认对象 |
| --- | --- |
| Core | `configure`、`plugin`、`upgrade`、`version`、`auto-completion`、`mock`、`utils` |
| Utils | `list-supported-pricing-apis`、`mcp-proxy`、`go-migrate` |
| Extension | `oss` legacy、`ossutil`、`agentbay`、`otsutil`、`kmscli`、`ecctl` 等独立 CLI |
| Products | 内置 Canonical 全部产品 |

新增规范路径 `aliyun utils <name>`；原三个根入口继续执行同一实现，但不在根 Help 重复展示。

### 4.2 Global Flags

默认展示：`--profile/-p`、`--region`、`--language`、`--version/--api-version`、`--output/-o`、`--cli-query`、`--cli-output`、`--dryrun`、`--yes/-y`、`--cli-ai-mode`、`--help/-h`、`--help-all`、`--help-search`。

低频公开 Flag 仅在 All/Search 中出现。Hidden/internal Flag 在 Default/All/Search 中均不可见。分类需要显式元数据，不允许仅按 Flag 名临时猜测。

## 5. Utils 输出边界

`aliyun utils --help|--help-all|--help-search` 和 JSON Help 使用统一 Help 操作。Utility 的执行结果是否支持 `--cli-output json` 逐项声明；未完成公开契约的 Utility 可不在根 Help 展示，但兼容入口不删除。

## 6. 主要验收

- 模拟已安装 Go/Meta 插件，确保所有产品 Help 参数原样转发，宿主输出为空。
- 断网时根 Help 内容、耗时和退出码不变。
- 裸命令与两种根 Help 入口在相同模式/格式下生成相同 Document。
- 驼峰/烤串产品版本、Action/参数名和后续命令保持风格一致。
- 新旧 Utils 路径调用同一 handler，根 Help 只展示新路径。
