# m65c02

A cycle-accurate, observable **65C02 CPU emulator in Go**, with an original-ROM
**KIM-1 machine**, a browser-based front panel and logic analyzer, and a
micro-operation viewer called Visible Computer.

The CPU targets the **Apple IIe-compatible CMOS 65C02 instruction profile**.
This is not yet an Apple IIe emulator. It deliberately excludes Rockwell
RMB/SMB/BBR/BBS and WDC WAI/STP instructions. Unsupported opcode bytes follow
the selected test profile's NOP behavior, not necessarily NMOS illegal-opcode
behavior.

The CPU lives in `internal/cpu`. It offers instruction, cycle, and
micro-operation stepping; PRE/BUS/POST execution phases; bus transactions;
interrupt/reset handling; snapshots; and optional internal observation.
Snapshots of CPU state **do not** rewind external memory or devices. Machine-wide
reverse execution is not implemented.

## Requirements

- Go (the module declares Go 1.16 compatibility).
- Linux for the KIM-1 terminal/PTY integration.
- Original KIM-1 ROM images for ROM-backed emulation.
- For the interactive BASIC launcher: `picocom` and `x-terminal-emulator`
  (the latter is commonly supplied through a desktop terminal alternative).
- For the browser front panel: a local web browser; `xdg-open` is used for
  automatic opening.

The ROMs, test corpora, and BASIC tape images are **not distributed in this
repository**. Download them separately into a sibling directory, by default
`../m65c02-resources`. See
**[External resource setup](docs/external-resources.md)** for sources,
the expected file layout, and custom test fixture instructions. Override the
default with `make RESOURCES_DIR=/absolute/path ...`.

## Quick start: KIM-1 front panel

On Linux, install Go and a web browser, then clone this repository:

```sh
git clone https://github.com/decuser/m65c02.git
cd m65c02
```

