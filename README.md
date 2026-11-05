# Apple 65C02 Processor Laboratory

**Architecture Baseline:** 0.3 (frozen 2026-11-04)  
**Status:** design handoff; no production CPU implementation yet.  
**Language:** Go 1.16. **Target:** Apple IIe-compatible base CMOS 65C02 (no Rockwell bit operations or WAI/STP).

Read in this order:
1. `docs/architecture-baseline-0.3.md` — frozen contracts and compatibility target.
2. `docs/working-agreement.md` — development conventions, cuts, tests and reviews.
3. `docs/schema-spike-mandate.md` — disposable four-template spike.
4. `docs/reference-collection.md` — test corpora, reference texts and inspirations.

**Next:** implement the disposable spike, report its results to the user/Claude, then draft Technical Specification 0.1. The selected SingleStepTests 65C02 set is the default cycle/bus oracle; Dormann covers functional semantics. Record a deliberate deviation rather than silently changing the compatibility profile. If external tests cannot be downloaded, state that clearly.

The eventual Visible Computer/Wails UI and complete Apple IIe host are not part of the initial headless CPU milestone.

## Disposable interactive spike monitor

Run `make run`, then `help`, `trace on`, `cycle 1`, `cycle 10`, `step`,
`micro`, `regs`, `mem 0300`, or `run`. `demo counter` resets a finite demo
with ten `INC $0300,X` instructions and BRK. The counter ends at `$0A`.
This is a teaching/validation adapter, not a complete 65C02 or M2 supervisor.
