# Security policy

HearthMesh is a pre-alpha demonstrator and has not received a security audit. Do not expose the MVP directly to the public internet and do not use it for confidential, personal, regulated, or operationally critical information.

Known limits include unauthenticated publishing, plain HTTP transport, no quota enforcement, no publisher trust policy, no revocation, no secure key store, and no availability guarantee. Signed content proves integrity and possession of a publisher key; it does not prove that the publisher is trustworthy or that the information is true.

For a future coordinated disclosure process, publish a dedicated security contact before accepting external deployments.

Functional validation evidence is recorded in `VALIDATION.md`. That evidence must not be interpreted as a security audit, penetration test, or production certification.
