# Aliyun CLI Command Tree 与 Help V2 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans for assigned tasks. Use focused regression tests for mechanical changes and test-first coverage for routing, parser, projection, search, and error contracts.

**Goal:** Implement the approved plugin-first L0–L3 Help model, global Help syntax, grouped Command Tree, shared Text/JSON projection, AI defaults, and typed local recovery on the latest `origin/unify-meta-test`, then ship one pushed, packaged, strictly accepted integration branch.

**Architecture:** Parse every host Help invocation into a stable `HelpTarget`, resolve installed-plugin ownership before host parsing/rendering, then build one style-aware structured Help Document. Separate target/routing, document projection, search/policy, parameter/section views, and local error recovery. Parallel worktrees own non-overlapping modules; the integration branch alone owns high-conflict dispatch wiring and executable acceptance.

**Tech Stack:** Go, `testify`, Canonical Meta v12, Git worktrees, Makefile embedded-meta build.

---

## Shared contracts

- Base: `origin/unify-meta-test@395a2a01a3168679f0c3f43fa2dede4d6559e48f`.
- Meta gitlink: `405d3a27746425db3f2c21f2764a9a1b26af0563`.
- Specs: `docs/superpowers/specs/2026-08-28-command-tree-help-00-overview-design.md` through `06-integration-acceptance-design.md`.
- Installed plugins always win, including AI/JSON/Search/All/Section.
- Public syntax is `--help`, `--help-all`, `--help-search`, `--cli-output json`, and prefix `--cli-section`.
- Workers must not update the gitlink or commit binaries/generated packs.

## Task 1: Help options, target primitives, root tree, and Utils

**Worktree:** `/private/tmp/aliyun-cli-command-tree-routing`

**Ownership:**

- Create `cli/help_options.go`, `openapi/help_target.go`, `openapi/root_help.go`, `main/utils_command.go` and focused tests.
- Modify `openapi/flags.go`, `openapi/help_options.go`, root registration/metadata in `main/main.go` and minimal `cli.Command` metadata/raw-argv support.
- Do not implement Canonical projection/search/error recovery and do not rewrite `openapi/commando.go` dispatch.

**Steps:**

- [ ] Replace old Help modifier names and model Default/All/Search/output/section as typed options.
- [ ] Reject duplicates, conflicts, empty search, and unsupported CLI output; `--cli-output` alone must not trigger Help.
- [ ] Add `HelpTarget`/style-aware command builder primitives for L0–L3 and Sections.
- [ ] Add explicit root command groups and offline Root Document input metadata.
- [ ] Add `utils` canonical subtree and hidden compatible root aliases using separate command instances/shared handlers.
- [ ] Add focused parser/root/utils tests, run them, commit, and report the commit hash.

## Task 2: Shared Help documents, policy, search, parameter and Response views

**Worktree:** `/private/tmp/aliyun-cli-command-tree-projection`

**Ownership:**

- Create focused document/policy/renderer files such as `help_document.go`, `help_policy.go`, `help_parameter.go`, `help_response_document.go` and tests.
- Modify `openapi/machine_help.go`, `openapi/help_projection.go`, `openapi/help_search.go`, `openapi/response_help.go`, `canonicalmeta/types.go` as needed.
- Do not modify `openapi/commando.go`, `openapi/commando_help.go`, `main/main.go`, or AI error files.

**Steps:**

- [ ] Build style-aware Root/Product/Action/Parameter/Section documents before rendering.
- [ ] Implement Text/JSON default policy, 100-line complete-object budget, required safety exception, `result/next`, and internal switches.
- [ ] Implement Product and Action Default/All semantics, Title/Description fallbacks, public globals, examples and query navigation.
- [ ] Extend tokenizer (including alpha-digit), deterministic rank, all-layer candidates, full-rank then top-20, and reusable recovery validation.
- [ ] Implement complete L3 parameter tree/search and full Responses + reachable Components projection without `$ref` inlining.
- [ ] Reuse or Golden-test the runtime/plugin parameter rendering intersection.
- [ ] Run focused canonical/openapi/runtime tests, commit, and report the commit hash.

## Task 3: AI Mode output and typed local recovery

**Worktree:** `/private/tmp/aliyun-cli-command-tree-errors`

**Ownership:**

- Modify `openapi/agent_error.go`, `cli/agent_error.go` and their focused tests.
- Add a small effective-output or recovery command helper file if needed.
- Do not modify Help routing/rendering hotspots; consume interfaces from Tasks 1/2, using temporary narrow interfaces where required.

**Steps:**

- [ ] Update all recovery commands to final suffix Help/Search and prefix Section forms.
- [ ] Preserve style/version, publish Search only after shared validation, and use L3 only for real top-level flags.
- [ ] Remove command for option conflicts; map Header/Body file errors to direct parameter Help.
- [ ] Enforce compact Envelope invariants and recursive omission of empty optional data.
- [ ] Preserve strict local-error allowlist and excluded-error behavior.
- [ ] Add focused JSON/Hints/Recovery tests, run them, commit, and report the commit hash.

## Task 4: Integration routing and L0–L3 dispatch

**Owner:** `/private/tmp/aliyun-cli-command-tree-help-v2`

**Steps:**

- [ ] Commit specs/plan, create the three task worktrees from the exact base, and dispatch Tasks 1–3 in parallel.
- [ ] Cherry-pick the three commits, preserving the Canonical gitlink.
- [ ] Implement the narrow pre-parse seam required to delegate installed-plugin Help with original argv.
- [ ] Replace old `NormalizeMachineHelpArgs` public path and wire `--cli-output json` plus AI implicit JSON.
- [ ] Route Root/Product/Action/Parameter/Section through shared target/document/search/policy/rendering.
- [ ] Ensure default `help <product> <action>` remains complete Request Section and all generated non-Section commands use suffix syntax.
- [ ] Remove `ALIBABA_CLOUD_ORIGINAL_PRODUCT_HELP` as a plugin override while retaining baseline style selection for no-plugin Help.
- [ ] Run integration suites, commit, and push `codex/command-tree-help-v2`.

## Task 5: Build, strict acceptance, rapid fixes, and final push

**Steps:**

- [ ] Run focused suites, broad module suites, then `go test ./... -count=1` outside sandbox constraints; compare failures against clean base when needed.
- [ ] Run `make VERSION=5.0.0 test-release build` without publish targets.
- [ ] From isolated HOME/outside repo, execute every row in `06-integration-acceptance-design.md` across AI/non-AI, Text/JSON, camel/kebab, Default/All/Search, plugin/no-plugin, L0–L3 and request/response.
- [ ] Execute every emitted Search Recovery command and assert at least one match.
- [ ] Fix failures directly on the integration branch with focused regressions; rebuild and rerun affected/full acceptance.
- [ ] Produce executable and `.tgz`, verify unpacked smoke/version, record SHA-256, and keep artifacts untracked.
- [ ] Commit all fixes and push the final branch.
