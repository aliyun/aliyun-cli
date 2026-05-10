# CMS2 插件实现方案（独立可执行文件接入 Aliyun CLI Wrapper）

> 本方案遵循《独立可执行文件接入 Aliyun CLI（Wrapper）规范说明》，参考 `ossutil`（环境变量封装 JSON）和 `otsutil`（本地配置文件）两种参考实现。

## 1. 背景

### 1.1 目标

在 `aliyun-cli` 中新增 `cms2` 子命令，使用户可以通过 `aliyun cms2 <subcommand> [args...]` 的方式调用 CMS CLI（`aliyuncms` 二进制），实现规范中定义的 Wrapper 模式——注册子命令、下载/更新二进制（可选）、组装凭证与环境、剥离主 CLI 专有参数后 exec 独立二进制。

### 1.2 CMS2 (aliyuncms) 概况

- **源码仓库**: `code.alibaba-inc.com/ali-prometheus/aliyuncms`（本地 checkout: `/Users/tingbin.ctb/GolandProjects/aliyun-cms-cli`）
- **可执行文件**: 本地开发构建产物 `aliyuncms`
- **框架**: spf13/cobra
- **凭证解析优先级** (从低到高):
  1. aliyun-cli 配置 (`~/.aliyun/config.json`) —— 已内置回退支持
  2. 本地配置 (`~/.aliyuncms/config.json`)
  3. 环境变量: `ALIYUN_CMS_CLI_ACCESS_KEY_ID`, `ALIYUN_CMS_CLI_ACCESS_KEY_SECRET`, `ALIYUN_CMS_CLI_SECURITY_TOKEN`, `ALIYUN_CMS_CLI_REGION`, `ALIYUN_CMS_CLI_ENDPOINT`
  4. CLI flags: `--access-key-id`, `--access-key-secret`, `--security-token`, `--region`, `--endpoint`
- **发行物 OSS 路径**: `https://o11y-addon-hangzhou-public.oss-cn-hangzhou.aliyuncs.com/share/aliyuncms/`
  - 版本目录: `<version>/aliyuncms-<os>-<arch>[.exe]`
  - Latest 目录: `latest/aliyuncms-<os>-<arch>[.exe]`
- **支持平台**: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64, windows/arm64

### 1.3 凭证传递方式选型

规范定义两种凭证传递形态：

| 方式 | 代表实现 | 描述 |
|------|---------|------|
| 5.1 环境变量封装 JSON | ossutil | Profile 序列化为 JSON → Base64 → 写入单一环境变量 |
| 5.2 本地配置文件 | otsutil | Profile 中 AK/STS 写入本地配置文件，下游读取 |

**cms2 选型：独立环境变量（5.1 变体）**

理由：
1. cms2 已原生支持 `ALIYUN_CMS_CLI_*` 环境变量，优先级高于 aliyun-cli 配置文件回退，无需新增解析逻辑
2. 不需要额外的 JSON 反序列化/Base64 约定，下游零改动即可接入
3. 环境变量方式不持久化凭证到磁盘，比配置文件方式更安全
4. cms2 当前仅需 AK/SK/STS Token/Region/Endpoint 五个字段，不涉及 RamRoleArn 等复杂模式的直传（由 Wrapper 层完成凭证刷新后传递最终 AK/SK/Token）

---

## 2. 总体架构

```
用户: aliyun cms2 integration-policy list --region cn-hangzhou
         │
         ▼
┌──────────────────────────────────────────────────────────────┐
│  aliyun-cli (rootCmd)                                        │
│    └── cms2 子命令 (cliext/cms2 包)                           │
│         ├── 1. InitBasicInfo: 配置目录、平台检测              │
│         ├── 2. EnsureInstalledAndUpdated: 下载/版本管理       │
│         ├── 3. PrepareEnv: Profile → ALIYUN_CMS_CLI_* 环境变量│
│         ├── 4. RemoveFlagsForMainCli: 剥离 config 类 flags   │
│         └── 5. exec aliyuncms <child-args>                    │
└──────────────────────────────────────────────────────────────┘
         │
         ▼
┌──────────────────────────────────────────────────────────────┐
│  aliyuncms (独立二进制)                                       │
│    凭证从 ALIYUN_CMS_CLI_* 环境变量获取（优先级高于配置文件）  │
│    正常执行 cobra 命令树                                       │
└──────────────────────────────────────────────────────────────┘
```

