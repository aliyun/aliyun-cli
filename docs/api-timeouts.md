# API read timeout defaults

Canonical API metadata optionally carries `timeout_ms`. The generator omits this field when the collected backend timeout is at most 10000 milliseconds, and otherwise emits `min(backend_timeout_ms + 2000, 62000)`. This is the final default read timeout, including the fixed 2-second margin. Clients consume it directly. It applies to each ordinary API request, including page requests and retry attempts; waiter completion and SSE streams keep their existing behavior.

Priority for ordinary API requests in legacy PascalCase commands, the embedded `aliyun-openapi-runtime`, and compiled Go plugins:

1. Explicit `--read-timeout` or the resolved profile timeout.
2. `timeout_ms` milliseconds, without adding another margin.
3. Leave the timeout unset and let the underlying SDK apply its existing internal behavior.

CLI/profile `read-timeout` values remain seconds; the metadata field is the final timeout in milliseconds. Missing, non-positive or overflowing duration values leave the timeout unset. The 62-second metadata cap does not limit explicit user settings. The CLI does not query or maintain SDK API timeout tables.

For example, ECS CreateInstance with backend `80000` emits `timeout_ms: 62000` and uses a 62-second read timeout. ECS DescribeInstances with backend `6000` omits the field and uses the SDK's existing behavior. RDS CreateDBInstance with backend `55000` emits `timeout_ms: 57000` and uses 57 seconds. User configuration overrides these defaults on every route.

Legacy commands use the official `github.com/aliyun/alibaba-cloud-sdk-go` dependency without SDK changes or a fork replacement.

Compiled Go plugins receive the metadata timeout through generated `http.WithTimeoutMS(...)` options and the updated `aliyun-cli-runtime`. Product plugins must be regenerated, rebuilt and republished to use the new defaults. Existing environment variables continue to transport user configuration; API defaults are embedded in plugin source.

Metadata and plugin artifacts produced with the previous pre-margin `timeout_ms` semantics must be regenerated before release with these consumers.
