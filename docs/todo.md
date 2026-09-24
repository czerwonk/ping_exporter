# TODO

## Current/upcoming tasks

- [ ] (medium) Verify multi-arch distroless build (arm64, arm/v7) in CI, not just local amd64
- [ ] (low) Consider a `nonroot` distroless variant (`:nonroot` tag); file capability (`cap_net_raw`) works independent of UID, so this should be compatible

## Completed tasks

- [x] Switch runtime image from `alpine:3.24.2` to `gcr.io/distroless/static-debian13` (pinned by digest)
- [x] Move `setcap cap_net_raw+ep` into the builder stage (distroless has no shell/package manager to run it in the final stage)
- [x] Replace shell-form `CMD` (relied on `/bin/sh` for `$CONFIG_FILE`/`$CMD_FLAGS` expansion) with exec-form `ENTRYPOINT`; app now reads `CONFIG_FILE` via kingpin `Envar` and parses `CMD_FLAGS` itself (`main.go`)
- [x] Fixed a pre-existing break in `Dockerfile` (introduced in `b108307`, same day): `COPY internal ./internal` / `COPY pkg ./pkg` referenced directories that don't exist in this repo, and `config/` (imported by `main.go`) was never copied at all — build was broken on `main`

## Development decisions + rationale

- File capabilities (`security.capability` xattr) **do** survive `COPY --from=builder` across build stages on this BuildKit/podman setup — verified with `getcap` on the extracted binary post-copy. Confirmed via local testing 2026-09-24, not a general guarantee across all builders/storage drivers.
- `CMD_FLAGS` is split with `strings.Fields` (plain whitespace splitting, no shell quoting semantics) — matches prior de-facto usage and keeps it simple; document if callers need quoted args.

## Recurring issues + solutions

- Local docker/podman here is **rootless**: file-capability-granted binaries (`cap_net_raw` for ICMP) fail with "Operation not permitted" unless the container is run with `--cap-add=NET_RAW`. This reproduces identically on both the old alpine image and the new distroless image — it's an environment limitation, not a regression. Production (non-rootless) runtimes are expected to honor `setcap` normally.

## Technical debt

- None new from this change.