---

## 3. 详细设计

### 3.1 代码目录结构

```
aliyun-cli/
├── cliext/
│   └── cms2/
│       ├── main.go          # NewCms2Command() 命令定义
│       ├── cms2.go          # Context 结构、Run 核心逻辑
│       ├── cms2_test.go     # 单元测试
│       └── main_test.go     # 命令定义测试
├── main/
│   └── main.go             # +import + rootCmd.AddSubCommand(cms2.NewCms2Command())
└── go.mod                  # 无新外部依赖
```

### 3.2 子命令注册 (`cliext/cms2/main.go`)

按规范第 3 节要求：

```go
package cms2

import (
    "github.com/aliyun/aliyun-cli/v3/cli"
    "github.com/aliyun/aliyun-cli/v3/i18n"
)

func NewCms2Command() *cli.Command {
    return &cli.Command{
        Name: "cms2",
        Short: i18n.T(
            "Alibaba Cloud CloudMonitor (CMS) CLI — manage monitoring integrations, Prometheus, alert rules, and PromQL.",
            "阿里云云监控 CLI — 管理监控集成、Prometheus 实例、告警规则和 PromQL 查询。"),
        Usage:             "aliyun cms2 <command> [args...] [options...]",
        Hidden:            false,
        EnableUnknownFlag: true,   // 允许未知 flag 透传给下游
        KeepArgs:          true,   // 保留原始参数供下游使用
        SkipDefaultHelp:   true,   // 不使用主 CLI 默认 help/version
        Run: func(ctx *cli.Context, args []string) error {
            // Help 规范化: 若主 CLI 检测到 help 请求，确保 args 中包含 help 标记
            // 以便下游 cobra 能正确处理
            if ctx.IsHelp() {
                hasHelp := false
                for i, arg := range args {
                    if arg == "help" {
                        hasHelp = true
                        break
                    } else if arg == "--help" {
                        args[i] = "help"
                        hasHelp = true
                        break
                    }
                }
                if !hasHelp {
                    args = append(args, "help")
                }
            }
            c := NewContext(ctx)
            return c.Run(args)
        },
    }
}
```

### 3.3 执行链路 (`cliext/cms2/cms2.go`)

按规范第 4 节推荐顺序实现：

#### 3.3.1 Context 结构

```go
type Context struct {
    originCtx                 *cli.Context
    configPath                string // aliyun 配置目录 (~/.aliyun/)
    checkVersionCacheFilePath string // 版本检查缓存: Unix 时间戳
    versionFilePath           string // 本地版本号文件
    execFilePath              string // aliyuncms 二进制路径
    installed                 bool
    versionLocal              string
    versionRemote             string
    osType                    string
    osArch                    string
    osSupport                 bool
    downloadPathSuffix        string // e.g. "darwin-arm64"
    envMap                    map[string]string // 传递给子进程的环境变量
}
```

#### 3.3.2 Run 主流程

