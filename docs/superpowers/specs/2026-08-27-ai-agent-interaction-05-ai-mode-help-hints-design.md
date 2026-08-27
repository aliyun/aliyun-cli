# AI Mode Activation and Help Hints

## 1. 职责

本分册负责有效 AI Mode 的计算，以及非 AI Mode Help 中的启用引导。非 AI Mode 错误 Hint 由本地错误分册负责，避免两套错误分类。

## 2. 有效模式

当前宿主只读取配置和命令行 Flag，尚未读取 `ALIBABA_CLOUD_CLI_AI_MODE`。本期必须让评审文案中的环境变量真实生效。

优先级：

```text
--no-cli-ai-mode
> --cli-ai-mode
> ALIBABA_CLOUD_CLI_AI_MODE
> 本地 ai-mode.json 配置
```

- `1`、`true` 表示开启；`0`、`false` 表示关闭，忽略大小写和首尾空格。
- 环境变量为空或非法时回退到本地配置。
- 有效 AI Mode 开启时，宿主文本输出关闭颜色。

## 3. 非 AI 文本 Help

未安装插件且使用宿主 Canonical 的 Root/Product/API 文本 Help，在 stdout 末尾追加完全相同的文案：

```text
For AI agents, run:
  export ALIBABA_CLOUD_CLI_AI_MODE=1

This enables compact Help, structured JSON errors, and actionable recovery guidance.
```

- 只在有效 AI Mode 关闭时追加一次。
- 不追加到已安装 Meta/Go 插件的文本 Help。
- 不改变原 Help 主体和命令顺序。

## 4. 非 AI JSON Help

Root/Product/API 的 `--format json` 增加：

```json
{
  "aiModeHint": {
    "command": "export ALIBABA_CLOUD_CLI_AI_MODE=1",
    "message": "Enable AI Mode for compact Help, structured JSON errors, and actionable recovery guidance."
  }
}
```

- JSON 前后不得打印 Hint 文本。
- AI Mode 已开启时省略 `aiModeHint`。
- 空字段遵循 Machine Help 的统一省略规则。
- Machine JSON 始终由宿主 Canonical 生成，因此是否安装产品插件不改变 `aiModeHint` 规则。

## 5. 已开启 AI Mode

现有文本 Help 中的关闭提示继续保留，包括：

```text
aliyun configure ai-mode disable
```

该提示保持现有文本路径，不得写到 Machine JSON 前后。Machine JSON 必须始终是单一合法 JSON 文档。

## 6. 测试

- 配置、环境变量、两个命令行 Flag 的全部优先级组合；
- 环境变量合法、非法和空值；
- Root/Product/API × Text/JSON；
- AI 关闭时启用 Hint，AI 开启时关闭提示仍存在；
- 已安装插件文本 Help 不追加宿主 Hint；Machine JSON 仍按宿主模式输出结构化 Hint；
- JSON stdout 可独立解析，stderr 不破坏 JSON 契约；
- AI Mode 宿主输出无 ANSI Color。
