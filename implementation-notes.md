# Implementation Notes

## Design Decisions

- The ZIP's Vue runtime sources already match `client/src` byte-for-byte. The exported design HTML, thumbnail, and `support.js` are treated as design-tool artifacts rather than application runtime code, so they are not copied into the production bundle.
- Ticket applications and issued tickets remain separate domain objects. The API exposes the current user's applications separately instead of fabricating `PENDING` or `WON` ticket objects.
- Event list responses are enriched with prediction odds and pool data on a best-effort basis because the redesigned home cards consume those fields. Events remain visible if prediction state is unavailable.
- UI state restrictions are mirrored at the API boundary: bets require `PREDICTION_OPEN`, while applications and lotteries require `TICKET_OPEN`; lottery size cannot exceed the event allocation.
- Fabric runtime images are pinned to 2.5.15 because that LTS patch is the first 2.5 release compatible with Docker Engine 29+; the previous floating 2.5 image resolved locally to 2.5.10 and could not install chaincode.
- The Vite API proxy keeps port 3000 as its default but accepts `VITE_API_PROXY_TARGET` so local integration can coexist with another service already bound to that port.

## Deviations

- No source overwrite was performed because it would produce no changes; integration work focuses on the mismatched frontend/backend contracts discovered during verification.

## Tradeoffs

- Enriching the event list adds up to two read-only Fabric queries per event. This is acceptable for the small demo dataset and keeps the frontend contract simple; a production system should batch, cache, or expose a dedicated market-summary query.

## Open Questions

- None.
