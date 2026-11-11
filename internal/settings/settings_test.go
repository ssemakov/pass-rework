package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestSettingsRestartAndPrivateNonSecretFormat(t *testing.T) {
	name := filepath.Join(t.TempDir(), "preferences", "settings.json")
	s := New(name)
	v, err := s.Load()
	if err != nil || v.StoragePath != "" || v.IdleMinutes != 5 {
		t.Fatal(v, err)
	}
	want := Values{filepath.Join(t.TempDir(), "Unicode 🍋 storage.pass"), 15}
	if err = s.Save(want); err != nil {
		t.Fatal(err)
	}
	got, err := New(name).Load()
	if err != nil || got != want {
		t.Fatal(got, err)
	}
	st, _ := os.Stat(name)
	if st.Mode().Perm() != 0600 {
		t.Fatal("settings permissions", st.Mode())
	}
	st, _ = os.Stat(filepath.Dir(name))
	if st.Mode().Perm() != 0700 {
		t.Fatal("settings directory permissions", st.Mode())
	}
	b, _ := os.ReadFile(name)
	var fields map[string]any
	if json.Unmarshal(b, &fields) != nil || len(fields) != 2 || fields["default_storage_path"] != want.StoragePath || fields["idle_minutes"] != float64(15) {
		t.Fatal("unexpected settings fields")
	}
}

func TestSettingsRejectMalformedOversizedAndSymlink(t *testing.T) {
	dir := t.TempDir()
	name := filepath.Join(dir, "settings.json")
	s := New(name)
	for _, raw := range []string{`{`, `{} {}`, `{"default_storage_path":"relative.pass"}`, `{"idle_minutes":0}`, `{"password":"must never be accepted"}`, string(make([]byte, 32769))} {
		if os.WriteFile(name, []byte(raw), 0600) != nil {
			t.Fatal("fixture")
		}
		if _, err := s.Load(); err == nil {
			t.Fatal("accepted invalid settings")
		}
	}
	os.Remove(name)
	target := filepath.Join(t.TempDir(), "untouched.json")
	if os.WriteFile(target, []byte("original"), 0600) != nil || os.Symlink(target, name) != nil {
		t.Fatal("fixture")
	}
	if _, err := s.Load(); err == nil {
		t.Fatal("read settings symlink")
	}
	if err := s.Save(Values{IdleMinutes: 5}); err == nil {
		t.Fatal("replaced settings symlink")
	}
	b, _ := os.ReadFile(target)
	if string(b) != "original" {
		t.Fatal("changed symlink target")
	}
}

func TestAtomicSettingsAndFailedSavePreservePreviousFile(t *testing.T) {
	dir := t.TempDir()
	name := filepath.Join(dir, "settings.json")
	s := New(name)
	v := Values{filepath.Join(dir, "storage.pass"), 5}
	if err := s.Save(v); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			b, err := os.ReadFile(name)
			var x Values
			if err != nil || json.Unmarshal(b, &x) != nil || x.StoragePath != v.StoragePath || !valid(x) {
				t.Error("reader observed partial settings")
				return
			}
		}
	}()
	for _, minutes := range []int{1, 15, 5, 1, 5} {
		v.IdleMinutes = minutes
		if err := s.Save(v); err != nil {
			t.Fatal(err)
		}
	}
	wg.Wait()
	before, _ := os.ReadFile(name)
	if os.Chmod(dir, 0500) != nil {
		t.Fatal("chmod")
	}
	defer os.Chmod(dir, 0700)
	if s.Save(Values{filepath.Join(dir, "different.pass"), 15}) == nil {
		t.Fatal("saved into read-only directory")
	}
	after, _ := os.ReadFile(name)
	if string(before) != string(after) {
		t.Fatal("failed save changed settings")
	}
}
