# 配置与凭证

[文档索引](../README.md) | [English](../en/configuration.md)

阿里云 CLI 默认将具名 Profile 保存在 `~/.aliyun/config.json`。

## 创建和管理 Profile

本地交互使用推荐从 OAuth 开始：

```sh
aliyun configure --mode OAuth --profile default
```

需要交互式创建基础 AK Profile 时，可以运行 `aliyun configure`。为新 Profile 配置且没有指定模式时，CLI 使用 AK 模式：

```text
$ aliyun configure --profile default
Configuring profile 'default' in 'AK' authenticate mode...
Access Key Id []: <AccessKeyId>
Access Key Secret []: <AccessKeySecret>
Default Region Id []: cn-hangzhou
Default Output Format [json]: json (Only support json)
Default Language [zh|en] zh:
Saving profile[default] ...Done.
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

交互式配置示例：

```text
$ aliyun configure --mode StsToken --profile StsProfile
Configuring profile 'StsProfile' in 'StsToken' authenticate mode...
Access Key Id []: STS.NUr5xxxxx
Access Key Secret []: 7Bshxxxxx
Sts Token []: CAISxxxxxxxxxxxxxxxx...
Default Region Id []: cn-hangzhou
Default Output Format [json]: json (Only support json)
Default Language [zh|en] zh:
Saving profile[StsProfile] ...Done.
```

非交互式配置示例：

```sh
aliyun configure set \
  --profile StsProfile \
  --mode StsToken \
  --access-key-id STS.NUr5xxxxx \
  --access-key-secret 7Bshxxxxx \
  --sts-token CAISxxxxxxxxxxxxxxxx... \
  --region cn-hangzhou
```

该模式不会自动续期。凭证过期后需要重新配置，或者改用可续期的 `RamRoleArn`、`EcsRamRole`、`OIDC` 等身份模式。

也可以通过环境变量补全 Profile 中缺失的凭证字段：

```sh
export ALIBABA_CLOUD_ACCESS_KEY_ID="STS.xxx"
export ALIBABA_CLOUD_ACCESS_KEY_SECRET="temporary-secret"
export ALIBABA_CLOUD_SECURITY_TOKEN="security-token"
```

更多说明见：[配置 StsToken 临时凭证](https://help.aliyun.com/zh/cli/temporary-security-credentials-sts-token)。

## 角色与工作负载凭证

### RAM 角色扮演

`RamRoleArn` 使用基于 AccessKey 的身份调用 AssumeRole，换取临时凭证：

```text
$ aliyun configure --mode RamRoleArn --profile subaccount
Configuring profile 'subaccount' in 'RamRoleArn' authenticate mode...
Access Key Id []: <AccessKeyId>
Access Key Secret []: <AccessKeySecret>
Sts Region []: cn-hangzhou
Ram Role Arn []: acs:ram::<账号ID>:role/<角色名>
Role Session Name []: aliyun-cli
Expired Seconds []: 900
Default Region Id []: cn-hangzhou
Default Output Format [json]: json (Only support json)
Default Language [zh|en] zh:
Saving profile[subaccount] ...Done.
```

### ECS 实例角色

`EcsRamRole` 从当前 ECS 实例绑定的 RAM 角色获取凭证：

```sh
aliyun configure --mode EcsRamRole --profile ecs-role
```

### 链式 RAM 角色

`ChainableRamRoleArn` 使用已有 Profile 作为源身份，再扮演配置的 RAM 角色：

```sh
aliyun configure set \
  --profile chained-role \
  --mode ChainableRamRoleArn \
  --source-profile source-profile \
  --ram-role-arn 'acs:ram::<账号ID>:role/<角色名>' \
  --role-session-name aliyun-cli
