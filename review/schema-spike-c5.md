# c5 disposable cycle-schema spike results

Status: **local structural tests pass; external corpus conformance unavailable.**
This is not production M1, not Technical Specification 0.1, and not an
implementation of the full CPU or supervisor.

## Test fixtures and provisional cycle expectations

The following are locally scripted model expectations, **not verified
SingleStepTests fixtures**. Physical bus order and 65C02 RMW dummy
behavior must be checked against the chosen corpus when available.

| Example | Initial PC / operand | X | Cycles in spike | Bus order (R=read, W=write, S=SYNC opcode read) |
|---|---|---:|---:|---|
| LDA # | $0200: A9 80 | 00 | 2 | S0200, R0201 |
| STA abs,X no cross | $0200: 9D 00 30 | 02 | 5 | S0200, R0201, R0202, R3002 dummy, W3002 |
| STA abs,X cross | $0200: 9D FF 30 | 02 | 5 | S0200, R0201, R0202, R3001 dummy, W3101 |
| INC abs,X no cross | $0200: FE 00 30 | 02 | 7 | S0200, R0201, R0202, R3002 dummy, R3002, R3002 dummy, W3002 |
| INC abs,X cross | $0200: FE FF 30 | 02 | 7 | S0200, R0201, R0202, R3001 dummy, R3101, R3101 dummy, W3101 |
| BRK | $0200: 00 EA, vector $5678 | 00 | 7 | S0200, R0201, W01FF, W01FE, W01FD, RFFFE, RFFFF |

The following unit tests pass for each example: instruction vs cycle vs
microoperation stepping equivalence (final CPU state and bus trace), snapshot
and resume at every microstep cursor, and preparation of next SYNC fetch
without resolving it. The RAM bus is copied alongside the CPU snapshot.

Basic RDY read stalls reissue authentic reads and preserve final CPU results
for ordinary RAM. Repeated reads against a side-effecting bus are deliberately
*not* assumed to preserve results. Snapshot and noninvasive inspection do
not issue transactions. Write-stall electrical behavior is deferred.

## Schema findings

- Explicit `PreBus(n) -> Resolve -> PostBus(n)` works for these templates.
- The resolved reply is retained inside value-only CPU state and is not
  resolved again when resuming from post-bus positions.
- Pre-bus labels are idempotent descriptions; mutations are post-bus.
- An RMW operand requires a retained operand latch distinct from the most
  recently resolved bus data (which dummy accesses overwrite).
- BRK can be an explicit exception without introducing a second engine.
- This spike uses dynamic per-cycle label slices and a switch for dispatch.
  That representation is intentionally not approved for production; its
  benchmark has allocations, and a declarative/reusable encoding still needs
  design review.
- The current per-cycle `Result.Ready` models only simple RDY read stalls.
  It does not attempt write-side electrical fidelity.

## Conformance classification

- Local unit/structural fixtures: **pass**.
- SingleStepTests selected 65C02 corpus: **unavailable, not run**.
- Dormann semantic test: **unavailable, not run**.
- Explicit corpus exceptions: **none claimed**.
- Full opcode compatibility: **not attempted**.

## Gate recommendation

Review the phase cursor, explicit operand latch, and dynamic schema costs
with the user and Claude before Technical Specification 0.1. Discard all
`internal/spike` and `cmd/m65c02-spike` code once findings are accepted.
Retain this report and the approved schema design, not the spike as M1.
