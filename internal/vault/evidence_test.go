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
	"reflect"
	"runtime"
	"testing"
	"time"
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

func TestPinnedEngineConfiguration(t *testing.T) {
	p := filepath.Join(t.TempDir(), "engine.pass")
	if e := newVault(p, testKey); e != nil {
		t.Fatal(e)
	}
	d, e := cipher.Open(p, testKey, 4, true)
	if e != nil {
		t.Fatal(e)
	}
	defer d.Close()
	for _, pragma := range []string{"cipher_version", "cipher_provider", "cipher_provider_version", "cipher_settings", "temp_store"} {
		r, e := d.Query("PRAGMA " + pragma)
		if e != nil {
			t.Fatal(e)
		}
		t.Log(pragma, r)
	}
	r, e := d.Query("PRAGMA cipher_settings")
	expected := [][]cipher.Value{
		{cipher.S("PRAGMA kdf_iter = 256000;")},
		{cipher.S("PRAGMA cipher_page_size = 4096;")},
		{cipher.S("PRAGMA cipher_use_hmac = 1;")},
		{cipher.S("PRAGMA cipher_plaintext_header_size = 0;")},
		{cipher.S("PRAGMA cipher_hmac_algorithm = HMAC_SHA512;")},
		{cipher.S("PRAGMA cipher_kdf_algorithm = PBKDF2_HMAC_SHA512;")},
	}
	if e != nil || !reflect.DeepEqual(r, expected) {
		t.Fatal("unexpected cipher settings", r, e)
	}
	provider := "openssl"
	if runtime.GOOS == "darwin" {
		provider = "commoncrypto"
	}
	r, e = d.Query("PRAGMA cipher_provider")
	if e != nil || !reflect.DeepEqual(r, [][]cipher.Value{{cipher.S(provider)}}) {
		t.Fatal("unexpected crypto provider", r, e)
	}
	r, e = d.Query("PRAGMA temp_store")
	if e != nil || !reflect.DeepEqual(r, [][]cipher.Value{{cipher.I(2)}}) {
		t.Fatal("temporary storage must be memory", r, e)
	}
}

func TestCRUDCrashChild(t *testing.T) {
	if os.Getenv("PASS_TEST_CRUD_CHILD") != "1" {
		t.Skip("subprocess helper")
	}
	p := os.Getenv("PASS_TEST_LIVE")
	d, e := cipher.Open(p, testKey, 4, false)
	if e != nil {
		t.Fatal(e)
	}
	d.Exec("PRAGMA cache_size=1")
	if _, e = d.Exec("BEGIN IMMEDIATE"); e != nil {
		t.Fatal(e)
	}
	if _, e = d.Exec("UPDATE login_items SET password='after-uncommitted-synthetic-password'"); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(p+".test-ready", []byte("synthetic transaction ready"), 0600); e != nil {
		t.Fatal(e)
	}
	time.Sleep(time.Hour)
}
func TestSQLiteRecoversKilledEditWithoutCustomRecovery(t *testing.T) {
	p := filepath.Join(t.TempDir(), "live.pass")
	m := New(nil, nil)
	if e := m.Create(p, testKey); e != nil {
		t.Fatal(e)
	}
	r, e := m.CreateEntry(m.Generation(), Input{Name: ptr("existing"), Password: ptr("before-committed-synthetic-password")})
	if e != nil {
		t.Fatal(e)
	}
	m.Close()
	child := exec.Command(os.Args[0], "-test.run=^TestCRUDCrashChild$")
	child.Env = append(os.Environ(), "PASS_TEST_CRUD_CHILD=1", "PASS_TEST_LIVE="+p)
	if e = child.Start(); e != nil {
		t.Fatal(e)
	}
	defer child.Process.Kill()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if _, e = os.Stat(p + ".test-ready"); e == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, e = os.Stat(p + "-journal"); e != nil {
		t.Fatal("no live journal", e)
	}
	if e = child.Process.Kill(); e != nil {
		t.Fatal(e)
	}
	child.Wait()
	recovered := New(nil, nil)
	defer recovered.Close()
	if e = recovered.Unlock(p, testKey); e != nil {
		t.Fatal("engine recovery failed", e)
	}
	row, e := recovered.Get(recovered.Generation(), r.ID)
	if e != nil || row.Password == nil || *row.Password != "before-committed-synthetic-password" {
		t.Fatal("uncommitted edit survived", e)
	}
}
