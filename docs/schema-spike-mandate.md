# Disposable schema spike: mandate for next session

**Not production cut M1.** Four exemplar cycle templates must pressure-test the proposed interfaces **before** Technical Specification 0.1 freezes the representation. Discard implementation source afterward; retain test results, counterexamples, the conformance results and any documented exceptions, and the proven schema in the spec.

## Templates/workloads

1. `LDA #imm`: minimal opcode/operand fetch, post-bus transfer, SYNC instruction boundary.
2. `STA abs,X`: indexed store, correct fixed/dummy intermediate access even without a crossing; with a crossing if useful.
3. `INC abs,X`: read-modify-write, run **twice**: no page crossing and page crossing. Use the selected SingleStepTests 65C02 timing and transaction expectations for both cases when available. If unavailable, explicitly mark provisional expectations as such; do not claim corpus conformance.
4. `BRK`: stack pushes, vector reads, P/PC changes, interrupt-entry mechanics, SYNC convention. Do not silently assert BRK == IRQ/NMI physically; document what tests do and do not establish.

## Contracts to exercise

- One phase cursor: `PreBus(n)`, `Resolved`, `PostBus(n)`; resolved read data and sampled controls stored in CPU state; once resolved, resuming never re-resolves the same transaction.
- Idempotent pre-bus teaching operations and non-idempotent post-bus changes; no full shadow state.
- Present/Resolve/Commit separately callable, plus `Clock` convenience operation (using the same engine).
- Microstep pause and inspection at every phase without time/device advance.
- CPU state as plain snapshotable data; full mid-cycle snapshot restoration.
- Stable instruction completion at next prepared SYNC fetch before Resolve, subject to documented exceptions.
- Profile-aware corpus harness: filter excluded Rockwell bit operations and WAI/STP; report passes, genuine failures, unavailable vectors and explicitly documented exceptions separately. Do not require a provenance matrix.

## Property tests and exploratory checks

1. **Step equivalence**: whole-instruction stepping, whole-cycle stepping, and one-microop-at-a-time stepping produce identical final machine state and bus transaction list.
2. **Snapshot round-trip**: snapshot at every cursor location (including Resolved and each PostBus microop); restore and run; no duplicate Resolve or changed result.
3. **RDY read-stall (basic, nonblocking)**: randomized/scripted RDY-low insertion in a deterministic, side-effect-free RAM machine, with fixed IRQ/NMI/RESET inputs; same final architectural state; trace = unstalled trace with precisely repeated held-address accesses inserted.
4. **Side-effect device (exploratory)**: repeat a read-to-clear or input FIFO status access during stalls; show repeated authentic accesses, not an erroneous attempt to suppress them. Do not require equal device state in stalled vs unstalled runs.
5. **Nominal vs crossed `INC abs,X`**: compare both cases to the selected corpus; record genuine differences and any explicit compatibility exception.
6. **Operator observational noninterference**: inspect pending cycle without causing additional bus access.

## Spike report required

For each workload report test input, selected corpus expectation (or unavailable status), observed cycles and bus transactions, stepping equivalence, snapshot/Resolve-once outcome, and whether the schema survived. Explain any difference without initiating chip-provenance research. Keep deferred RDY write-stall and unusual interrupt edge cases out of the exit gate.

## Exit gate

A successful spike proves the contract can express four representative sequences and the core property tests (stepping equivalence, mid-cycle snapshot, no duplicate Resolve and observer noninterference) without hidden execution state, duplicating bus accesses on resume, or introducing special-case shadow-state machinery. A failure is an architectural finding to discuss, not something to conceal with a convenient fallback. If the templates demonstrate a different phase ordering, bring it back before rewriting the frozen baseline.