```go
func (c *Context) Run(args []string) error {
    // Step 1: 初始化 — 配置目录、OS/Arch、二进制路径
    c.InitBasicInfo()
    c.CheckOsTypeAndArch()
    if !c.osSupport {
        return fmt.Errorf("your os type %s and arch %s is not supported now", c.osType, c.osArch)
    }

    // Step 2: 下载与版本管理（version.txt 不可用时降级为跳过）
    if err := c.EnsureInstalledAndUpdated(); err != nil {
        // 如果已安装则忽略版本检查/更新错误，继续使用本地版本
        if !c.installed {
            return err
        }
        // 已安装但更新失败，打印警告，不阻断执行
        _, _ = fmt.Fprintf(c.originCtx.Stderr(),
            "Warning: failed to check for cms2 updates: %v\n", err)
    }

    // 最终确认二进制可用
    if !c.installed {
        return fmt.Errorf("cms2 binary not found at %s, please install manually or set ALIYUN_CMS2_EXEC_PATH", c.execFilePath)
    }

    // Step 3: PrepareEnv — 从 Profile 加载凭证，转为 ALIYUN_CMS_CLI_* 环境变量
    if err := c.PrepareEnv(); err != nil {
        return err
    }

    // Step 4: RemoveFlagsForMainCli — 剥离 config 类 flags
    childArgs, err := c.RemoveFlagsForMainCli(args)
    if err != nil {
        return err
    }

    // Step 5: exec 子进程
    return c.Execute(childArgs)
}
```

**EnsureInstalledAndUpdated 降级策略**：

```go
func (c *Context) EnsureInstalledAndUpdated() error {
    if !c.installed {
        // 未安装: 尝试获取远程版本并下载
        latestVersion, err := getLatestCms2VersionFunc()
        if err != nil {
            // version.txt 不可用 → 无法自动安装，返回错误
            return fmt.Errorf("cms2 is not installed and auto-download failed: %v", err)
        }
        c.versionRemote = latestVersion
        return c.Install()
    }

    // 已安装: 按 TTL 检查是否需要更新
    if !c.NeedCheckVersion() {
        return nil
    }

    latestVersion, err := getLatestCms2VersionFunc()
    if err != nil {
        // version.txt 不可用 → 跳过更新，使用当前本地版本
        return nil
    }
    c.versionRemote = latestVersion

    if err := c.GetLocalVersion(); err != nil {
        return nil // 无法获取本地版本，跳过更新
    }
    if c.versionLocal != c.versionRemote {
        if err := c.Install(); err != nil {
            return nil // 下载失败，跳过更新，使用当前版本
        }
    }

    _ = c.UpdateCheckCacheTime()
    return nil
}
```

#### 3.3.3 PrepareEnv — 凭证与配置传递（规范第 5 节）

```go
func (c *Context) PrepareEnv() error {
    profile, err := config.LoadProfileWithContext(c.originCtx)
    if err != nil {
        return fmt.Errorf("config failed: %s", err.Error())
    }

    var accessKeyId, accessKeySecret, stsToken string

    switch profile.Mode {
    case config.AK:
        accessKeyId = profile.AccessKeyId
        accessKeySecret = profile.AccessKeySecret
    case config.StsToken:
        accessKeyId = profile.AccessKeyId
        accessKeySecret = profile.AccessKeySecret
        stsToken = profile.StsToken
    default:
        // RamRoleArn/EcsRamRole/OIDC/CredentialsURI 等动态模式
        // 必须通过 GetCredential() 完成 AssumeRole/OIDC Token Exchange 等操作，
        // 获取刷新后的最终临时 AK/SK/Token 传递给下游
        credential, err := profile.GetCredential(c.originCtx, nil)
        if err != nil {
            return fmt.Errorf("can't get credential: %s", err)
        }
        model, err := credential.GetCredential()
        if err != nil {
            return fmt.Errorf("can't get credential: %s", err)
        }
        accessKeyId = *model.AccessKeyId
        accessKeySecret = *model.AccessKeySecret
        if model.SecurityToken != nil {
            stsToken = *model.SecurityToken
        }
    }

    if accessKeyId == "" || accessKeySecret == "" {
        return fmt.Errorf("access key id or access key secret is empty, please run `aliyun configure` first")
    }

    // 构建 cms2 专用环境变量
    c.envMap = map[string]string{
        "ALIYUN_CMS_CLI_ACCESS_KEY_ID":     accessKeyId,
        "ALIYUN_CMS_CLI_ACCESS_KEY_SECRET": accessKeySecret,
    }
    if stsToken != "" {
        c.envMap["ALIYUN_CMS_CLI_SECURITY_TOKEN"] = stsToken
    }

    // Region: 优先从 argv 中用户显式传入的 --region 获取（保留在 argv 传给下游），
    // 仅当 argv 中无 --region 时从 profile 补充到环境变量作为 fallback
    if region := extractFlagValue(c.originCtx, "region"); region != "" {
        c.envMap["ALIYUN_CMS_CLI_REGION"] = region
    } else if profile.RegionId != "" {
        c.envMap["ALIYUN_CMS_CLI_REGION"] = profile.RegionId
    }

    // Endpoint: 从 ctx flags 提取（config 类 flag 会被 RemoveFlagsForMainCli 剥离，
    // 需要在此桥接到环境变量，否则下游无法获取）
    if endpoint := extractFlagValue(c.originCtx, "endpoint"); endpoint != "" {
        c.envMap["ALIYUN_CMS_CLI_ENDPOINT"] = endpoint
    }

    return nil
}
```

