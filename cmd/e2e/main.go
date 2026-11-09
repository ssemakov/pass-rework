//go:build e2e

// Synthetic test server only. It is not imported by the production application.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ssemakov/pass-rework/internal/service"
	"github.com/ssemakov/pass-rework/internal/settings"
	"github.com/ssemakov/pass-rework/internal/vault"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
)

type fakeClipboard struct {
	sync.Mutex
	token int64
	value string
}

func (c *fakeClipboard) Write(s string) (int64, bool) {
	c.Lock()
	defer c.Unlock()
	c.token++
	c.value = s
	return c.token, true
}
func (c *fakeClipboard) ClearIfOwned(t int64) {
	c.Lock()
	defer c.Unlock()
	if t == c.token {
		c.value = ""
		c.token++
	}
}

type testServer struct {
	sync.RWMutex
	facade       *service.Facade
	manager      *vault.Manager
	root         string
	pickMu       sync.Mutex
	picks        map[string]int
	cancelPicker bool
	openPath     string
}

func (s *testServer) reset() error {
	s.Lock()
	defer s.Unlock()
	if s.manager != nil {
		s.manager.Close()
		os.RemoveAll(s.root)
	}
	root, e := os.MkdirTemp("", "pass-ui-synthetic-")
	if e != nil {
		return e
	}
	s.root = root
	raw, e := os.ReadFile("testdata/legacy/journeyapps-4.0.0.pass")
	if e != nil {
		return e
	}
	os.WriteFile(filepath.Join(root, "legacy.pass"), raw, 0600)
	s.picks = map[string]int{}
	s.cancelPicker = false
	s.openPath = ""
	s.start()
	return nil
}

// Restart preserves the synthetic encrypted files and the real settings file.
func (s *testServer) start() {
	root := s.root
	s.manager = vault.New(&fakeClipboard{}, nil)
	backup := 0
	destination := 0
	lastBackup := ""
	picker := func(purpose string) (string, error) {
		s.pickMu.Lock()
		defer s.pickMu.Unlock()
		s.picks[purpose]++
		if s.cancelPicker {
			s.cancelPicker = false
			return "", nil
		}
		switch purpose {
		case "create", "open":
			if purpose == "open" && s.openPath != "" {
				return s.openPath, nil
			}
			return filepath.Join(root, "storage.pass"), nil
		case "backup":
			backup++
			lastBackup = filepath.Join(root, fmt.Sprintf("backup-%d.pass", backup))
			return lastBackup, nil
		case "source":
			if lastBackup != "" {
				return lastBackup, nil
			}
			return filepath.Join(root, "legacy.pass"), nil
		case "destination":
			destination++
			return filepath.Join(root, fmt.Sprintf("imported-%d.pass", destination)), nil
		}
		return "", service.ErrDenied
	}
	s.facade = service.NewWithSettings(s.manager, picker, func(ctx context.Context) bool { return ctx.Value("test-authorized") == true }, settings.New(filepath.Join(root, "settings.json")))
}
func (s *testServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Host != "127.0.0.1:4173" {
		http.Error(w, "denied", 403)
		return
	}
	if r.Method == "POST" && r.URL.Path == "/test-reset" {
		if s.reset() != nil {
			http.Error(w, "reset failed", 500)
		} else {
			w.WriteHeader(204)
		}
		return
	}
	if r.Method == "POST" && r.URL.Path == "/test-restart" {
		s.Lock()
		s.manager.Close()
		s.start()
		s.Unlock()
		w.WriteHeader(204)
		return
	}
	if r.Method == "POST" && r.URL.Path == "/test-picker-cancel" {
		s.pickMu.Lock()
		s.cancelPicker = true
		s.pickMu.Unlock()
		w.WriteHeader(204)
		return
	}
	if r.Method == "POST" && r.URL.Path == "/test-move-storage" {
		v, err := settings.New(filepath.Join(s.root, "settings.json")).Load()
		if err != nil || v.StoragePath == "" || os.Rename(v.StoragePath, v.StoragePath+".moved") != nil {
			http.Error(w, "move failed", 500)
			return
		}
		s.pickMu.Lock()
		s.openPath = v.StoragePath + ".moved"
		s.pickMu.Unlock()
		w.WriteHeader(204)
		return
	}
	if r.Method == "POST" && r.URL.Path == "/test-stats" {
		s.pickMu.Lock()
		defer s.pickMu.Unlock()
		b, _ := os.ReadFile(filepath.Join(s.root, "settings.json"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"picks": s.picks, "settings": string(b)})
		return
	}
	if r.Method != "POST" {
		http.FileServer(http.Dir("frontend/dist")).ServeHTTP(w, r)
		return
	}
	if !strings.HasPrefix(r.URL.Path, "/test-api/") {
		http.Error(w, "denied", 403)
		return
	}
	var args []json.RawMessage
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<20)).Decode(&args) != nil {
		http.Error(w, "invalid", 400)
		return
	}
	name := r.URL.Path[len("/test-api/"):]
	if !service.AllowedMethod("github.com/ssemakov/pass-rework/internal/service.Facade." + name) {
		http.Error(w, "denied", 403)
		return
	}
	s.RLock()
	f := s.facade
	s.RUnlock()
	method := reflect.ValueOf(f).MethodByName(name)
	typ := method.Type()
	if len(args) != typ.NumIn()-1 {
		http.Error(w, "invalid", 400)
		return
	}
	in := []reflect.Value{reflect.ValueOf(context.WithValue(context.Background(), "test-authorized", true))}
	for i, a := range args {
		v := reflect.New(typ.In(i + 1))
		if json.Unmarshal(a, v.Interface()) != nil {
			http.Error(w, "invalid", 400)
			return
		}
		in = append(in, v.Elem())
	}
	out := method.Call(in)
	last := out[len(out)-1]
	w.Header().Set("Content-Type", "application/json")
	if !last.IsNil() {
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]string{"error": last.Interface().(error).Error()})
		return
	}
	if len(out) == 1 {
		w.Write([]byte("null"))
	} else {
		json.NewEncoder(w).Encode(out[0].Interface())
	}
}
func main() {
	s := &testServer{}
	if s.reset() != nil {
		panic("synthetic fixture setup failed")
	}
	fmt.Println("Synthetic UI server listening on 127.0.0.1:4173")
	if http.ListenAndServe("127.0.0.1:4173", s) != nil {
		os.Exit(1)
	}
}
