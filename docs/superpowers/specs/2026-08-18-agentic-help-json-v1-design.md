# Agentic Help JSON v1 Design

## Context

Aone requirement 85538412 asks aliyun CLI to expose the Canonical OpenAPI help data through a stable, public, machine-readable interface. Today `--help` is a value-less flag, so `--help=json` fails parsing, while `GENERATE_METADATA=YES` exports only the static `cli.Command` tree and does not describe APIs served by the unified Canonical runtime.

This change is based on `origin/unify-meta-test`. Both legacy PascalCase OpenAPI commands and kebab-case commands are Canonical-backed on that branch, but they retain different command-line projections and may retain different default API versions.

## Goals

- Support equivalent machine-readable entry points:
  - `aliyun ... --help=json`
  - `aliyun help ... --format json`
- Support root, product, and API help.
- Support both PascalCase and kebab-case API names.
- Publish a stable `schemaVersion: "v1"` contract rather than serializing internal Go structs or raw Canonical files.
- Describe the actual command contract, including both legacy and kebab parameter projections.
- Operate entirely from local static/Canonical metadata without credentials, network access, API execution, or plugin installation.
- Preserve all existing text-help and command-execution behavior.

## Non-goals

- Changing Canonical generation or adding new source fields.
- Changing API execution, command routing, parameter serialization, or default-version selection.
- Inferring output schemas, pagination, risk, or recovery information when metadata does not provide it.
- Making the existing hidden `GENERATE_METADATA=YES` export part of the public contract.

## Command surface

The following pairs are equivalent:

```text
aliyun --help=json
aliyun help --format json

aliyun ecs --help=json
aliyun help ecs --format json

aliyun ecs DescribeInstances --help=json
aliyun help ecs DescribeInstances --format json

aliyun ecs describe-instances --help=json
aliyun help ecs describe-instances --format json
```

Plain `--help`, `-h`, and `aliyun help ...` without `--format json` keep their current text output.

Only `json` is accepted as the value of `--help` and `--format` in the machine-help forms. Unsupported values fail without falling through to API execution.

## Architecture

### Entry normalization

The CLI parser recognizes the two machine-help syntaxes and normalizes them into one request containing:

- target path after `aliyun`;
- requested output format;
- requested API version, when supplied;
- the requested command spelling, used to determine PascalCase versus kebab-case behavior.

The normalizer must not reinterpret ordinary API parameters named `format` or change the behavior of the existing value-less help flag.

### Stable help service

A dedicated Help v1 service owns the public DTOs and JSON rendering. It accepts a normalized request and returns one of three document kinds:

- `root`
- `product`
- `api`

It does not marshal `cli.Command`, runtime `meta.*`, `canonicalmeta.*`, or on-disk Canonical structs directly. Internal data is explicitly projected into the v1 DTO so future internal refactors do not silently change the external contract.

### Metadata resolution

- Root documents combine the static top-level command tree with the locally available product catalog.
- Product documents use product metadata plus the selected version index. They do not load every API JSON.
- API documents resolve the requested command to a Canonical `(product, version, APIName)` and then load that one API definition.
- PascalCase requests preserve the legacy default version and legacy parameter view.
- Kebab-case requests preserve the plugin/runtime default version and Canonical parameter view.
- An explicitly supplied version overrides only through the existing command-style rules; machine help does not introduce a new version-selection policy.

## JSON contract

### Common envelope

Every successful document contains:

```json
{
  "schemaVersion": "v1",
  "kind": "root",
  "target": {
    "path": ["aliyun"],
    "requestedStyle": "root"
  }
}
```

`kind` is `root`, `product`, or `api`. `requestedStyle` is `root`, `product`, `camel`, or `kebab`.

No timestamps, random identifiers, local paths, or other invocation-specific unstable values are emitted. Objects use deterministic field order through DTO definition order; maps and discovered lists are sorted before marshaling.

### Root document

The root document adds:

- `commands`: public top-level built-in commands with names and localized descriptions;
- `products`: product code, localized names, supported command styles, and whether Canonical machine help is available.

The root document is a discovery index. It does not inline every product or API schema.

### Product document

The product document adds:

- product code and localized names;
- API style;
- `legacyDefaultVersion`;
- `pluginDefaultVersion`;
- sorted `supportedVersions`;
- `selectedVersion`;
- APIs from the selected version index, with `name`, `cmdName`, localized descriptions, and deprecation state.

The version index keeps product help cheap and avoids opening every per-API JSON file.

### API document

The API document adds:

- Canonical API name and kebab command name;
- selected version and supported versions;
- localized descriptions and deprecation state;
- HTTP operation information;
- `activeParameterSet`, set to `camel` or `kebab`;
- `parameterSets.camel`, reflecting the legacy CLI projection actually accepted by PascalCase execution;
- `parameterSets.kebab`, reflecting the Canonical/runtime projection actually accepted by kebab execution;
- applicable global parameters;
- `examples.camel` and `examples.kebab`;
- the following stable nullable fields:

```json
{
  "outputSchema": null,
  "pagination": null,
  "risk": null,
  "recovery": null
}
```

Each parameter node contains the available subset of:

- logical name;
- wire/raw name;
- CLI options;
- type;
- location;
- required state;
- serialization style;
- localized help;
- example;
- recursive object fields, array element, or map value;
- source-backed constraints.

Missing constraints are represented explicitly as null values rather than guessed from API names, HTTP methods, descriptions, or examples.

## Error contract

Machine-help failures produce a JSON error document on stderr and a non-zero exit status. Stdout stays empty. The error document contains:

- `schemaVersion: "v1"`;
- a stable error code;
- a human-readable message;
- the requested target path;
- deterministic suggestions when available.

Initial stable codes cover invalid format, invalid target depth, unknown product, unknown version, unknown API/command, and unavailable machine help. Errors must never fall through into API execution or plugin auto-install.

## Compatibility

- Existing text help is byte-for-byte unchanged outside tests that intentionally exercise the new forms.
- Existing `--help` remains value-less for normal parsing; support for the exact `--help=json` form must not make `--help` consume a following positional argument.
- Root command, plugin routing, legacy OpenAPI routing, and unified runtime execution are unchanged.
- Default versions are reported, not unified: PascalCase and kebab requests may resolve different default versions by design.
- The service reads the same effective metadata source as the corresponding execution path and reports its selected version.

## Testing strategy

Implementation follows red-green-refactor. Tests cover:

1. parser recognition of both machine-help syntaxes and rejection of unsupported formats;
2. regression coverage proving plain `--help`, `-h`, and text `aliyun help ...` are unchanged;
3. deterministic root discovery output;
4. product output with multiple versions and distinct legacy/plugin defaults;
5. PascalCase and kebab resolution to Canonical API metadata;
6. correct active parameter set and both parameter trees;
7. RPC and ROA operations;
8. nested object, array, map, repeat-list, and body compatibility projections;
9. camel and kebab examples;
10. explicit nulls for unsupported output schema, pagination, risk, and recovery data;
11. stable JSON errors and non-zero exits;
12. proof that machine help does not load credentials, access the network, install plugins, or execute an API.

Targeted package tests run after each TDD cycle, followed by the repository's full Go test suite and build checks before push and PR creation.

## Delivery

- Base: latest `origin/unify-meta-test`.
- Source branch: `codex/agentic-help-json-v1` on the user's GitHub fork.
- Pull request target: `aliyun/aliyun-cli:unify-meta-test`.
- The PR is associated with Aone work item `85538412` where the available review tooling supports work-item linkage.
