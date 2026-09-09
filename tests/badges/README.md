# Badge real-Fabric concurrency harness

This harness is intentionally separate from unit and browser tests. It never
resets the ledger and always creates uniquely named activities, series and (for
the bulk mode) Fabric identities.

Run a practical smoke pass against the existing local API and three-peer Fabric:

```bash
node tests/badges/fabric-concurrency.mjs
```

Run the release-gate load scenarios:

```bash
node tests/badges/fabric-concurrency.mjs \
  --rounds 50 --bulk-clients 100
```

`FAB-01` and `FAB-03` construct and endorse all competing proposals first, then
release them together to the orderer. Invalid MVCC transactions are reconciled
through the production badge service using the original idempotency key.

`FAB-02` provisions fresh opaque StudentMSP identities directly through the
local CA, runs the real activity/ticket/check-in path, and supports any client
count of at least two. The requested release value is 100.

`FAB-05` bypasses Express and directly invokes administrative chaincode methods
with student, organizer, operator and verifier certificates. Every attempt must
be rejected without changing the series count.

The JSON output explicitly lists `FAB-04` fault injection and `FAB-06` upgrade
history as untested. They require controlled process/network lifecycle hooks and
must not be inferred from this harness.
