package vault

import (
	"encoding/hex"
	"errors"
	"github.com/ssemakov/pass-rework/internal/cipher"
)

// Classify only authenticated metadata before any writable open. Current V2
// rows may be mid-transaction after a crash; SQLite recovers and validates those
// on the normal handle. A legacy file is never recovered or changed here.
func classifyStorage(loc *location, key []byte, alive func() bool) error {
	journal := loc.noJournal() != nil
	for _, profile := range []int{4, 3} {
		if !alive() {
			return ErrLocked
		}
		db, err := cipher.OpenImmutable(loc.path, key, profile)
		if err != nil {
			if !errors.Is(err, cipher.ErrKey) {
				return err
			}
			continue
		}
		err = classifyMetadata(db, profile, journal)
		db.Close()
		if !alive() {
			return ErrLocked
		}
		return err
	}
	if !alive() {
		return ErrLocked
	}
	if journal {
		return ErrRecovery
	}
	return cipher.ErrKey
}
func classifyMetadata(db *cipher.DB, profile int, journal bool) error {
	u, err := db.Query("PRAGMA user_version")
	if err != nil || len(u) != 1 || len(u[0]) != 1 {
		return ErrFormat
	}
	if u[0][0].Int == 0 {
		if journal {
			return ErrRecovery
		}
		if err = schema(db, false); err != nil {
			return err
		}
		if err = db.Integrity(); err != nil {
			return err
		}
		return ErrLegacy
	}
	if u[0][0].Int != 1 || profile != 4 {
		return ErrFormat
	}
	rows, err := db.Query("SELECT type,name,tbl_name,sql FROM sqlite_schema WHERE name != 'sqlite_sequence' ORDER BY name")
	if err != nil || len(rows) != 3 {
		return ErrFormat
	}
	names := []string{"item_versions", "login_items", "pass_meta"}
	definitions := []string{createVersions, createItems, createMeta}
	for i, r := range rows {
		if len(r) != 4 || r[0].Text != "table" || r[1].Text != names[i] || r[2].Text != names[i] || r[3].Text != definitions[i] {
			return ErrFormat
		}
	}
	meta, err := db.Query("SELECT format,uuid FROM pass_meta")
	if err != nil || len(meta) != 1 || len(meta[0]) != 2 || meta[0][0].Text != "pass-rework-v1" || len(meta[0][1].Text) != 32 {
		return ErrFormat
	}
	if _, err = hex.DecodeString(meta[0][1].Text); err != nil {
		return ErrFormat
	}
	return nil
}
