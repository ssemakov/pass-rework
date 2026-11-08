# Migration and vault files

The supported route from `ssemakov/pass` is migration. The new app preserves the seven `login_items` fields and adds version/format tables; do not use the new vault in the legacy app. Legacy inputs support SQLCipher compatibility profiles 4 and 3, with the expected legacy schema. Other formats or unexpected schema objects are rejected. Historical SQLCipher raw-key literals can be used as the source password in the import/preservation flow; normal vault passwords and fresh destinations reject those reserved literal forms so password derivation cannot be bypassed.

## First migration

1. Close the old application normally. A source with `-journal`, `-wal` or `-shm` files is refused. Let its original application/engine finish recovery first; do not delete those files to force import.
2. Preserve the original encrypted file. The import dialog's **Preserve the original first** makes a fresh byte-identical encrypted backup, checks the copy hash, opens it with the source password and compares its entries. It never writes to or rekeys the source. An already preserved, clean encrypted backup can be selected directly.
3. Select that backup and a fresh destination filename. Enter the backup password and a new master password. The importer reads the backup without writing it and creates a private encrypted candidate. No plaintext intermediate or export file is used.
4. Before success, the importer closes/reopens the candidate with its new key, runs SQLCipher authentication and SQLite integrity/schema checks, and compares every field, NULL/empty distinction, signed integer ID, timestamp and AUTOINCREMENT sequence. Current-format restores also preserve revisions and vault identity.
5. Only the verified file is atomically published without replacing an existing destination. Keep the untouched original backup, then open the new file with its new password.

An interrupted attempt has no checkpoint. Retrying the same unused destination removes only its validated, app-owned incomplete staging directory and starts from the backup again. Unexpected files, unsafe ownership or permissions cause refusal. If interruption happened after final publication, the completed destination already exists: open it or choose another fresh name; it is never overwritten.

The repository's public synthetic fixture was produced by the actual legacy `@journeyapps/sqlcipher@4.0.0` driver and Knex version. Its observed profile is 4 despite the old source's attempted profile-3 setting; both real profile-3 and profile-4 fixtures are tested. See [fixture provenance](../testdata/legacy/README.md). Compatibility with an arbitrary modified legacy schema is not implied.

## Normal operation and backup

Pass exclusively owns one SQLCipher connection while unlocked. Edit/delete transactions use SQLite's `BEGIN IMMEDIATE`, FULL synchronization, fullfsync and standard rollback-journal recovery. A killed edit is recovered by SQLite on reopening; there is no custom recovery log or replay protocol.

**Save encrypted backup** writes a fresh encrypted snapshot from the owned connection, reopens it and compares every entry before publication. Restore uses the same fresh-destination import flow. It never overwrites the live vault.

**Change master password** verifies the current password, saves a verified encrypted old-password backup, creates/verifies a new-key candidate and atomically replaces the live filename after closing the old connection. The backup filename is reported on success and remains beside the vault. A failure or interruption can leave additional encrypted candidates/backups; the original or verified old-password backup remains the recovery source. This is file replacement for a password change, separate from ordinary transaction recovery.

## Portability and limits

After a normal lock/quit, the vault is one encrypted file and needs only its password on another supported machine. Native coordination files (`.<name>.pass-lock`) hold no key or entries and need not travel with it. Temporary staging, rollback journals and backups are also encrypted where they hold database content. A lock file may remain after quit; its presence alone does not mean the vault is locked by another process.

New vaults/backups use `0600` and staging directories `0700`. Normal opens require private permissions and reject symlinks, hard links, foreign ownership, oversized files and changed file identity. On a copied vault, `chmod 600 /path/to/vault.pass` may be needed. The implementation limits a file to 256 MiB, a result set to 100,000 rows and a bound value to 8 MiB; the UI applies smaller input limits.

Keep the parent directory private. Local SQLite locks and no-replace atomic renames are required. Network filesystems, simultaneous external replacement, cloud conflict resolution and uncooperative filesystem writers are outside v1. Backups and APFS snapshots can retain deleted entries and old passwords; deleting an entry or changing a password is not assured physical erasure of all copies.
