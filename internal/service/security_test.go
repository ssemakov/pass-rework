package service

import (
	"context"
	"github.com/ssemakov/pass-rework/internal/vault"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBridgeRequestGate(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	handler := SecureAssets(next, "public-test-capability", func() string { return "17" })
	for _, tc := range []struct {
		name, method, origin, cap, id, path string
		want                                int
	}{{"authorized", "POST", "null", "public-test-capability", "17", "/wails/runtime", 204}, {"no capability", "POST", "", "", "17", "/wails/runtime", 403}, {"foreign origin", "POST", "https://evil.test", "public-test-capability", "17", "/wails/runtime", 403}, {"missing window", "POST", "wails://localhost", "public-test-capability", "", "/wails/runtime", 403}, {"get forbidden", "GET", "", "public-test-capability", "17", "/wails/runtime", 403}, {"query forbidden", "POST", "", "public-test-capability", "17", "/wails/runtime?a=1", 403}} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, tc.path, strings.NewReader(`{"object":0,"method":0,"args":{"methodName":"github.com/ssemakov/pass-rework/internal/service.Facade.State","args":[],"call-id":"test"}}`))
			r.Header.Set("Origin", tc.origin)
			r.Header.Set("X-Pass-Capability", tc.cap)
			r.Header.Set("X-Wails-Window-Id", tc.id)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if w.Code != tc.want {
				t.Fatalf("got %d", w.Code)
			}
		})
	}
}
func TestFacadeDeniesCallerPathAndStaleSelections(t *testing.T) {
	m := vault.New(nil, nil)
	defer m.Close()
	f := New(m, func(string) (string, error) { return filepath.Join(t.TempDir(), "vault.pass"), nil }, func(ctx context.Context) bool { return ctx.Value("trusted") == true })
	ctx := context.WithValue(context.Background(), "trusted", true)
	if _, e := f.State(context.Background()); e == nil {
		t.Fatal("unauthorized caller")
	}
	if _, e := f.Choose(ctx, "/tmp/secret"); e == nil {
		t.Fatal("arbitrary path accepted")
	}
	s, e := f.Choose(ctx, "create")
	if e != nil {
		t.Fatal(e)
	}
	m.Lock()
	if e = f.Create(ctx, s.Token, "synthetic"); e == nil {
		t.Fatal("stale file selection reopened vault")
	}
}
func TestShutdownDoesNotWaitOnMainThread(t *testing.T) {
	mainJob := make(chan struct{})
	closed := make(chan struct{})
	quit := make(chan struct{})
	s := NewShutdown(func() { <-mainJob; close(closed) }, func() { close(quit) })
	returned := make(chan bool, 1)
	go func() { returned <- s.ShouldQuit() }()
	select {
	case ready := <-returned:
		if ready {
			t.Fatal("premature quit")
		}
	case <-time.After(time.Second):
		t.Fatal("UI thread blocked on cleanup")
	}
	close(mainJob)
	select {
	case <-quit:
	case <-time.After(time.Second):
		t.Fatal("cleanup did not finish")
	}
	<-closed
	if !s.ShouldQuit() {
		t.Fatal("completed cleanup not acknowledged")
	}
}
