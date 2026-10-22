# Concurrent OAuth credential refresh

Multiple CLI processes can load the same expired profile and consume the same
rotating refresh token. Atomic file replacement alone does not serialize this
read/refresh/write transaction. This change adds a configuration-file lock and
rereads the profile under that lock, so waiting processes reuse the winner's
credentials. Different profiles in one file cannot overwrite each other's updates.

The lock resolves file/directory symlinks and uses flock on Unix and LockFileEx on
Windows. A persistent sidecar inode prevents split-lock races. Process termination
releases the OS lock. Acquisition times out after 30 seconds.

OAuth and STS refresh starts 60 seconds before expiry. An omitted refresh_token
retains the existing token. Invalid or expired STS tuples are rejected. A rotated
OAuth generation is saved before the STS exchange, allowing another process to
recover if the first process dies during that exchange. A failed checkpoint stops
the STS request. Credential fields alone are merged onto the on-disk profile:
request region/endpoint overrides are not saved. The existing selected config-path
behavior is preserved.

The protocol coordinates participating OAuth refreshers, not old binaries,
CloudSSO or interactive configuration editors. Copies of the same rotating token
in different configuration files do not share a lock. A lost OAuth refresh
response or permanently unwritable disk can still require reauthentication.

## Scope of the fix

This fixes client-side OAuth credential lifecycle and persistence. It does not
change MaxCompute server-side principal resolution or make direct overseas
MaxCompute access work with every OAuth-issued STS credential.

In the retained 2026-10-21 tests, official-cli OAuth STS passed STS
GetCallerIdentity in Beijing, US East and Singapore, but direct MaxCompute
requests in US East and Singapore returned INVALID$v4_<USER_ID> while the same
RAM user's AK and Beijing OAuth STS resolved to p4_<USER_ID>. The exact
server-side cause remains unconfirmed; this is not evidence that STS is generally
restricted to China. These observations have not been revalidated as current
service behavior by this documentation update.

An upstream PR for this change should state that it fixes refresh races and
rotating-token persistence, not overseas MaxCompute authorization. Detailed
product-specific investigation belongs in a separate service issue, not in this
fix's acceptance criteria. No MaxCompute access claim follows from refresh tests.

## Verification

Run with a temporary HOME, retaining the development Go cache paths:

```sh
GOPATH="$(go env GOPATH)" GOCACHE="$(go env GOCACHE)" HOME="$(mktemp -d)" \
  go test -race ./config
```

Tests use synthetic credentials and local HTTP servers. Six subprocesses read a
stale snapshot before a barrier is released; only one refresh per profile may
consume the token. Coverage includes expired/near-expiry credentials, two profiles,
symlink aliases, process death during STS exchange, lock-owner death, malformed
responses, checkpoint failure and preservation of unrelated profile settings.
No MaxCompute SDK, Python runtime or CLI extension is required by this change.
