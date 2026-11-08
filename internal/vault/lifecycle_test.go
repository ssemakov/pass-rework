package vault

import (
	"github.com/ssemakov/pass-rework/internal/cipher"
	"path/filepath"
	"testing"
	"time"
)

type reviewBlockingClipboard struct{ entered, release chan struct{} }

func (c *reviewBlockingClipboard) Write(string) (int64, bool) { return 1, true }
func (c *reviewBlockingClipboard) ClearIfOwned(int64)         { close(c.entered); <-c.release }
func TestAdversarialQueuedUnlockAfterLock(t *testing.T) {
	p := filepath.Join(t.TempDir(), "synthetic.pass")
	key := []byte("public synthetic adversarial key")
	if e := newVault(p, key); e != nil {
		t.Fatal(e)
	}
	c := &reviewBlockingClipboard{make(chan struct{}), make(chan struct{})}
	m := New(c, nil)
	defer m.Close()
	if e := m.Unlock(p, key); e != nil {
		t.Fatal(e)
	}
	expected := m.Generation()
	m.mu.Lock()
	m.copied = 1
	m.mu.Unlock()
	locked := make(chan struct{})
	go func() { m.Lock(); close(locked) }()
	select {
	case <-c.entered:
	case <-time.After(3 * time.Second):
		t.Fatal("lock did not reach clipboard cleanup")
	}
	opened := make(chan error, 1)
	go func() { opened <- m.UnlockAt(p, key, expected) }()
	time.Sleep(20 * time.Millisecond)
	close(c.release)
	<-locked
	if e := <-opened; e == nil && !m.State().Locked {
		t.Fatal("queued unlock reopened the backend after admitted lock completed")
	}
}

func TestAdversarialCloseIsTerminal(t *testing.T) {
	p := filepath.Join(t.TempDir(), "synthetic.pass")
	key := []byte("public synthetic adversarial key")
	if e := newVault(p, key); e != nil {
		t.Fatal(e)
	}
	m := New(nil, nil)
	m.Close()
	if e := m.Unlock(p, key); e == nil {
		t.Fatal("closed manager reopened")
	}
}

func TestIdleLockInvalidatesBeforeWaitingForWorker(t *testing.T) {
	p := filepath.Join(t.TempDir(), "idle.pass")
	m := New(nil, nil)
	defer m.Close()
	if e := m.Create(p, testKey); e != nil {
		t.Fatal(e)
	}
	generation := m.Generation()
	m.idle.Store(int64(20 * time.Millisecond))
	m.mu.Lock()
	released := false
	defer func() {
		if !released {
			m.mu.Unlock()
		}
	}()
	deadline := time.Now().Add(2200 * time.Millisecond)
	for time.Now().Before(deadline) && m.Generation() == generation {
		time.Sleep(10 * time.Millisecond)
	}
	if m.Generation() == generation {
		t.Fatal("idle lock waited on the busy worker before invalidation")
	}
	m.mu.Unlock()
	released = true
}

func TestNormalSessionExcludesExternalSQLiteConnections(t *testing.T) {
	p := filepath.Join(t.TempDir(), "exclusive.pass")
	m := New(nil, nil)
	defer m.Close()
	if e := m.Create(p, testKey); e != nil {
		t.Fatal(e)
	}
	if other, e := cipher.Open(p, testKey, 4, true); e == nil {
		other.Close()
		t.Fatal("another SQLite connection read an unlocked vault")
	}
	m.Lock()
	other, e := cipher.Open(p, testKey, 4, true)
	if e != nil {
		t.Fatal("lock not released", e)
	}
	other.Close()
}
