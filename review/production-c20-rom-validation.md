# Cut 20 follow-up: explicit ROM validation

The original two 6530 binaries reside in the sibling `../tests` directory:

- `../tests/6530-002.bin` at `$1C00-$1FFF`.
- `../tests/6530-003.bin` at `$1800-$1BFF`.

Run `make validate-kim1` from the project root. Missing binaries are an error, not a skip. `TESTS_DIR`, `KIM_ROM002`, or `KIM_ROM003` can override the paths.

The test compares all 2,048 bytes of the original images against CPU-visible reads in the base, +`$2000`, and high ROM mirrors. It then checks RESET resolves to `$1C22` and executes 20 monitor instructions. This is not an interactive boot; the serial and keypad/display interfaces still need wiring.

Verified locally with the supplied original images: `make validate-kim1 GO=../.toolchains/go/bin/go TESTS_DIR=../upload` passed. Ordinary `make validate` passed as well. The external Synertek/Dormann CPU suites were not rerun.

Suggested commit: `Require and verify both KIM-1 ROMs with make validate-kim1`.
