// Package service defines the complete callable surface; storage objects and
// arbitrary paths are never registered with Wails.
package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"github.com/ssemakov/pass-rework/internal/vault"
	"path/filepath"
	"sync"
)

type Picker func(purpose string) (string, error)
type Selection struct {
	Token string `json:"token"`
	Label string `json:"label"`
}
type selected struct{ path, purpose, generation string }
type Facade struct {
	manager   *vault.Manager
	picker    Picker
	authorize func(context.Context) bool
	mu        sync.Mutex
	selected  map[string]selected
}

func New(m *vault.Manager, p Picker, authorize func(context.Context) bool) *Facade {
	return &Facade{manager: m, picker: p, authorize: authorize, selected: map[string]selected{}}
}

var ErrDenied = errors.New("request not authorized")
var ErrCancelled = errors.New("selection cancelled")

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
	if len(f.selected) > 32 {
		f.selected = map[string]selected{}
	}
	f.selected[token] = selected{path, purpose, generation}
	f.mu.Unlock()
	return Selection{token, filepath.Base(path)}, nil
}
func (f *Facade) State(ctx context.Context) (vault.Status, error) {
	if e := f.allow(ctx); e != nil {
		return vault.Status{}, e
	}
	return f.manager.State(), nil
}
func (f *Facade) Unlock(ctx context.Context, selection, password string) error {
	if e := f.allow(ctx); e != nil {
		return e
	}
	generation := f.manager.Generation()
	path, e := f.take(selection, "open")
	if e != nil {
		return e
	}
	key := []byte(password)
	defer clear(key)
	return f.manager.UnlockAt(path, key, generation)
}
func (f *Facade) Create(ctx context.Context, selection, password string) error {
	if e := f.allow(ctx); e != nil {
		return e
	}
	generation := f.manager.Generation()
	path, e := f.take(selection, "create")
	if e != nil {
		return e
	}
	key := []byte(password)
	defer clear(key)
	return f.manager.CreateAt(path, key, generation)
}
func (f *Facade) Lock(ctx context.Context) error {
	if e := f.allow(ctx); e != nil {
		return e
	}
	f.manager.Lock()
	f.mu.Lock()
	f.selected = map[string]selected{}
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
	return f.manager.SetIdle(g, minutes)
}
func (f *Facade) Backup(ctx context.Context, g, selection string) error {
	if e := f.allow(ctx); e != nil {
		return e
	}
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
	return f.manager.ImportAt(src, dst, oldKey, newKey, generation)
}
func (f *Facade) Rotate(ctx context.Context, g, oldPassword, newPassword string) (string, error) {
	if e := f.allow(ctx); e != nil {
		return "", e
	}
	oldKey, newKey := []byte(oldPassword), []byte(newPassword)
	defer clear(oldKey)
	defer clear(newKey)
	return f.manager.Rotate(g, oldKey, newKey)
}
