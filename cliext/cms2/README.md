# cms2 — CloudMonitor CLI Wrapper for Aliyun CLI

The `cms2` package integrates the standalone [Alibaba Cloud CMS CLI](https://help.aliyun.com/product/28615.html) (`aliyuncms`) into Aliyun CLI as a built-in subcommand. Users can invoke `aliyun cms2 <subcommand>` to manage various CloudMonitor applications, including Workspace, Integration Center, Application Monitoring, Browser Monitoring, Alert Center, Event Center, Cloud Dial Test, Prometheus Service, and more — using the same credentials and profiles configured in `aliyun configure`.

## How it works

```
aliyun cms2 integration-policy list --region cn-hangzhou
       │
       ├─ 1. Ensure the aliyuncms binary is installed (~/.aliyun/aliyuncms)
       ├─ 2. Load credentials from the current aliyun-cli profile
       ├─ 3. Bridge credentials via ALIYUN_CMS_CLI_* environment variables
       ├─ 4. Strip aliyun-cli-only flags (--profile, --config-path, etc.)
       └─ 5. exec aliyuncms integration-policy list --region cn-hangzhou
```

The wrapper follows the [Aliyun CLI External Binary Wrapper Specification](../../), alongside `ossutil` and `otsutil`.

## Usage

```bash
# List integration policies
aliyun cms2 integration-policy list

# Use a specific profile
aliyun --profile prod cms2 rule list

# Specify region
aliyun cms2 promql query --expr 'up' --region cn-hangzhou

# Pass credentials directly (configure mode)
aliyun --access-key-id AKID --access-key-secret SECRET \
       --region cn-hangzhou cms2 prometheus-instance list

# Pipe stdin
echo '{"name":"test"}' | aliyun cms2 integration-policy create --body @-

# Show help
aliyun cms2 help
aliyun cms2 rule --help
```

## Credential bridging

Credentials are resolved from the aliyun-cli profile (`aliyun configure`) and passed to `aliyuncms` via environment variables:

| Profile field | Environment variable |
|--------------|---------------------|
| AccessKeyId | `ALIYUN_CMS_CLI_ACCESS_KEY_ID` |
| AccessKeySecret | `ALIYUN_CMS_CLI_ACCESS_KEY_SECRET` |
| StsToken / SecurityToken | `ALIYUN_CMS_CLI_SECURITY_TOKEN` |
| RegionId | `ALIYUN_CMS_CLI_REGION` |
| Endpoint | `ALIYUN_CMS_CLI_ENDPOINT` |

All credential modes are supported: AK, StsToken, RamRoleArn, EcsRamRole, OIDC, CredentialsURI. For dynamic modes (RamRoleArn, OIDC, etc.), the wrapper performs credential exchange and passes the resulting temporary AK/SK/Token.

## Auto-download and version management

On first use, the wrapper automatically downloads the `aliyuncms` binary from the Alibaba Cloud CDN to `~/.aliyun/aliyuncms`. It checks for updates once per day (TTL = 86400s).

Supported platforms: `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`, `windows/arm64`.

If `version.txt` is unavailable (network error, etc.):
- **Already installed**: skips the update, prints a warning, continues with the local binary.
- **Not installed**: returns an error with install instructions.

## Development

Use the `ALIYUN_CMS2_EXEC_PATH` environment variable to point to a local build, bypassing auto-download:

```bash
ALIYUN_CMS2_EXEC_PATH=/path/to/aliyuncms aliyun cms2 version
```

Run tests:

```bash
go test ./cliext/cms2/ -v
```

## File structure

```
cliext/cms2/
├── main.go          # NewCms2Command() — command definition
├── cms2.go          # Core logic: Run, PrepareEnv, RemoveFlagsForMainCli, Install, Execute
├── main_test.go     # Command metadata tests
├── cms2_test.go     # Unit tests (28 cases)
├── DESIGN.md        # Implementation design document
├── README.md        # This file (English)
└── README-CN.md     # Chinese documentation
```