```

对应的 Profile 关系如下：

```json
{
  "profiles": [
    {
      "name": "chained-role",
      "mode": "ChainableRamRoleArn",
      "ram_role_arn": "acs:ram::<账号ID>:role/<角色名>",
      "ram_session_name": "aliyun-cli",
      "source_profile": "source-profile"
    },
    {
      "name": "source-profile",
      "mode": "AK",
      "access_key_id": "<AccessKeyId>",
      "access_key_secret": "<AccessKeySecret>"
    }
  ]
}
```

### OIDC 工作负载身份

在支持 OIDC 的 CI 或工作负载环境中，需要配置 Provider ARN、Token 文件、RAM 角色 ARN 和会话名称：

```text
$ aliyun configure --mode OIDC --profile oidc-profile
Configuring profile 'oidc-profile' in 'OIDC' authenticate mode...
OIDC Provider ARN []: acs:ram::<账号ID>:oidc-provider/<Provider名称>
OIDC Token File []: /path/to/oidc-token
RAM Role ARN []: acs:ram::<账号ID>:role/<角色名>
Role Session Name []: aliyun-cli
Default Region Id []: cn-hangzhou
Default Output Format [json]: json (Only support json)
Default Language [zh|en] zh:
Saving profile[oidc-profile] ...Done.
```

## 交互式身份登录

### OAuth

OAuth 会打开基于浏览器的登录流程，并将获得的临时凭证保存到 Profile：

```text
$ aliyun configure --mode OAuth --profile oauth-profile
Configuring profile 'oauth-profile' in 'OAuth' authenticate mode...
OAuth Site Type (CN: 0 or INTL: 1, default: CN): 0
请在浏览器中打开命令显示的地址并完成授权。
OAuth configuration completed.
Default Region Id []: cn-hangzhou
Saving profile[oauth-profile] ...Done.
```

### CloudSSO

CloudSSO 会通过配置的 CloudSSO 门户启动交互式登录流程：

```text
$ aliyun configure --mode CloudSSO --profile cloud-sso
Configuring profile 'cloud-sso' in 'CloudSSO' authenticate mode...
CloudSSO Sign In Url []: https://signin-cn-shanghai.alibabacloudsso.com/start/login
按照命令提示完成登录，并选择账号和访问配置。
Default Region Id []: cn-hangzhou
Saving profile[cloud-sso] ...Done.
```

## 外部凭证源

### 外部程序

`External` 会运行配置的本地命令，并将其输出作为凭证。外部程序需要遵循以下约定：

1. 将凭证响应写入标准输出。
2. 标准输出只能包含一个有效的 JSON 对象；诊断信息应写入标准错误。
3. 返回 `mode` 以及该模式要求的全部凭证字段。目前支持返回 `AK` 和 `StsToken` 两种模式。

AK 返回结构：

```json
{
  "mode": "AK",
  "access_key_id": "<AccessKeyId>",
  "access_key_secret": "<AccessKeySecret>"
}
```

StsToken 返回结构：

```json
{
  "mode": "StsToken",
  "access_key_id": "<AccessKeyId>",
  "access_key_secret": "<AccessKeySecret>",
  "sts_token": "<SecurityToken>"
}
```

交互式配置：

```text
$ aliyun configure --mode External --profile external
Configuring profile 'external' in 'External' authenticate mode...
Process Command []: <credential-command>
Default Region Id []: cn-hangzhou
Default Output Format [json]: json (Only support json)
Default Language [zh|en] zh:
Saving profile[external] ...Done.
```

### Credentials URI

`CredentialsURI` 从本地或远程 HTTP 地址获取临时凭证。交互式配置示例：

```text
$ aliyun configure --profile uri --mode CredentialsURI
Configuring profile 'uri' in 'CredentialsURI' authenticate mode...
Credentials URI []: http://127.0.0.1:6666/credentials
Default Region Id []: cn-hangzhou
Default Output Format [json]: json (Only support json)
Default Language [zh|en] zh:
Saving profile[uri] ...Done.
```

保存后的 Profile 等价于：

```json
{
  "profiles": [
    {
      "name": "uri",
      "mode": "CredentialsURI",
      "credentials_uri": "http://127.0.0.1:6666/credentials"
    }
  ]
}
```

也可以只为当前进程设置 `ALIBABA_CLOUD_CREDENTIALS_URI`，而不把 URI 保存到 Profile。

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

非 200 响应、响应格式错误或缺少凭证字段时，CLI 会将其视为凭证获取失败。

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
