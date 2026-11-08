package vault

import (
	"os"
	"path/filepath"
	"reflect"

	"github.com/ssemakov/pass-rework/internal/cipher"
)

// Import reads the preserved backup only. It never repairs/rekeys/writes it.
// Each call disposes the previous owned incomplete destination and starts over.
func Import(backup, destination string, oldKey, newKey []byte) (int, error) {
	return importChecked(backup, destination, oldKey, newKey, func() bool { return true })
}
func importChecked(backup, destination string, oldKey, newKey []byte, alive func() bool) (count int, err error) {
	if len(newKey) < 1 {
		return 0, ErrInput
	}
	src, e := locate(backup)
	if e != nil {
		return 0, e
	}
	defer src.close()
	before, e := src.check()
	if e != nil {
		return 0, e
	}
	dst, e := locate(destination)
	if e != nil {
		return 0, e
	}
	defer dst.close()
	if src.path == dst.path {
		return 0, ErrFile
	}
	dl, e := dst.lock()
	if e != nil {
		return 0, e
	}
	defer unlock(dl)
	sl, e := src.lock()
	if e != nil {
		return 0, e
	}
	defer unlock(sl)
	if e = src.noJournal(); e != nil {
		return 0, e
	}
	var a *cipher.DB
	for _, profile := range []int{4, 3} {
		a, e = cipher.Open(src.path, oldKey, profile, true)
		if e == nil {
			break
		}
	}
	if e != nil {
		return 0, e
	}
	defer a.Close()
	// Shared SQLite read lock prevents a cooperating DELETE-mode writer committing
	// during backup/verification. WAL inputs are rejected, never copied incompletely.
	if _, e = a.Exec("BEGIN"); e != nil {
		return 0, e
	}
	defer a.Exec("ROLLBACK")
	if e = a.Integrity(); e != nil {
		return 0, e
	}
	if e = src.noJournal(); e != nil {
		return 0, e
	}
	uv, e := a.Query("PRAGMA user_version")
	if e != nil {
		return 0, e
	}
	current := uv[0][0].Int == 1
	if !current && uv[0][0].Int != 0 {
		return 0, ErrFormat
	}
	if e = schema(a, current); e != nil {
		return 0, e
	}
	stage, e := newStaging(dst)
	if e != nil {
		return 0, e
	}
	defer stage.discard()
	loc, e := locate(stage.path)
	if e != nil {
		return 0, e
	}
	defer loc.close()
	f, e := loc.fresh()
	if e != nil {
		return 0, e
	}
	f.Close()
	b, e := cipher.Open(stage.path, newKey, 4, false)
	if e != nil {
		return 0, e
	}
	defer func() { b.Close() }()
	if e = initialise(b); e != nil {
		return 0, e
	}
	rows, e := a.Query("SELECT " + columns + " FROM login_items ORDER BY id")
	if e != nil {
		return 0, e
	}
	e = b.Transaction(func() error {
		for _, r := range rows {
			if !alive() {
				return ErrLocked
			}
			if _, e := b.Exec("INSERT INTO login_items("+columns+") VALUES(?,?,?,?,?,?,?)", r...); e != nil {
				return e
			}
			if _, e := b.Exec("INSERT INTO item_versions VALUES(?,1)", r[0]); e != nil {
				return e
			}
		}
		seq, e := sequence(a)
		if e != nil {
			return e
		}
		if _, e = b.Exec("DELETE FROM sqlite_sequence WHERE name='login_items'"); e != nil {
			return e
		}
		if seq != nil {
			if _, e = b.Exec("INSERT INTO sqlite_sequence(name,seq) VALUES('login_items',?)", seq[0]); e != nil {
				return e
			}
		}
		if current {
			versions, e := a.Query("SELECT id,revision FROM item_versions ORDER BY id")
			if e != nil {
				return e
			}
			if _, e = b.Exec("DELETE FROM item_versions"); e != nil {
				return e
			}
			for _, r := range versions {
				if _, e = b.Exec("INSERT INTO item_versions VALUES(?,?)", r...); e != nil {
					return e
				}
			}
			meta, e := a.Query("SELECT format,uuid FROM pass_meta")
			if e != nil {
				return e
			}
			if _, e = b.Exec("DELETE FROM pass_meta"); e != nil {
				return e
			}
			if _, e = b.Exec("INSERT INTO pass_meta VALUES(?,?)", meta[0]...); e != nil {
				return e
			}
		}
		return nil
	})
	if e != nil {
		return 0, e
	}
	if e = b.Close(); e != nil {
		return 0, e
	}
	b, e = cipher.Open(stage.path, newKey, 4, true)
	if e != nil {
		return 0, e
	}
	if e = schema(b, true); e != nil {
		return 0, e
	}
	if e = b.Integrity(); e != nil {
		return 0, e
	}
	if e = compare(a, b, current); e != nil {
		return 0, e
	}
	if e = b.Close(); e != nil {
		return 0, e
	}
	after, e := src.check()
	if e != nil || !os.SameFile(before, after) {
		return 0, ErrFile
	}
	if e = loc.noJournal(); e != nil {
		return 0, e
	}
	if !alive() {
		return 0, ErrLocked
	}
	if e = stage.publish(); e != nil {
		return 0, e
	}
	return len(rows), nil
}

