# cms2 — 阿里云云监控 CLI Wrapper

`cms2` 包将独立的[阿里云云监控 CLI](https://help.aliyun.com/product/28615.html)（`aliyuncms`）以内置子命令的方式集成到阿里云 CLI。用户可以通过 `aliyun cms2 <子命令>` 管理云监控各 APP 应用，包括工作空间、接入中心、应用监控、前端监控、告警中心、事件中心、云拨测、Prometheus 服务等，复用 `aliyun configure` 中配置的凭证和 Profile。

## 工作原理

```
aliyun cms2 integration-policy list --region cn-hangzhou
       │
       ├─ 1. 确保 aliyuncms 二进制已安装（~/.aliyun/aliyuncms）
       ├─ 2. 从当前 aliyun-cli Profile 加载凭证
       ├─ 3. 通过 ALIYUN_CMS_CLI_* 环境变量桥接凭证
       ├─ 4. 剥离 aliyun-cli 层面的 flags（--profile、--config-path 等）
       └─ 5. exec aliyuncms integration-policy list --region cn-hangzhou
```

本 Wrapper 遵循《独立可执行文件接入 Aliyun CLI（Wrapper）规范说明》，与 `ossutil`、`otsutil` 采用相同的集成模式。

## 使用方式

```bash
# 列出集成策略
aliyun cms2 integration-policy list

# 使用指定 profile
aliyun --profile prod cms2 rule list

# 指定 region
aliyun cms2 promql query --expr 'up' --region cn-hangzhou

# 直接传递凭证（configure mode）
aliyun --access-key-id AKID --access-key-secret SECRET \
       --region cn-hangzhou cms2 prometheus-instance list

# 管道输入
echo '{"name":"test"}' | aliyun cms2 integration-policy create --body @-

# 查看帮助
aliyun cms2 help
aliyun cms2 rule --help
```

## 凭证桥接

从 aliyun-cli Profile（`aliyun configure`）解析凭证后，通过环境变量传递给 `aliyuncms`：

| Profile 字段 | 环境变量 |
|-------------|---------|
| AccessKeyId | `ALIYUN_CMS_CLI_ACCESS_KEY_ID` |
| AccessKeySecret | `ALIYUN_CMS_CLI_ACCESS_KEY_SECRET` |
| StsToken / SecurityToken | `ALIYUN_CMS_CLI_SECURITY_TOKEN` |
| RegionId | `ALIYUN_CMS_CLI_REGION` |
| Endpoint | `ALIYUN_CMS_CLI_ENDPOINT` |

支持所有凭证模式：AK、StsToken、RamRoleArn、EcsRamRole、OIDC、CredentialsURI。对于动态凭证模式（RamRoleArn、OIDC 等），Wrapper 层会完成 AssumeRole / Token Exchange 后传递临时 AK/SK/Token。

## 自动下载与版本管理

首次使用时，Wrapper 自动从阿里云 CDN 下载 `aliyuncms` 二进制到 `~/.aliyun/aliyuncms`。每天检查一次更新（TTL = 86400 秒）。

支持平台：`linux/amd64`、`linux/arm64`、`darwin/amd64`、`darwin/arm64`、`windows/amd64`、`windows/arm64`。

降级策略（`version.txt` 不可用时）：
- **已安装**：跳过更新，打印警告，使用本地版本继续执行
- **未安装**：报错并提示安装方式

## 开发调试

通过环境变量 `ALIYUN_CMS2_EXEC_PATH` 指定本地构建产物，跳过自动下载：

```bash
ALIYUN_CMS2_EXEC_PATH=/path/to/aliyuncms aliyun cms2 version
```

运行测试：

```bash
go test ./cliext/cms2/ -v
```

## 文件结构

```
cliext/cms2/
├── main.go          # NewCms2Command() — 命令定义
├── cms2.go          # 核心逻辑：Run、PrepareEnv、RemoveFlagsForMainCli、Install、Execute
├── main_test.go     # 命令元数据测试
├── cms2_test.go     # 单元测试（28 个用例）
├── DESIGN.md        # 实现设计文档
├── README.md        # 英文文档
└── README-CN.md     # 本文件（中文文档）
```
