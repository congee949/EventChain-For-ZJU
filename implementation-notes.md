# Implementation Notes

## Design Decisions

- V2 targets the course-demo boundary. All seeded assets are explicitly non-monetary and the prediction flow defaults to non-redeemable `B_bonus`.
- The existing ledger will be reset and reseeded because V1 certificates expose student IDs and V1 AMM shares cannot be converted faithfully into pari-mutuel positions.
- Financial claims remain pending for seven days. Corrections replace the settlement epoch while claims are still reversible; matured balances are never made negative.
- A new `finance` chaincode owns A/B balances, positions, escrow shards, claims, service orders, and financial invariants.
- A new `activity` chaincode consolidates V2 event lifecycle, ticket lottery, QR claim, and check-in. The legacy `event`, `prediction`, `ticket`, and `token` chaincodes are retired from the V2 API; prediction accounting belongs exclusively to `finance`.
- Amounts use integer micro-units and API decimal strings. Floating-point arithmetic is forbidden in chaincode.
- Public self-registration is disabled by default. Course-demo identities are roster-bootstrapped into opaque Fabric account IDs; production must replace the demo bootstrap key with campus SSO or another authoritative identity source.
- The supported local runtime is Node 22. Fabric images are pinned to 2.5.15 because the current Docker Desktop rejects the API level used by the older peer image.
- The frontend keeps the original information architecture (`Home`, event detail, ticket hall, profile, and admin console) and adapts those views to V2 stores. The first V2-only views remain as implementation references but are no longer the primary routes.
- The design handoff is treated as a visual contract, not source code. Its editorial tokens and presentational primitives are reimplemented as Vue SFCs while the existing V2 stores, API calls, permissions, and route contracts remain unchanged.

## Deviations

- Paid and bonus funds are not mixed in one prediction pool. Each market selects exactly one stake bucket; the default is `B_bonus`. This prevents promotional points from becoming redeemable `B_paid` through winnings.
- The previously discussed 16 claim shards are logical private-state keys inside one PDC, not 16 collections.
- The legacy leaderboard, synthetic odds history, and public accuracy tracking are not restored during frontend fusion because V2 deliberately does not expose private positions or smart-money activity. Pages show only authoritative public snapshots and the current user's private balances/applications.
- Google Fonts are loaded from the public stylesheet endpoint to match the handoff typography. The CSS includes generic fallbacks, but a fully offline demo would need the three font families vendored locally.

## Tradeoffs

- `finance` private data is shared with PlatformMSP and StudentMSP peers and hidden from OrganizerMSP. This protects users from organizer-level tracking but does not hide data from peer administrators.
- Seven-day pending claims avoid either a 100% correction reserve or clawbacks, at the cost of delayed spendability.
- Check-in reward issuance is a two-step idempotent saga (`activity.CheckIn` then `finance.GrantCheckInBonus`) because separate gateway submissions cannot commit atomically. The API must expose retry/reconciliation instead of presenting the two commits as atomic.
- API throttling is process-local for the single-instance course demo. A multi-instance deployment needs a shared rate-limit store.

## Open Questions

- None. The user selected the recommended defaults: course Demo, reset/reseed, and seven-day pending claims.
