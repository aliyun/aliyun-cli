# 内置 OSS 显式机器契约（review 方案 C）

适用于内置 `aliyun oss`。方案 A/B 的本地改动继续保留；默认文本列表、旧参数名、过滤顺序和 `-f` 的授权语义不变。

## 启用与命令范围

```sh
aliyun oss ls oss://example/prefix/ --cli-output json --limited-num 100
aliyun oss ls oss://example/prefix/ --cli-output jsonl --limited-num 100
aliyun oss ls oss://example/prefix/ --cli-output json --limited-num 100 --cli-cursor '<next_cursor>'
aliyun oss ls oss://example/ --all-versions --cli-output json
aliyun oss ls oss://example/ -m --cli-output json
aliyun oss ls --cli-output json
```

- `--cli-output text` 或省略：保留旧成功输出。
- `ls --cli-output json/jsonl`：桶、对象、目录前缀、版本、删除标记、分片上传都有独立 item 类型；`-a` 依次分页返回对象（或版本）和分片上传。
- `cat` 即使指定 JSON/JSONL，stdout 仍是对象原始字节；不会包裹 JSON、添加耗时或缓冲整个对象。失败可以使用结构化 stderr。
- 其他命令尚无结构化成功结果，显式 JSON/JSONL 会在凭证获取/执行之前拒绝。签名 URL、XML 等成功输出不被强行包装；这些命令可单独启用 AI 错误。
- `--cli-ai-mode`、`ALIBABA_CLOUD_CLI_AI_MODE` 和宿主保存的 AI 配置启用 OSS 结构化错误；`--no-cli-ai-mode` 优先关闭 AI 错误。但显式 JSON/JSONL 仍要求结构化错误。
- 机器模式参数由宿主桥接消费，不传入 OSS 旧解析器。支持等号、空格和 `--` 终止符。
- 本批不实现 validate/plan；宿主 dry-run 参数在 OSS 上明确拒绝，包括继承的前置参数，防止用户以为预览却执行写入。

## JSON 与 JSONL

JSON 示例：

```json
{"schema_version":"1","items":[{"kind":"object","bucket":"example","key":"a.txt","size":5}],"returned":1,"complete":false,"truncated":true,"next_cursor":"opaque-token"}
```

`kind` 取值为 `bucket`、`object`、`prefix`、`version`、`delete_marker`、`upload`。按类型返回 `key`、`size`、`etag`、`storage_class`、`last_modified`、`created`、`region`、`version_id`、`is_latest`、`upload_id`。零字节对象仍含 `size: 0`；时间使用 RFC3339。字段只增不改，消费者应忽略不认识的字段。

JSONL 每个 item 一行：`{"type":"item","schema_version":"1","item":{...}}`。最后独立一行 `type: summary`，包含 `schema_version`、`returned`、`complete`、`truncated`、可选 `next_cursor`。没有汇总行，或进程退出非零，表示结果未完整交付，不应当作成功。写入 stdout 失败会返回错误。

JSON/JSONL 列表 stdout 不含标题、数量提示、进度或耗时。错误交给宿主在 stderr 渲染。AI/非交互模式下 cp/sync/rm 等传输、删除命令的旧进度和提示移至 stderr；消费者不能假定这些命令的 stderr 每一行都是 JSON。

## 有界分页

