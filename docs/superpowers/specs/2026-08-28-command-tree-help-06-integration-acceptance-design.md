# 集成、构建与验收

## 1. 集成所有权

并行分支只提交各自模块；集成分支独占下列高冲突接线：

- `openapi/commando.go`、`openapi/commando_help.go`：Provider-first 路由与最终 dispatch；
- `main/main.go`：旧 Machine Help normalize 移除、根注册与统一入口；
- `cli/command.go` / `cli/context.go`：仅在确有必要时增加原始 argv / pre-parse plugin seam；
- `openapi/machine_help.go`：并行文档模块的最终组装，不在此重写各算法。

合并必须保持 `aliyun-openapi-meta` gitlink 为 `405d3a27746425db3f2c21f2764a9a1b26af0563`，除非用户明确提供新的 Meta commit。不得提交本地生成包、个人配置或临时 manifest。

## 2. 测试层次

1. 每个正交模块运行 focused unit tests。
2. 集成后运行 `go test ./cli ./openapi ./canonicalmeta ./aliyun-openapi-runtime/... ./main -count=1`。
3. 在允许本地 socket/DNS 的环境运行 `go test ./... -count=1`；若失败，在未改基线上复跑同一用例区分环境/存量失败。
4. `go vet` 或仓库现有静态检查仅在不引入无关环境依赖时执行。
5. 使用正式打包命令构建 embedded Canonical 二进制，再做进程级 E2E。

Codex 环境存在 `NO_COLOR`/Agent 自动检测时，非 AI 测试必须显式清理相关环境或使用 `--no-cli-ai-mode`，不能把环境污染误判为产品行为。

## 3. 端到端环境

- 临时空 `HOME`、空插件 manifest、无 `ALIYUN_CLI_META_DIR`，证明读取 embedded Canonical。
- 在仓库外目录运行成品二进制。
- AI 与非 AI 分别执行；JSON 用标准解析器验证，文本检查 ANSI/Hints。
- 插件优先用隔离 manifest 与 stub Go/Meta plugin 验证原始 argv 转发。
- 所有 Help 用例禁止外网，并监控不存在 Plugin Index 请求。

## 4. 必验矩阵

| 能力 | 必验命令/断言 |
| --- | --- |
| 旧语法移除 | `--cli-all`、`--cli-search`、`--help=json`、`help --format json` 不再作为公开 Help 语法 |
| 操作组合 | Default/All/Search 与 JSON 正交；三种 Help 操作两两冲突、重复和空 query exit 2 |
| Root | `aliyun`、`aliyun --help`、`aliyun help` 同一内容；分组、离线、Utils 规范路径正确 |
| Product | 驼峰/烤串默认版本、Action 名称、Default/All/Search 描述和 20 条规则正确 |
| Action | required 全保留、optional 预算、All 完整、全局参数边界、Title/Description/Example 正确 |
| Parameter | camel/kebab L3、alias、嵌套完整、已赋值不误判、歧义/未知参数正确 |
| Sections | request/response 完整不截断，response 保留 `$ref` 和可达 Components，循环安全 |
| Query Example | 有 Array 才出现，只选一个代表路径，命令风格/版本正确 |
| Text/JSON | 同一对象选择与顺序；Text 100 行完整对象；JSON `result/next` 与空字段规则正确 |
| AI Help | 默认 JSON、无 ANSI、stderr 空、无冗余 JSON 提示；All/Search 行为正确 |
| 非 AI Hint | 宿主 Help/白名单本地文本错误恰好一次；插件/服务端/网络/未类型化不追加 |
| AI Error | stderr 单 JSON、stdout 空、exit 2、字段 allowlist、无空值、一个 Recovery |
| Recovery | 普通命令后置、Section 前置、style/version 保持；所有 Search 命令实际有命中 |
| Plugin-first | Go/Meta 插件在 Text/JSON/AI/Search/All/Section 下均完整接管 |
| Extension | 不继承宿主专属行为，不被 `--cli-output json` 强制改写 |

至少用 ECS 覆盖大参数 Action/Array Response，用一个 ROA 产品覆盖 Method/Path，用一个多版本产品覆盖默认版本；具体产品以 embedded Meta 可用性为准并记录。

## 5. 构建与包

本地验收使用仓库安全 target，不调用任何会上传 OSS 的 publish/release target：

```shell
make VERSION=5.0.0 test-release
make VERSION=5.0.0 build
```

成品必须验证 `aliyun version == 5.0.0`。按当前宿主平台生成本地可执行文件和 `.tgz`；若生成 Linux 包，使用仓库交叉编译约定。记录路径、字节数和 SHA-256，解包后再次跑 version、Root Help、AI Help、AI local error smoke。

## 6. 完成标准

- 所有分册的对客语法和所有权边界有自动测试或成品 E2E 证据。
- 完整测试失败均已归因；不能把未验证项描述为通过。
- 集成分支已提交并 push；验收修复继续提交并再次 push。
- 工作树除明确的本地包外干净，包不进入 Git。
