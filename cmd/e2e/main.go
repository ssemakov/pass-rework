//go:build e2e

// Synthetic test server only. It is not imported by the production application.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ssemakov/pass-rework/internal/service"
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
	facade  *service.Facade
	manager *vault.Manager
	root    string
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
	s.manager = vault.New(&fakeClipboard{}, nil)
	backup := 0
	destination := 0
	lastBackup := ""
	picker := func(purpose string) (string, error) {
		switch purpose {
		case "create", "open":
			return filepath.Join(root, "vault.pass"), nil
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
	s.facade = service.New(s.manager, picker, func(ctx context.Context) bool { return ctx.Value("test-authorized") == true })
	return nil
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
