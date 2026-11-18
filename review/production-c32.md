# C32: Grouped hexadecimal bus waveforms

- Made ADDRESS (16-bit) and DATA (8-bit) grouped waveforms the default presentation.
- Draws bus transition crosses and hex-value capsules on each sampled cycle, with compact/normal/wide cycle widths and horizontal scrolling.
- Provides an `Expand address/data bits` checkbox to restore A15–A0 and D7–D0 individual signals.
- Keeps R/W, SYNC, IRQ, NMI, RESET, RDY, sampled Φ2, amber instruction boundaries, disassembly and transaction table.
- Changes only `cmd/m65c02-panel/analyzer.html`; no changes to CPU timing or bus capture semantics.
- Checks: analyzer JavaScript syntax, `make validate`, `make validate-kim1`.
