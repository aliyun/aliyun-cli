# OAuth credential process

`aliyun configure oauth-credential --profile NAME [--config-path /absolute/config.json]`
returns one JSON object with Version=1, Type=sts, AccessKeyId, AccessKeySecret,
SecurityToken and UTC RFC3339 Expiration. Output is sensitive: consume it through
a private subprocess pipe, never logs or shell command substitution.

Only an existing OAuth profile is accepted. No login is started and no fallback
identity is selected. Refresh uses the same cross-process lock, rotation checkpoint
and atomic persistence as normal CLI OAuth. OAuth access/refresh tokens are never
exported. Refresh errors deliberately omit upstream bodies and diagnostics.

This opt-in command is intended for SDK credential providers. It does not depend
on MaxCompute, Python, the maxc wrapper, or MCP. It requires [the OAuth refresh fix (#1483)](https://github.com/aliyun/aliyun-cli/pull/1483);
older CLI versions do not provide this command and consumers must fail clearly.

Validation: config OAuth tests, including a six-process barrier test of this
command over a rotating local token endpoint. No production login is used by tests.

The proposed Python consumer is [MaxC RFC #52](https://github.com/aliyun/aliyun-odps-console/pull/52).
This command does not issue ODPS bearer tokens, select MaxCompute regions, or
establish cross-region authorization guarantees. The consumer RFC remains optional
and subject to its own service-contract review.

Without --profile, the existing current profile is used; an explicit profile must
exist and use OAuth. Unknown profiles, unreadable/malformed configuration,
non-OAuth modes and positional arguments fail without exporting credentials.
A cached export does not rewrite the configuration. Output contains exactly the
six documented fields; diagnostics must not be mixed into its JSON stream.

Tests additionally cover explicit profile selection, invalid input sanitization,
JSON/expiry contracts, and six-process exports across one/two profiles with expired
or near-expiry credentials and configuration symlink aliases. A checkpointed OAuth
rotation followed by failed STS exchange can resume using the saved generation.
