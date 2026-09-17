# 内置 OSS 自动化

[文档首页](../README.md) | [English](../en/oss.md)

以下接口适用于 v3.5.1 内置的 `aliyun oss`，不适用于单独安装的 `ossutil`。各命令参数见 `aliyun oss <command> --help`。宿主 CLI 的 profile 提供凭证和地域，超时及重试设置会传递到 OSS；可刷新的宿主凭证在请求时获取。请通过原安装方式升级宿主 CLI，`aliyun oss update` 不会调用独立 ossutil 的更新器。

## 有界列表

```sh
aliyun oss ls oss://example-bucket/prefix/ --cli-output json --limited-num 100
aliyun oss ls oss://example-bucket/prefix/ --cli-output json --cli-cursor '<next_cursor>'
```

`--cli-output json` 输出 `schema_version: "1"`、`items`、`returned`、`complete`、`truncated` 及可选的 `next_cursor`。JSONL 先输出包含 `item` 的 `type: "item"` 记录，最后输出带分页字段的 `type: "summary"` 记录；每条记录均含 `schema_version: "1"`。列表项以 `kind` 标识类型，并包含适用的 bucket、key、大小、ETag、版本或上传信息。

默认最多返回 100 项，`--limited-num` 范围为 1–1000。Bucket、对象、版本、目录前缀及未完成分片上传列表使用对应的命令选项。续查时保持查询条件不变，将游标视为不透明字符串。过滤后的页面可能为空但仍有 `next_cursor`，应继续查询直到 `complete` 为 true。列表不是快照；续查时重读的页面发生变化可能要求重新开始。

## 本地校验与只读计划

```sh
aliyun oss cp ./example.txt oss://example-bucket/example.txt --cli-validate
aliyun oss cp ./example.txt oss://example-bucket/example.txt --cli-plan
aliyun oss rm oss://example-bucket/example.txt --cli-plan
```

`--cli-validate` 只做本地校验，不发送 OSS 请求；`--cli-plan` 额外读取目标元数据。两者互斥，均输出 JSON，绝不执行上传或删除。仅支持单个普通本地文件上传到明确的对象 key，以及单对象删除（可带 `--version-id`）。递归传输、下载、云端复制、`sync` 及未支持的业务参数会在执行前被拒绝。

计划包含 `schema_version: "1"`、`mode`、`command`、`complete`、`side_effects`、`items`、`checks` 和 `limitations`。它不验证写权限，也不锁定目标；元数据只是快照，不是锁或执行凭据。内置 OSS 拒绝 `--dryrun`、`--cli-dry-run` 和 `--cli-dry-run-json`，请使用上述受支持的预览参数。

## 错误与确认

使用 `--cli-ai-mode` 获取结构化错误，使用 `--cli-non-interactive` 禁止读取确认输入。AI 模式或显式 JSON/JSONL 输出也会禁止交互输入，这些参数均不代表写操作授权。需要确认时返回 `ConfirmationRequired`。获得授权后，可按操作需要使用其已有的强制选项；多项操作中，之前的部分工作可能已经完成。

AI 模式下，共享错误信封包含 `schemaVersion: "v1"`；非 AI 模式显式请求的 JSON 错误保持原有格式。结构化错误写入 stderr，包含 Agent 错误字段及 `oss` 对象，后者带有 `schema_version: "1"`、`phase`、`status`、`side_effects`、`retryable`，适用时还包含失败报告信息。这类经过适配的 OSS 错误退出码为 **1**，与 OpenAPI Agent 错误不同。`--no-cli-ai-mode` 关闭 AI 错误，但显式 JSON/JSONL 输出仍使用结构化错误。进入 OSS 桥接层之前的初始配置加载错误，在启动 AI 模式启用时也使用共享的版本化信封。

`cat` 始终保持 stdout 原始字节，即使指定 `--cli-output json`。除此以外，JSON/JSONL 成功输出仅支持 `ls` 和预览模式；不支持的组合在执行前失败。仅开启 AI 模式不会把所有成功输出转换为 JSON。

## 失败报告与重试

执行 `cp` 或 `sync` 时可添加 `--cli-failure-report <新文件路径>`。报告使用独占创建方式及 `0600` 权限，目标文件已存在时会拒绝执行。内容为 JSONL `failed_item` 记录和最后的 `summary`，包含 `schema_version: "1"`、结果信息及 `automatic_retry: false`。不能与校验或计划模式组合使用。

失败报告不会重放操作，也不能证明失败的写请求没有产生效果。重试选定条目前，应检查报告和实际资源状态，不要自动重放整个同步或删除操作。允许重试的临时故障使用带抖动的退避；永久错误或结果不明确的写失败会停止重试。
