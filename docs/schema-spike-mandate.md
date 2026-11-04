# Disposable schema spike: mandate for next session

**Not production cut M1.** Four exemplar cycle templates must pressure-test the proposed interfaces **before** Technical Specification 0.1 freezes the representation. Discard implementation source afterward; retain test results, counterexamples, the conformance/evidence map, and the proven schema in the spec.

## Templates/workloads

1. `LDA #imm`: minimal opcode/operand fetch, post-bus transfer, SYNC instruction boundary.
2. `STA abs,X`: indexed store, correct fixed/dummy intermediate access even without a crossing; with a crossing if useful.
3. `INC abs,X`: read-modify-write, run **twice**: no page crossing and page crossing. Provisional Apple/NCR behavior is **6 (+1 crossing)** pending exact datasheet interpretation; do NOT turn a WDC 7-cycle baseline into an NCR requirement. Compare period documents before corpus interpretation.
4. `BRK`: stack pushes, vector reads, P/PC changes, interrupt-entry mechanics, SYNC convention. Do not silently assert BRK == IRQ/NMI physically; document what tests do and do not establish.

## Contracts to exercise

- One phase cursor: `PreBus(n)`, `Resolved`, `PostBus(n)`; resolved read data and sampled controls stored in CPU state; once resolved, resuming never re-resolves the same transaction.
- Idempotent pre-bus teaching operations and non-idempotent post-bus changes; no full shadow state.
- Present/Resolve/Commit separately callable, plus `Clock` convenience operation (using the same engine).
- Microstep pause and inspection at every phase without time/device advance.
- CPU state as plain snapshotable data; full mid-cycle snapshot restoration.
- Stable instruction completion at next prepared SYNC fetch before Resolve, subject to documented exceptions.
- Class-aware corpus harness: `Authoritative`, `Transferred`, `Excluded`, `Unresolved`. Classified known divergence is **not** an ordinary regression failure.

## Property tests

1. **Step equivalence**: whole-instruction stepping, whole-cycle stepping, and one-microop-at-a-time stepping produce identical final machine state and bus transaction list.
2. **Snapshot round-trip**: snapshot at every cursor location (including Resolved and each PostBus microop); restore and run; no duplicate Resolve or changed result.
3. **RDY read-stall**: randomized/scripted RDY-low insertion in a deterministic, side-effect-free RAM machine, with fixed IRQ/NMI/RESET inputs; same final architectural state; trace = unstalled trace with precisely repeated held-address accesses inserted.
4. **Side-effect device**: repeat a read-to-clear or input FIFO status access during stalls; show repeated authentic accesses, not an erroneous attempt to suppress them. Do not require equal device state in stalled vs unstalled runs.
5. **Nominal vs crossed `INC abs,X`**: compare selected reference and log each corpus variance with exact chip variant, source and classification.
6. **Operator observational noninterference**: inspect pending cycle without causing additional bus access.

## Evidence report required

Produce a short table for each workload: expected cycles from Apple Table A-1; NCR manufacturer's timing and applicable footnotes; applicable variant test vectors; observed cycles/transactions; evidence category; discrepancies; whether phase/schema design survived. Highlight RDY write stall bus-data details and interrupt sampling as **open** unless target-chip evidence resolves them.

## Exit gate

A successful spike proves the contract can express four representative sequences and all property tests without hidden execution state, duplicating bus accesses on resume, or introducing special-case shadow-state machinery. A failure is an architectural finding to discuss, not something to conceal with a convenient fallback. If the templates demonstrate a different phase ordering, bring it back before rewriting the frozen baseline.
