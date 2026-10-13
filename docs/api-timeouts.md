# API read timeout defaults

Canonical API metadata optionally carries `backend_timeout_ms`, a positive integer in milliseconds copied from the backend execution budget. The client adds a fixed 10-second margin when selecting a default read timeout. It applies to each ordinary API request, including page requests and retry attempts; waiter completion and SSE streams keep their existing behavior.

Priority for ordinary API requests in legacy PascalCase commands, the embedded `aliyun-openapi-runtime`, and compiled Go plugins:

1. Explicit `--read-timeout` or the resolved profile timeout.
2. `backend_timeout_ms + 10000` milliseconds.
3. Leave the timeout unset and let the underlying SDK apply its existing internal behavior.

CLI/profile `read-timeout` values remain seconds; the metadata field remains raw milliseconds. Missing or invalid metadata leaves the timeout unset. The CLI does not query or maintain SDK API timeout tables.

For example, ECS CreateInstance with backend `80000` uses a 90-second read timeout, ECS DescribeInstances with backend `6000` uses 16 seconds, and RDS CreateDBInstance with backend `55000` uses 65 seconds. User configuration overrides these defaults on every route.

Legacy commands use the official `github.com/aliyun/alibaba-cloud-sdk-go` dependency without SDK changes or a fork replacement.

The metadata submodule temporarily uses `pl0301/aliyun-openapi-meta` so a fresh checkout can fetch the regenerated metadata commit on this feature branch. After that commit is available upstream, restore the official metadata repository URL.

Compiled Go plugins receive the raw budget through generated `http.WithBackendTimeoutMS(...)` options and the updated `aliyun-cli-runtime`. Product plugins must be rebuilt and republished to use the new defaults. Existing environment variables continue to transport user configuration; API defaults are embedded in plugin source.
