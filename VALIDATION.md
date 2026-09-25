# Validation record

## v0.1.0-prealpha — 24 September 2026

The HearthMesh pre-alpha MVP was exercised locally on Windows x86-64 with:

- Go 1.27.1 (`windows/amd64`);
- Docker Engine 29.8.0;
- Docker Compose 5.5.1;
- Windows Subsystem for Linux 2.7.14.0.

The following checks completed successfully:

1. All four Go tests passed, including identity persistence, tamper rejection, three-node replication and repair, and unexpected ZIP entry rejection.
2. The Windows executable built successfully.
3. A persistent Ed25519 publisher identity was created.
4. A one-file HearthPack was signed and verified. Its pack ID was `624d2df5a2b2a5c20ff1ac37ef45e6c9558b71e2600adc318ec7741220cadbc0`.
5. In a manual two-node run, the pack replicated to the peer, deliberate local corruption was detected, and the peer restored a verified copy.
6. Docker Compose built and started three separate node containers with separate persistent volumes.
7. Publishing to node A resulted in the same verified pack appearing on nodes A, B, and C.
8. After node A was stopped, nodes B and C continued to serve the verified pack.
9. With node A still offline, node B's stored ZIP was deliberately corrupted. Node B detected the invalid copy and repaired it from node C.
10. Docker Compose shut down cleanly without deleting the named data volumes.

This record documents a user-observed local acceptance run. It is not an independent security audit, a production certification, or evidence of resilience across separate physical failure domains.
