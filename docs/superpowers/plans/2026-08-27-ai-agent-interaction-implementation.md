# Aliyun CLI Agent Interaction Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan.

**Goal:** Implement the approved Canonical Response Help, Help search/listing, AI-mode hints, and typed local error recovery on top of the latest `origin/unify-meta-test`, then ship one tested integration branch and executable binary.

**Architecture:** Keep Canonical response parsing, schema search/query selection, AI-mode/error policy, and command rendering as separate modules. Parallel workers add pure models and algorithms in isolated worktrees; the integration branch alone owns Help flags, provider routing, final text/JSON assembly, cross-module tests, and packaging.

**Tech Stack:** Go, `github.com/stretchr/testify`, local Canonical JSON fixtures, Git worktrees.

---

## Shared contracts

- Base commit: `origin/unify-meta-test` at `2f176bb5595615bf9ba891a0d3b5b8ad3f45f1d4`.
- Approved design: `docs/superpowers/specs/2026-08-27-ai-agent-interaction-00-overview-design.md` through `07-integration-acceptance-design.md`.
- New flags are `--cli-section`, `--cli-search`, and `--cli-all`.
- Workers do not modify `openapi/commando_help.go`, `openapi/machine_help.go`, or `openapi/flags.go`; the integration task owns those files.
- Text Help remains plugin-owned when an installed plugin handles it. Machine JSON remains host Canonical-owned.
- Request/Response sections and search results are complete. Only uns searched AI-mode Root/Product lists are capped at 20.
- No network lookup. Server, network, plugin, credential, Canonical-constraint, and untyped errors do not receive the new local recovery envelope.

## Task 1: Canonical Response model and projection

**Worktree:** `/private/tmp/aliyun-cli-ai-agent-meta`

**Files:**

- Modify: `canonicalmeta/types.go`
- Create: `canonicalmeta/response_schema.go`
- Create: `canonicalmeta/response_schema_test.go`

**Steps:**

1. Add failing tests for lazy `responses`/`components` decoding, status/content-type selection, direct `schema`, missing schema, malformed JSON, reachable component closure, cyclic refs, and missing refs.
2. Store `responses` and `components` as `json.RawMessage` on `canonicalmeta.API` so normal execution does not eagerly expand schemas.
3. Implement a stable `ResponseSchemaDocument` containing selected status/content type, raw-compatible schema, reachable `components.schemas`, warnings, and a no-schema state.
4. Select `200`, then the numerically smallest other 2xx, then `default`; select JSON, then `+json`, then stable first content type, then direct `schema`.
5. Preserve `$ref`, walk only local `#/components/schemas/...` refs, protect cycles, and warn rather than fail for missing refs.
6. Run `go test ./canonicalmeta -count=1`, commit, and report the commit hash.

## Task 2: Help search, listing, and response query guidance

**Worktree:** `/private/tmp/aliyun-cli-ai-agent-search`

**Files:**

- Create: `openapi/help_search.go`
- Create: `openapi/help_search_test.go`
- Create: `openapi/response_query_example.go`
- Create: `openapi/response_query_example_test.go`

**Steps:**

1. Add failing table tests for case/separator-insensitive matching, deterministic ranking, active parameter set filtering, response tree pruning, array/items preservation, reachable-component recomputation, empty matches, and search results over 20.
2. Implement pure search helpers over explicit DTOs or `map[string]any`; do not wire command flags or renderers.
3. Implement AI listing projection: cap only uns searched Root/Product lists at 20, emit listing metadata only when truncated, and bypass the cap for search/`--cli-all`.
4. Add failing tests for inline/ref/cyclic/missing-ref arrays, pagination arrays, resource-name matching, common result names, and stable-first fallback.
5. Implement `ResponseQueryExample` selection and PascalCase/kebab command rendering using supplied product/API/version/style context.
6. Expose a local search-validation result that error recovery can use without invoking a command or network service.
7. Run focused `go test ./openapi -run 'Search|Listing|ResponseQuery' -count=1`, commit, and report the commit hash.

