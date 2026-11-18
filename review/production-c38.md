# C38: Individually selectable logic analyzer signals

- The analyzer overlays CPU, 6530-002, and 6530-003 signals on one cycle timeline instead of switching between exclusive probe views.
- Each device has independent All/None controls and checkboxes for each signal. CPU, Display, Keypad, Serial, All, and None presets configure those checkboxes together.
- Expanded mode exposes individual bus and port bit lanes with their own toggles. Grouped mode retains the decoded hexadecimal buses.
- The selected-cycle detail and transaction table always include both RIOT port pairs. The independent RIOT change navigator still selects one device as its search source.
- This cut changes only the analyzer HTML; it neither changes trace collection nor advances the CPU when the presentation changes.

## Validation

- Embedded JavaScript parsed successfully with Node.js.
- `make validate GO=../.toolchains/go/bin/go` passed.
- Interactive browser behavior should be verified after applying the overlay: toggle individual signals, switch presets, expand bits, select waveforms, and navigate RIOT changes.
