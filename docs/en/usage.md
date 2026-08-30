# Commands, output, and automation

[Documentation index](../README.md) | [简体中文](../zh-CN/usage.md)

## Command forms

For RPC APIs, use either the traditional PascalCase operation name or the kebab-case command name:

```sh
aliyun <product> <Operation> [--Parameter value ...]
aliyun <product> <operation-name> [--parameter value ...]

aliyun ecs DescribeInstances --RegionId cn-hangzhou
aliyun ecs describe-instances --biz-region-id cn-hangzhou
```

The two command forms call the same API; only the operation and parameter spelling differs. For example:

```sh
# List regions
aliyun ecs DescribeRegions
aliyun ecs describe-regions

# List instances
aliyun ecs DescribeInstances --RegionId cn-hangzhou --PageSize 10
aliyun ecs describe-instances --biz-region-id cn-hangzhou --page-size 10

# Show one instance
aliyun ecs DescribeInstanceAttribute --InstanceId i-1234567890abcdef
aliyun ecs describe-instance-attribute --instance-id i-1234567890abcdef
```

In practice:

- PascalCase commands use the original API operation name, such as `DescribeInstances`, and usually keep the original API parameter names such as `--RegionId` and `--InstanceId`.
- Kebab-case commands use the metadata-derived command name, and parameter names are also exposed in kebab-case where defined by metadata. Use the local command help for the exact flag names. For example, `describe-instances` uses `--biz-region-id`, while `describe-instance-attribute` uses `--instance-id`.

The subcommand casing decides which engine handles the call: an all-lowercase subcommand is routed to the plugin/runtime engine, while a subcommand containing uppercase letters goes through the built-in OpenAPI path. Do not mix the two spellings in one command — `aliyun ecs DescribeInstances --biz-region-id cn-hangzhou` fails with an unknown parameter error; pair `--RegionId` with `DescribeInstances` and `--biz-region-id` with `describe-instances`.

Product-level help follows the same split. Without an installed product plugin, `aliyun ecs --help` shows the PascalCase command list by default; products that exist only in the bundled metadata show the kebab-case list. Two environment variables switch the style explicitly:

```sh
# kebab-case product help (no product plugin installed)
ALIBABA_CLOUD_BASELINE_PRODUCT_HELP=true aliyun ecs --help

# legacy PascalCase product help (a product plugin is installed)
ALIBABA_CLOUD_ORIGINAL_PRODUCT_HELP=true aliyun ecs --help
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

If the request uses an API or parameter that is not present in the local metadata, the CLI returns `unknown api` or `unknown parameter`. Use `--force` to skip those local checks and continue the call.

`--force` skips API and parameter metadata checks; it does not bypass server-side authorization or validation.

When `--force` is used, these options should normally be provided explicitly:

- `--version`: the API version. Get the version from the product API documentation. For example, ECS commonly uses `2014-05-26`.
- `--endpoint`: the product endpoint. Use the endpoint documented for that product.

Use `--header X-foo=bar` repeatedly to add custom HTTP headers. Use `--secure` to force HTTPS. `--skip-secure-verify` disables certificate verification and is not recommended.

## Validate without invoking

Use `--cli-dry-run` before a mutating call:

```sh
aliyun ecs run-instances ... --cli-dry-run
```

Support depends on the API metadata. CLI dry-run validation is not a substitute for reviewing permissions and resource impact.

## Filter and format output

### Table output

OpenAPI responses are JSON by default. For example, a `DescribeInstances` response may contain a collection like this:

```json
{
  "PageNumber": 1,
  "TotalCount": 2,
  "PageSize": 10,
  "RequestId": "2B76ECBD-A296-407E-BE17-7E668A609DDA",
  "Instances": {
    "Instance": [
      {
        "InstanceId": "i-12345678912345678123",
        "Status": "Stopped"
      },
      {
        "InstanceId": "i-abcdefghijklmnopqrst",
        "Status": "Running"
      }
    ]
  }
}
```

Use `--output` to select the collection and render selected fields as a table:

```sh
aliyun ecs DescribeInstances \
  --output cols=InstanceId,Status rows=Instances.Instance[]
