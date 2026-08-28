# 配置与凭证

[文档索引](../README.md) | [English](../en/configuration.md)

阿里云 CLI 默认将具名 Profile 保存在 `~/.aliyun/config.json`。需要隔离配置时，可以使用 `--config-path <文件>`。

## 创建和管理 Profile

本地交互使用推荐从 OAuth 开始：

```sh
aliyun configure --mode OAuth --profile default
```

Profile 管理命令：

```sh
aliyun configure list
aliyun configure get --profile default
aliyun configure switch --profile default
aliyun configure delete --profile old-profile
```

在 API 命令中使用 `--profile <名称>`，或设置 `ALIBABA_CLOUD_PROFILE`，可以选择 Profile 而不修改默认值：

```sh
aliyun ecs DescribeRegions --profile production
export ALIBABA_CLOUD_PROFILE=production
```

## 凭证模式

| 模式 | 适用场景 |
| --- | --- |
| `OAuth` | 本地用户交互登录，避免保存长期 AccessKey |
| `CloudSSO` | 阿里云 CloudSSO 人员身份访问 |
| `AK` | 静态 AccessKey ID 和 Secret；仅在无法使用更安全方式时采用 |
| `StsToken` | 已签发的临时 AccessKey 和 SecurityToken；不会自动续期 |
| `RamRoleArn` | 使用 AccessKey 身份扮演 RAM 角色 |
| `ChainableRamRoleArn` | 使用另一个 CLI Profile 作为源身份扮演 RAM 角色 |
| `EcsRamRole` | 从 ECS 实例 RAM 角色获取凭证 |
| `OIDC` | 使用 OIDC Token 换取角色凭证，常用于工作负载或 CI |
| `External` | 执行本地程序并从 JSON 输出读取凭证 |
| `CredentialsURI` | 从 HTTP 地址获取凭证 |
| `BearerToken` | 为支持 Bearer 认证的 API 提供 Token |
| `Anonymous` | 不使用凭证调用，仅适用于允许匿名访问的 API |

各模式支持的字段可以通过 `aliyun configure --help` 查看。

## 非交互配置

下面的示例在无提示的情况下写入 AK Profile：

```sh
aliyun configure set \
  --profile ci \
  --mode AK \
  --access-key-id '<AccessKeyId>' \
  --access-key-secret '<AccessKeySecret>' \
  --region cn-hangzhou \
  --language zh
```

不要把真实密钥写进脚本、Shell 历史、源码仓库或 Issue。优先使用密钥管理系统注入、工作负载身份或临时凭证。

## STS 临时凭证

已经获得三个临时凭证字段时，可以使用 `StsToken`：

```sh
aliyun configure set \
  --profile sts \
  --mode StsToken \
  --access-key-id 'STS....' \
  --access-key-secret '<临时 Secret>' \
  --sts-token '<SecurityToken>' \
  --region cn-hangzhou
```

该模式不会自动续期。凭证过期后需要重新配置，或者改用可续期的 `RamRoleArn`、`EcsRamRole`、`OIDC` 等身份模式。

## 角色与工作负载凭证

交互式配置角色扮演：

```sh
aliyun configure --mode RamRoleArn --profile assumed-role
```

使用已有 Profile 进行链式角色扮演：

```sh
aliyun configure set \
  --profile chained-role \
  --mode ChainableRamRoleArn \
  --source-profile source-profile \
  --ram-role-arn 'acs:ram::<账号ID>:role/<角色名>' \
  --role-session-name aliyun-cli
```

在 ECS 上，`EcsRamRole` 会读取实例 RAM 角色。在支持 OIDC 的 CI 或工作负载环境中，需要配置 Provider ARN、Token 文件、角色 ARN 和会话名称：

```sh
aliyun configure --mode OIDC --profile workload
```

## 外部凭证源

### 外部程序

`External` 会运行配置的命令，并从标准输出读取一个 JSON 对象。外部程序的诊断信息应写入标准错误，不能混入标准输出。

```json
{
  "mode": "StsToken",
  "access_key_id": "STS....",
  "access_key_secret": "temporary-secret",
  "sts_token": "security-token"
}
```

交互式配置：

```sh
aliyun configure --mode External --profile external
```

### Credentials URI

交互式配置本地或远程地址：

```sh
aliyun configure --profile uri --mode CredentialsURI
# Credentials URI []: http://127.0.0.1:6666/credentials
```

也可以为当前进程设置 `ALIBABA_CLOUD_CREDENTIALS_URI`。

服务必须返回 HTTP 200，并提供类似以下结构的凭证：

```json
{
  "Code": "Success",
  "AccessKeyId": "STS....",
  "AccessKeySecret": "temporary-secret",
  "SecurityToken": "security-token",
  "Expiration": "2030-01-02T15:04:05Z"
}
```

在受限环境中设置 `ALIBABA_CLOUD_DISABLE_EXTERNAL_PROCESS=true`，可以同时禁止外部进程执行和 CredentialsURI 请求。

## 环境变量

环境变量可以选择 Profile，或补全命令参数/Profile 中未提供的字段。常用变量包括：

| 环境变量 | 用途 |
| --- | --- |
| `ALIBABA_CLOUD_PROFILE` | 选择 Profile |
| `ALIBABA_CLOUD_IGNORE_PROFILE=TRUE` | 忽略 Profile 文件 |
| `ALIBABA_CLOUD_PROFILE_MODE` | 选择 `Anonymous` 等模式 |
| `ALIBABA_CLOUD_ACCESS_KEY_ID` | AccessKey ID |
| `ALIBABA_CLOUD_ACCESS_KEY_SECRET` | AccessKey Secret |
| `ALIBABA_CLOUD_SECURITY_TOKEN` | STS SecurityToken |
| `ALIBABA_CLOUD_REGION_ID` | 默认地域 |
| `ALIBABA_CLOUD_STS_ENDPOINT` | 覆盖 STS Endpoint |
| `ALIBABA_CLOUD_ENDPOINT` | 覆盖产品 Endpoint |
| `ALIBABA_CLOUD_CREDENTIALS_URI` | CredentialsURI 地址 |
| `ALIBABA_CLOUD_OIDC_PROVIDER_ARN` | OIDC Provider ARN |
| `ALIBABA_CLOUD_OIDC_TOKEN_FILE` | OIDC Token 文件 |
| `ALIBABA_CLOUD_ROLE_ARN` | RAM 角色 ARN |
| `ALIBABA_CLOUD_BEARER_TOKEN` | Bearer Token |
| `ALIBABA_CLOUD_LANGUAGE` | `en` 或 `zh` 输出语言 |
| `DEBUG=sdk` | 打印 SDK HTTP 诊断信息；分享前请检查敏感内容 |

命令行参数对当前调用具有更高优先级。不要在共享机器上全局导出敏感凭证。

## 语言与自动补全

使用 `--language en` 或 `--language zh` 可以设置单次命令语言，也可以在 Profile 中保存默认语言。启用 Shell 自动补全：

```sh
aliyun auto-completion
```

使用 `aliyun auto-completion --uninstall` 删除已安装的自动补全。

下一步：[调用 API 和处理输出](./usage.md)。
