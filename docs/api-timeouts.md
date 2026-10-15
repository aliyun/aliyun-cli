# API read timeout defaults

Canonical API metadata optionally carries `timeout_ms`. The generator omits this field when the collected backend timeout is at most 10000 milliseconds, and otherwise emits the collected value capped at 60000 milliseconds. The client adds a fixed 2-second margin when selecting a default read timeout. It applies to each ordinary API request, including page requests and retry attempts; waiter completion and SSE streams keep their existing behavior.

Priority for ordinary API requests in legacy PascalCase commands, the embedded `aliyun-openapi-runtime`, and compiled Go plugins:

1. Explicit `--read-timeout` or the resolved profile timeout.
2. `timeout_ms + 2000` milliseconds.
3. Leave the timeout unset and let the underlying SDK apply its existing internal behavior.

CLI/profile `read-timeout` values remain seconds; the metadata field is milliseconds before the client margin. Missing or invalid metadata leaves the timeout unset. The 60-second metadata cap does not limit explicit user settings. The CLI does not query or maintain SDK API timeout tables.

For example, ECS CreateInstance with backend `80000` emits `timeout_ms: 60000` and uses a 62-second read timeout. ECS DescribeInstances with backend `6000` omits the field and uses the SDK's existing behavior. RDS CreateDBInstance with backend `55000` emits `timeout_ms: 55000` and uses 57 seconds. User configuration overrides these defaults on every route.

Legacy commands use the official `github.com/aliyun/alibaba-cloud-sdk-go` dependency without SDK changes or a fork replacement.

Compiled Go plugins receive the metadata timeout through generated `http.WithTimeoutMS(...)` options and the updated `aliyun-cli-runtime`. Product plugins must be regenerated, rebuilt and republished to use the new defaults. Existing environment variables continue to transport user configuration; API defaults are embedded in plugin source.
