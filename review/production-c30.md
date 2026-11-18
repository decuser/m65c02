# C30: Cycle-level KIM-1 logic analyzer

- Added an opt-in bounded 8192-sample bus recorder at the backplane transfer point. Each sample captures physical cycle number, address, read/write data, R/W, SYNC, IRQ, NMI, RESET and RDY.
- Open the analyzer from the original front panel in a separate browser window. The original panel is unchanged apart from the launch button.
- The analyzer draws all 16 address and 8 data bit lanes plus control lines, and includes an exact transaction table with selectable cycle details. Φ2 is explicitly depicted as the active transfer phase.
- Provides LIVE, OLDER, NEWER, machine PAUSE, STEP CYCLE, STEP INSTRUCTION and CLEAR TRACE controls. The history controls never restore or rewind the machine.
- The history buffer is bounded; 128 samples are returned per API request. Captures reset on cold power-on.
- Added ring-buffer/pagination and HTTP-control tests; both `make validate` and original-ROM `make validate-kim1` pass.

Follow-up: dedicated reverse execution/snapshotting requires full machine and peripheral state restoration and is outside C30.
