# 内置 OSS 计划与恢复（review 方案 D）

本批次在本地 A/B/C 改动之上实现显式预览、失败项清单、保守重试和宿主凭证刷新。所有新增预览均独立于实际执行路径，不调用业务 RunCommand 来模拟 dry-run。

## 能力范围

| 操作 | `--cli-validate` | `--cli-plan` | `--cli-failure-report PATH` |
| --- | --- | --- | --- |
| cp 单个本地普通文件 → 明确的 OSS 对象 key | 本地参数、URI、源文件检查 | 本地检查 + HEAD 目标 | 支持 |
| rm 单个明确对象，可带 version-id | 本地参数、URI 检查 | HEAD 对象/指定版本 | 不支持 |
| cp 下载、云端复制、递归传输 | 拒绝 | 拒绝 | 支持传输失败项 |
| sync | 拒绝 | 拒绝 | 支持传输失败项和删除阶段状态 |
| 其他命令 | 拒绝 | 拒绝 | 拒绝 |

预览暂不支持目录目标、尾斜杠、过滤、update、encoding-type、payer 等业务选项。支持范围以代码中的允许列表为准；不支持的选项在获取凭证前拒绝。旧 `--dryrun`、`--cli-dry-run` 等参数继续拒绝，不会误执行写操作。

```sh
aliyun oss cp ./sample.txt oss://example-bucket/sample.txt --cli-validate
aliyun oss cp ./sample.txt oss://example-bucket/sample.txt --cli-plan
aliyun oss rm oss://example-bucket/sample.txt --version-id VERSION --cli-plan
aliyun oss cp ./data/ oss://example-bucket/prefix/ -r -f --cli-failure-report ./failed-items.jsonl
```

validate 无需凭证、endpoint 或网络。plan 使用现有配置优先级和鉴权，只发 HEAD；不上传、不删除、不创建目标目录。两者互斥，成功均输出一个 JSON 文档（schema_version=1），包括 items、complete、side_effects、checks 和 limitations。`complete` 仅表示当前受支持的单对象集合完整，不代表权限验证或执行成功。计划是读取时的快照；ETag 只供检查，不能把旧计划当作执行凭据、事务锁或自动确认。

HEAD 的明确 NoSuchKey/NoSuchVersion 可以表示不存在；403、桶不存在以及无法区分原因的 404 都返回错误，不猜测为可安全覆盖的空目标。计划会产生只读请求；不验证写权限。

## 失败项清单

`--cli-failure-report` 在执行前以 O_EXCL、0600 创建 JSONL 文件，路径已存在即失败，不覆盖旧清单。父目录由用户提前准备。并发 worker 使用同一带锁 writer，路径中的空格和换行由 JSON 编码，无需解析人类日志。

每个 `failed_item` 包含 operation、source、target、源 version_id（适用时）、error、outcome=unknown、automatic_retry=false；最后一条 summary 记录命令成功/失败及错误。sync --delete 的 summary 额外给出 delete_phase=not_started/started/completed，机器错误也引用 failure_report 与 delete_phase。扫描、初始化或删除阶段错误保留在 summary；清单仅承诺定位传输 worker 报告的失败项，不是全量对象账本。进程被杀或写盘失败时可能没有 summary，消费者应视为不完整；写盘失败会让命令返回失败。

清单不包含可执行 shell 命令，也不提供自动整批重放。收到超时/5xx 后服务端可能已经写入，版本桶重试可能产生新版本；须先检查失败项状态，再明确选择需要重试的 source/target/version。成功项不会进入 failed_item。

传输失败会停止安排后续项并等待在途 worker 结束，再关闭清单和恢复桥接状态；显式继续处理的原有语义保留。同步修复了进度快照与结束状态的数据竞争，进度 goroutine 随调用结束回收。

旧文本失败报告继续保留，文件名增加随机后缀避免同秒碰撞、权限设为 0600、并发写入加锁。清理空报告只移除空目录，不能递归删除另一并发任务的报告。错误、SDK 日志与报告统一脱敏，覆盖刷新后的 AK/Secret/Token。

## 重试与凭证

共享的列表/元数据请求，以及 ls、lcb、stat、read-symlink、cp、rm、mb、restore、set-acl、set-meta、create-symlink 和 probe 的现有外层重试，统一使用同一分类与带抖动的指数退避（首轮 100–200ms，上限 3.2–6.4s）。retry-times/retry-count 仍限制总尝试次数，默认数值与配置优先级不变。

- 400/403/404、输入/文件错误与 credential refresh 错误不重试。
- 429 是明确拒绝，可在预算内重试。
- 读请求的网络错误、EOF/意外断流、5xx 可重试。
- 写操作的网络错误/5xx 不再由外层循环自动重放，以免未知结果导致重复版本或重复变更。多段操作保留 SDK 的 checkpoint 机制，不宣称多步操作是原子的。

这是行为修复，旧脚本可能更早看到写失败；应检查结果而不是自动重跑整个 sync。预算是这些外层操作的尝试次数，不是整个递归任务的请求总量或总时限。单次网络调用仍由现有 connect/read timeout 控制；本批次没有引入端到端 deadline，也没有改变 append 等未使用这些循环的命令语义。

宿主 provider 贯穿 OSS HTTP 请求（SDK CredentialsProviderE），每次签名读取完整凭证快照；锁保护多段 worker 的并发访问，缓存/到期刷新由原 provider 负责。刷新错误保留错误链并在发请求前返回，不能退回旧 token 或换身份。执行结束恢复 provider 全局状态。签名 URL 的 SDK 接口没有刷新错误返回通道，因此 sign 仍使用入口已成功解析的快照；不承诺签名链接能超出 STS 有效期。

## 验证

使用本地 httptest 服务和虚构凭证，覆盖离线 validate、不支持组合拒绝、plan 仅 HEAD、版本目标、轮换 token、刷新失败零请求、重试分类/预算/退避、写入 503 不重放、失败清单精确路径/脱敏/并发/文件保护、旧报告同秒命名与清理，并回归 A/B/C。B 的预算测试改用 503，403 的不重试行为由 D 单独验证。没有访问真实 OSS 账号或运行旧账号集成套件。
