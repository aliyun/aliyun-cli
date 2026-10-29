# Configuration and credentials

[Documentation index](../README.md) | [简体中文](../zh-CN/configuration.md)

Alibaba Cloud CLI stores named profiles in `~/.aliyun/config.json` by default.

For telemetry settings and the information collected, see [Telemetry data collection](./telemetry.md).

## Create and manage profiles

For interactive local use, OAuth is the recommended starting point:

```sh
aliyun configure --mode OAuth --profile default
```

To create a basic AK profile interactively, run `aliyun configure`. When no mode is specified for a new profile, the CLI uses AK mode:

```text
$ aliyun configure --profile default
Configuring profile 'default' in 'AK' authenticate mode...
Access Key Id []: <AccessKeyId>
Access Key Secret []: <AccessKeySecret>
Default Region Id []: cn-hangzhou
Default Output Format [json]: json (Only support json)
Default Language [zh|en] en:
Saving profile[default] ...Done.
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
| `CloudSSO` | CloudSSO workforce access with shared sessions across profiles |
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

Interactive example:

```text
$ aliyun configure --mode StsToken --profile StsProfile
Configuring profile 'StsProfile' in 'StsToken' authenticate mode...
Access Key Id []: STS.NUr5xxxxx
Access Key Secret []: 7Bshxxxxx
Sts Token []: CAISxxxxxxxxxxxxxxxx...
Default Region Id []: cn-hangzhou
Default Output Format [json]: json (Only support json)
Default Language [zh|en] en:
Saving profile[StsProfile] ...Done.
```

Non-interactive example:

```sh
aliyun configure set \
  --profile StsProfile \
  --mode StsToken \
  --access-key-id STS.NUr5xxxxx \
  --access-key-secret 7Bshxxxxx \
  --sts-token CAISxxxxxxxxxxxxxxxx... \
  --region cn-hangzhou
```

This mode does not renew the token. Reconfigure after expiration, or use a renewable identity mode such as `RamRoleArn`, `EcsRamRole`, or `OIDC`.

Missing credential fields can also be supplied through environment variables:

```sh
export ALIBABA_CLOUD_ACCESS_KEY_ID="STS.xxx"
export ALIBABA_CLOUD_ACCESS_KEY_SECRET="temporary-secret"
export ALIBABA_CLOUD_SECURITY_TOKEN="security-token"
```

See [Configure StsToken credentials](https://help.aliyun.com/en/cli/temporary-security-credentials-sts-token) for more information.

## Role and workload credentials

### RAM role assumption

`RamRoleArn` uses an AccessKey-based identity to call AssumeRole and obtain temporary credentials:

```text
$ aliyun configure --mode RamRoleArn --profile subaccount
Configuring profile 'subaccount' in 'RamRoleArn' authenticate mode...
Access Key Id []: <AccessKeyId>
Access Key Secret []: <AccessKeySecret>
Sts Region []: cn-hangzhou
Ram Role Arn []: acs:ram::<account-id>:role/<role-name>
Role Session Name []: aliyun-cli
Expired Seconds []: 900
Default Region Id []: cn-hangzhou
Default Output Format [json]: json (Only support json)
Default Language [zh|en] en:
Saving profile[subaccount] ...Done.
```

### ECS instance role

`EcsRamRole` obtains credentials from the RAM role attached to the current ECS instance:

```sh
aliyun configure --mode EcsRamRole --profile ecs-role
```

### Chainable RAM role

`ChainableRamRoleArn` uses an existing profile as its source identity, then assumes the configured RAM role:

```sh
aliyun configure set \
  --profile chained-role \
  --mode ChainableRamRoleArn \
  --source-profile source-profile \
  --ram-role-arn 'acs:ram::<account-id>:role/<role-name>' \
  --role-session-name aliyun-cli
