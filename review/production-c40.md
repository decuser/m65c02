# C40: Separate Visible Computer from KIM-1 logic analyzer

- Removed MICRO TRACE toggle and internal CPU-state dump from `/analyzer`. It remains focused on external CPU bus/control and RIOT pins.
- Added `/visible`, a distinct CPU-observation interface listing physical bus cycles and their executed pre-bus, bus, and post-bus micro-operations with register/latch deltas.
- Added a VISIBLE COMPUTER launch button to the KIM-1 front panel.
- Microstep capture is opt-in in `/visible`; the ordinary `/api/trace` response no longer includes internal events unless explicitly requested using `?micro=1`. This avoids the large internal-state payload during ordinary logic-analyzer refreshes.
- The CPU callback and bounded internal history from C39 remain intact. No CPU or RIOT execution semantics changed.

## Validation

- All three embedded HTML scripts parsed successfully in Node.js.
- Server route test checks `/visible`.
- `make validate` and `make validate-kim1` passed.
- Manual user review of the Visible Computer layout remains to be done.
