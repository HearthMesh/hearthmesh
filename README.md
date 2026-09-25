# HearthMesh

**Version:** `0.1.0-prealpha`

**Tagline:** When Infrastructure Fails, Knowledge Remains.

HearthMesh is an experimental, identity-aware continuity network: a global public mesh built node by node, with optional private organizational cells. Its purpose is to keep selected, signed emergency information available when ordinary infrastructure becomes unreliable.

This pre-alpha package contains:

- `mvp/` — a three-node Go prototype with signed HearthPacks, replication, integrity checking, and automatic repair;
- `site/` — the English project explainer and interactive failure demonstration;
- `brand/` — SVG and PNG logo assets plus concise usage guidance;
- `docs/` — the High-Level Design in DOCX and PDF, plus its source generator.

## Try the MVP

Requires Go 1.22+. On Linux or macOS, the automated demo also requires `curl`:

```sh
cd mvp
go test ./...
sh demo.sh
```

On Windows PowerShell:

```powershell
cd mvp
go test ./...
.\demo.ps1
```

Both demonstrations build the binary, start three loopback nodes, publish and replicate a signed pack, take the publisher node offline, deliberately corrupt another node's copy, and verify repair from the surviving peer.

Or run the three persistent nodes with Docker Compose:

```sh
cd mvp
docker compose up --build
```

Open `http://127.0.0.1:8081`, `:8082`, and `:8083` for the local read-only status portals.

## What this version proves

The prototype demonstrates content-addressed, Ed25519-signed public packs moving between explicitly configured peers. Every receiving node verifies the publisher signature, manifest, file sizes, and SHA-256 hashes. If a local copy is corrupted, a node can replace it with a verified copy from another peer.

It does **not** yet provide production authentication, encrypted transport, DHT discovery, quotas, deletion, private cells, mobile clients, or a global operational network. See `mvp/README.md` and the HLD for the exact boundaries.

The full three-node acceptance drill passed on Windows on 24 September 2026 with Go 1.27.1, Docker Engine 29.8.0, and Docker Compose 5.5.1. The result is recorded in [VALIDATION.md](VALIDATION.md); it is not an independent security audit or a multi-host resilience test.

## Product position

HearthMesh is not anonymous storage, an onion network, or a generic file-sharing service. Public publishing is intended to be attributable. Private enterprise or community cells are a later layer, not a replacement for the shared global public mesh.

## Status

Pre-alpha technical demonstrator. Do not use it for real emergency operations or sensitive data.

See the [release notes](RELEASE_NOTES_v0.1.0-prealpha.md), [roadmap](ROADMAP.md), and [security policy](SECURITY.md). Released under the [Apache License 2.0](LICENSE).
