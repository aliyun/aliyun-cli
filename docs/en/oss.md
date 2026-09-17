# Built-in OSS automation

[Documentation index](../README.md) | [简体中文](../zh-CN/oss.md)

These interfaces apply to built-in `aliyun oss` in v3.5.1, not the separately installed `ossutil` tool. Run `aliyun oss <command> --help` for command-specific options. Host profiles supply credentials and region; host timeout and retry settings reach OSS execution. Refreshable host credentials are resolved at request time. Update the host CLI through its installation method; `aliyun oss update` does not run the standalone updater.

## Bounded listings

```sh
aliyun oss ls oss://example-bucket/prefix/ --cli-output json --limited-num 100
aliyun oss ls oss://example-bucket/prefix/ --cli-output json --cli-cursor '<next_cursor>'
```

`--cli-output json` emits `schema_version: "1"`, `items`, `returned`, `complete`, `truncated`, and optional `next_cursor`. JSONL emits `type: "item"` records containing `item`, followed by one `type: "summary"` record with pagination fields; every record has `schema_version: "1"`. Items identify their `kind` and contain applicable bucket, key, size, ETag, version, or upload fields.

The default bound is 100 items; `--limited-num` accepts 1–1000. Bucket, object, version, prefix, and multipart-upload listings use the applicable command options. Keep the list query unchanged when resuming and treat cursors as opaque. A filtered page may be empty while still having `next_cursor`; continue until `complete` is true. Listing is not a snapshot: a changed replayed page can require restarting.

## Validation and read-only plans

```sh
aliyun oss cp ./example.txt oss://example-bucket/example.txt --cli-validate
aliyun oss cp ./example.txt oss://example-bucket/example.txt --cli-plan
aliyun oss rm oss://example-bucket/example.txt --cli-plan
```

`--cli-validate` checks locally without an OSS request. `--cli-plan` additionally reads target metadata. Both emit JSON and never execute the upload or deletion. They are mutually exclusive and support only a single regular local file upload to an explicit object key, or single-object deletion (optionally `--version-id`). Recursive transfers, downloads, cloud-to-cloud copies, `sync`, and unsupported business options are rejected before execution.

The plan contains `schema_version: "1"`, `mode`, `command`, `complete`, `side_effects`, `items`, `checks`, and `limitations`. It does not verify write permission or reserve the target; metadata is a snapshot, not a lock or an execution token. Built-in OSS rejects `--dryrun`, `--cli-dry-run`, and `--cli-dry-run-json`; use the supported preview flags above.

## Errors and confirmation

Use `--cli-ai-mode` for structured errors and `--cli-non-interactive` to prevent confirmation input. AI mode or explicit JSON/JSONL output also blocks interactive input; none of these flags authorizes a write. A blocked prompt returns `ConfirmationRequired`. After obtaining authorization, use the operation's existing force option where appropriate; earlier work in a multi-item command may already have completed.

In AI mode, the shared error envelope includes `schemaVersion: "v1"`; non-AI explicit JSON errors keep their existing format. Structured errors go to stderr and include the Agent error fields plus an `oss` object with `schema_version: "1"`, `phase`, `status`, `side_effects`, and `retryable`; failure-report details appear when available. These adapted OSS errors exit with status **1**, unlike the OpenAPI Agent error status. `--no-cli-ai-mode` disables AI errors, but explicit JSON/JSONL output still requests structured errors. Initial configuration-loading errors also use the shared versioned envelope when startup AI mode is enabled.

`cat` always preserves raw stdout bytes, even with `--cli-output json`. JSON/JSONL success output is otherwise limited to `ls` and the preview modes; unsupported combinations fail before execution. AI mode alone does not convert all successful command output to JSON.

## Failure reports and retries

For an executing `cp` or `sync`, add `--cli-failure-report <new-path>`. The report is a new file created exclusively with mode `0600`; an existing path is rejected. It contains JSONL `failed_item` records and a final `summary`, using `schema_version: "1"`, outcome information, and `automatic_retry: false`. It cannot be combined with validation or planning.

A failure report does not replay operations or prove that a failed write had no effect. Inspect the report and actual resource state before retrying selected items; never replay a whole sync/delete operation automatically. Eligible transient retries use jittered backoff; permanent errors and ambiguous write failures stop retrying.
