# C41 release-candidate audit

No CPU execution code was changed.

- Audited the uploaded `dev-v0.1.0` source archive and the frozen 0.3
  architecture contract.
- Confirmed local `make validate` and supplied-original-ROM
  `make validate-kim1` pass.
- Recorded the developer's successful `make validate-full` as reported
  rather than independently reproduced: external corpus files were not
  available in this audit environment.
- Fixed `make baseline` to retain tracked root `.tar.gz` files and fail
  loudly when tracked working-tree files are missing. The uploaded archive
  omitted the tracked `m65c02-dormann-profile-fixtures.tar.gz`.
- Added `docs/release-candidate-c41.md` detailing support, exclusions,
  conformance scope, snapshot guarantees, and release acceptance steps.
- Updated the README's outdated M1-status statements.

Do not infer that a release tag was created or pushed by this cut.
