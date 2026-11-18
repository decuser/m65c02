# C31: Instruction-aware logic analyzer

- Marks captured SYNC opcode-fetch cycles with amber vertical separators over the bus waveforms.
- Adds opcode metadata to trace records using the authoritative CPU instruction table, without reading machine memory.
- Reconstructs operand bytes from subsequent captured instruction bus reads and formats instruction addressing modes; missing bytes at page boundaries remain unknown (`??`).
- Groups exact cycle transactions beneath their instruction in the table. Selecting a cycle reveals its nearest in-page instruction context.
- Leaves the original KIM-1 panel and bounded capture/history controls unchanged.
- Validation: `make validate`, `make validate-kim1`, and JavaScript syntax check.
