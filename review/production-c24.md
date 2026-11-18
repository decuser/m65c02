# Production c24: Linux PTY serial adapter and separate trace window

## Goal

Allow a conventional Linux serial terminal (for example Minicom) to connect to the original KIM-1 ROM as if through a serial adapter. Preserve the emulated 6530 pin waveforms and run only the logic trace in a new terminal window.

## Delivered

- Linux PTY pair from `/dev/ptmx` with the slave path printed as `/dev/pts/N`.
- Nonblocking master reader detects the client opening the slave and moves incoming bytes into the emulator's serial input channel.
- The emulator converts host bytes into cycle-timed PA7 input, and sends decoded PB0 characters back through the PTY master.
- Automatic RUBOUT handshake when the first PTY client connects; no auto-RUBOUT before that connection.
- `make run-kim1-pty` and `make run-kim1-pty-trace` targets.
- `make run-kim1-trace` now opens **only** an additional logic trace window while the KIM-1 stays in the current terminal. Traces are streamed into a temporary file observed by `tail` in `x-terminal-emulator`.
- Terminal-trace window closure does not stop the emulator; the trace process exits when the emulator exits.

## Validation

- `make validate`: passed, including unit tests for PTY device allocation, client attachment, and byte transfer.
- `make validate-kim1 GO=../.toolchains/go/bin/go TESTS_DIR=../upload`: passed original ROM tests, including a PTY client receiving the ROM's `KIM` greeting and `1234 FF` response after sending `1234 ` through the PTY.
- A separately installed Minicom executable and the graphical `x-terminal-emulator` window were not exercised in the development container. Verify on the user's Devuan/Cinnamon host.

## Usage

1. Run `make run-kim1-pty` in the existing shell.
2. Read the printed `/dev/pts/N` device path.
3. Connect from another shell with `minicom -D /dev/pts/N -b 110 -8` (8N1, local echo enabled).
4. Type `1234 ` (space required) and expect `1234 FF`.
5. Use `make run-kim1-pty-trace` if a second terminal window should show the raw logic trace.

## Limits

- The PTY transports bytes. Its Linux termios baud setting does not regulate emulated CPU timing; `KIM_BIT_CYCLES=9091` controls nominal 110-baud transmission at a 1 MHz emulated clock.
- The KIM-1 firmware's ability to negotiate faster baud rates has not been demonstrated here.
- Auto-RUBOUT is triggered at first client attachment; reopening the PTY after a disconnect does not yet guarantee a renewed handshake.
- Initial decoder NUL bytes remain unexplained.
- No minicom-specific UI or termios configuration is performed by the emulator; the user configures Minicom's local echo.

## Suggested commit

`Expose KIM-1 serial port as Linux PTY and split trace into a new window`

## Next cut

c25: six-digit multiplexed LED display, or a focused device-reconnection and baud-rate qualification cut if PTY interoperability reveals gaps.