// PreserveBackup copies only a clean encrypted source while holding SQLite's
// read transaction, then reopens and compares every entry before success.
func PreserveBackup(source, destination string, key []byte) error {
	src, e := locate(source)
	if e != nil {
		return e
	}
	defer src.close()
	dst, e := locate(destination)
	if e != nil {
		return e
	}
	defer dst.close()
	if src.path == dst.path {
		return ErrFile
	}
	sl, e := src.lock()
	if e != nil {
		return e
	}
	defer unlock(sl)
	dl, e := dst.lock()
	if e != nil {
		return e
	}
	defer unlock(dl)
	var a *cipher.DB
	profile := 4
	for _, p := range []int{4, 3} {
		a, e = cipher.Open(src.path, key, p, true)
		if e == nil {
			profile = p
			break
		}
	}
	if e != nil {
		return e
	}
	defer a.Close()
	if _, e = a.Exec("BEGIN"); e != nil {
		return e
	}
	defer a.Exec("ROLLBACK")
	if e = a.Integrity(); e != nil {
		return e
	}
	if e = copyEncrypted(src, dst); e != nil {
		return e
	}
	b, e := cipher.Open(dst.path, key, profile, true)
	if e != nil {
		return e
	}
	defer b.Close()
	if e = b.Integrity(); e != nil {
		return e
	}
	x, e := a.Query("SELECT " + columns + " FROM login_items ORDER BY id")
	if e != nil {
		return e
	}
	y, e := b.Query("SELECT " + columns + " FROM login_items ORDER BY id")
	if e != nil || !reflect.DeepEqual(x, y) {
		return ErrFormat
	}
	return nil
}
func newVault(path string, key []byte) error {
	dst, e := locate(path)
	if e != nil {
		return e
	}
	defer dst.close()
	lk, e := dst.lock()
	if e != nil {
		return e
	}
	defer unlock(lk)
	s, e := newStaging(dst)
	if e != nil {
		return e
	}
	defer s.discard()
	loc, e := locate(s.path)
	if e != nil {
		return e
	}
	defer loc.close()
	f, e := loc.fresh()
	if e != nil {
		return e
	}
	f.Close()
	d, e := cipher.Open(s.path, key, 4, false)
	if e != nil {
		return e
	}
	defer d.Close()
	if e = initialise(d); e != nil {
		return e
	}
	if e = d.Integrity(); e != nil {
		return e
	}
	if e = d.Close(); e != nil {
		return e
	}
	return s.publish()
}
func sameParentCandidate(path string) string {
	return filepath.Join(filepath.Dir(path), ".pass-rotate-"+randomID())
}
