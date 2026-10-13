# API read timeout defaults

Canonical API metadata optionally carries `backend_timeout_ms`, a positive integer in milliseconds copied from the backend execution budget. The client adds a fixed 10-second margin when selecting a default read timeout. It applies to each ordinary API request, including page requests and retry attempts; waiter completion and SSE streams keep their existing behavior.

Priority for legacy PascalCase commands:

1. Explicit `--read-timeout` or the resolved profile timeout.
2. An existing entry in the legacy SDK API timeout table.
3. `backend_timeout_ms + 10000` milliseconds.
4. The SDK's existing default.

The embedded `aliyun-openapi-runtime` uses the user timeout, then the metadata budget plus 10 seconds, then its original unset timeout. It does not consult the legacy SDK table. Missing metadata keeps the existing fallback. CLI/profile `read-timeout` values remain seconds; the metadata field remains raw milliseconds.

For example, ECS CreateInstance with backend `80000` keeps the legacy SDK table's 86 seconds on the legacy route and uses a 90-second read timeout on the embedded runtime route. RDS CreateDBInstance with backend `55000` uses a 65-second fallback on both routes. User configuration overrides these defaults.

The legacy SDK lookup is currently supplied by the pinned `pl0301/alibaba-cloud-sdk-go` fork commit `c91cb18aa1a17ab58d0833ef585eda899a76da26`, based on SDK v1.63.107. The change only exports the existing table lookup. Once upstream publishes that accessor, replace the fork dependency with the corresponding official SDK version.

Compiled Go plugins receive the raw budget through generated `http.WithBackendTimeoutMS(...)` options and the updated `aliyun-cli-runtime`. Product plugins must be rebuilt and republished to use the new defaults. Existing environment variables continue to transport user configuration; API defaults are embedded in plugin source.