## Task 3: AI-mode state, hints, and local error protocol

**Worktree:** `/private/tmp/aliyun-cli-ai-agent-errors`

**Files:**

- Modify: `sysconfig/aimode/config.go`
- Modify: `sysconfig/aimode/config_test.go`
- Modify: `cli/agent_error.go`
- Modify: `cli/agent_error_test.go`
- Modify: `cli/command.go`
- Modify: `cli/command_test.go`
- Modify: `openapi/agent_error.go`
- Modify: `openapi/agent_error_test.go`
- Modify as needed: typed errors in `openapi/errors.go` and `aliyun-openapi-runtime/engine/errors.go`

**Steps:**

1. Test effective AI-mode precedence: `--no-cli-ai-mode` > `--cli-ai-mode` > valid `ALIBABA_CLOUD_CLI_AI_MODE` > config; accept trimmed/case-insensitive `1/true/0/false` and ignore invalid values.
2. Add reusable text and JSON AI-mode hint models; do not append text to JSON or plugin/server/network errors.
3. Replace the old broad envelope with `message`, optional `did_you_mean`, and exactly one `recovery` containing `action`, optional validated command, and hint. Empty values must be omitted.
4. Preserve typed local causes for unknown product/API/command/flag, missing required parameter, invalid argument, invalid option combination, invalid header, and invalid body file.
5. Narrow normalization so credentials, Canonical constraints, server/Tea/SDK, network, external plugin, post-processing, and untyped errors bypass the new local envelope.
6. Make command generation style-aware and injectable for local search validation. Do not implement Help search itself.
7. Verify no old `ok/category/code/details/requestId/retryable` fields appear and non-AI local text gets exactly one AI-mode hint.
8. Run `go test ./sysconfig/aimode ./cli ./openapi ./aliyun-openapi-runtime/engine -count=1`, commit, and report the commit hash.

## Task 4: Help command surface and renderer integration

**Owner:** Integration branch `/private/tmp/aliyun-cli-ai-agent-integration`

**Files:**

- Modify: `openapi/flags.go`, `openapi/machine_help_args.go`, `openapi/commando_help.go`, `openapi/machine_help.go`
- Modify tests: corresponding `*_test.go` files
- Add cross-module fixtures/tests as needed

**Steps:**

1. Merge Tasks 1-3 and resolve only shared-interface conflicts.
2. Add `--cli-section request|response`, `--cli-search`, and `--cli-all` to the host Help command surface and compatibility entry points.
3. Keep request as default; reject response section without API; keep `--format json` orthogonal.
4. Wire Root/Product/API search and AI listing projection without scanning more metadata than the performance budget permits.
5. Render Response text/JSON with selected schema, reachable components, warnings, notice, and optional query example.
6. Append query guidance to existing Request Help only when a reliable array exists and preserve user command style/version.
7. Add non-AI text/JSON hints and keep installed-plugin text routing unchanged.
8. Omit empty optional values from enhanced Machine Help while keeping `schemaVersion: v1` and existing stable identity/request fields.

## Task 5: Integration verification and packaging

**Files:**

- Create or modify focused integration tests under `openapi/` and `main/`
- Do not commit generated binaries or user-supplied Meta unless explicitly requested.

**Steps:**

1. Run module tests for each workstream, then `go test ./...`.
2. Run formatting/static checks used by the repository and ensure the worktree is clean except intentional source changes.
3. Obtain the colleague-provided updated Canonical Meta from the user immediately before binary end-to-end validation.
4. Build macOS and Linux executables using repository build conventions.
5. Execute the command matrix from `07-integration-acceptance-design.md`, covering AI/non-AI, text/JSON, PascalCase/kebab, request/response, search/all, arrays, and typed local errors.
6. Verify provider boundaries and that server/network/plugin/untyped errors are not given invented recovery.
7. Commit the final integration work and report the branch, binary paths/checksums, test results, and any explicitly deferred acceptance item.
