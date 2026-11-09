package service

import (
	"context"
	"errors"
	"github.com/ssemakov/pass-rework/internal/cipher"
	"github.com/ssemakov/pass-rework/internal/settings"
	"github.com/ssemakov/pass-rework/internal/vault"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func trusted(context.Context) bool { return true }

func TestRememberedStorageWrongKeyLockAndRealRestart(t *testing.T) {
	ctx := context.Background()
	prefs := settings.New(filepath.Join(t.TempDir(), "settings.json"))
	picks := 0
	picker := func(string) (string, error) { picks++; return prefs.DefaultStorage(), nil }
	m := vault.New(nil, nil)
	f := NewWithSettings(m, picker, trusted, prefs)
	s, err := f.State(ctx)
	if err != nil || s.Storage != nil || s.NewStorage == nil {
		t.Fatal(s, err)
	}
	if err = f.Create(ctx, s.NewStorage.Token, "public-master"); err != nil {
		t.Fatal(err)
	}
	s, _ = f.State(ctx)
	if err = f.SetIdle(ctx, s.Generation, 15); err != nil {
		t.Fatal(err)
	}
	if err = f.Lock(ctx); err != nil {
		t.Fatal(err)
	}
	s, _ = f.State(ctx)
	if s.Storage == nil {
		t.Fatal("lock discarded remembered storage")
	}
	token := s.Storage.Token
	for i := 0; i < 80; i++ {
		next, e := f.State(ctx)
		if e != nil || next.Storage.Token != token {
			t.Fatal("polling invalidated grant")
		}
	}
	before, _ := os.ReadFile(filepath.Join(filepath.Dir(prefs.DefaultStorage()), "settings.json"))
	for i := 0; i < 3; i++ {
		if err = f.Unlock(ctx, token, "wrong-password"); !errors.Is(err, cipher.ErrKey) {
			t.Fatal(err)
		}
		x, _ := f.State(ctx)
		if !x.Locked || x.Storage.Token != token {
			t.Fatal("failed login changed state")
		}
	}
	after, _ := os.ReadFile(filepath.Join(filepath.Dir(prefs.DefaultStorage()), "settings.json"))
	if string(before) != string(after) || picks != 0 {
		t.Fatal("rejected key changed settings or opened picker")
	}
	if err = f.Unlock(ctx, token, "public-master"); err != nil {
		t.Fatal("correct retry", err)
	}
	m.Close()
	m = vault.New(nil, nil)
	defer m.Close()
	f = NewWithSettings(m, picker, trusted, settings.New(filepath.Join(filepath.Dir(prefs.DefaultStorage()), "settings.json")))
	s, err = f.State(ctx)
	if err != nil || !s.Locked || s.Storage == nil {
		t.Fatal("restart", s, err)
	}
	if err = f.Unlock(ctx, s.Storage.Token, "public-master"); err != nil {
		t.Fatal(err)
	}
	s, _ = f.State(ctx)
	if s.Locked || s.IdleMinutes != 15 || picks != 0 {
		t.Fatal("restart did not restore preferences", s, picks)
	}
}

func TestVerifiedImportRemembersDestinationButFailureDoesNot(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	prefs := settings.New(filepath.Join(dir, "settings.json"))
	dst := filepath.Join(dir, "imported.pass")
	legacy, err := filepath.Abs("../../testdata/legacy/journeyapps-4.0.0.pass")
	if err != nil {
		t.Fatal(err)
	}
	m := vault.New(nil, nil)
	f := NewWithSettings(m, func(p string) (string, error) {
		if p == "source" {
			return legacy, nil
		}
		return dst, nil
	}, trusted, prefs)
	source, _ := f.Choose(ctx, "source")
	dest, _ := f.Choose(ctx, "destination")
	if _, err = f.Import(ctx, source.Token, dest.Token, "wrong", "new-public-master"); err == nil {
		t.Fatal("wrong import password accepted")
	}
	v, _ := prefs.Load()
	if v.StoragePath != "" {
		t.Fatal("failed import persisted destination")
	}
	if _, err = f.Import(ctx, source.Token, dest.Token, "public-fixture-password", "new-public-master"); err != nil {
		t.Fatal(err)
	}
	v, _ = prefs.Load()
	if v.StoragePath != dst {
		t.Fatal(v)
	}
	m.Close()
	m = vault.New(nil, nil)
	defer m.Close()
	f = NewWithSettings(m, func(string) (string, error) { t.Fatal("unexpected picker"); return "", nil }, trusted, prefs)
	s, _ := f.State(ctx)
	if s.Storage == nil || f.Unlock(ctx, s.Storage.Token, "new-public-master") != nil {
		t.Fatal("cannot open imported destination after restart")
	}
}

func TestCancelledAndRejectedSelectionPreserveRememberedChoice(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	prefs := settings.New(filepath.Join(dir, "settings.json"))
	m := vault.New(nil, nil)
	defer m.Close()
	path := prefs.DefaultStorage()
	f := NewWithSettings(m, func(string) (string, error) { return path, nil }, trusted, prefs)
	s, _ := f.State(ctx)
	if err := f.Create(ctx, s.NewStorage.Token, "original"); err != nil {
		t.Fatal(err)
	}
	f.Lock(ctx)
	before, _ := prefs.Load()
	path = ""
	if _, err := f.Choose(ctx, "open"); err != ErrCancelled {
		t.Fatal(err)
	}
	path = filepath.Join(dir, "unavailable.pass")
	choice, _ := f.Choose(ctx, "open")
	if err := f.Unlock(ctx, choice.Token, "wrong"); err == nil {
		t.Fatal("missing file opened")
	}
	after, _ := prefs.Load()
	if before != after {
		t.Fatal("picker/failure changed remembered path")
	}
	s, _ = f.State(ctx)
	if s.Storage == nil || f.Unlock(ctx, s.Storage.Token, "original") != nil {
		t.Fatal("original selection lost")
	}
}

func TestChoosingAnotherVaultPersistsOnlyAfterItsPasswordIsAccepted(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	prefs := settings.New(filepath.Join(dir, "settings.json"))
	other := filepath.Join(dir, "other.pass")
	separate := vault.New(nil, nil)
	if err := separate.Create(other, []byte("other-public-master")); err != nil {
		t.Fatal(err)
	}
	separate.Close()
	m := vault.New(nil, nil)
	defer m.Close()
	f := NewWithSettings(m, func(string) (string, error) { return other, nil }, trusted, prefs)
	s, _ := f.State(ctx)
	if err := f.Create(ctx, s.NewStorage.Token, "original"); err != nil {
		t.Fatal(err)
	}
	f.Lock(ctx)
	before, _ := prefs.Load()
	choice, _ := f.Choose(ctx, "open")
	after, _ := prefs.Load()
	if before != after {
		t.Fatal("Choose changed persisted storage")
	}
	if err := f.Unlock(ctx, choice.Token, "wrong"); !errors.Is(err, cipher.ErrKey) {
		t.Fatal(err)
	}
	after, _ = prefs.Load()
	if before != after {
		t.Fatal("wrong key changed persisted storage")
	}
	if err := f.Unlock(ctx, choice.Token, "other-public-master"); err != nil {
		t.Fatal(err)
	}
	after, _ = prefs.Load()
	if after.StoragePath != other {
		t.Fatal("accepted storage was not saved", after)
	}
}

func TestRepeatedSourceChoicesPreserveDestinationAndDefaultGrants(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	prefs := settings.New(filepath.Join(dir, "settings.json"))
	m := vault.New(nil, nil)
	defer m.Close()
	f := NewWithSettings(m, func(p string) (string, error) { return filepath.Join(dir, p+".pass"), nil }, trusted, prefs)
	s, _ := f.State(ctx)
	createToken := s.NewStorage.Token
	dst, err := f.Choose(ctx, "destination")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 80; i++ {
		if _, err = f.Choose(ctx, "source"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = f.take(dst.Token, "destination"); err != nil {
		t.Fatal("rechoosing source lost destination", err)
	}
	if err = f.Create(ctx, createToken, "public-master"); err != nil {
		t.Fatal("rechoosing source lost default destination", err)
	}
}

type failingPreferences struct{ *settings.Store }

func (f failingPreferences) Save(settings.Values) error { return settings.ErrSettings }

func TestSettingsSaveFailureDoesNotReportFailedVaultCreation(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	prefs := failingPreferences{settings.New(filepath.Join(dir, "settings.json"))}
	m := vault.New(nil, nil)
	defer m.Close()
	f := NewWithSettings(m, nil, trusted, prefs)
	s, _ := f.State(ctx)
	if err := f.Create(ctx, s.NewStorage.Token, "public-master"); err != nil {
		t.Fatal("created vault presented as failed", err)
	}
	s, _ = f.State(ctx)
	if s.Locked || !strings.Contains(s.SettingsError, "settings could not be saved") {
		t.Fatal(s)
	}
	if _, err := os.Stat(prefs.DefaultStorage()); err != nil {
		t.Fatal("vault missing", err)
	}
}

func TestHeavyOperationsDoNotQueueAndLockCancelsPendingPicker(t *testing.T) {
	ctx := context.Background()
	m := vault.New(nil, nil)
	defer m.Close()
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	f := NewWithSettings(m, func(string) (string, error) {
		close(entered)
		<-release
		return filepath.Join(t.TempDir(), "storage.pass"), nil
	}, trusted, settings.New(filepath.Join(t.TempDir(), "settings.json")))
	go func() { _, err := f.Choose(ctx, "open"); done <- err }()
	<-entered
	if err := f.Unlock(ctx, "invalid", "wrong"); err != ErrWorking {
		t.Fatal("auth queued behind picker", err)
	}
	locked := make(chan error, 1)
	go func() { locked <- f.Lock(ctx) }()
	select {
	case err := <-locked:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("lock waited on picker")
	}
	close(release)
	if err := <-done; err != ErrDenied {
		t.Fatal("cancelled picker grant accepted", err)
	}
}