Download the two original 1 KiB KIM-1 monitor ROMs from
[Zimmers.NET's KIM-1 archive](https://www.zimmers.net/anonftp/pub/cbm/src/kim-1/index.html):

- [6530-002.bin](https://www.zimmers.net/anonftp/pub/cbm/src/kim-1/6530-002.bin)
- [6530-003.bin](https://www.zimmers.net/anonftp/pub/cbm/src/kim-1/6530-003.bin)

```sh
mkdir -p ../m65c02-resources
curl -fL -o ../m65c02-resources/6530-002.bin https://www.zimmers.net/anonftp/pub/cbm/src/kim-1/6530-002.bin
curl -fL -o ../m65c02-resources/6530-003.bin https://www.zimmers.net/anonftp/pub/cbm/src/kim-1/6530-003.bin
wc -c ../m65c02-resources/6530-00{2,3}.bin  # each should be 1024 bytes
make run-kim1-panel
```

The front panel opens in your browser, normally at
`http://127.0.0.1:6580/`. If it does not open automatically, visit that
address yourself. Use **POWER ON**, then press **RS** to reset/start the
original monitor. No external test corpora or BASIC tapes are needed for
this first run. The ROMs are third-party archival files and remain outside
the Git repository.

For serial-terminal operation, install `picocom` and use the commands below.
For Microsoft BASIC, also download the optional paper tapes described below.

## Build and test

```sh
make build              # compile Go packages
make test               # local Go tests
make validate           # build, go vet, and tests
make validate-kim1      # original ROM and KIM-1 integration checks
make validate-full      # includes validate AND validate-kim1,
                        # then Synertek and three Dormann tests
```

`make validate` can skip optional external-data tests if their fixtures are not
configured. `make validate-kim1` fails without the two original ROM images.
`make validate-full` requires both ROMs, the Synertek 65C02 JSON corpus,
the upstream Dormann 6502 functional binary, and **two customized** Dormann
65C02/CMOS-interrupt binaries. Missing required files cause a failure.
The full gate runs 2,560,000 Synertek vectors in four groups, plus the three
Dormann programs. See [Dormann conformance](docs/dormann-conformance.md)
for the required configuration and success addresses.

## Run the KIM-1

Put `6530-002.bin` and `6530-003.bin` in
`../m65c02-resources/`, then:

```sh
make run-kim1-panel       # graphical front panel in your browser
make run-kim1             # original ROM monitor in this terminal
make run-kim1-pty         # serial monitor exposed through /dev/pts/N
make run-kim1-trace       # monitor with serial-pin trace
make run-kim1-pty-trace   # PTY plus separate trace terminal
```

The browser panel normally serves `http://127.0.0.1:6580/`. If the browser
does not open automatically, visit that URL manually. Set
`PANEL_OPEN=0` to prevent automatic browser launching, or
`PANEL_ADDR=127.0.0.1:6581` to change the listen address.

The panel models the six multiplexed LED digits, keypad matrix, TTY jumper,
reset and stop inputs, and single-step switch using the original ROM's
interaction with the 6530 port pins. Power ON requires pressing RS before the
CPU starts clocking. The **OPEN LOGIC ANALYZER** view shows cycle-aligned CPU
bus and RIOT signals; **VISIBLE COMPUTER** shows executed PRE/BUS/POST
micro-operations. The analyzer retains a bounded 8,192-cycle history and can
export it as CSV or text. Internal microstep capture is optional.

The terminal targets use the genuine monitor's serial protocol. The PTY target
prints its assigned device and the corresponding terminal command. For
example, in a second terminal:

```sh
picocom --echo --emap delbs --omap delbs -b 1200 /dev/pts/4
```

Use the **actual** `/dev/pts/N` printed by your run. Exit picocom with
**Ctrl-A, then Ctrl-X**.

## Run KIM-1 Microsoft BASIC

Download the original KB-9 V1.1 and patched V1.2/V2 paper tapes as described in
[External resource setup](docs/external-resources.md). They belong in
`../m65c02-resources/programs/kim1/` with these names:

| Selection | File | Monitor GO address |
| --- | --- | --- |
| 1: Original V1.1 | `kb9-v1.1.ptp` | `$4065` |
| **2: Patched V1.2 (default)** | `kb9-v1.2.ptp` | `$3FB1` |
| 3: Patched V2 | `kb9-v2.ptp` | `$2000` |

```sh
make run-kim1-basic-pty
# or select a version without prompting:
make KIM_BASIC=3 run-kim1-basic-pty
```

This target loads the chosen tape into the emulated KIM-1's expansion RAM,
initializes the monitor's decimal-mode/stack setup locations, and sets the
monitor's GO address. It **automatically launches picocom in another
terminal** using the allocated PTY. In picocom, type **G** at the monitor's
address prompt to enter BASIC.

For BASIC, use **CAPS LOCK** for commands. The launched picocom uses local echo
and maps DEL/backspace to the expected character. Exit picocom with
**Ctrl-A, then Ctrl-X**. At BASIC's `MEMORY SIZE?` and `TERMINAL WIDTH?`
prompts, press Enter to accept its defaults, then answer the optional math
functions prompt as desired.

The `run-kim1-basic-pty` target does not automatically enter `G` for you.
The BASIC program images remain external: the Microsoft BASIC source retained
in this repository is **reference material**, not the executable tape images.

## Other development entry points

```sh
make run               # interactive teaching monitor
make run-prod          # production CPU demonstration
make bench             # Go benchmarks
make help              # available Make targets
```

The teaching monitor accepts commands such as `help`, `step`, `cycle 1`,
`micro`, `regs`, and `mem 0300`.

## Design notes and project status

- [Architecture baseline](docs/architecture-baseline-0.3.md): CPU contracts
  and profile decisions.
- [Release candidate](docs/release-candidate-c41.md): supported behavior,
  verification evidence, and limits as of the C41 review.
- [External resources](docs/external-resources.md): reproducible test data,
  monitor ROMs, and BASIC tapes.
- [Reference collection](docs/reference-collection.md): historical references.
- [Working agreement](docs/working-agreement.md): development conventions.

The current implementation includes the 65C02 CPU, the KIM-1 host, the
browser panel, the cycle-based signal analyzer, and the observational Visible
Computer view. A complete Apple IIe machine, native Wails interface, and
reversible machine-wide execution are not provided.

## Acknowledgments

Development of m65c02 has been assisted by ChatGPT (OpenAI), and Grok,
including design discussions, implementation, testing, and
documentation. All changes are reviewed and integrated by
the project author.

## License and provenance

Project code is licensed under the
[BSD 2-Clause License](LICENSE), copyright 2026 Will Senn.

`third_party/microsoft-basic-m6502/` contains historical assembly source
from [Microsoft's BASIC-M6502 repository](https://github.com/microsoft/BASIC-M6502).
That directory has its **own MIT license and provenance note**. The root
BSD license does not replace Microsoft's terms. Other archival ROMs, test
datasets, and BASIC tape binaries must be obtained separately and are not
distributed here.