- 默认每次最多请求 100 项；显式 `--limited-num` 必须为 1–1000。旧文本模式的无限列表语义不变。
- 每次调用只读取一个逻辑服务页，沿用旧重试预算；SDK 请求携带对应 max-keys/max-uploads。过滤可能产生 0 项但 `truncated: true` 的结果，调用方仍需使用游标继续。
- `complete` 表示已遍历当前查询的所有阶段；`truncated` 表示还有页或后续阶段，和 `next_cursor` 同步。
- 游标绑定桶、前缀、endpoint、过滤顺序、目录/版本/分片模式以及初始 marker。继续时保留原查询参数，只添加 `--cli-cursor`。输出格式和展示限额可以改变。
- 页中截断保存当前服务页 marker、页大小、页内偏移及内容摘要，不使用下一页 marker 跳过未输出项。版本和分片页保留双 marker。
- 游标是可解码的状态，不是凭证，不含 AK/Token，也不授予权限。格式错误、查询不匹配、无进展的服务游标会报错。不要依赖游标内部字段或自行构造它。
- 这不是快照：页内重放发现内容变化会拒绝续页并要求重新列举；跨页并发修改仍遵循 OSS 列举语义，不能保证全程一致快照。游标不用于恢复写操作。

## 错误与确认

沿用 `cli.AgentErrorEnvelope` 的 `message`、`error_code`、`status_code`、`request_id`、`recovery`，增加 OSS 扩展：

```json
{"message":"...","error_code":"AccessDenied","status_code":403,"request_id":"...","recovery":{"action":"check_permissions","hint":"..."},"oss":{"schema_version":"1","phase":"execution","status":"failed","side_effects":"none","retryable":false}}
```

- 阶段为 `validation`、`configuration`、`execution`。本批没有虚构逐项 BatchResult、精确成功计数或传输/删除阶段日志，这些仍属于后续计划与恢复能力。
- 参数错误给 `InvalidArgument`，配置阶段错误给 `ConfigurationError`；服务端错误按真实 ServiceError 分类，保留 request ID 和解包链。未知执行错误保留脱敏消息，不凭字符串猜测错误类型。
- AccessDenied、资源不存在、凭证过期、地域/签名地域错误有对应检查建议。429/5xx/网络错误仅对这里已明确识别的读取命令给可重试提示；写操作的副作用保守标为 `unknown`，不生成自动重放指令。`retryable: false` 不代表永久错误，只表示本适配器没有证明可安全自动重试。
- OSS 错误保持退出码 1，成功为 0；不沿用 AgentError 默认的本地用法退出码 2。
- `--cli-non-interactive` 不读 stdin。AI/JSON 模式也不询问确认；需要确认时返回 `ConfirmationRequired`，`status: confirmation_required`，不会自动添加 `-f`。普通模式仍保留终端交互及原有拒绝行为。
- 被拒绝的项不算已授权成功。整条命令最终失败，即使旧路径把该项记为 skipped。sync 在传输阶段出现确认缺失时，不进入删除阶段。此时已有其他项完成是可能的，因此不能盲目重跑整批。
- 所有旧 Scanln 确认/值输入入口及密码输入均受统一拦截。交互式 `oss config` 在上述模式明确拒绝，避免绕过 Scanln 的读取阻塞。`-f` 只继续发挥各命令已有作用。

## 验证与边界

`oss/lib/phase_c_test.go` 使用虚构凭证和本地 HTTP 服务，覆盖 JSON/JSONL、对象/目录/版本/删除标记/分片/桶、多阶段游标、过滤空页、页内偏移、页变化、错误恢复、secret 脱敏、stdout 写入失败、非交互确认、显式 force、raw cat、状态恢复、入口 stderr 和实际进程退出码。

没有访问真实 OSS，也不运行依赖真实账号的旧集成测试。单例和进程级 argv/stdout 的旧结构仍然存在，不支持并发嵌入；每次桥接调用结束恢复状态。统一重试策略、刷新 provider、在线 plan、逐项失败清单和自动恢复仍属于方案 D。

额外 `-race` 检查发现旧进度监控的已知竞态：`CPMonitor.progressBar` 对 `finish` 的访问，以及 `RMMonitor` 对 `seekAheadEnd` 的访问；分别由已有 sync 上传失败测试和显式 force 删除测试触发。`monitor.go` 本批未修改。普通 A/B/C 回归、CLI/OpenAPI 包测试与主程序编译通过，但不能声称完整竞态检查通过。
