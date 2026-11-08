package vault

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/ssemakov/pass-rework/internal/cipher"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestHistoricalJourneyappsFixture(t *testing.T) {
	src := filepath.Join("..", "..", "testdata", "legacy", "journeyapps-4.0.0.pass")
	before, e := os.ReadFile(src)
	if e != nil {
		t.Fatal(e)
	}
	dst := filepath.Join(t.TempDir(), "migrated.pass")
	key := []byte("public-fixture-password")
	if n, e := Import(src, dst, key, testKey); e != nil || n != 2 {
		t.Fatal("historic driver migration", n, e)
	}
	after, _ := os.ReadFile(src)
	if !bytes.Equal(before, after) {
		t.Fatal("historic fixture changed")
	}
	a, e := cipher.Open(src, key, 4, true)
	if e != nil {
		t.Fatal("actual historic profile is 4", e)
	}
	defer a.Close()
	b, e := cipher.Open(dst, testKey, 4, true)
	if e != nil {
		t.Fatal(e)
	}
	defer b.Close()
	if e = compare(a, b, false); e != nil {
		t.Fatal(e)
	}
	var expected struct {
		Rows []struct {
			ID       int64  `json:"id"`
			Password string `json:"password"`
		}
	}
	raw, _ := os.ReadFile(filepath.Join("..", "..", "testdata", "legacy", "expected.json"))
	if json.Unmarshal(raw, &expected) != nil {
		t.Fatal("fixture provenance")
	}
	for _, r := range expected.Rows {
		rows, e := b.Query("SELECT password FROM login_items WHERE id=?", cipher.I(r.ID))
		if e != nil || len(rows) != 1 || rows[0][0].Text != r.Password {
			t.Fatal("historic value mismatch", r.ID, e)
		}
	}
	sum := sha256.Sum256(before)
	t.Log("historic fixture SHA256", hex.EncodeToString(sum[:]))
}
func TestPlainSQLiteCannotReadVaultAndPlaintextCannotOpen(t *testing.T) {
	p := filepath.Join(t.TempDir(), "encrypted.pass")
	if e := newVault(p, testKey); e != nil {
		t.Fatal(e)
	}
	script := `import sqlite3,sys
try:
 db=sqlite3.connect('file:'+sys.argv[1]+'?mode=ro',uri=True); db.execute('select * from sqlite_schema').fetchall()
except sqlite3.DatabaseError: sys.exit(0)
sys.exit(1)`
	if out, e := exec.Command("python3", "-c", script, p).CombinedOutput(); e != nil {
		t.Fatalf("plain SQLite could read encrypted file, or probe failed: %v %s", e, out)
	}
	plain := filepath.Join(t.TempDir(), "plain.sqlite")
	if out, e := exec.Command("python3", "-c", `import sqlite3,sys;db=sqlite3.connect(sys.argv[1]);db.execute('create table login_items(id integer)');db.commit()`, plain).CombinedOutput(); e != nil {
		t.Fatal(e, string(out))
	}
	if d, e := cipher.Open(plain, testKey, 4, true); e == nil {
		d.Close()
		t.Fatal("plaintext accepted as vault")
	}
}
func TestLiveRollbackJournalContainsNoPlaintext(t *testing.T) {
	p := filepath.Join(t.TempDir(), "journal.pass")
	if e := newVault(p, testKey); e != nil {
		t.Fatal(e)
	}
	d, e := cipher.Open(p, testKey, 4, false)
	if e != nil {
		t.Fatal(e)
	}
	defer d.Close()
	secret := "public-long-sentinel-password-that-must-not-be-in-the-journal"
	if _, e = create(d, Input{Name: ptr("sentinel-entry"), Password: &secret}); e != nil {
		t.Fatal(e)
	}
	e = d.Transaction(func() error {
		if _, e := d.Exec("UPDATE login_items SET password='other public synthetic password'"); e != nil {
			return e
		}
		journal, e := os.ReadFile(p + "-journal")
		if e != nil {
			return e
		}
		if bytes.Contains(journal, []byte(secret)) || bytes.Contains(journal, []byte("sentinel-entry")) {
			t.Fatal("journal leaked plaintext")
		}
		return nil
	})
	if e != nil {
		t.Fatal(e)
	}
}
