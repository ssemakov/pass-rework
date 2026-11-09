// Package settings stores non-secret application preferences, separate from the
// encrypted vault. Paths never come from renderer-supplied filenames.
package settings

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
)

var ErrSettings = errors.New("settings could not be read or saved")

type Values struct {
	StoragePath string `json:"default_storage_path"`
	IdleMinutes int    `json:"idle_minutes"`
}

type Store struct {
	path string
	mu   sync.Mutex
}

func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", ErrSettings
	}
	return filepath.Join(dir, "Pass Rework", "settings.json"), nil
}

func New(path string) *Store            { return &Store{path: path} }
func (s *Store) DefaultStorage() string { return filepath.Join(filepath.Dir(s.path), "storage.pass") }

func valid(v Values) bool {
	return (v.IdleMinutes == 1 || v.IdleMinutes == 5 || v.IdleMinutes == 15) &&
		(v.StoragePath == "" || filepath.IsAbs(v.StoragePath) && len(v.StoragePath) <= 16384 && !strings.ContainsRune(v.StoragePath, 0))
}

func private(st os.FileInfo) bool {
	u, ok := st.Sys().(*syscall.Stat_t)
	return ok && u.Uid == uint32(os.Getuid()) && st.Mode().Perm()&0077 == 0
}

func (s *Store) root() (*os.Root, error) {
	dir := filepath.Dir(s.path)
	if !filepath.IsAbs(s.path) || os.MkdirAll(dir, 0700) != nil {
		return nil, ErrSettings
	}
	st, err := os.Lstat(dir)
	if err != nil || !st.IsDir() || !private(st) {
		return nil, ErrSettings
	}
	r, err := os.OpenRoot(dir)
	if err != nil {
		return nil, ErrSettings
	}
	actual, err := r.Stat(".")
	if err != nil || !os.SameFile(st, actual) {
		r.Close()
		return nil, ErrSettings
	}
	return r, nil
}

func (s *Store) Load() (Values, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := Values{IdleMinutes: 5}
	r, err := s.root()
	if err != nil {
		return v, err
	}
	defer r.Close()
	f, err := r.OpenFile(filepath.Base(s.path), os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if os.IsNotExist(err) {
		return v, nil
	}
	if err != nil {
		return v, ErrSettings
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() || !private(st) || st.Sys().(*syscall.Stat_t).Nlink != 1 || st.Size() > 32768 {
		return v, ErrSettings
	}
	b, err := io.ReadAll(io.LimitReader(f, 32769))
	if err != nil || len(b) > 32768 {
		return v, ErrSettings
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(&v) != nil || d.Decode(new(any)) != io.EOF || !valid(v) {
		return Values{IdleMinutes: 5}, ErrSettings
	}
	return v, nil
}

func (s *Store) Save(v Values) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !valid(v) {
		return ErrSettings
	}
	r, err := s.root()
	if err != nil {
		return err
	}
	defer r.Close()
	name := filepath.Base(s.path)
	if st, e := r.Lstat(name); e == nil {
		if !st.Mode().IsRegular() || !private(st) {
			return ErrSettings
		}
	} else if !os.IsNotExist(e) {
		return ErrSettings
	}
	// A standard atomic settings-file replacement, independent of SQLite's
	// transaction/recovery mechanisms for the encrypted storage.
	var nonce [16]byte
	if _, err = rand.Read(nonce[:]); err != nil {
		return ErrSettings
	}
	tmp := ".settings-" + hex.EncodeToString(nonce[:])
	f, err := r.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return ErrSettings
	}
	defer r.Remove(tmp)
	defer f.Close()
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return ErrSettings
	}
	if _, err = f.Write(append(b, '\n')); err != nil || f.Sync() != nil || f.Close() != nil {
		return ErrSettings
	}
	if r.Rename(tmp, name) != nil {
		return ErrSettings
	}
	dir, err := r.Open(".")
	if err != nil {
		return ErrSettings
	}
	defer dir.Close()
	if dir.Sync() != nil {
		return ErrSettings
	}
	return nil
}
