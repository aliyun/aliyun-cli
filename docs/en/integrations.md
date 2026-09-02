# MCP proxy, OpenTelemetry, and machine-readable interfaces

[Documentation index](../README.md) | [简体中文](../zh-CN/integrations.md)

## MCP proxy

The MCP proxy exposes discovered Alibaba Cloud API MCP servers through one local endpoint and handles the OAuth flow used to reach the upstream service.

Start it with the public command path:

```sh
aliyun utils mcp-proxy --region-type CN --port 8088
```

The defaults are:

| Option | Default | Meaning |
| --- | --- | --- |
| `--host` | `127.0.0.1` | Local listen address |
| `--port` | `8088` | Local listen port |
| `--region-type` | `CN` | Endpoint family; use `CN` or `INTL` |
| `--scope` | `/acs/mcp-server` | OAuth scope |
| `--no-browser` | off | When set, use manual authorization-code input instead of opening a browser |
| `--oauth-app-name` | empty | Reuse an existing OAuth application by name |
| `--upstream-url` | empty | Override the configured upstream MCP endpoint |

After startup, the CLI prints the local MCP and SSE URLs for the discovered servers. Configure the MCP client with the printed URL rather than constructing a path from the server name.

### Access control

Use `--allowed-servers` and `--blocked-servers` with comma-separated server names, IDs, or path prefixes:

```sh
aliyun utils mcp-proxy \
  --allowed-servers ecs-tools,/mcp/approved \
  --blocked-servers deprecated-server
```

The blocked list takes precedence over the allowed list. For backward compatibility, omitting both lists allows every discovered server. Keep the default loopback host where possible. Before listening on `0.0.0.0` or another non-loopback address, configure an allowed list and apply host-level network controls.

### Timeouts and streaming

The local server limits request-header reads to 10 seconds and idle keep-alive connections to 2 minutes. Waiting for upstream response headers is limited to 2 minutes. There is no server-wide write timeout or HTTP client total timeout, so an established SSE stream is not terminated by these limits.

### Logging and sensitive data

Proxy logs retain the HTTP method, URL origin/path, status, timing, and payload sizes needed for diagnostics. URL user information, query strings, fragments, request bodies, OAuth authorization codes, OAuth error bodies, and SSE payload lines are not logged. Do not put credentials in MCP server path segments, because paths remain visible for routing diagnostics.

## OpenTelemetry trace-context propagation

Alibaba Cloud CLI can propagate W3C trace context to OpenAPI requests. This feature injects headers into outgoing requests; it does not create spans or export telemetry by itself.

| Environment variable | Behavior |
| --- | --- |
| `ALIBABA_CLOUD_OTEL_TRACEPARENT` | Injected as the `traceparent` header after format validation |
| `ALIBABA_CLOUD_OTEL_BAGGAGE` | Injected as the `baggage` header without changing its value |
| `ALIBABA_CLOUD_OTEL_ENABLED` | `false`, `0`, or `off` disables propagation; other values do not create headers unless traceparent or baggage is present |

Example:

```sh
export ALIBABA_CLOUD_OTEL_TRACEPARENT=00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
export ALIBABA_CLOUD_OTEL_BAGGAGE='workflow.id=deploy-42,environment=staging'

aliyun ecs DescribeRegions
```

`traceparent` must use lowercase hexadecimal fields in this shape:

```text
00-<32 hex trace id>-<16 hex parent id>-<2 hex flags>
```

An invalid `traceparent` value is ignored and produces a warning on stderr. A valid, non-empty `traceparent` or `baggage` environment value takes precedence over a header with the same name supplied through `--header`. Compatible child plugins receive the same values through an internal encoded header environment; users should not set that internal variable directly.

Treat baggage as data sent to the target service. Do not include AccessKeys, tokens, authorization codes, passwords, or other secrets.

## Machine-readable Help

Request JSON Help explicitly with `--cli-output json` and a Help operation:

```sh
aliyun --help --cli-output json
aliyun ecs --help --cli-output json
aliyun ecs DescribeInstances --help --cli-output json
aliyun ecs DescribeInstances --cli-section request --help --cli-output json
aliyun ecs DescribeInstances --cli-section response --help --cli-output json
aliyun ecs --help-search instance --cli-output json
```

`--cli-output json` selects the Help format only. It is not an execution-output flag; successful OpenAPI calls already return JSON by default. Use `--output` or `--cli-query` to format an API response.

Machine Help writes one JSON document plus a trailing newline to stdout. Its common fields are:

| Field | Meaning |
| --- | --- |
| `schemaVersion` | Machine Help contract version; currently `v1` |
| `helpLevel` | `root`, `utility`, `product`, `api`, or `parameter` |
| `name` | Current Help target |
| `description` | Description in the effective output language |
| `target` | Resolved command path, style, and API version where applicable |
| `commands`, `parameters`, `flags` | Entries available at the current level |
| `result.shown`, `result.total`, `result.truncated` | Projection and paging information |
| `next` | Suggested commands for search, complete output, or another Help section |

Use `--help-all` to remove the normal result cap. Use `--help-search <text>` to filter the current Help level; combine it with `--help-all` to return all matches. Consumers should branch on `schemaVersion` and `helpLevel`, ignore unknown fields, and use `next` instead of constructing follow-up commands themselves.

Machine Help request errors are written to stderr and exit with status 2. They use a versioned envelope:

```json
{
  "schemaVersion": "v1",
  "error": {
    "code": "INVALID_CLI_OUTPUT",
    "message": "--cli-output only supports json",
    "target": ["aliyun"],
    "suggestions": []
  }
}
```

## Agent-mode errors

Enable agent behavior globally or for one command:

```sh
export ALIBABA_CLOUD_CLI_AI_MODE=1
# or
aliyun ecs describe-instances --cli-ai-mode
```

Supported local usage, query, transport, OAuth, and server errors are written as one compact JSON object to stderr. Success output remains on stdout. Optional fields are omitted when unavailable:

```json
{
  "message": "unknown flag --instnace-type",
  "did_you_mean": ["--instance-type"],
  "recovery": {
    "action": "search_parameter",
    "command": "aliyun ecs describe-instances --help-search instance-type",
    "hint": "Search request parameters related to instance-type."
  }
}
```

Remote server errors may additionally include `error_code`, `status_code`, and `request_id`. `did_you_mean` and `recovery.command` are also optional; `message`, `recovery.action`, and `recovery.hint` are present in every structured Agent error.

The Agent error object is a separate compact interface and currently has no `schemaVersion`; the Machine Help `v1` contract does not apply to it. Not every error is normalized yet, so consumers must also tolerate human-readable stderr.

| Exit status | Meaning |
| --- | --- |
| `0` | Successful command or Help response |
| `1` | General execution failure |
| `2` | Usage error, structured Agent error, or Machine Help request error |
| `3` | Failure accompanied by a CLI recovery tip |

For request inspection without invoking the API, `--cli-dry-run-json` emits structured request details. `--cli-dry-run` is the human-readable form.

Next: [commands, output, and automation](./usage.md).
