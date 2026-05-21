# cliext/maxc Runbook

End-to-end verification log for the `aliyun maxc` launcher. Refer back here
when reproducing release smoke tests or debugging install-time issues.

## OSS layout (spec § 3)

```
maxcompute-repo.oss-cn-hangzhou.aliyuncs.com/maxc-cli/
├── versions/latest                    # plain text, single version string
└── <version>/<platform>/
    ├── maxc.tar.gz                    # PyInstaller onedir, top dir = maxc/
    └── maxc.tar.gz.sha256             # hex digest, single line
```

`<platform>` is one of: `linux-amd64`, `linux-arm64`, `darwin-amd64`,
`darwin-arm64`, `windows-amd64`, `windows-arm64`.

## Build a tarball locally

```bash
cd <maxc-cli repo>
bash scripts/build_release.sh
# → dist/release/maxc.tar.gz
# → dist/release/maxc.tar.gz.sha256
```

Produces a native-platform onedir bundle. Tar entries preserve the
top-level `maxc/` directory; `COPYFILE_DISABLE=1` strips macOS
AppleDouble noise.

## Phase 4.1 — first-install smoke

Tested 2026-05-21 on darwin-arm64, Python 3.11, against a local
http.server fake serving the layout above.

1. Build the parent CLI with the maxc cliext registered:

   ```bash
   cd <aliyun-cli repo>
   go build -o /tmp/aliyun ./main/
   ```

2. Wipe any existing install so we exercise the first-install path:

   ```bash
   rm -rf ~/.aliyun/maxc ~/.aliyun/maxc.old.*
   ```

3. Stage a local OSS-shaped tree and serve it:

   ```bash
   mkdir -p /tmp/maxc-staging/{0.0.0-local/darwin-arm64,versions}
   cp dist/release/maxc.tar.gz dist/release/maxc.tar.gz.sha256 \
      /tmp/maxc-staging/0.0.0-local/darwin-arm64/
   echo -n "0.0.0-local" > /tmp/maxc-staging/versions/latest
   ( cd /tmp/maxc-staging && python3 -m http.server 8765 )
   ```

4. Run the launcher pointing at the local mirror:

   ```bash
   ALIBABA_CLOUD_MAXC_DOWNLOAD_BASE_URL=http://localhost:8765 \
     /tmp/aliyun maxc --version
   ```

   Expected output: `maxc 0.2.5` (the bundled maxc-cli version).

5. Verify install side-effects:

   ```bash
   ls -la ~/.aliyun/maxc/
   file ~/.aliyun/maxc/maxc       # → Mach-O 64-bit executable arm64
   cat ~/.aliyun/maxc/.version    # → 0.0.0-local
   ```

## Phase 4.1 — real query against MaxCompute

Requires a configured aliyun profile with MaxCompute access. We used
the active default profile pointing at project `bird` in cn-shanghai.

```bash
ALIBABA_CLOUD_MAXC_NO_UPDATE_CHECK=1 /tmp/aliyun maxc auth whoami
```

Returned a populated envelope with `status: success`, principal masked
as `LTAI***5m7j`, project `bird`, endpoint cn-shanghai. Credentials
were sourced from the parent aliyun profile (zero overlap with
`~/.maxc/config.yaml` — the launcher injected
`ALIBABA_CLOUD_ACCESS_KEY_ID/SECRET` + `MAXCOMPUTE_REGION` directly).

```bash
ALIBABA_CLOUD_MAXC_NO_UPDATE_CHECK=1 /tmp/aliyun maxc query "select 1 as v" --json
```

Returned a populated envelope:

```json
{
  "status": "success",
  "data": { "result": { "rows": [{"v": 1}], "row_count": 1 } },
  "metadata": { "project": "bird", "elapsed_ms": 6000 }
}
```

## Phase 4.4 — update + offline-fallback smoke

1. Stale cache, latest unchanged → silent no-op:

   ```bash
   touch -t 202001010000 ~/.aliyun/maxc/.version_check
   ALIBABA_CLOUD_MAXC_DOWNLOAD_BASE_URL=http://localhost:8765 \
     /tmp/aliyun maxc --version
   ```

   No stderr, exit 0, install untouched.