```

The resulting profile relationship is equivalent to:

```json
{
  "profiles": [
    {
      "name": "chained-role",
      "mode": "ChainableRamRoleArn",
      "ram_role_arn": "acs:ram::<account-id>:role/<role-name>",
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

### OIDC workload identity

In an OIDC-enabled CI or workload environment, configure the provider ARN, token file, RAM role ARN, and session name:

```text
$ aliyun configure --mode OIDC --profile oidc-profile
Configuring profile 'oidc-profile' in 'OIDC' authenticate mode...
OIDC Provider ARN []: acs:ram::<account-id>:oidc-provider/<provider-name>
OIDC Token File []: /path/to/oidc-token
RAM Role ARN []: acs:ram::<account-id>:role/<role-name>
Role Session Name []: aliyun-cli
Default Region Id []: cn-hangzhou
Default Output Format [json]: json (Only support json)
Default Language [zh|en] en:
Saving profile[oidc-profile] ...Done.
```

## Interactive identity sign-in

### OAuth

OAuth opens a browser-based sign-in flow and stores the resulting temporary credentials in the profile:

```text
$ aliyun configure --mode OAuth --profile oauth-profile
Configuring profile 'oauth-profile' in 'OAuth' authenticate mode...
OAuth Site Type (CN: 0 or INTL: 1, default: CN): 0
Please open the displayed URL in your browser to authorize.
OAuth configuration completed.
Default Region Id []: cn-hangzhou
Saving profile[oauth-profile] ...Done.
```

### CloudSSO

CloudSSO supports a shared session for multiple profiles. The session stores the sign-in URL and AccessToken; each profile selects its own Alibaba Cloud account and access configuration and stores its own STS credentials.

#### Shared session

Configure the session and log in in one step:

```sh
aliyun configure sso login --sso-session company \
  --cloud-sso-sign-in-url 'https://signin-example.alibabacloudsso.com/device/login'
```

Replace the example URL with your CloudSSO sign-in URL. The command saves the configuration and opens a browser for authorization. If you omit the URL, an existing session reuses its saved URL; a new session prompts for one. If authorization fails, the configuration remains available for retry. A failed login with an unchanged URL preserves the previous token.

To save configuration without logging in, use:

```sh
aliyun configure sso set --sso-session company \
  --cloud-sso-sign-in-url 'https://signin-example.alibabacloudsso.com/device/login'
```

After logging in, configure a profile interactively to select an account and access configuration:

```sh
aliyun configure --mode CloudSSO --profile prod-readonly --sso-session company
```

Configure another profile using the same session without interactive account selection:

```sh
aliyun configure set --mode CloudSSO --profile test-admin \
  --sso-session company \
  --cloud-sso-account-id '<AccountId>' \
  --cloud-sso-access-config '<AccessConfigurationId>' \
  --region cn-hangzhou
```

Replace the account and access configuration IDs with values assigned to your CloudSSO user. Different profiles can use different accounts and permissions while sharing the same login:

```sh
aliyun sts GetCallerIdentity --profile prod-readonly
aliyun sts GetCallerIdentity --profile test-admin
```

While the shared session is valid, the CLI obtains or refreshes each profile's STS credentials as needed without opening a browser. When the session expires, log in once for all referencing profiles:

```sh
aliyun configure sso login --sso-session company
# Alternatively, select the session through a profile.
aliyun configure sso login --profile prod-readonly
```

Logging in again does not clear still-valid STS credentials. Use separate session names for different user identities. Changing a profile's session, account, or access configuration clears that profile's cached STS credentials.

#### Delete a session

Delete an unused session with:

```sh
aliyun configure sso delete --sso-session company
```

Deletion is refused while any profile references the session; the error lists those profiles. Reassign or explicitly delete the profiles first.

#### Existing profile-based flow

Existing CloudSSO profiles without `sso_session` continue to use the original flow:

```sh
aliyun configure --mode CloudSSO --profile cloud-sso
aliyun sts GetCallerIdentity --profile cloud-sso
```

Follow the prompts to enter the sign-in URL, authorize in the browser, and select an account and access configuration. The URL, AccessToken, and STS credentials remain in that profile. When its login expires, run the same `configure` command again.

Existing profiles are not migrated automatically. To attach one to a shared session, log in to the session and run:

```sh
aliyun configure --mode CloudSSO --profile cloud-sso --sso-session company
```

The profile reuses its account and access configuration if they are still available. Once saved, the profile references the shared session and no longer stores its own CloudSSO URL or AccessToken.


## External credential sources

### External process

`External` runs a configured local command and uses its output as credentials. The external program must follow these rules:

1. Write the credential response to standard output.
2. Write exactly one valid JSON object to standard output. Diagnostics belong on standard error.
3. Include `mode` and all credential fields required by that mode. The supported response modes are `AK` and `StsToken`.

AK response:

```json
{
  "mode": "AK",
  "access_key_id": "<AccessKeyId>",
  "access_key_secret": "<AccessKeySecret>"
}
```

StsToken response:

```json
{
  "mode": "StsToken",
  "access_key_id": "<AccessKeyId>",
  "access_key_secret": "<AccessKeySecret>",
  "sts_token": "<SecurityToken>"
}
```

Configure it interactively:

```text
$ aliyun configure --mode External --profile external
Configuring profile 'external' in 'External' authenticate mode...
Process Command []: <credential-command>
Default Region Id []: cn-hangzhou
Default Output Format [json]: json (Only support json)
Default Language [zh|en] en:
Saving profile[external] ...Done.
```

### Credentials URI

`CredentialsURI` obtains temporary credentials from a local or remote HTTP endpoint. Configure it interactively:

```text
$ aliyun configure --profile uri --mode CredentialsURI
Configuring profile 'uri' in 'CredentialsURI' authenticate mode...
Credentials URI []: http://127.0.0.1:6666/credentials
Default Region Id []: cn-hangzhou
Default Output Format [json]: json (Only support json)
Default Language [zh|en] en:
Saving profile[uri] ...Done.
```

The stored profile is equivalent to:

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

Alternatively, set `ALIBABA_CLOUD_CREDENTIALS_URI` for the current process instead of storing the URI in a profile.

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

Any non-200 response or malformed/missing credential field is treated as a credential retrieval failure.

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
