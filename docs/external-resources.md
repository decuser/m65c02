# External resources: reproducible setup

This repository intentionally contains **no** third-party test corpus, KIM-1
monitor ROM, or Microsoft BASIC tape image. Keep these resources in a sibling
directory, outside Git. The default is `../m65c02-resources` relative to the
repository root. Override it with `make RESOURCES_DIR=/absolute/path ...`.

## 1. Get the two upstream test repositories

From the `m65c02` repository root:

```sh
mkdir -p ../m65c02-resources
git clone https://github.com/SingleStepTests/65x02.git ../m65c02-resources/65x02
git clone https://github.com/Klaus2m5/6502_65C02_functional_tests.git ../m65c02-resources/6502_65C02_functional_tests
```

The required SingleStepTests corpus is
`65x02/synertek65c02/v1/{00,...,ff}.json`. Other processor variants may be
present, but `validate-full` does not use them. Keep a record of each upstream
revision with `git -C ../m65c02-resources/65x02 rev-parse HEAD` and the
equivalent command for `6502_65C02_functional_tests`.

The required unmodified Dormann binary is:

`6502_65C02_functional_tests/bin_files/6502_functional_test.bin`

For reproducibility the original project fixture was documented against
Dormann commit `7954e2dbb49c469ea286070bf46cdd71aeb29e4b`. To reproduce
that exact revision, run:

```sh
git -C ../m65c02-resources/6502_65C02_functional_tests checkout 7954e2dbb49c469ea286070bf46cdd71aeb29e4b
```

## 2. Build the two *customized* Dormann fixtures

The upstream binaries are **not** interchangeable with these two fixtures.
Create `../m65c02-resources/dormann-profile/` and work from copies of the
upstream assembly sources, leaving the cloned upstream sources untouched:

```sh
mkdir -p ../m65c02-resources/dormann-profile
cp ../m65c02-resources/6502_65C02_functional_tests/65C02_extended_opcodes_test.a65c ../m65c02-resources/dormann-profile/65C02_profile.a65c
cp ../m65c02-resources/6502_65C02_functional_tests/6502_interrupt_test.a65 ../m65c02-resources/dormann-profile/6502_interrupt_cmos.a65
```

Edit only the configuration definitions in the copied files:

- `65C02_profile.a65c`: set `rkwl_wdc_op = 2`, `wdc_op = 1`, and `skip_nop = 0`.
- `6502_interrupt_cmos.a65`: set `D_clear = 1`.

With Klaus Dormann's `as65` assembler installed and available as `as65`:

```
cd ../m65c02-resources/dormann-profile
wget https://github.com/Klaus2m5/6502_65C02_functional_tests/raw/refs/heads/master/as65_142.zip
```

```sh
cd ../m65c02-resources/dormann-profile
./as65 -x -qm -l65C02_profile.lst -o65C02_profile.bin 65C02_profile.a65c
./as65 -x -qm -l6502_interrupt_cmos.lst -o6502_interrupt_cmos.bin 6502_interrupt_cmos.a65
cd ..
```


## 3. Supply the original KIM-1 monitor ROMs

Obtain the original 1,024-byte images `6530-002.bin` and `6530-003.bin`
from an archival KIM-1 source, such as
[zimmers.net's KIM-1 ROM collection](https://www.zimmers.net/anonftp/pub/cbm/src/kim-1/).
Place both files directly in `../m65c02-resources/`:

```
wget https://www.zimmers.net/anonftp/pub/cbm/src/kim-1/6530-002.bin
wget https://www.zimmers.net/anonftp/pub/cbm/src/kim-1/6530-003.bin
cd ../../m65c02
```

```sh
wc -c ../m65c02-resources/6530-002.bin ../m65c02-resources/6530-003.bin
sha256sum ../m65c02-resources/6530-002.bin ../m65c02-resources/6530-003.bin
```

Each file must be exactly 1024 bytes. Record the hashes of the images you
actually obtained. Do not use a combined 2 KiB ROM dump without splitting
and verifying its bank order.

79c6dbd54aa3911f53c1068c4310aabceb7a6ce9714d1c402b73af76ea109237  ../m65c02-resources/6530-002.bin
80f5c6393e0bbcfc8a5b68fd73858da34e6d6a415f32d4820f21edb52fecafec  ../m65c02-resources/6530-003.bin

## 4. Optional Microsoft KIM BASIC paper tapes
Get from http://retro.hansotten.nl/6502-sbc/kim-1-manuals-and-software/kim-1-software/microsoft-kb-9-basic/kb-9-downloads/

The original KIM-1 KB-9 Microsoft Basic V1.1, cassette audio wave, binary and papertape format
The updated KIM-1 KB-9 Microsoft Basic V1.2, binary and papertape format
The updated (see below) KIM-1 KB-9 Microsoft Basic V2, binary and papertape format

Then copy renaming to m65c02-resources/programs/kim1
```
mkdir -p ../m65c02-resources/programs/kim1

cp ~/Downloads/kb9v11/kb9v11.pap ../m65c02-resources/programs/kim1/kb9-v1.1.ptp
cp ~/Downloads/kb9v12/kb9v2.pap  ../m65c02-resources/programs/kim1/kb9-v1.2.ptp
cp ~/Downloads/kb9v2/kb9V2.ptp   ../m65c02-resources/programs/kim1/kb9-v2.ptp
```

Sources for historical/modified KB-9 versions include
[Hans Otten's KB-9 BASIC collection](https://retro.hansotten.nl/6502-sbc/kim-1-manuals-and-software/kim-1-software/microsoft-kb-9-basic/)
and [W9OMS KIM-1 Software](https://w9oms.com/kim-1-software/).
Preserve original downloads, record their provenance and checksums, and
rename/copy each *matching* tape to the filenames above. Do not merely
rename arbitrary binary images to `.ptp`: these tests parse the checksummed
KIM monitor paper-tape format. In particular `kb9.kim` and `kb9-v1.1.ptp`
are required by a test to yield identical loaded memory images.

The BASIC tape tests skip when the entire directory is absent or empty.
Once it contains any files, missing required tapes are treated as errors.

## 5. Verify

From the repository root:

```sh
make validate                       # builds, vets, runs local tests
make validate-kim1                  # requires both original ROM images
make validate-full                  # includes validate, validate-kim1,
                                    # Synertek and all three Dormann fixtures
```

Use `make RESOURCES_DIR=/absolute/path/to/resources validate-full` for a
nonstandard resource location. To check individual requirements before the
full suite, run:

```sh
test -f ../m65c02-resources/65x02/synertek65c02/v1/00.json
test -f ../m65c02-resources/65x02/synertek65c02/v1/ff.json
test -f ../m65c02-resources/6502_65C02_functional_tests/bin_files/6502_functional_test.bin
test -f ../m65c02-resources/dormann-profile/65C02_profile.bin
test -f ../m65c02-resources/dormann-profile/6502_interrupt_cmos.bin
```

Do not commit or redistribute third-party files as part of this emulator's
source tree. The root BSD 2-Clause license covers this project's code, not
the external ROMs and test collections.
