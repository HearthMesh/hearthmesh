# HearthMesh v0.1.0-prealpha

Initial technical demonstrator release.

## Included

- A standard-library Go node and command-line client.
- Persistent Ed25519 node and publisher identities.
- Signed, content-addressed HearthPacks with bounded ZIP validation.
- Explicit peer replication across three nodes.
- Periodic integrity checking and automatic repair from a valid peer.
- A local read-only node status portal and HTTP API.
- Docker Compose topology with separate persistent volumes.
- Automated demonstrations for POSIX shells and Windows PowerShell.
- An interactive public project explainer, brand assets, and the v0.1 High-Level Design.

## Acceptance status

The complete three-node Docker acceptance drill passed on Windows on 24 September 2026. It covered publication, replication, publisher-node loss, continued availability, deliberate corruption, and repair from a surviving peer. See [VALIDATION.md](VALIDATION.md) for the environment and exact evidence.

## Quick start

From `mvp/`:

```powershell
go test ./...
.\demo.ps1
```

Or start persistent Docker nodes:

```powershell
docker compose up --build
```

Then open `http://127.0.0.1:8081`, `http://127.0.0.1:8082`, and `http://127.0.0.1:8083`.

## Important limits

This is a pre-alpha demonstrator, not a production service. It has no publisher authorization, encrypted transport, peer authentication, DHT discovery, quotas, deletion or supersession protocol, private cells, mobile clients, or availability guarantee. All Docker nodes still share one physical host. Do not expose it directly to the public internet or use it for real emergency operations or sensitive data.

## License

Apache License 2.0. See [LICENSE](LICENSE).
