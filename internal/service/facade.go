// Package service defines the complete callable surface; storage objects and
// arbitrary paths are never registered with Wails.
package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"github.com/ssemakov/pass-rework/internal/settings"
	"github.com/ssemakov/pass-rework/internal/vault"
	"path/filepath"
	"strconv"
	"sync"
)

type Picker func(purpose string) (string, error)
type Selection struct {
	Token string `json:"token"`
	Label string `json:"label"`
}
type selected struct{ path, purpose, generation string }
type Preferences interface {
	Load() (settings.Values, error)
	Save(settings.Values) error
	DefaultStorage() string
}
type State struct {
	vault.Status
	Storage       *Selection `json:"storage"`
	NewStorage    *Selection `json:"newStorage"`
	SettingsError string     `json:"settingsError"`
}
type Facade struct {
	manager             *vault.Manager
	picker              Picker
	authorize           func(context.Context) bool
	mu                  sync.Mutex
	selected            map[string]selected
	work                sync.Mutex
	prefs               Preferences
	values              settings.Values
	settingsError       string
	selectionGeneration string
	storage, newStorage *Selection
}

func New(m *vault.Manager, p Picker, authorize func(context.Context) bool) *Facade {
	return NewWithSettings(m, p, authorize, nil)
}
func NewWithSettings(m *vault.Manager, p Picker, authorize func(context.Context) bool, prefs Preferences) *Facade {
	f := &Facade{manager: m, picker: p, authorize: authorize, selected: map[string]selected{}, prefs: prefs, values: settings.Values{IdleMinutes: 5}}
	if prefs != nil {
		v, err := prefs.Load()
		if err == nil {
			f.values = v
		} else {
			f.settingsError = "Could not read settings. Open or create storage using the File menu."
		}
	}
	return f
}

var ErrDenied = errors.New("request not authorized")
var ErrCancelled = errors.New("selection cancelled")
var ErrWorking = errors.New("another storage operation is in progress")

// Called under f.mu. Stable per-generation grants survive arbitrary State polls.
func (f *Facade) grant(path, purpose, generation string) (*Selection, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	token := hex.EncodeToString(b)
	f.selected[token] = selected{path, purpose, generation}
	return &Selection{token, filepath.Base(path)}, nil
}

func nextGeneration(g string) string {
	n, _ := strconv.ParseUint(g, 10, 64)
	return strconv.FormatUint(n+1, 10)
}

func (f *Facade) remember(path, generation string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if generation != f.manager.Generation() {
		return false
	}
	f.values.StoragePath = path
	f.selectionGeneration = ""
	f.saveSettings()
	return generation == f.manager.Generation()
}
func (f *Facade) saveSettings() {
	if f.prefs == nil {
		return
	}
	if f.prefs.Save(f.values) != nil {
		f.settingsError = "Storage is ready, but settings could not be saved. Select it again after restarting."
	} else {
		f.settingsError = ""
	}
}
func (f *Facade) authenticated(path, generation string) error {
	f.mu.Lock()
	idle := f.values.IdleMinutes
	f.mu.Unlock()
	if err := f.manager.SetIdle(generation, idle); err != nil {
		return err
	}
	if !f.remember(path, generation) {
		return vault.ErrLocked
	}
	return nil
}

