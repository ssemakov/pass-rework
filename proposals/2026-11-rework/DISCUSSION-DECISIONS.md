# Recorded discussion decisions and clarifications

These chat decisions now apply to the proposal documents and supersede conflicting recommendations in the preserved original reviews. No product option has been selected.

| Topic | Agreed scope / clarification |
|---|---|
| Restartable migration | Preserve the original encrypted backup untouched. Each attempt creates a fresh encrypted destination, verifies every entry before success, and discards incomplete output/restarts from backup after interruption. No checkpoints or resumable migration |
| Normal edit/delete | Use SQLCipher/SQLite transactions and its standard journal recovery. No custom recovery log/replay layer |
| Option A macOS scope | Minimum macOS 15 (Sequoia), tested on 15 and 26. No requirement to go back to 13/14. The reported legacy failure on 26 is user context; no failure cause/date was independently reproduced |
| Comparable B/C/D evaluation | Use 15/26 as a planning assumption to compare alternatives. This extends the evaluation baseline, not the owner’s explicit A-only OS commitment. Test both CPU architectures where supported; a next beta is exploratory, not an advance compatibility guarantee |
| External security review | Omit the commissioned external human review from v1. Encryption/migration tests and adversary-agent reviews remain. No paid audit is needed for v1 completion, and no audit/certification claim is made |
| Option D | Add Wails + Go + React/TypeScript + SQLCipher through CGo. Go is an alternative backend/framework choice, not a change inside Tauri. No selection of D is implied |
| UI comparison | Compare native feel, legacy workflow continuity, frontend reuse, CRUD, keyboard/accessibility, secret display/lock and ownership cost across A/B/C/D. B/D can use essentially the same React interface |

System SQLite risk: SQLCipher shares much of SQLite’s API. Incorrect linkage can select ordinary SQLite; it silently ignores an unknown `PRAGMA key` and can create plaintext. Using the SQLCipher-specific `sqlite3_key` C API helps make a missing cipher fail during build/load, but packaged-binary identity, wrong-password, plain-SQLite rejection and tamper tests still verify the actual path. This is a packaging/configuration risk, not a new SQLCipher vulnerability.

KDF settings: the key derivation function turns a master password into an encryption key. More computation (and, for some KDFs, memory) slows both unlocking and offline guesses. New SQLCipher 4 vaults use its verified modern profile, normally PBKDF2-HMAC-SHA512 with 256,000 iterations. Legacy settings are retained only to read the source. See [encryption evidence](ENCRYPTION-EVIDENCE.md); no custom crypto wrapper is proposed.

Framework terms: Tauri 2 combines a web UI with a Rust backend and uses macOS’s WebView instead of bundling Chromium. Wails provides a similar web-UI approach with Go. The D proposal uses the current stable Wails v3 line; exact versions and native SQLCipher integration remain a storage/build spike, not an implemented claim. Garbage-collected Go/JS and Swift UI strings can leave copies of secrets; do not promise complete memory erasure in any option.