#### 3.3.4 RemoveFlagsForMainCli — 参数剥离（规范第 6 节）

> 原则：仅移除主 CLI 已解析且 category 为 "config" 的 flag（如 --config-path、--profile 等），避免传给下游报错。

```go
func (c *Context) RemoveFlagsForMainCli(args []string) ([]string, error) {
    if c.originCtx.Flags() == nil || c.originCtx.Flags().Flags() == nil {
        return append([]string(nil), args...), nil
    }

    longNeedsValue := make(map[string]bool)
    shortNeedsValue := make(map[string]bool)
    for _, f := range c.originCtx.Flags().Flags() {
        if !f.IsAssigned() || f.Category != "config" {
            continue
        }
        needsValue := f.AssignedMode != cli.AssignedNone
        if f.Name != "" {
            longNeedsValue["--"+f.Name] = needsValue
        }
        if f.Shorthand != 0 {
            shortNeedsValue["-"+string(f.Shorthand)] = needsValue
        }
    }

    out := make([]string, 0, len(args))
    for i := 0; i < len(args); i++ {
        a := args[i]
        if needs, ok := longNeedsValue[a]; ok {
            if needs && i+1 < len(args) {
                i++ // skip value
            }
            continue
        }
        if needs, ok := shortNeedsValue[a]; ok {
            if needs && i+1 < len(args) {
                i++ // skip value
            }
            continue
        }
        out = append(out, a)
    }
    return out, nil
}
```

#### 3.3.5 exec 子进程

```go
func (c *Context) Execute(childArgs []string) error {
    cmd := execCommandFunc(c.execFilePath, childArgs...)

    // 合并完整 os.Environ() 与包装层追加的环境变量
    // 先过滤掉与 envMap key 冲突的已有变量，再追加新值，确保不产生重复
    envs := filterEnv(os.Environ(), c.envMap)
    for k, v := range c.envMap {
        envs = append(envs, k+"="+v)
    }
    cmd.Env = envs
    cmd.Stdout = c.originCtx.Stdout()
    cmd.Stderr = c.originCtx.Stderr()
    cmd.Stdin = os.Stdin // 关键: 透传 stdin

    if err := cmd.Run(); err != nil {
        return fmt.Errorf("failed to execute %s %v: %v", c.execFilePath, childArgs, err)
    }
    return nil
}

// filterEnv 从 base 中移除与 override keys 同名的条目
func filterEnv(base []string, overrides map[string]string) []string {
    result := make([]string, 0, len(base))
    for _, item := range base {
        key, _, _ := strings.Cut(item, "=")
        if _, conflict := overrides[key]; conflict {
            continue
        }
        result = append(result, item)
    }
    return result
}
```

