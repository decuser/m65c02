# Historical evidence, corpus transfer and conflicts

This file operationalizes Baseline 0.2 section 11. It is not another competing priority table.

## Subject-specific precedence

1. **Target machine and supported instructions:** Apple *Apple IIe Technical Reference Manual* (1985).
2. **Target processor timing/signals/instruction semantics:** NCR 65C02 original datasheet, including Apple's reproduced copy. Apple-authored commentary in the same volume is separate text, not independent hardware verification.
3. **Contemporary Apple IIe timing and operating context:** Jim Sather, *Understanding the Apple IIe*, and other contemporary Apple documentation.
4. **Observed physical behavior:** controlled, documented measurements of actual target-reference chips override conflicting printed behavior. Record erratum and conditions; verify measurement identity.
5. **Gaps in documented bus-cycle detail:** SingleStepTests/65x02 and independent traces, provided their tested variant and provenance are identified and a specific transfer justification is recorded.
6. **Secondary implementation comparison:** AppleWin. Instruction semantics may be useful; its execution path is not presumed to generate accurate per-cycle bus traces.
7. **Teaching intent:** *The Visible Computer: 6502* governs teaching vocabulary, without overriding physical bus behavior.
8. **Software design precedent only:** Aiki, Harpy, Nezha, V6 and PDP-11 projects; no dependencies.

## Required per-claim columns

Topic/opcode, instruction/mode, assertion, source title, edition/date, page/table/test ID, **source origin and provenance chain**, physical silicon variant, independently measured vs documentation vs derived, evidence category, provisional implementation choice, conflict, next check.

## Classifications

- **Authoritative:** direct target-processor documentary evidence or verified target silicon data (record which kind). For a conflict, verified target hardware behavior prevails with erratum.
- **Transferred:** strongly justified imported result from a different variant. Distinguish test passing from physical NCR proof.
- **Excluded:** a case outside the Apple instruction profile, or a foreign variant behavior explicitly inapplicable to this test gate.
- **Unresolved:** source conflict or missing target evidence. Prefer documented target behavior provisionally if executable; otherwise `UnresolvedOpcode` explicit stop. **Do not** silently treat a foreign test failure as a target conformance failure.

## First issue ledger

- `INC abs,X`, `DEC abs,X`: Apple Table A-1 states 6 cycles; possible page-cross +1; WDC/Rockwell evidence elsewhere reports fixed 7. NCR-specific conditional timing **unresolved**. Check original NCR table AND footnotes before test corpus.
- RDY-on-write: NCR documentation supports CPU stopping on writes; precise repeated write bus data/side effects need chip-specific evidence. Read-stall tests do not settle writes.
- IRQ/NMI sampling: penultimate-cycle and branch-sampling details must be directly established or qualified as transferred, never asserted as generic NMOS/CMOS identity.
- Opcode-column NOP lengths/cycles: use NCR documented invalid-opcode table where present. Unknowns stop explicitly.
- CMOS dummy reads: strongly variant-sensitive, especially with Apple IIe soft-switch read effects; record actual addresses rather than only cycle totals.

See `review/evidence-matrix-seed.tsv` for initial classified records. Filling exact page/test identifiers from collected source files is part of the next session's first documentary pass; do not fabricate citations.
