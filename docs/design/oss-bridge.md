# 内置 OSS 桥接与错误边界（review 方案 B）

本批次承接方案 A，仅修改内置 `aliyun oss`，不接入 OSS JSON/AI 输出、在线 plan 或自动恢复。

## 参数与错误

- OSS 子命令使用宿主的 `RawArgs` 路由。宿主只在选项位置识别帮助，完整 token 交给 OSS；`help` 文件名和 `--` 后的横线路径不再被误处理。
- `OptionMap` 同时提供注册与解析所需的类型、整数范围和枚举。布尔参数不取值；每个列表参数只消费一个值；标量重复时最后一次生效。include/exclude 保留原顺序与次数。
- 空格和等号写法均支持，包括空字符串。空值继续沿用旧 OSS 的默认值补全语义。缺值、未知选项、非法整数/枚举返回错误，不再通过 goopt 打印帮助并退出进程。
- 参数数量与可用选项在获取凭证之前验证。校验顺序固定；类型校验使用公开的选项名。
- BucketError/ObjectError/FileError/CopyError 支持 `Unwrap`，桥接错误使用 `%w`，调用者可以通过 `errors.Is/As` 获取服务错误及 RequestId。

## 配置与执行

- 一次加载宿主 profile，复用宿主的账号、环境变量与 flag 解析规则，不改变既有账号选择规则。环境凭证/region 仍遵循宿主已有的空值回退行为。
- 显式 endpoint 优先，然后是 profile endpoint，最后按 region/endpoint-type 构建。region 同时传入 OSS SDK 的 v4 签名配置。
- read-timeout/connect-timeout 在两端均为秒；显式值优先于 profile。`--retry-timeout` 作为宿主 read-timeout 的别名消费。
- 当前宿主 runtime 的 retry-count 与 OSS retry-times 均表示最大尝试次数。显式 OSS retry-times 优先于宿主 retry-count，再优先于 profile retry_count；未配置时保留旧 OSS 默认预算。retry-count=0 沿用宿主的默认预算语义。未统一各命令的重试错误分类。
- 显式代理参数保留；宿主 skip-secure-verify 映射为 OSS skip-verify-cert。不会自动降级 TLS。
- 解析后的配置和凭证在执行期间单独传递，不追加到 argv。SDK 使用宿主解析出的 AK/STS 快照，不再次执行 role assume。日志继续使用方案 A 的统一脱敏。
- 宿主配置优先于旧 OSS config-file 的同名配置，旧配置的剩余字段继续补全。宿主 endpoint 的桶级映射覆盖行为与方案 A 一致。
- hash/help 不获取云端凭证；config 保持旧 OSS 配置文件操作；内置 update 明确拒绝，避免独立 ossutil 自更新逻辑替换宿主程序。
- 返回时恢复 argv、桥接配置与本次修改的 flag 状态；密码输入缓存每次初始化清空。旧命令单例和进度通道仍存在，不支持并发嵌入；长任务 provider 刷新属于方案 D。

## 验证

回归覆盖 token 等价性、前置选项、重复标量、过滤顺序、空值、缺值、终止符、无凭证本地校验、宿主别名、配置优先级、错误解包与连续调用状态恢复。

本地 HTTP 模拟服务验证 profile region 出现在 v4 签名范围中；profile retry_count=2 实际发出 2 次请求，显式 retry-count=1 发出 1 次，retry-count=3 与 retry-times=1 同时出现时发出 1 次。服务错误仍能解包为 OSS ServiceError。

没有访问真实 OSS 账号，也没有运行依赖真实账号的旧集成测试。