**Exit code 处理**：不在 Execute 中直接调用 `os.Exit()`（会跳过 defer 清理），而是返回 error。在最顶层由 aliyun-cli 框架处理退出。如需精确传递子进程 exit code，定义自定义 error 类型：

```go
type ExitError struct {
    Code int
}

func (e *ExitError) Error() string {
    return fmt.Sprintf("subprocess exited with code %d", e.Code)
}

// 在 Execute 中:
if err := cmd.Run(); err != nil {
    if exitErr, ok := err.(*exec.ExitError); ok {
        return &ExitError{Code: exitErr.ExitCode()}
    }
    return fmt.Errorf("failed to execute %s %v: %v", c.execFilePath, childArgs, err)
}

// 在 Run (cli.Command) 入口处，aliyun-cli 框架拿到 error 后可检查类型决定 exit code
```

### 3.4 二进制分发与版本（规范第 7 节）

| 配置项 | 值 |
|-------|------|
| 下载基地址 | `https://o11y-addon-hangzhou-public.oss-cn-hangzhou.aliyuncs.com/share/aliyuncms/` |
| 版本文件 URL | `{baseURL}latest/version.txt`（纯文本版本号，如 `v1.2.0`） |
| 包命名 | `aliyuncms-{os}-{arch}[.exe]`（裸二进制，非 zip） |
| 本地安装路径 | `~/.aliyun/aliyuncms` (或 `aliyuncms.exe`) |
| 版本检查缓存 | `~/.aliyun/.cms2_version_check` (Unix 时间戳) |
| 本地版本记录 | `~/.aliyun/.cms2_version` |
| 检查 TTL | 86400 秒 (1天) |
| 支持平台矩阵 | linux-amd64, linux-arm64, darwin-amd64, darwin-arm64, windows-amd64, windows-arm64 |

**版本获取**:

```go
func GetLatestCms2Version() (string, error) {
    url := downloadBaseURL + "latest/version.txt"
    // HTTP GET → 读取纯文本版本号 → strings.TrimSpace
}
```

**下载安装**（裸二进制，无需解压，原子替换）:

```go
func (c *Context) Install() error {
    url := fmt.Sprintf("%s%s/aliyuncms-%s", downloadBaseURL, c.versionRemote, c.downloadPathSuffix)

    // 1. 下载到与目标同目录的临时文件（确保同一文件系统，支持原子 rename）
    tmpFile := c.execFilePath + ".tmp"
    if err := downloadFile(url, tmpFile); err != nil {
        _ = os.Remove(tmpFile) // 清理残留
        return fmt.Errorf("failed to download cms2 from %s: %v", url, err)
    }

    // 2. chmod 0755 (非 Windows)
    if runtimeGOOSFunc() != "windows" {
        if err := os.Chmod(tmpFile, 0755); err != nil {
            _ = os.Remove(tmpFile)
            return fmt.Errorf("failed to set exec permission: %v", err)
        }
    }

    // 3. 原子替换: os.Rename 在同一文件系统下是原子操作
    //    如果目标已存在则先移除（Windows 不支持 rename 覆盖）
    if runtimeGOOSFunc() == "windows" && fileExists(c.execFilePath) {
        _ = os.Remove(c.execFilePath)
    }
    if err := os.Rename(tmpFile, c.execFilePath); err != nil {
        // 回退到 copy 策略（跨文件系统场景）
        if copyErr := util.CopyFileAndRemoveSource(tmpFile, c.execFilePath); copyErr != nil {
            return fmt.Errorf("failed to install cms2 binary: %v", copyErr)
        }
    }

    // 4. 记录版本号
    c.versionLocal = c.versionRemote
    return c.SaveLocalVersion()
}
```

与 ossutil/otsutil 的关键差异：**cms2 发行物是裸二进制**（不是 zip），下载后直接原子 rename 即可，不需要解压步骤。下载中断时临时文件会被清理，不会留下损坏的二进制。

