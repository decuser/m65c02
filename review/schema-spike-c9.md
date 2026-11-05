# Spike c9: one-cycle fetch, branch transitions, and runnable programs

Disposable spike, not production CPU. The selected compatibility oracle is
SingleStepTests Synertek 65C02 v1. These local tests do not claim the full
external corpus has been run against this implementation.

- Opcode 03 completes on the shared SYNC fetch (one cycle). Two consecutive
  03 instructions yield consecutive SYNC bus cycles and correct counters.
- BNE exercises not-taken (2), taken same-page (3), and taken crossed-page
  (4) paths. Snapshots, cycle stepping, and microstep stepping use one engine.
- EA (two-cycle NOP) and JMP absolute (three cycles) run through that engine.
- Indexed dummy reads use the final instruction byte, PC+2, not the NMOS
  uncarried effective address. The updated trace test asserts that address.
- The monitor's counter demo now uses an INC/BNE loop (10 increments,
  starting at F6 and wrapping to 00), then BRK. It completes after 106 cycles
  and 21 instructions. `demo figure8` provides two JMP instructions forming
  an endless loop; use `step` or `cycle N` rather than `run`.

The spike still uses dynamic string-coded explanatory operations and allocates;
those implementation techniques are explicitly excluded from production.
Interrupt timing and the complete Synertek corpus remain production conformance
work. The NCR reserved-opcode differences (including 5C) remain a short
compatibility note rather than a new authority hierarchy.
