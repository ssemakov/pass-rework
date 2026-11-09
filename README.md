# Pass

An offline macOS password manager implemented with Go, Wails, React/TypeScript and SQLCipher. Work is on `implementation/option-d`; the owner chooses what enters `main` and when to release.

- Create, open, search, add, edit and delete login entries. Editing metadata preserves the existing password unless explicitly replaced.
- Reveal credentials on demand or copy through the native clipboard. Auto-lock after 1, 5 or 15 minutes, on sleep/screen lock, or with **⌘~**. Clipboard content is cleared after 30 seconds only while Pass still owns it.
- One portable encrypted vault, without an account, cloud service, remembered machine key or plaintext export.
- Preserve a legacy encrypted backup, migrate or restore into a fresh destination, and verify every field of every entry before success. An interrupted import discards its incomplete destination on retry and starts over.
- Save verified encrypted backups and change the master password while retaining an encrypted backup protected by the old password.

The UI mirrors legacy Pass: its gray background, indigo controls, centered password field, search/count header, stacked secret cards and full-page add form. The default outer window is **904 × 534**. Use **File → Open Storage…** to choose a vault, **File → New Storage…** to create one, and the plus button to add a secret. Search displays matches after three characters, as in the legacy app. Copy/reveal controls appear on hover or keyboard focus; edit/delete controls use the same icon style. Backup, import, password change and settings live in the native File menu. See [UI matching and evidence](docs/UI-RESTORATION.md).

The deployment floor is **macOS 15**, with native CI for 15/26 on Apple Silicon and Intel. Future beta compatibility is exploratory. The [validation record](docs/VALIDATION.md) distinguishes tested builds from support promises.

Build on macOS with Xcode Command Line Tools, Go **1.27.3**, Node **24**, Python 3, Make and curl:

```sh
git switch implementation/option-d
./scripts/prepare-sqlcipher.sh
npm --prefix frontend ci --ignore-scripts --no-audit --no-fund
./scripts/package-macos.sh
open build/Pass.app
```

SQLCipher source and generated C/header hashes are checked before building; no system SQLite driver is used. The native app embeds its frontend. A frontend dev server alone cannot open a vault. Local/CI bundles are ad-hoc signed; the [distribution instructions](docs/DISTRIBUTION.md) cover Developer ID signing and notarization.

For your first migration, close the legacy app, choose **File → Import Storage…**, and use **Preserve original backup** if you do not already have a closed, clean encrypted backup. Select that preserved backup, choose a new unused destination and enter the old/new passwords. Keep the original and preserved backup. Open the new vault only after the every-entry verification succeeds. See [migration and file handling](docs/MIGRATION.md).

Use a strong master passphrase: there is no password reset. Lock or quit before moving a vault in Finder; use **File → Save Encrypted Backup…** for a consistent copy while unlocked. Keep files on a local filesystem with working SQLite locks and atomic renames. Do not synchronize an open vault through a cloud folder.

See [storage encryption evidence and threat model](docs/SECURITY.md), [adversarial implementation review](docs/IMPLEMENTATION-REVIEW.md), and [test commands/results](docs/VALIDATION.md). A commissioned external human audit is omitted from v1 as agreed; this project makes no audit or certification claim. No real vaults, passwords or exports belong in this repository or CI.

The earlier [four-option proposal](proposals/2026-11-rework/OPTIONS.md), [UI comparison](proposals/2026-11-rework/UI-COMPARISON.md) and [discussion decisions](proposals/2026-11-rework/DISCUSSION-DECISIONS.md) are preserved as design history. This implementation selects D and supersedes their decision-pending wording.
