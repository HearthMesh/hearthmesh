# HearthMesh pre-alpha MVP

Experimental, English-only, public-pack prototype. **Not production-secure.** One Go binary, standard library only, no AI, no anonymity/onion routing. A node persists an Ed25519 identity in `identity.key`; a publisher uses its own persistent key. A HearthPack is a ZIP containing `manifest.json` and `files/…`. The manifest lists publisher public key, sorted relative paths, file sizes and SHA-256 hashes, and an Ed25519 signature over canonical JSON manifest fields. Its ID is SHA-256 of canonical JSON including the signature.

## Quick start

Requires Go 1.22+. On Linux or macOS, the demo also requires `curl`:

```sh
go test ./...
sh demo.sh
```

On Windows PowerShell:

```powershell
go test ./...
.\demo.ps1
```

The demos start three loopback nodes using explicit peers, create and sign a pack, publish to A, wait for B and C to replicate, stop A, corrupt B's stored ZIP, and wait for B to recover a verified copy from C. Demo data is temporary and removed on exit.

Run nodes in separate terminals, or use Docker Compose (`docker compose up --build -d`):

```sh
go build -o hearthmesh ./cmd/hearthmesh
./hearthmesh serve -data ./node-a -listen 127.0.0.1:8081 -peers http://127.0.0.1:8082,http://127.0.0.1:8083
./hearthmesh serve -data ./node-b -listen 127.0.0.1:8082 -peers http://127.0.0.1:8081,http://127.0.0.1:8083
./hearthmesh serve -data ./node-c -listen 127.0.0.1:8083 -peers http://127.0.0.1:8081,http://127.0.0.1:8082
```

Publish a directory (in a separate terminal):

```sh
./hearthmesh identity -key ./publisher.key
./hearthmesh pack -src ./public-files -out ./public.zip -key ./publisher.key
./hearthmesh verify -pack ./public.zip
./hearthmesh publish -pack ./public.zip -node http://127.0.0.1:8081
```

Create `public-files` first and put at least one regular file in it. Use the same `publisher.key` for later publications under that identity. The publisher private key is never included in a pack. Docker Compose exposes portals/API at [node A](http://127.0.0.1:8081), [node B](http://127.0.0.1:8082), and [node C](http://127.0.0.1:8083). Compose uses separate persistent volumes, and the three services name each other as bootstrap peers. `docker compose down` stops the nodes without deleting their volumes.

## HTTP API

| Route | Method | Purpose |
| --- | --- | --- |
| `/` | GET | Read-only local status portal |
| `/v1/status` | GET | Node public key, peers, verified pack IDs, corrupt pack IDs |
| `/v1/packs` | GET | List verified pack IDs |
| `/v1/packs/{64-digit-lowercase-hex-id}` | GET | Download ZIP only if still verified |
| `/v1/packs/{id}` | PUT | Publish a ZIP; accepted only if signature, all hashes, and ID verify |

The node checks local packs each sync cycle (default two seconds) and at serving time. It discovers IDs from explicit peers, downloads missing or corrupt packs, checks signatures/hashes locally, then replaces damaged files atomically. It has no DHT, implicit peer discovery, mutable content, or deletion propagation.

## Security and scope limits

- **Public only:** anyone with HTTP access can publish, list, or download; there are no accounts, quotas, authorization, or publisher trust policy. Restrict the listener to trusted networks. The default `serve` listener is loopback-only; widening it is an explicit operator decision.
- **No transport security:** HTTP permits traffic observation, blocking, or replay. Signatures detect content tampering, not peer identity, freshness, availability, or malicious-but-valid publications. Explicit peers are not authenticated.
- ZIP input is capped at 32 MiB (also bounds individual entries and total declared uncompressed file bytes) and 256 files. This is only a basic resource limit; status scans read and verify all packs on each request, and there is no total storage cap.
- Identity files are written with mode `0600`, but no encrypted key storage, rotation, revocation, fsync/durable transactions, or backup mechanism exists. Treat a lost publisher key as permanent identity loss.
- No sandboxing or extraction: a verified ZIP is stored and served; its files are not executed or rendered by the portal. Clients should still treat downloaded data as untrusted.
- Replication is best-effort polling and depends on a reachable honest peer holding a valid copy. There is no consensus or availability guarantee. Local corrupt files remain marked corrupt if repair fails.
- Repository CI lives at `../.github/workflows/ci.yml` and tests the MVP on Linux and Windows, with an additional Linux race-detector job.

## Validated environment

The complete manual and Docker acceptance drill passed on Windows x86-64 on 24 September 2026 with Go 1.27.1, Docker Engine 29.8.0, Docker Compose 5.5.1, and WSL 2.7.14.0. See the repository-level `VALIDATION.md`. This is single-host functional evidence, not a security audit or production certification.
