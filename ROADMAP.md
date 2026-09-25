# Roadmap

## Stage 0 — technical demonstrator (completed 24 September 2026)

- [x] signed, content-addressed HearthPacks;
- [x] explicit three-node replication;
- [x] integrity detection and repair;
- [x] local read-only status portal;
- [x] Docker Compose demonstration;
- [x] Windows manual and automated test path.

Exit evidence: all Go tests pass; the three-node single-host drill demonstrated publication, replication, continued availability after node A stopped, corruption detection on node B, and repair from node C. See `VALIDATION.md`. Stage 0 completion does not imply production readiness.

## Stage 1 — safe local pilot

- mutual TLS between enrolled nodes;
- invitation-based publisher enrollment and role policy;
- quotas, rate limits, audit events, and basic observability;
- retention classes and signed supersession records;
- reproducible releases and an independent security review.

## Stage 2 — federated regional pilot

- resilient peer discovery without anonymous routing;
- organization/community cells connected to the public mesh;
- encrypted private packs with explicit recipient groups;
- governance, abuse response, publisher revocation, and key recovery;
- degraded-network synchronization and bandwidth budgets.

## Stage 3 — public continuity network

- multi-region seed operators and transparent service health;
- interoperable node implementations and protocol versioning;
- verified emergency-source directories;
- independent governance and ongoing adversarial testing.

Each stage requires exit criteria for security, abuse handling, reliability, and operator accountability before the next begins.
