# Reference collection manifest

User says the documentation and test collection is assembled. **No assumptions about archive names, integrity, or availability in the next session.** Ask for or inspect the collection present in that session before claiming a test file exists; verify ZIP/TAR integrity and identify chip variant/test revision.

## Normative historical inputs

1. Apple, *Apple IIe Technical Reference Manual* (1985), including Appendix A and Apple comparison Table A-1. Defines Apple CPU profile and machine requirements. Source is Apple-authored material together with a *reproduced* NCR processor document: record provenance separately.
2. NCR original 65C02 datasheet (in Apple appendix and/or independent original). Processor semantics, signal behavior, timing table and notes. Important: actual cycle sequences may not be fully documented.
3. Jim Sather, *Understanding the Apple IIe* (1985). Contemporary IIe hardware and timing discussion.
4. *The Visible Computer: 6502*, Software Masters (1982). Pedagogical vocabulary and interaction inspiration only; primarily NMOS-era, not a silicon specification for Apple's CMOS variant.

## Executable and implementation evidence

5. `SingleStepTests/65x02` corpus, organized and tagged by actual processor variant. Confirm whether relevant Apple/NCR sequences are directly covered. Do not treat Rockwell/WDC vectors as automatic NCR truth. Known classified disagreement should not fail a target-profile test.
6. AppleWin source archive, for documented instruction handling and Apple machine integration. Not automatically cycle-bus-accurate.

## Design inspiration, not code dependencies

7. Aiki baseline — DTrace-inspired profiling architecture and observer invariance.
8. Harpy baseline — low-volume structured logging of operator actions/errors (not trace).
9. V6 emulator and PDP-11/40 experiments — SIMH-like supervisor, inspection/run separation.
10. Nezha baseline — future Wails/frontend adapter pattern.

## Suggested collection structure (user can choose different layout)

```
references/
  apple/           # Apple IIe manual and contemporary books
  ncr/             # NCR original/reproduced CPU docs
  teaching/        # Visible Computer manual
  tests/           # 65x02 test corpora
  implementation/  # AppleWin source
  inspiration/     # Aiki, Harpy, Nezha, V6, PDP-11
```

Do **not** copy copyrighted manuals or large upstream test archives into incremental code overlays unless the user explicitly chooses an appropriate license/distribution method. Use a manifest and local paths.
