# Apple 65C02 Processor Laboratory — frozen architecture handoff

**Architecture Baseline:** 0.2 (amended, frozen 2026-11-04)
**Status:** design handoff; **no production implementation exists in this package**.
**Language:** Go. **Target:** Apple-documented CMOS 65C02 profile, NCR as reference part.

Read in this order:
1. `docs/architecture-baseline-0.2.md` — frozen architecture and authority rules.
2. `docs/working-agreement.md` — how to collaborate with the user: cuts, tests, nested archives, progress reports, and versioning.
3. `docs/schema-spike-mandate.md` — next task: disposable four-template exploration and tests.
4. `docs/evidence-policy.md` and `review/evidence-matrix-seed.tsv` — source precedence and classified test outcomes.
5. `docs/reference-collection.md` — historical and engineering source inventory.

**Next action:** verify supplied reference collection, perform the throwaway schema spike, discuss its findings with the user/Claude, then write Technical Specification 0.1. Do not silently turn the spike into production code.

Do not confuse the desired later Visible Computer/Wails UI or a complete Apple IIe host with the first headless simulator milestone.
