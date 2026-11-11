package vault

import (
	"bytes"
	"errors"
	"github.com/ssemakov/pass-rework/internal/cipher"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestV1DetectionIsReadOnlyForBothProfilesAndPublicPermissions(t *testing.T) {
	for _, profile := range []int{3, 4} {
		t.Run(strconv.Itoa(profile), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "legacy ?#% 🍋.pass")
			legacyFixture(t, path, profile)
			os.Chmod(path, 0644)
			before, _ := os.ReadFile(path)
			st, _ := os.Stat(path)
			m := New(nil, nil)
			defer m.Close()
			if err := m.Unlock(path, []byte("wrong password")); !errors.Is(err, cipher.ErrKey) {
				t.Fatal(err)
			}
			if err := m.Unlock(path, testKey); !errors.Is(err, ErrLegacy) {
				t.Fatal("legacy not detected", err)
			}
			after, _ := os.ReadFile(path)
			actual, _ := os.Stat(path)
			if !bytes.Equal(before, after) || actual.Mode() != st.Mode() || !actual.ModTime().Equal(st.ModTime()) {
				t.Fatal("V1 detection touched original")
			}
			if !m.State().Locked {
				t.Fatal("legacy opened as current")
			}
			dst := filepath.Join(filepath.Dir(path), "migrated.pass")
			if count, err := m.Import(path, dst, testKey, testKey); err != nil || count != 2 {
				t.Fatal(count, err)
			}
			if err := m.Unlock(dst, testKey); err != nil {
				t.Fatal(err)
			}
			mode, _ := os.Stat(dst)
			if mode.Mode().Perm() != 0600 {
				t.Fatal("destination not private")
			}
			after, _ = os.ReadFile(path)
			if !bytes.Equal(before, after) {
				t.Fatal("migration changed source")
			}
		})
	}
}

func TestKilledLegacyEditIsNotRecoveredDuringV1Detection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dirty-legacy.pass")
	legacyFixture(t, path, 4)
	child := exec.Command(os.Args[0], "-test.run=^TestCRUDCrashChild$")
	child.Env = append(os.Environ(), "PASS_TEST_CRUD_CHILD=1", "PASS_TEST_LIVE="+path)
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	defer child.Process.Kill()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path + ".test-ready"); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := os.Stat(path + ".test-ready"); err != nil {
		t.Fatal("legacy child did not prepare transaction", err)
	}
	if err := child.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	child.Wait()
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	journal, err := os.ReadFile(path + "-journal")
	if err != nil || len(journal) == 0 {
		t.Fatal("no legacy journal", err)
	}
	m := New(nil, nil)
	defer m.Close()
	if err = m.Unlock(path, testKey); !errors.Is(err, ErrRecovery) {
		t.Fatal("legacy journal was recovered or opened", err)
	}
	after, _ := os.ReadFile(path)
	actual, _ := os.ReadFile(path + "-journal")
	if !bytes.Equal(before, after) || !bytes.Equal(journal, actual) {
		t.Fatal("legacy main file/journal changed")
	}
}

func TestHistoricalDriverDetectionAndMalformedFileClassification(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "historical.pass")
	b, err := os.ReadFile("../../testdata/legacy/journeyapps-4.0.0.pass")
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(path, b, 0644)
	m := New(nil, nil)
	defer m.Close()
	if err = m.Unlock(path, []byte("public-fixture-password")); !errors.Is(err, ErrLegacy) {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(b, after) {
		t.Fatal("historical source changed")
	}
	bad := filepath.Join(dir, "bad.pass")
	legacyFixture(t, bad, 4)
	d, err := cipher.Open(bad, testKey, 4, false)
	if err != nil {
		t.Fatal(err)
	}
	d.Exec("CREATE TRIGGER unexpected AFTER INSERT ON login_items BEGIN SELECT 1; END")
	d.Close()
	before, _ := os.ReadFile(bad)
	if err = m.Unlock(bad, testKey); !errors.Is(err, ErrFormat) {
		t.Fatal("unknown schema treated as V1", err)
	}
	after, _ = os.ReadFile(bad)
	if !bytes.Equal(before, after) {
		t.Fatal("malformed source changed")
	}
}

func TestLegacyWithJournalAndDamagedMetadataNeverOpensWritable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.pass")
	legacyFixture(t, path, 4)
	journal := []byte("synthetic dirty journal; must remain untouched")
	os.WriteFile(path+"-journal", journal, 0600)
	before, _ := os.ReadFile(path)
	m := New(nil, nil)
	defer m.Close()
	if err := m.Unlock(path, testKey); !errors.Is(err, ErrRecovery) {
		t.Fatal("dirty legacy source was not refused", err)
	}
	after, _ := os.ReadFile(path)
	j, _ := os.ReadFile(path + "-journal")
	if !bytes.Equal(before, after) || !bytes.Equal(journal, j) {
		t.Fatal("dirty source/journal modified")
	}
	before[200] ^= 0xff
	os.WriteFile(path, before, 0600)
	if err := m.Unlock(path, testKey); !errors.Is(err, ErrRecovery) {
		t.Fatal("damaged metadata fallback", err)
	}
	after, _ = os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("damaged source modified")
	}
}

func TestCurrentRawKeyStillCannotBypassPasswordKDF(t *testing.T) {
	path := filepath.Join(t.TempDir(), "raw-current.pass")
	os.WriteFile(path, nil, 0600)
	key := []byte("x'" + strings.Repeat("a", 64) + "'")
	d, err := cipher.Open(path, key, 4, false)
	if err != nil {
		t.Fatal(err)
	}
	if err = initialise(d); err != nil {
		t.Fatal(err)
	}
	d.Close()
	m := New(nil, nil)
	defer m.Close()
	if err = m.Unlock(path, key); !errors.Is(err, ErrMasterFormat) {
		t.Fatal("raw current password accepted", err)
	}
	if !m.State().Locked {
		t.Fatal("raw current file opened")
	}
}
