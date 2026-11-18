# C28: Power-on and original MOS panel proportions

- Shrink the six segment-rendered LEDs relative to the compact keypad; retain the shorter gap between the four address digits and two data digits.
- Arrange the first row `GO ST RS SST`; depict SST as an OFF-left, ON-right slide switch. SST's electrical single-step function is deferred to C29.
- Separate applying power from pressing RS. Power-on creates the board and holds CPU reset, with dark LEDs and no CPU clocks. Press and release RS to start monitor execution; subsequent RS presses preserve RAM.
- Cold power cycles reinitialize all 1024 main RAM bytes and both 64-byte 6530 RAM banks. The default `device` mode generates a synthetic, reproducible, device-biased cold-start image with small inter-cycle variation. The `zero` mode provides deterministic regression fixtures. Neither mode claims to reproduce a particular physical SRAM's actual cell state.
- Expose `PANEL_RAM_MODE` and `PANEL_RAM_SEED` as Make variables, and show `PRESS RS TO START` in the browser status area.

Validation: `make validate` and `make validate-kim1` using original ROM binaries.
