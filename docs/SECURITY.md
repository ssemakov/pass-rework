# Storage encryption and security boundaries

This document describes the implemented configuration and reproducible evidence. Tests establish behavior of these source/binary versions; they are not a mathematical proof of the application or an external security certification. A commissioned human audit is outside v1 by the owner's decision. The separate [adversarial agent review](IMPLEMENTATION-REVIEW.md) records findings and corrections.

## Cryptographic construction

The application uses upstream **SQLCipher 4.20.0 Community**, commit `355d4db92d3404e71faa8088d071a71d767807f4`, through its C API. There is no application-designed encryption wrapper. See the upstream [design](https://www.zetetic.net/sqlcipher/design/), [API](https://www.zetetic.net/sqlcipher/sqlcipher-api/) and [pinned source](https://github.com/sqlcipher/sqlcipher/tree/355d4db92d3404e71faa8088d071a71d767807f4).

| New-vault property | Configuration |
|---|---|
| Page encryption | SQLCipher AES-256-CBC with per-page IVs |
| Authentication | HMAC-SHA512 per page |
| Password derivation | PBKDF2-HMAC-SHA512, 256,000 iterations, random per-file salt |
| Page size | 4096 bytes |
| Plaintext database header | 0 bytes; no SQLite schema/header in plaintext |
| macOS crypto provider | Apple's CommonCrypto |
| Linux test provider | OpenSSL; Linux is a storage/browser-test host, not a shipping GUI target |

Normal vault passwords reject SQLCipher's exact reserved raw-key literal forms (`x'hex'`/`X'hex'`, with 64, 96 or 160 hex digits). Those upstream forms bypass password derivation even when a settings readback reports 256,000 iterations; rejecting them keeps the new-vault password-KDF guarantee accurate. Near-match ordinary passphrases remain supported. Read-only legacy import/preservation accepts historical raw-key sources and migrates them to an ordinary new passphrase.

SQLCipher's format, randomness, key separation, encryption and authentication implementation supply the cryptographic construction. File size, filesystem timestamps and access patterns remain observable. A strong password is required: possession of the file permits offline guesses. Legacy profiles are tried only for read-only import and never become the default for a fresh destination.

Each connection verifies `cipher_version`, `cipher_status` and the exact effective cipher profile settings before use, then authenticates a schema read after keying. `sqlite3_key` receives an explicit byte length; passwords are never interpolated into SQL. Prepared statements bind all entry values with explicit lengths and copy semantics. NULL, empty, embedded NUL, Unicode and signed 64-bit integers are preserved across the native boundary. C-owned transient buffers are wiped/freed; results are copied before statements are finalized.

The build sets `SQLITE_HAS_CODEC`, SQLCipher initialization/shutdown hooks, `SQLITE_TEMP_STORE=3`, and disables loadable extensions and double-quoted string literals. Connections enable `cipher_memory_security`, memory temporary storage, untrusted schema protection and secure deletion. Normal writes use DELETE rollback journals, synchronous FULL and fullfsync. SQLite handles transaction commit/rollback and journal recovery.

## Evidence chain

1. **Source identity:** `prepare-sqlcipher.sh` downloads the exact upstream commit over HTTPS and checks the archive SHA-256 `fc5394c916beeabb1f43e2ac97d0827f36e8917f6675a00cca54537c32cb36a2`. It verifies generated C/header hashes against `scripts/sqlcipher-generated.sha256` even on a cached build. The upstream license is retained. Go checksums and the npm lockfile pin the other dependencies.
2. **Native integration:** `internal/cipher` compiles that amalgamation into the Go binary, directly references SQLCipher-specific `sqlite3_key`, and fails closed on the expected cipher version/status. No ordinary SQLite driver or connection pool is used. Production build tags exclude alternate GUI/server/debug routes.
3. **Runtime settings:** `TestPinnedEngineConfiguration` checks the actual provider, PBKDF2 iteration count, HMAC/KDF algorithms, page size, header policy and memory temp setting against exact expected values. This runs on all native target jobs.
4. **Negative encryption tests:** the real storage path rejects wrong passwords and plaintext SQLite input; ordinary Python SQLite cannot read a produced vault; a modified authenticated page is rejected. Sentinel tests inspect a vault and its live rollback journal for known synthetic secrets. Sentinel absence alone is not treated as proof: the settings, engine identity and authenticated reads are separate checks.
5. **Data safety:** real driver-produced legacy/profile-3/profile-4 fixtures exercise every-field verification, original-backup byte preservation, sequence and 64-bit round trips, killed migration restart, standard SQLite recovery of a killed edit, backup/restore and old/new-password behavior after rotation.
6. **Shipping artifact:** `check-artifact.sh` records native linkage, symbols, build tags/module metadata and SHA-256; rejects undefined SQLite storage calls; requires the bundled key API; and verifies the macOS signature. `govulncheck` scans the packaged executable. Native WebKit assertions exercise the production frontend/bridge and native clipboard in a separate synthetic test app; its test hooks are absent from release binaries.

Run commands and platform results are in [VALIDATION.md](VALIDATION.md). CI artifacts retain platform/toolchain details, native smoke output, binary metadata, linkage, symbols, hash and vulnerability-scan output. The tests use public synthetic passwords only. No real user vault was used.

## Frontend, native bridge and lock

Bundled assets are the only content allowed in the main WebView. CSP denies remote scripts/connects, frames, plugins and forms. A native WebKit navigation policy permits only the main local document and rejects external/data/file navigation and popups. Vault text renders as React text; it never becomes HTML.

The runtime bridge requires a per-run random capability and the window ID overwritten by the native Wails asset request handler. It accepts only a strict POST envelope and exact facade method allowlist, rejects query/chunk alternate routes and foreign origins, and bounds ordinary concurrent dispatch. Authenticated Lock bypasses ordinary capacity limits. Only opaque generation-bound tokens from native file dialogs can identify paths; SQL, storage objects, arbitrary file/shell helpers and bulk export are not exposed.

One manager serializes native DB work. Session generations invalidate immediately when lock is admitted; SQLite interruption and safe draining complete before lock acknowledgment. Idle timing uses Go's monotonic clock. Stale queued unlocks and closed-manager opens are rejected. The frontend separately fences awaited responses and reveal selections so queued data cannot restore secrets after concealment/lock. Backend metadata lists omit passwords. Sleep, screen-lock, explicit lock, window close/navigation and quit initiate locking.

Native clipboard writes carry the concealed-data marker and a change-count ownership token. The 30-second timer/lock/quit clears only Pass-owned content and never erases a later foreign clipboard value. Native shutdown drains off the AppKit main thread to avoid deadlocking pending clipboard callbacks. Clipboard history/sync tools may retain copies independently.

## Limits of the guarantee

Encryption protects a closed vault file against reading/modification without its password, subject to the algorithms, dependencies and password strength. It cannot prevent malware or an administrator from reading an unlocked process, a keylogger/screenshot, a compromised authorized frontend from invoking permitted reveal/edit methods, destructive deletion, or rollback to a valid older encrypted file. No authenticated remote rollback authority exists in this offline design.

Password entry and revealed values transiently exist in JavaScript/Go strings and framework serialization. Lock clears UI state, closes native handles and wipes owned mutable key buffers, but Go/JS garbage collection cannot guarantee erasure of every heap copy, swap or crash dump. Secure deletion does not erase backups, APFS snapshots, SSD history or clipboard managers. Filename/path metadata is not encrypted.

The owner-controlled local directory, OS, WebKit, C toolchain and dependencies remain trusted. File-identity/permission checks and SQLite locks detect common unsafe aliases and concurrent opens; v1 does not defend against a malicious same-user process racing arbitrary filesystem changes or synchronize open vaults on network/cloud filesystems. Apple OS/WebKit security updates remain part of deployment maintenance.
