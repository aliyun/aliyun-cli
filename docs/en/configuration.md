# Configuration and credentials

[Documentation index](../README.md) | [简体中文](../zh-CN/configuration.md)

Alibaba Cloud CLI stores named profiles in `~/.aliyun/config.json` by default. Use `--config-path <file>` when an isolated configuration file is required.

## Create and manage profiles

For interactive local use, OAuth is the recommended starting point:

```sh
aliyun configure --mode OAuth --profile default
```

Profile management commands:

```sh
aliyun configure list
aliyun configure get --profile default
aliyun configure switch --profile default
aliyun configure delete --profile old-profile
```

Use `--profile <name>` on an API command, or set `ALIBABA_CLOUD_PROFILE`, to select a profile without changing the default:

```sh
aliyun ecs DescribeRegions --profile production
export ALIBABA_CLOUD_PROFILE=production
```

## Credential modes

| Mode | Intended use |
| --- | --- |
| `OAuth` | Interactive sign-in for a local user; avoids storing a long-lived AccessKey |
| `CloudSSO` | Alibaba Cloud CloudSSO workforce access |
| `AK` | Static AccessKey ID and secret; use only when a stronger option is unavailable |
| `StsToken` | An already-issued temporary AccessKey plus SecurityToken; does not refresh automatically |
| `RamRoleArn` | Assume a RAM role from an AccessKey-based identity |
| `ChainableRamRoleArn` | Assume a RAM role using another CLI profile as the source identity |
| `EcsRamRole` | Obtain credentials from an ECS instance RAM role |
| `OIDC` | Exchange an OIDC token for role credentials, commonly in workload/CI environments |
| `External` | Execute a local program that returns credentials as JSON |
| `CredentialsURI` | Fetch credentials from an HTTP endpoint |
| `BearerToken` | Use a bearer token for APIs that support bearer authentication |
| `Anonymous` | Call an API without credentials; only valid for APIs that allow anonymous access |

Run `aliyun configure --help` for the fields supported by each mode.

## Non-interactive configuration

The following example writes an AK profile without prompts:

```sh
aliyun configure set \
  --profile ci \
  --mode AK \
  --access-key-id '<AccessKeyId>' \
  --access-key-secret '<AccessKeySecret>' \
  --region cn-hangzhou \
  --language en
```

Do not put real secrets in scripts, shell history, source control, or issue reports. Prefer environment injection from a secret manager, a workload identity, or a temporary credential mode.

## Temporary STS credentials

Use `StsToken` when all three credential values have already been issued:

```sh
aliyun configure set \
  --profile sts \
  --mode StsToken \
  --access-key-id 'STS....' \
  --access-key-secret '<temporary-secret>' \
  --sts-token '<security-token>' \
  --region cn-hangzhou
```

This mode does not renew the token. Reconfigure after expiration, or use a renewable identity mode such as `RamRoleArn`, `EcsRamRole`, or `OIDC`.

## Role and workload credentials

Interactive role assumption:

```sh
aliyun configure --mode RamRoleArn --profile assumed-role
```

For a role chained from an existing profile:

```sh
aliyun configure set \
  --profile chained-role \
  --mode ChainableRamRoleArn \
  --source-profile source-profile \
  --ram-role-arn 'acs:ram::<account-id>:role/<role-name>' \
  --role-session-name aliyun-cli
```

On ECS, `EcsRamRole` reads the instance role. In OIDC-enabled CI or workload environments, configure the provider ARN, token file, role ARN, and session name:

```sh
aliyun configure --mode OIDC --profile workload
```

## External credential sources

### External process

`External` runs the configured command and reads one JSON object from standard output. The command must write diagnostics to standard error, not standard output.

```json
{
  "mode": "StsToken",
  "access_key_id": "STS....",
  "access_key_secret": "temporary-secret",
  "sts_token": "security-token"
}
```

Configure it interactively:

```sh
aliyun configure --mode External --profile external
```

### Credentials URI

Configure a local or remote endpoint interactively:

```sh
aliyun configure --profile uri --mode CredentialsURI
# Credentials URI []: http://127.0.0.1:6666/credentials
```

Alternatively, set `ALIBABA_CLOUD_CREDENTIALS_URI` for the current process.

The endpoint must return HTTP 200 with a credential response such as:

```json
{
  "Code": "Success",
  "AccessKeyId": "STS....",
  "AccessKeySecret": "temporary-secret",
  "SecurityToken": "security-token",
  "Expiration": "2030-01-02T15:04:05Z"
}
```

Set `ALIBABA_CLOUD_DISABLE_EXTERNAL_PROCESS=true` to block both external process execution and CredentialsURI fetching in a restricted environment.

## Environment variables

Environment variables can select a profile or fill fields not supplied by flags/profile configuration. Important variables include:

| Variable | Purpose |
| --- | --- |
| `ALIBABA_CLOUD_PROFILE` | Select a profile |
| `ALIBABA_CLOUD_IGNORE_PROFILE=TRUE` | Ignore profile files |
| `ALIBABA_CLOUD_PROFILE_MODE` | Select a mode such as `Anonymous` |
| `ALIBABA_CLOUD_ACCESS_KEY_ID` | AccessKey ID |
| `ALIBABA_CLOUD_ACCESS_KEY_SECRET` | AccessKey secret |
| `ALIBABA_CLOUD_SECURITY_TOKEN` | STS SecurityToken |
| `ALIBABA_CLOUD_REGION_ID` | Default region |
| `ALIBABA_CLOUD_STS_ENDPOINT` | Override the STS endpoint |
| `ALIBABA_CLOUD_ENDPOINT` | Override the product endpoint |
| `ALIBABA_CLOUD_CREDENTIALS_URI` | CredentialsURI source |
| `ALIBABA_CLOUD_OIDC_PROVIDER_ARN` | OIDC provider ARN |
| `ALIBABA_CLOUD_OIDC_TOKEN_FILE` | OIDC token file |
| `ALIBABA_CLOUD_ROLE_ARN` | RAM role ARN |
| `ALIBABA_CLOUD_BEARER_TOKEN` | Bearer token |
| `ALIBABA_CLOUD_LANGUAGE` | `en` or `zh` output language |
| `DEBUG=sdk` | Print SDK HTTP diagnostics; review output before sharing |

Command-line flags take precedence for that invocation. Avoid globally exporting secrets on shared machines.

## Language and shell completion

Use `--language en` or `--language zh` for one command, or save a language in the profile. Enable shell completion with:

```sh
aliyun auto-completion
```

Remove installed completion with `aliyun auto-completion --uninstall`.

Next: [call APIs and process output](./usage.md).
