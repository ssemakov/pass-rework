package vault

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"github.com/ssemakov/pass-rework/internal/cipher"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func legacyFixture(t *testing.T, p string, profile int) {
	t.Helper()
	if e := os.WriteFile(p, nil, 0600); e != nil {
		t.Fatal(e)
	}
	d, e := cipher.Open(p, testKey, profile, false)
	if e != nil {
		t.Fatal(e)
	}
	defer d.Close()
	if _, e = d.Exec(createItems); e != nil {
		t.Fatal(e)
	}
	for _, r := range [][]cipher.Value{{cipher.I(7), cipher.N(), cipher.S(""), cipher.S("first synthetic sentinel"), cipher.N(), cipher.N(), cipher.I(-19)}, {cipher.I(9007199254740993), cipher.S("O'Connor 🔐\x00tail"), cipher.N(), cipher.S("last synthetic secret\x00🍋"), cipher.S("https://example.test"), cipher.I(9223372036854775807), cipher.I(-123)}} {
		if _, e = d.Exec("INSERT INTO login_items("+columns+") VALUES(?,?,?,?,?,?,?)", r...); e != nil {
			t.Fatal(e)
		}
	}
	if _, e = d.Exec("UPDATE sqlite_sequence SET seq=? WHERE name='login_items'", cipher.I(9007199254741093)); e != nil {
		t.Fatal(e)
	}
}
func TestMigrationProfilesEveryEntryAndSequence(t *testing.T) {
	for _, profile := range []int{3, 4} {
		t.Run(strconv.Itoa(profile), func(t *testing.T) {
			dir := t.TempDir()
			backup := filepath.Join(dir, "backup.pass")
			destination := filepath.Join(dir, "new.pass")
			legacyFixture(t, backup, profile)
			original, _ := os.ReadFile(backup)
			key := []byte("new public synthetic migration key")
			count, e := Import(backup, destination, testKey, key)
			if e != nil || count != 2 {
				t.Fatal("import", count, e)
			}
			after, _ := os.ReadFile(backup)
			if !bytes.Equal(original, after) {
				t.Fatal("backup modified")
			}
			a, e := cipher.Open(backup, testKey, profile, true)
			if e != nil {
				t.Fatal(e)
			}
			defer a.Close()
			b, e := cipher.Open(destination, key, 4, true)
			if e != nil {
				t.Fatal(e)
			}
			if e = compare(a, b, false); e != nil {
				t.Fatal(e)
			}
			b.Close()
			m := New(nil, nil)
			defer m.Close()
			if e = m.Unlock(destination, key); e != nil {
				t.Fatal(e)
			}
			all, e := m.List(m.Generation(), "")
			if e != nil || len(all) != 2 {
				t.Fatal(e)
			}
			r, e := m.CreateEntry(m.Generation(), Input{Name: ptr("next")})
			if e != nil || r.ID != "9007199254741094" {
				t.Fatal("sequence loss", r.ID, e)
			}
			m.Lock()
			if _, e = Import(backup, destination, testKey, key); !errors.Is(e, ErrExists) {
				t.Fatal("destination overwritten", e)
			}
		})
	}
}
func TestImportCancelRestartAndWrongKey(t *testing.T) {
	dir := t.TempDir()
	src, dst := filepath.Join(dir, "backup.pass"), filepath.Join(dir, "new.pass")
	legacyFixture(t, src, 3)
	original, _ := os.ReadFile(src)
	attempts := 0
	if _, e := importChecked(src, dst, testKey, testKey, func() bool { attempts++; return attempts < 2 }); !errors.Is(e, ErrLocked) {
		t.Fatal("cancel ignored", e)
	}
	if _, e := os.Stat(dst); !os.IsNotExist(e) {
		t.Fatal("partial destination published")
	}
	if _, e := Import(src, dst, []byte("wrong"), testKey); e == nil {
		t.Fatal("wrong-key import succeeded")
	}
	if _, e := Import(src, dst, testKey, testKey); e != nil {
		t.Fatal("restart failed", e)
	}
	after, _ := os.ReadFile(src)
	if !bytes.Equal(original, after) {
		t.Fatal("source changed")
	}
}
func TestStagingInterruptedSetupAndAtomicPublication(t *testing.T) {
	for _, owner := range []string{"absent", "", "pass-rework disposable"} {
		t.Run(owner, func(t *testing.T) {
			dir := t.TempDir()
			dst, e := locate(filepath.Join(dir, "new.pass"))
			if e != nil {
				t.Fatal(e)
			}
			defer dst.close()
			s, e := newStaging(dst)
			if e != nil {
				t.Fatal(e)
			}
			name := s.name
			s.root.Close()
			os.Remove(filepath.Join(dir, name, "owner"))
			if owner != "absent" {
				os.WriteFile(filepath.Join(dir, name, "owner"), []byte(owner), 0600)
			}
			s, e = newStaging(dst)
			if e != nil {
				t.Fatal("setup not restartable", e)
			}
			defer s.discard()
			if e = os.WriteFile(s.path, []byte("encrypted candidate fixture"), 0600); e != nil {
				t.Fatal(e)
			}
			if e = s.publish(); e != nil {
				t.Fatal(e)
			}
			st, e := dst.check()
			if e != nil || !regular(st) {
				t.Fatal("publication left hardlinks", e)
			}
		})
	}
}
func TestUnexpectedStageNeverDeleted(t *testing.T) {
	dst, _ := locate(filepath.Join(t.TempDir(), "new.pass"))
	defer dst.close()
	s, e := newStaging(dst)
	if e != nil {
		t.Fatal(e)
	}
	defer s.discard()
	s.root.WriteFile("unrelated", []byte("keep"), 0600)
	s.root.Close()
	if _, e = newStaging(dst); e == nil {
		t.Fatal("unexpected stage accepted")
	}
	b, e := os.ReadFile(filepath.Join(filepath.Dir(s.path), "unrelated"))
	if e != nil || string(b) != "keep" {
		t.Fatal("unrelated file deleted")
	}
}
func TestBackupRotateAndRestore(t *testing.T) {
	dir := t.TempDir()
	path, backup, restored := filepath.Join(dir, "live.pass"), filepath.Join(dir, "backup.pass"), filepath.Join(dir, "restored.pass")
	m := New(nil, nil)
	defer m.Close()
	if e := m.Create(path, testKey); e != nil {
		t.Fatal(e)
	}
	g := m.Generation()
	if _, e := m.CreateEntry(g, Input{Name: ptr("entry"), Password: ptr("secret-sentinel")}); e != nil {
		t.Fatal(e)
	}
	if e := m.Backup(g, backup); e != nil {
		t.Fatal(e)
	}
	backupBytes, _ := os.ReadFile(backup)
	next := []byte("replacement public synthetic master password")
	rollback, e := m.Rotate(g, testKey, next)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = os.Stat(rollback); e != nil {
		t.Fatal("old backup missing")
	}
	m.Lock()
	if e = m.Unlock(path, testKey); e == nil {
		t.Fatal("old password still works")
	}
	if e = m.Unlock(path, next); e != nil {
		t.Fatal(e)
	}
	m.Lock()
	if _, e = m.Import(backup, restored, testKey, next); e != nil {
		t.Fatal(e)
	}
	if e = m.Unlock(restored, next); e != nil {
		t.Fatal(e)
	}
	all, e := m.List(m.Generation(), "")
	if e != nil || len(all) != 1 {
		t.Fatal("restore lost data", e)
	}
	after, _ := os.ReadFile(backup)
	if !bytes.Equal(backupBytes, after) {
		t.Fatal("backup rekeyed")
	}
}
func TestSchemaTamperingAndAliases(t *testing.T) {
	for _, mutation := range []string{"CREATE TRIGGER sqliteXwipe AFTER UPDATE ON login_items BEGIN DELETE FROM login_items; END", "DROP TABLE item_versions; CREATE TABLE item_versions(id INTEGER,revision INTEGER)", "INSERT INTO item_versions VALUES(777,1)"} {
		t.Run(mutation, func(t *testing.T) {
			p := filepath.Join(t.TempDir(), "vault.pass")
			if e := newVault(p, testKey); e != nil {
				t.Fatal(e)
			}
			d, e := cipher.Open(p, testKey, 4, false)
			if e != nil {
				t.Fatal(e)
			}
			d.Exec("PRAGMA foreign_keys=OFF")
			for _, sql := range splitSQL(mutation) {
				if _, e = d.Exec(sql); e != nil {
					t.Fatal(e)
				}
			}
			d.Close()
			m := New(nil, nil)
			defer m.Close()
			if e = m.Unlock(p, testKey); e == nil {
				t.Fatal("malformed schema accepted")
			}
		})
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "vault.pass")
	newVault(p, testKey)
	alias := filepath.Join(dir, "alias.pass")
	os.Symlink(p, alias)
	m := New(nil, nil)
	defer m.Close()
	if e := m.Unlock(alias, testKey); e == nil {
		t.Fatal("symlink accepted")
	}
	os.Remove(alias)
	os.Link(p, alias)
	if e := m.Unlock(alias, testKey); e == nil {
		t.Fatal("hardlink accepted")
	}
}
func splitSQL(s string) []string {
	if s == "DROP TABLE item_versions; CREATE TABLE item_versions(id INTEGER,revision INTEGER)" {
		return []string{"DROP TABLE item_versions", "CREATE TABLE item_versions(id INTEGER,revision INTEGER)"}
	}
	return []string{s}
}
func TestImportCrashChild(t *testing.T) {
	if os.Getenv("PASS_TEST_IMPORT_CHILD") != "1" {
		t.Skip("subprocess helper")
	}
	n := 0
	_, _ = importChecked(os.Getenv("PASS_TEST_BACKUP"), os.Getenv("PASS_TEST_DEST"), testKey, testKey, func() bool {
		n++
		if n == 2 {
			time.Sleep(time.Hour)
		}
		return true
	})
	t.Fatal("child did not pause")
}
func TestKilledImportRestartsFromBackup(t *testing.T) {
	dir := t.TempDir()
	src, dst := filepath.Join(dir, "backup.pass"), filepath.Join(dir, "new.pass")
	legacyFixture(t, src, 3)
	before, _ := os.ReadFile(src)
	hash := sha256.Sum256(before)
	child := exec.Command(os.Args[0], "-test.run=^TestImportCrashChild$")
	child.Env = append(os.Environ(), "PASS_TEST_IMPORT_CHILD=1", "PASS_TEST_BACKUP="+src, "PASS_TEST_DEST="+dst)
	if e := child.Start(); e != nil {
		t.Fatal(e)
	}
	defer child.Process.Kill()
	deadline := time.Now().Add(10 * time.Second)
	found := false
	for time.Now().Before(deadline) {
		stages, _ := filepath.Glob(filepath.Join(dir, ".pass-stage-*", "candidate-journal"))
		if len(stages) > 0 {
			found = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !found {
		t.Fatal("no interrupted transaction observed")
	}
	if e := child.Process.Kill(); e != nil {
		t.Fatal("child not alive at interruption", e)
	}
	child.Wait()
	if child.ProcessState.ExitCode() != -1 {
		t.Fatal("child was not signal-interrupted")
	}
	if _, e := Import(src, dst, testKey, testKey); e != nil {
		t.Fatal("fresh restart failed", e)
	}
	after, _ := os.ReadFile(src)
	if sha256.Sum256(after) != hash {
		t.Fatal("original encrypted backup changed")
	}
}

func TestSignedIDsAndUnsupportedText(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "legacy.pass")
	legacyFixture(t, src, 4)
	d, e := cipher.Open(src, testKey, 4, false)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = d.Exec("INSERT INTO login_items(id,name) VALUES(-1,'negative id'),(0,'zero id')"); e != nil {
		t.Fatal(e)
	}
	d.Close()
	dst := filepath.Join(dir, "new.pass")
	if n, e := Import(src, dst, testKey, testKey); e != nil || n != 4 {
		t.Fatal("signed IDs lost", n, e)
	}
	m := New(nil, nil)
	defer m.Close()
	if e = m.Unlock(dst, testKey); e != nil {
		t.Fatal(e)
	}
	if r, e := m.Get(m.Generation(), "-1"); e != nil || r.ID != "-1" {
		t.Fatal("negative ID unavailable", e)
	}
	if r, e := m.Get(m.Generation(), "0"); e != nil || r.ID != "0" {
		t.Fatal("zero ID unavailable", e)
	}
	m.Lock()
	d, e = cipher.Open(src, testKey, 4, false)
	if e != nil {
		t.Fatal(e)
	}
	d.Exec("UPDATE login_items SET name=? WHERE id=0", cipher.S(string([]byte{0xff})))
	d.Close()
	if _, e = Import(src, filepath.Join(dir, "unsupported.pass"), testKey, testKey); e == nil {
		t.Fatal("unrepresentable UTF-8 text accepted")
	}
}
