# Adversarial review C — KeePassXC adoption and KDBX migration

Reviewer: separate adversary agent `/root/adversary_keepassxc`. Date: 2026-11-08. Scope: Option C and its shared migration/evidence documents as drafted. This is a design review, not an implementation audit. I independently checked the official download/audit pages and selected KeePassXC 2.8.0 import/serialization source. No executable or macOS tests were run.

## Verdict

**Conditionally viable and attractive for reducing long-term ownership, but not ready to migrate a real vault.** Adopting upstream CRUD and a documented interoperable format is credible. A safe, lossless converter remains an implementation project whose library, serialization behavior, license and validation have not been demonstrated. “Small utility” describes its product scope, not its security difficulty.

## Findings, highest severity first

### C-H1 — High: there is no selected, proven conversion path

The draft correctly identifies writer selection as unresolved. Until a concrete writer/version and integration route is selected, neither “no plaintext on disk” nor cross-reader interoperability is established. The stock [2.8.0 CLI import implementation](https://github.com/keepassxreboot/keepassxc/blob/2.8.0/src/cli/Import.cpp) imports an XML pathname; it is not a ready-made SQLCipher migration API. A shell pipeline, FIFO or `/dev/fd` trick must not be treated as a verified solution without testing its handling of source data, password prompting, errors and subprocess output.

Required gate: demonstrate the exact supported writer integration using synthetic legacy data, with filesystem-write tracing, no secret command arguments, no plaintext staging and independent KeePassXC reopen. Reusing upstream code needs an explicit GPL-compatible distribution decision. A permissively licensed third-party KDBX crate may have a different risk profile from the reviewed KeePassXC implementation and cannot inherit its audit claims.

### C-H2 — High: native field fidelity is stricter than retaining an archival copy

The proposed metadata fallback is necessary but underspecified. In [KeePassXC 2.8.0's XML writer](https://github.com/keepassxreboot/keepassxc/blob/2.8.0/src/format/KdbxXmlWriter.cpp), KDBX4 native timestamps serialize whole seconds; unprotected string and custom-data paths filter invalid XML characters. A naive custom string cannot preserve every original value. This does not prove ordinary passwords will change: inner-protected attributes use a different serialization path. It does prove that an unspecified “encrypted custom data” fallback is not enough.

Required change: define a versioned, typed, binary-safe encoding for legacy integers, null flags and other non-native representations, such as base64 of explicit UTF-8/byte payloads plus type tags. Do not round millisecond timestamps and call the migration exact. Test source values containing NUL/control characters, CR/LF sequences, noncharacters, null and empty values, timestamps outside ordinary date ranges, duplicate titles and long strings. Compare the actual fields KeePassXC will reveal/copy after open-save-reopen. If a value survives only as archival metadata but cannot be used faithfully in its native field, report it and stop for a mapping decision rather than declaring complete behavioral compatibility.

### C-H3 — High: preserving original secrets can defeat later edits and deletion expectations

Copying an entire original row into custom metadata would keep its old password after editing the Password field unless the archival copy is also maintained. Entry history, recycle-bin entries and automatic backup saves create additional retained copies. These are encrypted, but they matter after a future key disclosure and when a user expects deletion from the active vault.

Required change: preserve only metadata needed for lossless mapping, rather than duplicating every secret by default. Where raw original secret material is essential, define how native edits/delete affect it; either synchronize/purge it or label the field as an archival secret with explicit retention. Verify behavior after KeePassXC edit, delete, recycle-bin empty, history clear, master-password change and save. The proposal already correctly excludes assured erasure of APFS snapshots and external backups.

### C-M1 — Medium: proof must cover the converter and actual macOS build

The draft appropriately limits its audit claims. The [official audit list](https://keepassxc.org/audits/) distinguishes the 2.7.4 source review from the 2.7.9 Windows 10 certification. Neither proves a new converter safe or certifies KeePassXC 2.8.0 on macOS. The KDBX format's authenticated encryption construction also does not establish a particular writer's correctness.

Required gate: tie field, tamper, wrong-password, parameter-bound, interruption and filesystem-artifact tests to exact converter/client hashes. Review the chosen implementation and changes relevant to storage since reviewed versions. Classify this proposal's agent review separately from an independent security audit.

### C-M2 — Medium: upstream OS support is not an application compatibility test

The [current download page](https://keepassxc.org/download/) advertises 2.8.0, macOS 12+ and separate Intel/Apple Silicon downloads. This supports feasibility, not a guarantee for an unspecified oldest macOS release or future beta. Migrator toolchain/provider support must be assessed separately from the GUI.

Required gate: resolve the oldest required macOS version, verify both converter and selected upstream binaries there, and test newest/prior releases and the next available beta/RC. Do not treat an obsolete upstream build as a supported permanent fallback. A beta failure remains a limitation even if the product works on stable releases.

### C-M3 — Medium: single-file storage still has save and configuration behavior

The encrypted committed vault is one file, but save staging, lock files, backups and client configuration can exist. Decryption must remain possible with just the final KDBX file and master password. Key files, device-local remembered unlock secrets or mandatory additional factors would change that requirement.

Required gate: inspect chosen KeePassXC save/backup settings and crash artifacts on macOS; verify temporary content is encrypted and atomic-save failure preserves a recoverable vault. Test copying the closed final file to a clean test account/machine and unlocking without local configuration or Keychain state. Disable optional browser/network integrations for the initial migration pilot unless independently needed and assessed.

## Residual risk and scope

Even after the gates pass, C is a one-way application/format migration, not DB-level drop-in compatibility. Bespoke UI and direct old-app interoperability are lost. Upstream supplies features and maintenance but can raise its OS floor; KDBX interoperability allows another client, whose quality must then be evaluated. Offline guessing, unlocked-process compromise, clipboard replication, valid older-file rollback, history and backups remain material risks.

The strongest case for C is reducing the amount of password-manager code this project must own. That benefit survives this review; the claim of an already-proven migration does not. The proposal should present C with the unresolved writer and fidelity gates prominently, alongside its stronger existing ecosystem/review trail.

## Author disposition communicated during review

The author plans to constrain archival metadata to a versioned typed/base64 representation of IDs, exact timestamps and null flags; reject native field values that cannot round-trip byte-exactly; and prohibit duplicate full-row/secret archives. These changes address C-H2/C-H3 at the design level if implemented as stated. C-H1 remains an explicit blocking spike, and all runtime/evidence gates remain unpassed. This review records the original draft assessment and does not claim the planned fixes were executed or tested.