```

The result is:

```text
InstanceId                    | Status
----------                    | ------
i-12345678912345678123        | Stopped
i-abcdefghijklmnopqrst        | Running
```

`--output` supports the following fields:

- `cols` is required. It is a comma-separated list of fields evaluated relative to each row. The field names are also used as the table headers.
- `rows` is optional. It is a JMESPath expression that selects the array used as table rows. For nested API responses, it is normally required.
- `num=true` is optional and adds a zero-based `Num` column.

For example:

```sh
aliyun ecs DescribeInstances \
  --output cols=InstanceId,Status rows=Instances.Instance[] num=true
```

The `rows` expression must resolve to an array. A missing `cols` value, an invalid JMESPath expression, or a non-array row result causes the command to fail instead of silently producing an incomplete table.

### JMESPath query

Use `--cli-query` to filter or reshape JSON output:

```sh
aliyun ecs describe-instances \
  --cli-query 'Instances.Instance[].{Id:InstanceId,Status:Status}'
```

Use `--quiet` to suppress normal output where supported.

## Pagination

For pageable APIs, `--pager` repeatedly calls the API and merges the collection from every response:

```sh
aliyun ecs describe-instances --pager
```

With no additional fields, the CLI uses the common response fields `PageNumber`, `PageSize`, `TotalCount`, and `NextToken`, and attempts to recognize a nested result array automatically. If the collection cannot be recognized, specify its JMESPath explicitly:

```sh
aliyun ecs DescribeInstances \
  --pager path='Instances.Instance[]'
```

APIs that use non-standard pagination fields can override the mapping:

```sh
aliyun <product> <operation> \
  --pager \
  path='Data.Items[]' \
  PageNumber='PageInfo.PageNumber' \
  PageSize='PageInfo.PageSize' \
  TotalCount='PageInfo.TotalCount'
```

Token-based APIs can specify the token field in the same way:

```sh
aliyun <product> <operation> \
  --pager path='Data.Items[]' NextToken='Data.NextToken'
```

The pager fields are:

| Field | Purpose |
| --- | --- |
| `path` | JMESPath expression selecting the result array to merge |
| `PageNumber` | Page-number request field and response expression |
| `PageSize` | Page-size request field and response expression |
| `TotalCount` | Response expression for the total item count |
| `NextToken` | Next-token request field and response expression |

The final JSON contains the merged collection under `path`; per-page metadata such as request IDs and page numbers is not preserved. Pagination can issue many requests and retain all merged items in memory, so use it carefully for very large result sets. It cannot be combined with `--waiter` or `--cli-dry-run`.

## Wait for a state

`--waiter` repeatedly invokes the same API until a JMESPath expression in the response equals the target value. A typical use case is waiting for an ECS instance to enter the `Running` state after creation:

```sh
aliyun ecs DescribeInstances \
  --InstanceIds '["i-12345678912345678123"]' \
  --waiter expr='Instances.Instance[0].Status' to=Running
```

When the expression matches, the CLI stops polling and prints the final API response. The supported fields are:

- `expr` is required. It is the JMESPath expression evaluated against each response.
- `to` is required. It is the expected value, compared with the evaluated result as text.
- `timeout` is optional. It defaults to 180 seconds; use a value from 1 to 600 seconds for compatibility across command forms.
- `interval` is optional. It defaults to 5 seconds; use a value from 2 to 10 seconds for compatibility across command forms.

To override the polling limits:

```sh
aliyun ecs DescribeInstances \
  --InstanceIds '["i-12345678912345678123"]' \
  --waiter \
  expr='Instances.Instance[0].Status' \
  to=Running \
  interval=5 \
  timeout=300
```

The waiter stops immediately if the API call or JMESPath evaluation fails. If the timeout is reached, the error includes the expression, expected value, and last observed value. `--waiter` cannot be combined with `--pager` or `--cli-dry-run`.

## Safety and non-interactive use

Potentially destructive operations may be subject to CLI safety policy and confirmation. Manage policy with:

```sh
aliyun configure safety-policy --help
```

`--yes` skips a confirmation prompt for non-interactive use, but does not override a deny policy. Review the exact command and scope before using it in automation.

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

To disable automatic agent mode for one command, use `--no-cli-ai-mode`:

```sh
aliyun ecs describe-instances --no-cli-ai-mode
```

## PascalCase argument edge case

This rule applies only to the traditional PascalCase command form. When a parameter value begins with `-`, use `--name=value` so the value is not parsed as another flag:

```sh
aliyun ecs SomeOperation --PortRange=-1/-1
```

Next: [manage product plugins](./plugins.md).