### 3.5 主入口注册 (`main/main.go`)

```go
import "github.com/aliyun/aliyun-cli/v3/cliext/cms2"

// 在 newRootCommand 函数中添加:
rootCmd.AddSubCommand(cms2.NewCms2Command())
```

---

## 4. 规范合规性对照

### 4.1 对接 Checklist（规范第 10 节）

| # | 检查项 | 状态 | 说明 |
|---|--------|------|------|
| 1 | 子命令名、Usage、help 行为与用户文档一致，不与现有插件名冲突 | ✅ | `cms2` 不与现有 `oss`/`ossutil`/`otsutil`/`agentbay` 等冲突；help 已规范化 |
| 2 | 凭证与 Region 等与 aliyun configure Profile 对齐 | ✅ | 通过 `config.LoadProfileWithContext` 加载，支持所有凭证模式（含 RamRoleArn 动态刷新） |
| 3 | 约定 env / 配置文件格式并完成联调 | ✅ | 使用 cms2 已有的 `ALIYUN_CMS_CLI_*` 环境变量协议；endpoint 正确桥接 |
| 4 | RemoveFlagsForMainCli 覆盖所有需剥离的主 CLI 参数 | ✅ | 仅移除 category="config" 的 assigned flags |
| 5 | 验证管道 stdin 透传 | ✅ | `cmd.Stdin = os.Stdin` |
| 6 | --user-agent 保留在 argv | ✅ | 不在 RemoveFlagsForMainCli 中剥离 |
| 7 | Safety Policy 文件路径 | 🔲 | 待确认 cms2 侧是否需要支持 `ALIBABA_CLOUD_CLI_SAFETY_POLICY_FILE` |
| 8 | 发布地址、version.txt、多平台包可用 | ✅ | OSS 路径已约定，version.txt 由 cms2 发布流程提供 |
| 9 | 主仓库 main/main.go 注册子命令 | ✅ | `rootCmd.AddSubCommand(cms2.NewCms2Command())` |

### 4.2 与规范要求的细节对照

| 规范条目 | 实现方式 |
|---------|---------|
| §3 EnableUnknownFlag: true | ✅ |
| §3 KeepArgs: true | ✅ |
| §3 SkipDefaultHelp: true | ✅ |
| §4 初始化 | `InitBasicInfo()` + `CheckOsTypeAndArch()` |
| §4 下载与版本管理 | `EnsureInstalledAndUpdated()` |
| §4 PrepareEnv | 从 Profile 转为 `ALIYUN_CMS_CLI_*` 环境变量 |
| §4 RemoveFlagsForMainCli | 仅剥离 category="config" 的已赋值 flags |
| §4 exec | 透传 stdout/stderr/stdin + 合并 os.Environ() |
| §5 凭证传递 | 独立环境变量方式（5.1 变体） |
| §5.3 根级 --user-agent | 保留在 argv，由下游解析 |
| §7 version.txt | 纯文本版本号 |
| §7 落地目录 | `~/.aliyun/aliyuncms` |

---

## 5. 测试计划

### 5.1 单元测试 (`cms2_test.go`)

