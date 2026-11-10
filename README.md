# PassV2

An offline macOS password manager implemented with Go, Wails, React/TypeScript and SQLCipher. Work is on `implementation/option-d`; the owner chooses what enters `main` and when to release.

- Create, open, search, add, edit and delete login entries. Editing metadata preserves the existing password unless explicitly replaced.
- Reveal credentials on demand or copy through the native clipboard. Auto-lock after 1, 5 or 15 minutes, on sleep/screen lock, or with **⌘~**. Clipboard content is cleared after 30 seconds only while Pass still owns it.
- One portable encrypted vault, without an account, cloud service, remembered machine key or plaintext export.
- Preserve a legacy encrypted backup, migrate or restore into a fresh destination, and verify every field of every entry before success. An interrupted import discards its incomplete destination on retry and starts over.
- Save verified encrypted backups and change the master password while retaining an encrypted backup protected by the old password.

The UI mirrors legacy Pass: its gray background, indigo controls, centered password field, search/count header, stacked secret cards and full-page add form. The default outer window is **904 × 534**. First launch shows Create Storage with Import/Open alternatives. Later launches read the remembered storage path and show the password field; a rejected password only shows “Wrong password”. Use **File → Open Storage…** to switch vaults, **File → New Storage…** to create another, and the plus button to add a secret. Search displays matches after three characters, as in the legacy app. Copy/reveal controls appear on hover or keyboard focus; edit/delete controls use the same icon style. Backup, import, password change and settings live in the native File menu. See [UI matching and evidence](docs/UI-RESTORATION.md) and the [complete flow trace](docs/FLOWS.md).

Settings are saved at **`~/Library/Application Support/PassV2/settings.json`**, with the last successfully opened/created/imported `default_storage_path` and `idle_minutes`. This private, unencrypted settings file contains no passwords, keys or entries. The initial vault location is `storage.pass` in that directory; the folder button lets you choose another location. Passwords are never remembered. Settings from the earlier Pass Rework directory are not migrated. Existing storage files are not moved or modified; select one using Open Storage if needed.

Run **`./scripts/verify-local.sh`** to validate locally without GitHub Actions. It runs race/vet/build/unit/browser checks and restores production assets. On macOS it additionally runs the two-process native restart/incorrect-password test, builds the app, checks its encryption linkage/signature and scans the packaged binary. Logs and a local results manifest are written under `artifacts/`. Only temporary synthetic vaults are used; the native smoke does not read your real app settings or vault.

The deployment floor is **macOS 15**, targeting 15/26 on Apple Silicon and Intel. Earlier versions have recorded native platform evidence; current startup changes are locally validated on Linux and require the native local command for macOS execution evidence. The [validation record](docs/VALIDATION.md) separates those results. Verification does not depend on CI.

Build on macOS with Xcode Command Line Tools, Go **1.27.3**, Node **24**, Python 3, Make and curl:

```sh
git switch implementation/option-d
./scripts/prepare-sqlcipher.sh
npm --prefix frontend ci --ignore-scripts --no-audit --no-fund
./scripts/package-macos.sh
open build/Pass.app
```

SQLCipher source and generated C/header hashes are checked before building; no system SQLite driver is used. The native app embeds its frontend. A frontend dev server alone cannot open a vault. Local/CI bundles are ad-hoc signed; the [distribution instructions](docs/DISTRIBUTION.md) cover Developer ID signing and notarization.

For migration, close the original app and choose **File → Import Storage…**. Select the original encrypted file and a new unused destination, enter the original password and matching new/confirmation passwords, then import. The source is always kept unchanged; there is no extra backup-preservation prompt. “Imported N secrets” remains on the import screen; **Open storage** takes you to the new file's login without another picker. Opening a validated V1 file offers the same migration after explicit consent. See [migration and file handling](docs/MIGRATION.md).

Use a strong master passphrase: there is no password reset. Lock or quit before moving a vault in Finder; use **File → Save Encrypted Backup…** for a consistent copy while unlocked. Keep files on a local filesystem with working SQLite locks and atomic renames. Do not synchronize an open vault through a cloud folder.

See [storage encryption evidence and threat model](docs/SECURITY.md), [adversarial implementation review](docs/IMPLEMENTATION-REVIEW.md), and [test commands/results](docs/VALIDATION.md). A commissioned external human audit is omitted from v1 as agreed; this project makes no audit or certification claim. No real vaults, passwords or exports belong in this repository or CI.

The earlier [four-option proposal](proposals/2026-11-rework/OPTIONS.md), [UI comparison](proposals/2026-11-rework/UI-COMPARISON.md) and [discussion decisions](proposals/2026-11-rework/DISCUSSION-DECISIONS.md) are preserved as design history. This implementation selects D and supersedes their decision-pending wording.

Application version is **2.0.2**. `VERSION` supplies the native About/bundle version; `scripts/bump-version.py patch` or `minor` keeps package metadata synchronized. Patch bumps are for fixes; minor bumps are for features. The major version changes only on the owner's explicit instruction. See [2.0.2 changes and verification](docs/UX-2.0.2.md).
