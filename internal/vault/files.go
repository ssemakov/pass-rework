package vault

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

var ErrFile = errors.New("unsafe, unavailable, or changed vault file")
var ErrBusy = errors.New("vault is already open in another process")
var ErrExists = errors.New("destination already exists; choose a fresh filename")

type location struct {
	root       *os.Root
	path, name string
}

func locate(path string) (*location, error) {
	a, e := filepath.Abs(path)
	if e != nil {
		return nil, ErrFile
	}
	parent, e := filepath.EvalSymlinks(filepath.Dir(a))
	if e != nil {
		return nil, ErrFile
	}
	r, e := os.OpenRoot(parent)
	if e != nil {
		return nil, ErrFile
	}
	name := filepath.Base(a)
	if name == "." || name == "/" {
		r.Close()
		return nil, ErrFile
	}
	return &location{r, filepath.Join(parent, name), name}, nil
}
func (l *location) close() { l.root.Close() }
func regular(f os.FileInfo) bool {
	st, ok := f.Sys().(*syscall.Stat_t)
	return ok && f.Mode().IsRegular() && st.Nlink == 1 && st.Uid == uint32(os.Getuid())
}
func (l *location) check() (os.FileInfo, error) {
	f, e := l.root.Lstat(l.name)
	if e != nil || !regular(f) || f.Size() > 256<<20 {
		return nil, ErrFile
	}
	absolute, e := os.Lstat(l.path)
	if e != nil || !regular(absolute) || !os.SameFile(f, absolute) {
		return nil, ErrFile
	}
	return f, nil
}
func (l *location) noJournal() error {
	for _, s := range []string{"-journal", "-wal", "-shm"} {
		if _, e := l.root.Lstat(l.name + s); e == nil || !os.IsNotExist(e) {
			return errors.New("close the old app and complete database recovery before copying this vault")
		}
	}
	return nil
}
func (l *location) lock() (*os.File, error) {
	f, e := l.root.OpenFile("."+l.name+".pass-lock", os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0600)
	if e != nil {
		return nil, ErrFile
	}
	st, e := f.Stat()
	if e != nil || !regular(st) || st.Mode().Perm()&0077 != 0 {
		f.Close()
		return nil, ErrFile
	}
	if syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) != nil {
		f.Close()
		return nil, ErrBusy
	}
	return f, nil
}
func unlock(f *os.File) {
	if f != nil {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}
}
func (l *location) fresh() (*os.File, error) {
	f, e := l.root.OpenFile(l.name, os.O_CREATE|os.O_EXCL|os.O_RDWR|syscall.O_NOFOLLOW, 0600)
	if os.IsExist(e) {
		return nil, ErrExists
	}
	if e != nil {
		return nil, ErrFile
	}
	return f, nil
}
func (l *location) sync() error {
	f, e := l.root.Open(".")
	if e != nil {
		return ErrFile
	}
	defer f.Close()
	if f.Sync() != nil {
		return ErrFile
	}
	return nil
}
func randomID() string {
	b := make([]byte, 16)
	if _, e := rand.Read(b); e != nil {
		panic("system randomness unavailable")
	}
	return hex.EncodeToString(b)
}
func wipe(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
func copyEncrypted(src, dst *location) (err error) {
	before, e := src.check()
	if e != nil {
		return e
	}
	if src.noJournal() != nil {
		return ErrFile
	}
	in, e := src.root.OpenFile(src.name, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if e != nil {
		return ErrFile
	}
	defer in.Close()
	st, e := in.Stat()
	if e != nil || !os.SameFile(before, st) {
		return ErrFile
	}
	out, e := dst.fresh()
	if e != nil {
		return e
	}
	ok := false
	defer func() {
		out.Close()
		if !ok {
			_ = dst.root.Remove(dst.name)
		}
	}()
	h1 := sha256.New()
	if _, e = io.Copy(io.MultiWriter(out, h1), in); e != nil {
		return ErrFile
	}
	if out.Sync() != nil {
		return ErrFile
	}
	if _, e = out.Seek(0, 0); e != nil {
		return ErrFile
	}
	h2 := sha256.New()
	if _, e = io.Copy(h2, out); e != nil {
		return ErrFile
	}
	after, e := src.check()
	if e != nil || !os.SameFile(before, after) || string(h1.Sum(nil)) != string(h2.Sum(nil)) || src.noJournal() != nil {
		return ErrFile
	}
	if e = dst.sync(); e != nil {
		return e
	}
	ok = true
	return nil
}

// Staging is disposable ownership metadata, not a checkpoint or resume record.
// The next attempt discards its entire incomplete candidate before importing.
type staging struct {
	parent     *location
	root       *os.Root
	name, path string
}

const marker = "pass-rework disposable encrypted migration output v1\n"

func newStaging(dst *location) (*staging, error) {
	sum := sha256.Sum256([]byte(dst.name))
	name := ".pass-stage-" + hex.EncodeToString(sum[:8])
	if _, e := dst.root.Lstat(dst.name); e == nil {
		return nil, ErrExists
	} else if !os.IsNotExist(e) {
		return nil, ErrFile
	}
	info, e := dst.root.Lstat(name)
	if e == nil {
		st, ok := info.Sys().(*syscall.Stat_t)
		if !ok || !info.IsDir() || info.Mode().Perm() != 0700 || st.Uid != uint32(os.Getuid()) {
			return nil, ErrFile
		}
		r, e := dst.root.OpenRoot(name)
		if e != nil {
			return nil, ErrFile
		}
		b, markerErr := r.ReadFile("owner")
		completeMarker := markerErr == nil && string(b) == marker
		partialMarker := (os.IsNotExist(markerErr) || markerErr == nil && len(b) < len(marker) && string(b) == marker[:len(b)])
		if !completeMarker && !partialMarker {
			r.Close()
			return nil, ErrFile
		}
		dir, e := r.Open(".")
		if e != nil {
			r.Close()
			return nil, ErrFile
		}
		entries, e := dir.ReadDir(-1)
		dir.Close()
		if e != nil {
			r.Close()
			return nil, ErrFile
		}
		for _, x := range entries {
			if !completeMarker && x.Name() != "owner" {
				r.Close()
				return nil, ErrFile
			}
			if x.Name() != "owner" && x.Name() != "candidate" && x.Name() != "candidate-journal" {
				r.Close()
				return nil, ErrFile
			}
			fi, e := r.Lstat(x.Name())
			if e != nil || !regular(fi) {
				r.Close()
				return nil, ErrFile
			}
		}
		for _, x := range entries {
			if r.Remove(x.Name()) != nil {
				r.Close()
				return nil, ErrFile
			}
		}
		r.Close()
		if dst.root.Remove(name) != nil {
			return nil, ErrFile
		}
	} else if !os.IsNotExist(e) {
		return nil, ErrFile
	}
	if dst.root.Mkdir(name, 0700) != nil {
		return nil, ErrFile
	}
	r, e := dst.root.OpenRoot(name)
	if e != nil {
		return nil, ErrFile
	}
	if r.WriteFile("owner", []byte(marker), 0600) != nil {
		r.Close()
		return nil, ErrFile
	}
	path := filepath.Join(filepath.Dir(dst.path), name, "candidate")
	return &staging{dst, r, name, path}, nil
}
func (s *staging) discard() {
	for _, n := range []string{"candidate-journal", "candidate", "owner"} {
		_ = s.root.Remove(n)
	}
	s.root.Close()
	_ = s.parent.root.Remove(s.name)
}
func (s *staging) publish() error {
	from, e := s.root.Open(".")
	if e != nil {
		return ErrFile
	}
	defer from.Close()
	to, e := s.parent.root.Open(".")
	if e != nil {
		return ErrFile
	}
	defer to.Close()
	if e = renameExclusive(int(from.Fd()), "candidate", int(to.Fd()), s.parent.name); e != nil {
		return e
	}
	if from.Sync() != nil {
		return ErrFile
	}
	return s.parent.sync()
}