2. Stale cache, latest bumped → triggers re-download:

   ```bash
   mkdir -p /tmp/maxc-staging/0.0.0-local2/darwin-arm64
   cp .../maxc.tar.gz{,.sha256} /tmp/maxc-staging/0.0.0-local2/darwin-arm64/
   echo -n "0.0.0-local2" > /tmp/maxc-staging/versions/latest
   touch -t 202001010000 ~/.aliyun/maxc/.version_check
   ALIBABA_CLOUD_MAXC_DOWNLOAD_BASE_URL=http://localhost:8765 \
     /tmp/aliyun maxc --version
   cat ~/.aliyun/maxc/.version   # → 0.0.0-local2
   ```

3. Stale cache + unreachable server → warn but continue:

   ```bash
   touch -t 202001010000 ~/.aliyun/maxc/.version_check
   ALIBABA_CLOUD_MAXC_DOWNLOAD_BASE_URL=http://127.0.0.1:1/dead \
     /tmp/aliyun maxc --version
   ```

   stderr: `maxc: update check failed: ... connection refused`
   stdout: `maxc 0.2.5`
   exit: 0

## Env-var contract (spec § 4)

| Var | Effect |
|---|---|
| `ALIBABA_CLOUD_MAXC_DOWNLOAD_BASE_URL` | Replace the OSS base URL entirely (testing / private mirror) |
| `ALIBABA_CLOUD_MAXC_EXEC_PATH` | Bypass install completely — run this binary directly (BYO) |
| `ALIBABA_CLOUD_MAXC_NO_UPDATE_CHECK=1` | Skip the once-per-day latest-pointer fetch |

## Credentials injected into the child

| Var | Source |
|---|---|
| `ALIBABA_CLOUD_ACCESS_KEY_ID` | active profile (or STS exchange) |
| `ALIBABA_CLOUD_ACCESS_KEY_SECRET` | active profile (or STS exchange) |
| `ALIBABA_CLOUD_SECURITY_TOKEN` | only when profile mode is StsToken / RamRoleArn / etc. |
| `MAXCOMPUTE_REGION` | `profile.RegionId` when non-empty |

Profile resolution happens via `config.LoadProfileWithContext`, so
`--profile <name>`, `ALIBABA_CLOUD_PROFILE` env, and `default`
fall-through all work identically to the rest of the aliyun CLI.

## Parent-only flag stripping

These flags are consumed by the parent and **not** forwarded to the
child (full list in `stripFlags` in `maxc.go`):

- credential / auth: `--profile`, `--mode`, `--access-key-*`,
  `--ram-role-*`, `--sts-*`, `--oidc-*`, `--cloud-sso-*`, …
- runtime knobs: `--config-path`, `--read-timeout`, `--connect-timeout`,
  `--retry-count`, `--skip-secure-verify`, `--endpoint-type`, `RegionId`
- openapi-only: `--secure`, `--insecure`, `--header`, `--pager`,
  `--accept`, `--waiter`, `--dryrun`, `--quiet`, `--yes`, `--cli-query`,
  `--roa`, `--method`, `--user-agent`, `--cli-ai-mode`,
  `--no-cli-ai-mode`

All other flags (including shared names like `--region`, `--output`,
`--endpoint`) pass through so the child handles them with its own
semantics.

## Known caveats

- **Exit-code translation.** When the child exits non-zero we surface
  an `*ExitError{Code:N}` from `Context.Run`. The aliyun CLI's central
  `processError` doesn't currently honour `ExitCode()`, so the parent
  always exits 1 (not N). Same trade-off as `cliext/cms2`; addressing
  this requires a small patch to `cli/command.go:processError`.
- **macOS / Windows runners.** The Aone CI pipeline only has linux
  hosted runners. Full 6-platform builds run on GitHub Actions (see
  `.github/workflows/release.yml` in the maxc-cli repo).
- **Bucket state.** As of 2026-05-21 `maxc-cli/versions/latest` and the
  darwin/windows tarballs are not yet published. Local smoke uses the
  http.server-based mirror above. The first real release will populate
  the bucket via Phase 2 CI.

## Phase 4.2 — cross-platform [HUMAN, deferred]

Awaits the GitHub Actions matrix run + a clean VM/container for each
of linux-amd64 / darwin-arm64 / windows-amd64. Append a row per
machine: OS, build commit, date, OK/FAIL.

## Phase 4.3 — STS / RamRoleArn [HUMAN, deferred]

Awaits a real RAM role + parent AK. Expected check: `aliyun --profile
sts-role maxc auth whoami` returns the assumed role's principal, not
the parent AK.
