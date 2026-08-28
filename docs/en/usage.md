# Commands, output, and automation

[Documentation index](../README.md) | [简体中文](../zh-CN/usage.md)

## Command forms

For RPC APIs, use either the traditional PascalCase operation name or, where Canonical metadata is available, the kebab-case command name:

```sh
aliyun <product> <Operation> [--Parameter value ...]
aliyun <product> <operation-name> [--parameter value ...]

aliyun ecs DescribeInstances --RegionId cn-hangzhou
aliyun ecs describe-instances --region-id cn-hangzhou
```

Use the product/operation forms above for normal API calls. For APIs that can only be addressed by a raw method and path, the CLI retains the following RESTful compatibility form; it is not the recommended form for general use:

```sh
aliyun cs GET /clusters
aliyun cs POST /clusters --body "$(cat input.json)"
aliyun cs DELETE /clusters/<cluster-id>
```

Each API supports one API style. Check its OpenAPI documentation or local CLI help when unsure.

## Discover commands and parameters

Human-readable help:

```sh
aliyun --help
aliyun ecs DescribeInstances --help
aliyun ecs describe-instances --help
```

## Endpoints and unsupported metadata

The CLI normally resolves API versions and endpoints from metadata. To call an API not present in the bundled metadata, use `--force` together with an explicit version and endpoint:

```sh
aliyun newproduct SomeAction \
  --version 2025-01-01 \
  --endpoint newproduct.aliyuncs.com \
  --SomeParameter value \
  --force
```

`--force` skips API and parameter metadata checks; it does not bypass server-side authorization or validation.

Use `--header X-foo=bar` repeatedly to add custom HTTP headers. Use `--secure` to force HTTPS. `--skip-secure-verify` disables certificate verification and is not recommended.

## Validate without invoking

Use `--cli-dry-run` before a mutating call:

```sh
aliyun ecs run-instances ... --cli-dry-run
```

Support depends on the API metadata. CLI dry-run validation is not a substitute for reviewing permissions and resource impact.

## Filter and format output

### Table output

`--output` accepts columns and an optional JMESPath row selector:

```sh
aliyun ecs DescribeInstances \
  --output cols=InstanceId,Status rows=Instances.Instance[]
```

- `cols`: comma-separated fields shown as columns.
- `rows`: optional JMESPath expression selecting the row collection.

### JMESPath query

Use `--cli-query` to filter or reshape JSON output:

```sh
aliyun ecs describe-instances \
  --cli-query 'Instances.Instance[].{Id:InstanceId,Status:Status}'
```

Use `--quiet` to suppress normal output where supported.

## Pagination

For pageable APIs, `--pager` requests subsequent pages and merges the result:

```sh
aliyun ecs describe-instances --pager
```

Review the specific API's help because pagination fields vary by product.

## Wait for a state

`--waiter` polls until a JMESPath expression reaches a target value:

```sh
aliyun ecs DescribeInstances \
  --InstanceIds '["i-example"]' \
  --waiter expr='Instances.Instance[0].Status' to=Running interval=5 timeout=300
```

- `expr`: field selected from the JSON response.
- `to`: expected value.
- `interval`: optional polling interval in seconds.
- `timeout`: optional total timeout in seconds.

## Safety and non-interactive use

Potentially destructive operations may be subject to CLI safety policy and confirmation. Manage policy with:

```sh
aliyun configure safety-policy --help
```

`--yes` skips a confirmation prompt for non-interactive use, but does not override a deny policy. Review the exact command and scope before using it in automation.

The CLI supports `--language en` and `--language zh`. For deterministic scripts, prefer JSON command output rather than parsing localized human-readable text.

## Agent-aware optimization and AI mode

AI mode can be managed globally or per command:

```sh
aliyun configure ai-mode --help
aliyun ecs describe-instances --cli-ai-mode
```

When a supported agent environment is detected, in-process OpenAPI commands automatically enable agent-oriented interaction and execution optimizations. These optimizations currently include stricter metadata-based validation and more structured error output, but the exact behavior may change and is not a stable compatibility contract.

Requests made through this automatically enabled mode append the following generic User-Agent marker:

```text
AlibabaCloud-AIMode/enabled
```

The marker does not contain the detected agent name. To disable automatic agent mode for one command, use `--no-cli-ai-mode`:

```sh
aliyun ecs describe-instances --no-cli-ai-mode
```

## Argument edge cases

When a value begins with `-`, use the `--name=value` form so it is not parsed as another flag:

```sh
aliyun ecs SomeOperation --PortRange=-1/-1
```

Next: [manage product plugins](./plugins.md).