| 测试项 | 内容 |
|-------|------|
| `TestInitBasicInfo` | 验证路径拼接、Windows 后缀处理 |
| `TestCheckOsTypeAndArch` | 各平台支持矩阵（6 种有效 + N 种无效） |
| `TestNeedCheckVersion` | TTL 过期/未过期/缓存文件不存在/格式错误 |
| `TestPrepareEnv_AK` | AK 模式的环境变量生成 |
| `TestPrepareEnv_StsToken` | STS 模式包含 SecurityToken |
| `TestPrepareEnv_RamRoleArn` | RamRoleArn 走 GetCredential 动态获取临时凭证 |
| `TestPrepareEnv_DynamicCredential` | EcsRamRole/OIDC/CredentialsURI 等动态模式 |
| `TestPrepareEnv_Empty` | 凭证为空时的错误提示 |
| `TestPrepareEnv_EndpointBridge` | --endpoint flag 正确桥接到 ALIYUN_CMS_CLI_ENDPOINT |
| `TestPrepareEnv_RegionPriority` | argv --region 优先于 profile.RegionId |
| `TestRemoveFlagsForMainCli` | --profile/--config-path 等 config flags 被正确剥离 |
| `TestRemoveFlagsForMainCli_PreserveDownstream` | --output 等下游 flags 不被误删 |
| `TestFilterEnv` | 同名环境变量被正确过滤，无冲突的保留 |
| `TestExecute_ExitCode` | 子进程非零退出码通过 ExitError 类型返回 |
| `TestExecute_StdinPassthrough` | stdin 透传验证 |
| `TestExecute_EnvNoConflict` | 已有 ALIYUN_CMS_CLI_* 不会重复出现 |
| `TestRun_NotInstalled` | mock 下载流程验证 |
| `TestRun_Installed_NoUpdate` | 已安装且未过期时直接执行 |
| `TestGetLatestVersion_NetworkError` | HTTP 请求失败时返回明确错误 |
| `TestGetLatestVersion_Non200` | 非 200 状态码时返回明确错误 |
| `TestDownload_AtomicReplace` | 下载中断不留下损坏二进制 |
| `TestHelpNormalization` | --help 被正确转换为下游 help 子命令 |

### 5.2 命令定义测试 (`main_test.go`)

| 测试项 | 内容 |
|-------|------|
| `TestNewCms2Command_Metadata` | Name/Usage/Short 属性验证 |
| `TestNewCms2Command_Flags` | EnableUnknownFlag, KeepArgs, SkipDefaultHelp |

### 5.3 测试 Seams（可替换函数变量）

```go
var (
    getLatestCms2VersionFunc = GetLatestCms2Version
    downloadFileFunc         = downloadFile
    execCommandFunc          = exec.Command
    httpGetFunc              = http.Get
    timeNowFunc              = time.Now
    runtimeGOOSFunc          = func() string { return runtime.GOOS }
    runtimeGOARCHFunc        = func() string { return runtime.GOARCH }
    getConfigurePathFunc     = func() string { return config.GetConfigPath() }
)
```

---

## 6. 实现步骤

### Phase 1: 基础骨架 (MVP)
1. 创建 `cliext/cms2/main.go` — 命令定义
2. 创建 `cliext/cms2/cms2.go` — Context、Run、PrepareEnv、RemoveFlagsForMainCli、Execute
3. 修改 `main/main.go` — import + 注册 cms2 子命令
4. 编写基础单元测试

### Phase 2: 自动下载与版本管理
5. 实现 `GetLatestCms2Version()` — 从 OSS 获取 version.txt
6. 实现 `DownloadAndInstall()` — 下载裸二进制并安装
7. 实现 `EnsureInstalledAndUpdated()` — 含 TTL 版本检查缓存
8. 编写下载/安装相关测试
9. **[cms2 侧]** 发布流程增加 `latest/version.txt` 上传

### Phase 3: 完善
10. 开发调试模式支持 (`ALIYUN_CMS2_EXEC_PATH` 环境变量覆盖)
11. Safety Policy 支持（待与 CLI 团队确认）
12. 补充集成测试

---

## 7. 注意事项

### 7.1 cms2 已有 aliyun-cli 配置回退

cms2 的 `auth.Resolve()` 已有从 `~/.aliyun/config.json` 读取凭证的回退逻辑。通过环境变量桥接仍然必要，原因：
1. 确保 `--profile` flag 选择的 profile 生效（而非 cms2 自己读 default profile）
2. 支持 RamRoleArn/EcsRamRole/OIDC 等需要动态刷新的凭证模式（Wrapper 完成 AssumeRole/Token Exchange 后传递最终 AK/SK/Token）
3. 避免凭证来源不一致导致的用户困惑