func (f *Facade) allow(ctx context.Context) error {
	if f.authorize == nil || !f.authorize(ctx) {
		return ErrDenied
	}
	return nil
}
func (f *Facade) take(token, purpose string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.selected[token]
	if !ok || s.purpose != purpose || s.generation != f.manager.Generation() {
		return "", ErrDenied
	}
	// Grants remain reusable for retry within this exact generation. Fresh-output
	// filesystem operations still refuse to overwrite a completed destination.
	return s.path, nil
}
func (f *Facade) Choose(ctx context.Context, purpose string) (Selection, error) {
	if e := f.allow(ctx); e != nil {
		return Selection{}, e
	}
	switch purpose {
	case "open", "create", "source", "backup", "destination":
	default:
		return Selection{}, ErrDenied
	}
	if !f.work.TryLock() {
		return Selection{}, ErrWorking
	}
	defer f.work.Unlock()
	generation := f.manager.Generation()
	path, e := f.picker(purpose)
	if e != nil {
		return Selection{}, errors.New("file selection failed")
	}
	if generation != f.manager.Generation() {
		return Selection{}, ErrDenied
	}
	if path == "" {
		return Selection{}, ErrCancelled
	}
	b := make([]byte, 24)
	if _, e = rand.Read(b); e != nil {
		return Selection{}, ErrDenied
	}
	token := hex.EncodeToString(b)
	f.mu.Lock()
	if f.selectionGeneration != generation {
		f.selected = map[string]selected{}
		f.storage, f.newStorage = nil, nil
		f.selectionGeneration = generation
	}
	// One selected file per purpose, plus stable remembered/default grants.
	// Rechoosing a source must not invalidate its separately chosen destination.
	for token, grant := range f.selected {
		if grant.purpose == purpose && (f.storage == nil || token != f.storage.Token) && (f.newStorage == nil || token != f.newStorage.Token) {
			delete(f.selected, token)
		}
	}
	f.selected[token] = selected{path, purpose, generation}
	f.mu.Unlock()
	return Selection{token, filepath.Base(path)}, nil
}
func (f *Facade) State(ctx context.Context) (State, error) {
	if e := f.allow(ctx); e != nil {
		return State{}, e
	}
	s := State{Status: f.manager.State()}
	f.mu.Lock()
	defer f.mu.Unlock()
	if s.Generation != f.manager.Generation() {
		return State{}, vault.ErrLocked
	}
	s.SettingsError = f.settingsError
	if !s.Locked {
		return s, nil
	}
	if f.selectionGeneration != s.Generation {
		f.selected = map[string]selected{}
		f.storage, f.newStorage = nil, nil
		f.selectionGeneration = s.Generation
	}
	var err error
	if f.storage == nil && f.values.StoragePath != "" {
		f.storage, err = f.grant(f.values.StoragePath, "open", s.Generation)
		if err != nil {
			return State{}, err
		}
	}
	if f.newStorage == nil && f.prefs != nil {
		f.newStorage, err = f.grant(f.prefs.DefaultStorage(), "create", s.Generation)
		if err != nil {
			return State{}, err
		}
	}
	s.Storage, s.NewStorage = f.storage, f.newStorage
	return s, nil
}
func (f *Facade) Unlock(ctx context.Context, selection, password string) error {
	if e := f.allow(ctx); e != nil {
		return e
	}
	if !f.work.TryLock() {
		return ErrWorking
	}
	defer f.work.Unlock()
	generation := f.manager.Generation()
	path, e := f.take(selection, "open")
	if e != nil {
		return e
	}
	key := []byte(password)
	defer clear(key)
	if e = f.manager.UnlockAt(path, key, generation); e != nil {
		return e
	}
	return f.authenticated(path, nextGeneration(generation))
}
func (f *Facade) Create(ctx context.Context, selection, password string) error {
	if e := f.allow(ctx); e != nil {
		return e
	}
	if !f.work.TryLock() {
		return ErrWorking
	}
	defer f.work.Unlock()
	generation := f.manager.Generation()
	path, e := f.take(selection, "create")
	if e != nil {
		return e
	}
	key := []byte(password)
	defer clear(key)
	if e = f.manager.CreateAt(path, key, generation); e != nil {
		return e
	}
	return f.authenticated(path, nextGeneration(generation))
}
func (f *Facade) Lock(ctx context.Context) error {
	if e := f.allow(ctx); e != nil {
		return e
	}
	f.manager.Lock()
	f.mu.Lock()
	f.selected = map[string]selected{}
	f.selectionGeneration = ""
	f.mu.Unlock()
	return nil
}
func (f *Facade) List(ctx context.Context, g, q string) ([]vault.Entry, error) {
	if e := f.allow(ctx); e != nil {
		return nil, e
	}
	return f.manager.List(g, q)
}
func (f *Facade) Get(ctx context.Context, g, id string) (vault.Entry, error) {
	if e := f.allow(ctx); e != nil {
		return vault.Entry{}, e
	}
	return f.manager.Get(g, id)
}
func (f *Facade) CreateEntry(ctx context.Context, g string, i vault.Input) (vault.Entry, error) {
	if e := f.allow(ctx); e != nil {
		return vault.Entry{}, e
	}
	return f.manager.CreateEntry(g, i)
}
func (f *Facade) Update(ctx context.Context, g, id string, p vault.Patch) (vault.Entry, error) {
	if e := f.allow(ctx); e != nil {
		return vault.Entry{}, e
	}
	return f.manager.Update(g, id, p)
}
func (f *Facade) Delete(ctx context.Context, g, id, revision string) error {
	if e := f.allow(ctx); e != nil {
		return e
	}
	return f.manager.Delete(g, id, revision)
}
func (f *Facade) Copy(ctx context.Context, g, id, field string) error {
	if e := f.allow(ctx); e != nil {
		return e
	}
	return f.manager.Copy(g, id, field)
}
func (f *Facade) Activity(ctx context.Context, g string) error {
	if e := f.allow(ctx); e != nil {
		return e
	}
	return f.manager.Activity(g)
}
func (f *Facade) SetIdle(ctx context.Context, g string, minutes int) error {
	if e := f.allow(ctx); e != nil {
		return e
	}
	if e := f.manager.SetIdle(g, minutes); e != nil {
		return e
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if g != f.manager.Generation() {
		return vault.ErrLocked
	}
	f.values.IdleMinutes = minutes
	f.saveSettings()
	return nil
}
func (f *Facade) Backup(ctx context.Context, g, selection string) error {
	if e := f.allow(ctx); e != nil {
		return e
	}
	if !f.work.TryLock() {
		return ErrWorking
	}
	defer f.work.Unlock()
	path, e := f.take(selection, "backup")
	if e != nil {
		return e
	}
	return f.manager.Backup(g, path)
}
func (f *Facade) PreserveBackup(ctx context.Context, source, destination, password string) error {
	if e := f.allow(ctx); e != nil {
		return e
	}
	if !f.work.TryLock() {
		return ErrWorking
	}
	defer f.work.Unlock()
	generation := f.manager.Generation()
	src, e := f.take(source, "source")
	if e != nil {
		return e
	}
	dst, e := f.take(destination, "backup")
	if e != nil {
		return e
	}
	key := []byte(password)
	defer clear(key)
	return f.manager.PreserveBackup(src, dst, key, generation)
}
func (f *Facade) Import(ctx context.Context, source, destination, oldPassword, newPassword string) (int, error) {
	if e := f.allow(ctx); e != nil {
		return 0, e
	}
	if !f.work.TryLock() {
		return 0, ErrWorking
	}
	defer f.work.Unlock()
	generation := f.manager.Generation()
	src, e := f.take(source, "source")
	if e != nil {
		return 0, e
	}
	dst, e := f.take(destination, "destination")
	if e != nil {
		return 0, e
	}
	oldKey, newKey := []byte(oldPassword), []byte(newPassword)
	defer clear(oldKey)
	defer clear(newKey)
	count, err := f.manager.ImportAt(src, dst, oldKey, newKey, generation)
	if err != nil {
		return 0, err
	}
	if !f.remember(dst, generation) {
		return 0, vault.ErrLocked
	}
	return count, nil
}
func (f *Facade) Rotate(ctx context.Context, g, oldPassword, newPassword string) (string, error) {
	if e := f.allow(ctx); e != nil {
		return "", e
	}
	if !f.work.TryLock() {
		return "", ErrWorking
	}
	defer f.work.Unlock()
	oldKey, newKey := []byte(oldPassword), []byte(newPassword)
	defer clear(oldKey)
	defer clear(newKey)
	return f.manager.Rotate(g, oldKey, newKey)
}
