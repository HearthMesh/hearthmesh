# Contributing to HearthMesh

HearthMesh is an experimental pre-alpha project. Contributions should preserve its narrow claim: attributable signed public content, explicit peers, deterministic verification, and no anonymity or production-security promises.

## Development setup

Install Go 1.22 or later. From `mvp/`, run:

```sh
go test ./...
go vet ./...
```

Use `sh demo.sh` on Linux or macOS, or `.\demo.ps1` in Windows PowerShell, for the three-node acceptance drill. Docker Desktop or Docker Engine with Compose is required only for the persistent container topology.

## Pull requests

- Keep changes focused and explain the behavior or risk they address.
- Add or update tests for protocol, verification, storage, or recovery changes.
- Run `gofmt` on changed Go files.
- Update the HLD, security limits, validation record, or release notes when claims or boundaries change.
- Do not commit publisher keys, node data, generated executables, or test packs.

## Security-sensitive changes

Treat identity, signature verification, archive parsing, peer transport, authorization, and repair logic as security-sensitive. The current MVP has not received an independent security audit and must not be exposed directly to the public internet.
