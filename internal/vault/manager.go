package vault

import (
	"crypto/subtle"
	"errors"
	"github.com/ssemakov/pass-rework/internal/cipher"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

var ErrLocked = errors.New("vault locked; unlock again")

type Clipboard interface {
	Write(string) (int64, bool)
	ClearIfOwned(int64)
}
type Status struct {
	Generation  string `json:"generation"`
	Locked      bool   `json:"locked"`
	Label       string `json:"label"`
	IdleMinutes int    `json:"idleMinutes"`
}
type Manager struct {
	mu          sync.Mutex
	admit       sync.Mutex
	native      sync.RWMutex
	epoch       atomic.Uint64
	closed      atomic.Bool
	activeEpoch uint64
	opened      atomic.Bool
	d           *cipher.DB
	loc         *location
	identity    os.FileInfo
	owner       *os.File
	key         []byte
	clipboard   Clipboard
	copied      int64
	copyTimer   *time.Timer
	since       time.Time
	last        atomic.Int64
	idle        atomic.Int64
	stop        chan struct{}
	onLock      func(string)
}

func New(clipboard Clipboard, onLock func(string)) *Manager {
	m := &Manager{clipboard: clipboard, since: time.Now(), stop: make(chan struct{}), onLock: onLock}
	m.idle.Store(int64(5 * time.Minute))
	m.epoch.Store(1)
	go m.watch()
	return m
}
func (m *Manager) watch() {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-m.stop:
			return
		case <-t.C:
			if time.Since(m.since).Nanoseconds()-m.last.Load() > m.idle.Load() {
				if m.opened.Load() {
					m.Lock()
				}
			}
		}
	}
}
func (m *Manager) Close() {
	if !m.closed.CompareAndSwap(false, true) {
		return
	}
	close(m.stop)
	m.Lock()
}
func (m *Manager) Generation() string { return m.generation() }
func (m *Manager) generation() string { return strconv.FormatUint(m.epoch.Load(), 10) }
func (m *Manager) State() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := Status{Generation: m.generation(), Locked: m.d == nil || m.activeEpoch != m.epoch.Load(), IdleMinutes: int(time.Duration(m.idle.Load()) / time.Minute)}
	if m.loc != nil {
		s.Label = m.loc.name
	}
	return s
}
func (m *Manager) clearClipboard() {
	if m.copyTimer != nil {
		m.copyTimer.Stop()
		m.copyTimer = nil
	}
	if m.clipboard != nil && m.copied != 0 {
		m.clipboard.ClearIfOwned(m.copied)
	}
	m.copied = 0
}
func (m *Manager) Lock() {
	m.admit.Lock()
	m.epoch.Add(1)
	m.opened.Store(false)
	if m.onLock != nil {
		m.onLock(m.generation())
	}
	m.admit.Unlock()
	m.native.RLock()
	if m.d != nil {
		m.d.Interrupt()
	}
	m.native.RUnlock()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closeLocked()
	m.clearClipboard()
}
func (m *Manager) closeLocked() {
	m.opened.Store(false)
	m.native.Lock()
	if m.d != nil {
		_ = m.d.Close()
		m.d = nil
	}
	m.native.Unlock()
	wipe(m.key)
	m.key = nil
	unlock(m.owner)
	m.owner = nil
	if m.loc != nil {
		m.loc.close()
		m.loc = nil
	}
	m.identity = nil
}
func (m *Manager) openLocked(path string, key []byte) error {
	if e := validateMasterPassword(key); e != nil {
		return e
	}
	openingEpoch := m.epoch.Load()
	loc, e := locate(path)
	if e != nil {
		return e
	}
	ok := false
	defer func() {
		if !ok {
			loc.close()
		}
	}()
	before, e := loc.check()
	if e != nil {
		return e
	}
	if before.Mode().Perm()&0077 != 0 {
		return errors.New("vault permissions must be private (chmod 600)")
	}
	owner, e := loc.lock()
	if e != nil {
		return e
	}
	defer func() {
		if !ok {
			unlock(owner)
		}
	}()
	d, e := cipher.Open(loc.path, key, 4, false)
	if e != nil {
		return e
	}
	defer func() {
		if !ok {
			d.Close()
		}
	}()
	// SQLite's exclusive connection lock also excludes other SQLite clients
	// for the lifetime of a normal unlocked session; no custom recovery layer.
	if _, e = d.Exec("PRAGMA locking_mode=EXCLUSIVE"); e != nil {
		return e
	}
	if _, e = d.Exec("BEGIN EXCLUSIVE"); e != nil {
		return e
	}
	if _, e = d.Query("SELECT count(*) FROM sqlite_schema"); e != nil {
		d.Exec("ROLLBACK")
		return e
	}
	if _, e = d.Exec("COMMIT"); e != nil {
		return e
	}
	if e = schema(d, true); e != nil {
		return e
	}
	if e = d.Integrity(); e != nil {
		return e
	}
	after, e := loc.check()
	if e != nil || !os.SameFile(before, after) {
		return ErrFile
	}
	m.native.Lock()
	m.d = d
	m.native.Unlock()
	m.activeEpoch = openingEpoch
	m.loc = loc
	m.identity = after
	m.owner = owner
	m.key = append([]byte(nil), key...)
	m.last.Store(time.Since(m.since).Nanoseconds())
	m.opened.Store(true)
	ok = true
	return nil
}
func (m *Manager) Unlock(path string, key []byte) error { return m.UnlockAt(path, key, m.generation()) }
func (m *Manager) UnlockAt(path string, key []byte, expected string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed.Load() || expected != m.generation() {
		return ErrLocked
	}
	if m.d != nil {
		return errors.New("lock the current vault first")
	}
	if e := m.openLocked(path, key); e != nil {
		return e
	}
	return m.finishOpen(expected)
}
func (m *Manager) Create(path string, key []byte) error { return m.CreateAt(path, key, m.generation()) }
func (m *Manager) CreateAt(path string, key []byte, expected string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed.Load() || expected != m.generation() {
		return ErrLocked
	}
	if m.d != nil || len(key) == 0 {
		return ErrInput
	}
	if e := newVault(path, key); e != nil {
		return e
	}
	if expected != m.generation() || m.closed.Load() {
		return ErrLocked
	}
	if e := m.openLocked(path, key); e != nil {
		return e
	}
	return m.finishOpen(expected)
}
func (m *Manager) check(token string) error {
	if m.closed.Load() || m.d == nil || m.activeEpoch != m.epoch.Load() || token != m.generation() {
		return ErrLocked
	}
	st, e := m.loc.check()
	if e != nil || !os.SameFile(st, m.identity) {
		return ErrFile
	}
	m.last.Store(time.Since(m.since).Nanoseconds())
	return nil
}
func (m *Manager) List(token, q string) ([]Entry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e := m.check(token); e != nil {
		return nil, e
	}
	r, e := list(m.d, q)
	if token != m.generation() {
		return nil, ErrLocked
	}
	return r, e
}
func (m *Manager) Get(token, id string) (Entry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e := m.check(token); e != nil {
		return Entry{}, e
	}
	r, e := get(m.d, id)
	if token != m.generation() {
		return Entry{}, ErrLocked
	}
	return r, e
}
func (m *Manager) CreateEntry(token string, i Input) (Entry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e := m.check(token); e != nil {
		return Entry{}, e
	}
	r, e := create(m.d, i)
	r.Password = nil
	if token != m.generation() {
		return Entry{}, ErrLocked
	}
	return r, e
}
func (m *Manager) Update(token, id string, p Patch) (Entry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e := m.check(token); e != nil {
		return Entry{}, e
	}
	r, e := update(m.d, id, p)
	r.Password = nil
	if token != m.generation() {
		return Entry{}, ErrLocked
	}
	return r, e
}
func (m *Manager) Delete(token, id, revision string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e := m.check(token); e != nil {
		return e
	}
	return remove(m.d, id, revision)
}
func (m *Manager) Activity(token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.check(token)
}
func (m *Manager) SetIdle(token string, minutes int) error {
	if minutes != 1 && minutes != 5 && minutes != 15 {
		return ErrInput
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if e := m.check(token); e != nil {
		return e
	}
	m.idle.Store(int64(time.Duration(minutes) * time.Minute))
	return nil
}
func (m *Manager) Copy(token, id, field string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e := m.check(token); e != nil {
		return e
	}
	r, e := get(m.d, id)
	if e != nil {
		return e
	}
	var v *string
	switch field {
	case "password":
		v = r.Password
	case "login":
		v = r.Login
	case "url":
		v = r.URL
	default:
		return ErrInput
	}
	if v == nil {
		return ErrInput
	}
	m.admit.Lock()
	defer m.admit.Unlock()
	if token != m.generation() {
		return ErrLocked
	}
	if m.clipboard == nil {
		return errors.New("clipboard unavailable")
	}
	m.clearClipboard()
	ownership, ok := m.clipboard.Write(*v)
	if !ok {
		return errors.New("clipboard unavailable")
	}
	m.copied = ownership
	m.copyTimer = time.AfterFunc(30*time.Second, func() { m.mu.Lock(); defer m.mu.Unlock(); m.clearClipboard() })
	return nil
}
func (m *Manager) Backup(token, destination string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e := m.check(token); e != nil {
		return e
	}
	return m.snapshotLocked(token, destination, m.key)
}
func (m *Manager) snapshotLocked(token, destination string, key []byte) error {
	dst, e := locate(destination)
	if e != nil {
		return e
	}
	defer dst.close()
	owner, e := dst.lock()
	if e != nil {
		return e
	}
	defer unlock(owner)
	if _, e = m.d.Exec("BEGIN"); e != nil {
		return e
	}
	defer m.d.Exec("ROLLBACK")
	if e = m.d.Integrity(); e != nil {
		return e
	}
	if e = schema(m.d, true); e != nil {
		return e
	}
	_, e = writeSnapshot(m.d, true, dst, key, func() bool { return token == m.generation() && !m.closed.Load() }, func() error { return m.check(token) })
	return e
}
func (m *Manager) Import(backup, destination string, oldKey, newKey []byte) (int, error) {
	return m.ImportAt(backup, destination, oldKey, newKey, m.generation())
}
func (m *Manager) ImportAt(backup, destination string, oldKey, newKey []byte, expected string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.d != nil {
		return 0, errors.New("lock the current vault before migration or restore")
	}
	if m.closed.Load() || expected != m.generation() {
		return 0, ErrLocked
	}
	return importChecked(backup, destination, oldKey, newKey, func() bool { return expected == m.generation() && !m.closed.Load() })
}

// Rotate verifies a fresh encrypted candidate before closing every old handle.
// The rollback copy remains encrypted with the old password and is never deleted.
func (m *Manager) Rotate(token string, oldKey, newKey []byte) (rollback string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e := m.check(token); e != nil {
		return "", e
	}
	if e := validateMasterPassword(newKey); e != nil {
		return "", e
	}
	if subtle.ConstantTimeCompare(oldKey, m.key) != 1 {
		return "", cipher.ErrKey
	}
	var e error
	candidate := filepath.Join(filepath.Dir(m.loc.path), "Pass-before-password-change-"+time.Now().UTC().Format("20060102-150405")+"-"+randomID()+".pass")
	if e = m.snapshotLocked(token, candidate, oldKey); e != nil {
		return "", e
	}
	rollback = candidate
	target := sameParentCandidate(m.loc.path)
	if e = m.snapshotLocked(token, target, newKey); e != nil {
		return rollback, e
	}
	defer os.Remove(target)
	newer, e := cipher.Open(target, newKey, 4, true)
	if e != nil {
		return rollback, e
	}
	if e = compare(m.d, newer, true); e != nil {
		newer.Close()
		return rollback, e
	}
	newer.Close()
	if wrong, e := cipher.Open(target, oldKey, 4, true); e == nil {
		wrong.Close()
		return rollback, errors.New("new password must differ from the old password")
	}
	if token != m.generation() {
		return rollback, ErrLocked
	}
	path := m.loc.path
	m.native.Lock()
	e = m.d.Close()
	m.d = nil
	m.native.Unlock()
	if e != nil {
		return rollback, e
	}
	if st, e := m.loc.check(); e != nil || !os.SameFile(st, m.identity) {
		m.closeLocked()
		return rollback, ErrFile
	}
	if e = os.Rename(target, path); e != nil {
		m.closeLocked()
		return rollback, ErrFile
	}
	if e = m.loc.sync(); e != nil {
		m.closeLocked()
		return rollback, e
	}
	m.closeLocked()
	m.clearClipboard()
	if e = m.openLocked(path, newKey); e != nil {
		return rollback, e
	}
	// Reuse authentication admission: a Lock/Close admitted during replacement
	// must not be superseded by rotation reopening under a newer generation.
	return rollback, m.finishOpen(token)
}

func (m *Manager) PreserveBackup(source, destination string, key []byte, expected string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed.Load() || expected != m.generation() || m.d != nil {
		return ErrLocked
	}
	return PreserveBackup(source, destination, key)
}

func (m *Manager) finishOpen(expected string) error {
	m.admit.Lock()
	defer m.admit.Unlock()
	if expected != m.generation() || m.closed.Load() {
		m.closeLocked()
		return ErrLocked
	}
	m.activeEpoch = m.epoch.Add(1)
	return nil
}
