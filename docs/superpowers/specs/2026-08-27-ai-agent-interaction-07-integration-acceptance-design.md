# Integration, Compatibility, and Acceptance

## 1. 职责

本分册定义 01—06 的集成顺序、性能边界、异常降级和最终验收，不重新定义各模块算法。

## 2. 推荐合并顺序

1. 合入共享数据模型和稳定接口，不接用户入口；
2. 并行合入 Canonical Response、Search、Query Example、AI Mode 状态和 Local Error 模块；
3. Help Command Surface 统一接线；
4. 补跨模块集成测试和真实 Canonical Fixture；
5. 运行完整测试、构建和手工验收。

并行 Agent 应尽量新增独立文件。对以下高冲突文件的最终修改由 Help Surface 或集成工作流统一完成：

- `openapi/commando_help.go`
- `openapi/machine_help.go`
- `openapi/agent_error.go`
- `cli/command.go`

## 3. 兼容性不变量

- 未指定 Section 的 API Help 与现有 Request Help 行为一致。
- API 请求、序列化、Endpoint、鉴权和响应输出完全不变。
- PascalCase 使用 Legacy 参数视图和默认版本；kebab 使用 Canonical 参数视图和插件默认版本。
- 已安装 Meta/Go 插件继续拥有自己的文本 Help；Machine JSON 继续由宿主 Canonical 提供。
- `--help`、`--help=json` 继续兼容。
- Machine Help `schemaVersion` 保持 `v1`；本功能尚处 Unify 测试阶段，新增字段和空字段省略直接收敛进 v1。
- JSON 前后不得出现非 JSON 文本。

## 4. 性能预算

| 操作 | 允许读取 |
| --- | --- |
| Root Help/Search | `products.json`，不得遍历 API JSON |
| Product Help/Search | 对应 `version.json`，不得遍历每个 API JSON |
| API Request Help | 单个 API JSON；可惰性解析其 Response 以生成一个 Query Example |
| API Response Help/Search | 单个 API JSON 的 `responses/components` |
| Recovery Search 验证 | 与相应 Help Search 相同的本地索引，不访问网络 |

任何路径都不得全量解压 Canonical Pack、扫描全部产品 API 或调用开放元数据服务。

## 5. 降级规则

| 故障 | 行为 |
| --- | --- |
| API 没有 Response Schema | Help 成功，显示明确 Notice |
| Ref 循环/缺失 | 保留 Ref，输出可用 Schema 和 Warning |
| Query Example 无可靠 Array | 省略 Example，不影响原 Help |
| AI Mode 配置读取失败 | 仅把配置值视为关闭；命令行 Flag 和合法环境变量仍按既定优先级生效 |
| Search 无结果 | 返回空结果和明确消息；Recovery 不生成该 Search 命令 |
| 远程插件索引失败 | 不影响本地 Canonical Help |
| Canonical 主索引/单 API 损坏 | 返回现有 Machine Help 类型化错误或文本错误，不 Panic |
| 已安装插件的文本 Help | 回到插件现有 Help，不强行追加宿主输出 |
| 已安装插件的 Machine JSON | 保持宿主 Canonical Machine Help，不委托插件 |

## 6. 自动化验收矩阵

| 维度 | 覆盖值 |
| --- | --- |
| 模式 | AI / 非 AI |
| 输出 | Text / JSON |
| 命令风格 | PascalCase / kebab-case |
| Help Provider | 无插件 Canonical / Meta 插件文本 / Go 插件文本 / 已安装插件下的宿主 Machine JSON |
| Section | 默认 Request / 显式 Request / Response |
| 列表 | 少于 20 / 等于 20 / 超过 20 / Search / All |
| Response | Inline / Ref / 循环 Ref / 缺失 Ref / 无 Schema / 多 Array |
| 版本 | 默认版本 / 显式 `--api-version` / 多版本 API |
| 错误 | 九种本地错误 / 服务端 / 网络 / 插件 / 未类型化 |

重点断言：

- Request/Response Section 在两种模式下都完整；
- Search 结果超过 20 条仍全部输出；
- AI 默认 Product/API 列表最多 20 条；
- Response JSON 只包含可达 Components；
- Query Example 风格、版本和 Path 正确；
- 非 AI Help/本地错误 Hint 恰好一次；
- AI Error 只有允许字段且 Recovery 可实际返回有效 Help；
- 服务端、网络和插件错误没有被宿主编造 Recovery。

## 7. 命令级手工验收

```bash
# Request 默认兼容
aliyun help ecs DescribeInstances

# Response Text / JSON
aliyun help ecs DescribeInstances --cli-section response
aliyun help ecs DescribeInstances --cli-section response --format json

# Product/API/Parameter/Response Search
aliyun help --cli-search ecs
aliyun help ecs --cli-search instance
aliyun help ecs DescribeInstances --cli-search instance-id
aliyun help ecs DescribeInstances --cli-section response --cli-search instance-id

# AI 列表上限与完整列表
ALIBABA_CLOUD_CLI_AI_MODE=1 aliyun help ecs
ALIBABA_CLOUD_CLI_AI_MODE=1 aliyun help ecs --cli-all

# Camel/kebab Query Example
aliyun help ecs DescribeInstances
aliyun help ecs describe-instances

# 本地错误 JSON
ALIBABA_CLOUD_CLI_AI_MODE=1 aliyun ecs DescribeInstnaces
ALIBABA_CLOUD_CLI_AI_MODE=1 aliyun ecs describe-instances --page-size abc
```

## 8. 完成标准

- 01—06 分册测试全部通过；
- `go test ./...` 通过；
- Packed Meta 相关测试通过；
- Linux/macOS 构建通过；
- 手工验收命令符合本文；
- 未安装插件路径的新行为完成，已安装插件路径无回归；
- 不依赖尚未发布的远程服务，Fixture 可离线复现。
