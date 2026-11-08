package vault

import (
	"bytes"
	"errors"
	"github.com/ssemakov/pass-rework/internal/cipher"
	"os"
	"path/filepath"
	"testing"
)

var testKey = []byte("public synthetic passphrase O'Connor 🔐")

func ptr(s string) *string { return &s }
func TestEncryptedCRUD(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.pass")
	m := New(nil, nil)
	defer m.Close()
	if e := m.Create(path, testKey); e != nil {
		t.Fatal(e)
	}
	g := m.State().Generation
	input := Input{ptr("Email <script>"), ptr("user\x00name"), ptr("secret-sentinel-🍋\x00end"), nil}
	r, e := m.CreateEntry(g, input)
	if e != nil {
		t.Fatal(e)
	}
	found, e := m.List(g, "")
	if e != nil || len(found) != 1 || found[0].Password != nil {
		t.Fatal("list disclosed password or omitted entry", e)
	}
	actual, e := m.Get(g, r.ID)
	if e != nil || *actual.Password != *input.Password || *actual.Login != *input.Login {
		t.Fatal("byte roundtrip", e)
	}
	patch := Patch{Name: ptr("Edited"), Login: input.Login, URL: nil, Revision: r.Revision}
	edited, e := m.Update(g, r.ID, patch)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = m.Update(g, r.ID, patch); !errors.Is(e, ErrConflict) {
		t.Fatal("stale edit accepted", e)
	}
	actual, e = m.Get(g, r.ID)
	if e != nil || *actual.Password != *input.Password {
		t.Fatal("untouched password overwritten")
	}
	m.Lock()
	if _, e = m.Get(g, r.ID); !errors.Is(e, ErrLocked) {
		t.Fatal("stale session")
	}
	raw, _ := os.ReadFile(path)
	if bytes.HasPrefix(raw, []byte("SQLite format 3")) || bytes.Contains(raw, []byte("secret-sentinel")) {
		t.Fatal("plaintext persisted")
	}
	if e = m.Unlock(path, []byte("wrong")); e == nil {
		t.Fatal("wrong password unlocked")
	}
	if e = m.Unlock(path, testKey); e != nil {
		t.Fatal(e)
	}
	g = m.State().Generation
	if e = m.Delete(g, r.ID, edited.Revision); e != nil {
		t.Fatal(e)
	}
	found, e = m.List(g, "")
	if e != nil || len(found) != 0 {
		t.Fatal("delete", e)
	}
}
func TestCipherStatusAndTamper(t *testing.T) {
	p := filepath.Join(t.TempDir(), "vault.pass")
	if e := newVault(p, testKey); e != nil {
		t.Fatal(e)
	}
	d, e := cipher.Open(p, testKey, 4, true)
	if e != nil {
		t.Fatal(e)
	}
	if e = d.Integrity(); e != nil {
		t.Fatal(e)
	}
	d.Close()
	b, _ := os.ReadFile(p)
	b[len(b)-70] ^= 0x20
	os.WriteFile(p, b, 0600)
	d, e = cipher.Open(p, testKey, 4, true)
	if e == nil {
		defer d.Close()
		if d.Integrity() == nil {
			t.Fatal("tamper accepted")
		}
	}
}
