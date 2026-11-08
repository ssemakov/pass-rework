package vault

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/ssemakov/pass-rework/internal/cipher"
)

func TestRawKeyLiteralCannotBypassNewVaultPasswordKDF(t *testing.T) {
	for _, n := range []int{64, 96, 160} {
		for _, prefix := range []string{"x", "X"} {
			t.Run(prefix+strconv.Itoa(n), func(t *testing.T) {
				dir := t.TempDir()
				key := []byte(prefix + "'" + strings.Repeat("aB", n/2) + "'")
				fresh := filepath.Join(dir, "refused.pass")
				if e := newVault(fresh, key); !errors.Is(e, ErrMasterFormat) {
					t.Fatal("raw key accepted as new password", e)
				}
				if _, e := os.Stat(fresh); !os.IsNotExist(e) {
					t.Fatal("rejected password created a vault")
				}
				// This is a genuine upstream raw-key file; the reported KDF settings
				// still look normal. Old sources must remain readable for migration.
				source := filepath.Join(dir, "legacy-raw.pass")
				if e := os.WriteFile(source, nil, 0600); e != nil {
					t.Fatal(e)
				}
				db, e := cipher.Open(source, key, 4, false)
				if e != nil {
					t.Fatal(e)
				}
				if _, e = db.Exec(createItems); e != nil {
					t.Fatal(e)
				}
				if _, e = db.Exec("INSERT INTO login_items(name,password) VALUES(?,?)", cipher.S("raw legacy"), cipher.S("public synthetic source secret")); e != nil {
					t.Fatal(e)
				}
				db.Close()
				before, _ := os.ReadFile(source)
				normal := New(nil, nil)
				e = normal.Unlock(source, key)
				normal.Close()
				if !errors.Is(e, ErrMasterFormat) {
					t.Fatal("normal open accepted a raw-key password", e)
				}
				refused := filepath.Join(dir, "refused-import.pass")
				if _, e = Import(source, refused, key, key); !errors.Is(e, ErrMasterFormat) {
					t.Fatal("raw destination password accepted", e)
				}
				preserved := filepath.Join(dir, "preserved.pass")
				if e = PreserveBackup(source, preserved, key); e != nil {
					t.Fatal("raw source preservation", e)
				}
				copyBytes, _ := os.ReadFile(preserved)
				if !bytes.Equal(before, copyBytes) {
					t.Fatal("preservation changed raw-key source")
				}
				destination := filepath.Join(dir, "migrated.pass")
				if count, e := Import(preserved, destination, key, testKey); e != nil || count != 1 {
					t.Fatal("raw source migration", count, e)
				}
				after, _ := os.ReadFile(source)
				if !bytes.Equal(before, after) {
					t.Fatal("raw-key original modified")
				}
				m := New(nil, nil)
				defer m.Close()
				if e = m.Unlock(destination, testKey); e != nil {
					t.Fatal(e)
				}
				rows, e := m.List(m.Generation(), "")
				if e != nil || len(rows) != 1 {
					t.Fatal("migrated entry", e)
				}
				entry, e := m.Get(m.Generation(), rows[0].ID)
				if e != nil || entry.Password == nil || *entry.Password != "public synthetic source secret" {
					t.Fatal("migrated secret changed", e)
				}
			})
		}
	}
}

func TestRawKeyNearMissUsesOrdinaryPasswordAndRotationRejectsLiteral(t *testing.T) {
	for _, key := range [][]byte{[]byte("x'" + strings.Repeat("0", 63) + "g'"), []byte("x'0000'")} {
		p := filepath.Join(t.TempDir(), "ordinary.pass")
		m := New(nil, nil)
		if e := m.Create(p, key); e != nil {
			t.Fatal("ordinary passphrase refused", e)
		}
		g := m.Generation()
		if _, e := m.Rotate(g, key, []byte("x'"+strings.Repeat("0", 64)+"'")); !errors.Is(e, ErrMasterFormat) {
			t.Fatal("raw replacement accepted", e)
		}
		if m.Generation() != g || m.State().Locked {
			t.Fatal("rejected password changed session")
		}
		backups, _ := filepath.Glob(filepath.Join(filepath.Dir(p), "Pass-before-password-change-*"))
		if len(backups) != 0 {
			t.Fatal("rejected replacement produced a backup")
		}
		m.Lock()
		if e := m.Unlock(p, key); e != nil {
			t.Fatal("ordinary passphrase did not reopen", e)
		}
		m.Close()
	}
}
