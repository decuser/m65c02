# C29: Original KIM-1 single-step switch

- The SST slider sits to the right of GO, ST and RS; only its right-hand SST marking appears. The switch is interactive, with a right-hand ON position.
- Models SST on the motherboard as a real interrupt source driven by an opcode-fetch SYNC request, inhibited for the mirrored $1800–$1FFF KIM ROM addresses. ST drives NMI independently.
- Does not invoke CPU `StepInstruction()` from UI controls. The original monitor handles the NMI and returns control to the keypad, so GO executes one user instruction per press.
- Preserves physical SST switch position across power cycles.
- Adds electrical signal and CPU execution tests, plus UI control tests. The synthetic ROM test verifies the NMI handler is entered after executing a user instruction.
- Hardware manual prerequisite: load `$17FA=00`, `$17FB=1C` before using SST, or the NMI vector may point elsewhere after cold power-on.

Validated with `make validate` and original-ROM `make validate-kim1`.