### 7.2 Region 和 Endpoint 的优先级语义

Region 存在两个来源：
- **argv** 中的 `--region`（保留在 childArgs，由 cms2 下游 flags 解析，优先级最高）
- **环境变量** `ALIYUN_CMS_CLI_REGION`（由 Wrapper 从 profile 或 ctx flags 设置，作为 fallback）

由于 cms2 下游的 flags 优先级高于 env，两者同时存在时 argv 中的值生效。Wrapper 层在 PrepareEnv 中设置环境变量时已确保：如果 ctx flags 中有 --region 值，优先使用该值设置环境变量（保持一致性）。

Endpoint 的处理：`--endpoint` 属于 config 类 flag，会被 RemoveFlagsForMainCli 从 argv 中剥离，因此必须在 PrepareEnv 中桥接到 `ALIYUN_CMS_CLI_ENDPOINT` 环境变量，否则下游无法获取。

### 7.3 版本文件要求（规范第 7 节协同项）

cms2 的 OSS 发布流程（Makefile `upload` target）需要新增一步：
- 上传 `version.txt` 到 `latest/` 目录，内容为当前版本号纯文本（如 `v1.2.0`）

**降级策略**：如果 `version.txt` 不存在或网络异常：
- **已安装时**：跳过更新，使用本地已有版本继续执行（仅打印 warning）
- **未安装时**：报错提示用户手动安装或通过 `ALIYUN_CMS2_EXEC_PATH` 指定路径

### 7.4 开发调试模式

```go
if envPath := os.Getenv("ALIYUN_CMS2_EXEC_PATH"); envPath != "" {
    c.execFilePath = envPath
    c.installed = fileExists(envPath)
    // 跳过自动下载逻辑
}
```

### 7.5 与其他 Wrapper 的差异

| 维度 | ossutil | otsutil | cms2 |
|------|---------|---------|------|
| 凭证传递 | JSON+Base64 环境变量 | 本地配置文件 | 独立环境变量 |
| 兼容模式标识 | `OSSUTIL_COMPAT_MODE=alicli` | N/A | N/A（cms2 本身已适配） |
| 发行格式 | `.zip` (含子目录) | `.zip` (扁平) | 裸二进制 |
| 版本发现 | `version.txt` (含前缀解析) | `version.txt` (纯文本) | `version.txt` (纯文本) |
| 本地二进制名 | `ossutil` | `ts` | `aliyuncms` |
| AI Mode | 支持 | 不支持 | 暂不支持（后续可扩展） |

---

## 8. 依赖

- 无新外部依赖，仅使用 Go 标准库 + aliyun-cli 已有的 `cli`, `config`, `util`, `i18n` 包
- 不引入 cms2 源码仓库作为依赖

---

## 9. 命令使用示例

```bash
# 查看版本
aliyun cms2 version

# 使用当前 profile 的凭证
aliyun cms2 integration-policy list

# 指定 profile
aliyun --profile test cms2 rule list

# 指定 region（保留在 argv 传递给下游）
aliyun cms2 promql query --expr 'up' --region cn-hangzhou

# configure mode（动态凭证直传）
aliyun --access-key-id xxx --access-key-secret yyy --region cn-hangzhou cms2 prometheus-instance list

# 管道 stdin 透传
echo '{"name":"test"}' | aliyun cms2 integration-policy create --body @-

# 开发调试（跳过远程下载）
ALIYUN_CMS2_EXEC_PATH=/path/to/local/aliyuncms aliyun cms2 version
```

---

## 10. 联系与协同

按规范第 11 节，接口约定、下载域名与代码评审需与以下同学沟通：
- Aliyun CLI 维护: @朱明明(原根) @王栗(木西)
- 内部 CLI 支持与答疑钉钉群: 174090001814
- 产品相关疑问: @李港晨(礼川)
