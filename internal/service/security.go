package service

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

const ContentSecurityPolicy = "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; font-src 'self'; object-src 'none'; base-uri 'none'; frame-src 'none'; frame-ancestors 'none'; form-action 'none'"

// Native Wails injects/overwrites the window ID before this handler. A per-run
// capability additionally authenticates opaque-origin custom-scheme POSTs.
func SecureAssets(next http.Handler, capability string, windowID func() string) http.Handler {
	slots := make(chan struct{}, 8)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deny := func() { http.Error(w, "denied", http.StatusForbidden) }
		origin := r.Header.Get("Origin")
		if origin != "" && origin != "null" && origin != "wails://localhost" {
			deny()
			return
		}
		ref := r.Header.Get("Referer")
		if ref != "" && !strings.HasPrefix(ref, "wails://localhost/") {
			deny()
			return
		}
		if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
			deny()
			return
		}
		w.Header().Set("Content-Security-Policy", ContentSecurityPolicy)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		if r.URL.Path == "/wails/runtime" {
			if r.Method != "POST" || r.URL.RawQuery != "" || r.Header.Get("X-Wails-Chunk-Id") != "" || capability == "" || subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Pass-Capability")), []byte(capability)) != 1 || windowID() == "" || r.Header.Get("X-Wails-Window-Id") != windowID() {
				deny()
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, 32*1024*1024)
			body, e := io.ReadAll(r.Body)
			if e != nil {
				deny()
				return
			}
			var envelope struct {
				Object int `json:"object"`
				Method int `json:"method"`
				Args   struct {
					MethodName string            `json:"methodName"`
					Values     []json.RawMessage `json:"args"`
					CallID     string            `json:"call-id"`
				} `json:"args"`
			}
			decoder := json.NewDecoder(bytes.NewReader(body))
			decoder.DisallowUnknownFields()
			if decoder.Decode(&envelope) != nil || envelope.Object != 0 || envelope.Method != 0 || envelope.Args.CallID == "" || !AllowedMethod(envelope.Args.MethodName) {
				deny()
				return
			}
			if decoder.Decode(new(any)) != io.EOF {
				deny()
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))
			if !strings.HasSuffix(envelope.Args.MethodName, ".Lock") {
				select {
				case slots <- struct{}{}:
					defer func() { <-slots }()
				default:
					http.Error(w, "busy", http.StatusTooManyRequests)
					return
				}

			}
		}
		next.ServeHTTP(w, r)
	})
}

func AllowedMethod(name string) bool {
	const prefix = "github.com/ssemakov/pass-rework/internal/service.Facade."
	if !strings.HasPrefix(name, prefix) {
		return false
	}
	switch strings.TrimPrefix(name, prefix) {
	case "Choose", "State", "Unlock", "Create", "Lock", "List", "Get", "CreateEntry", "Update", "Delete", "Copy", "Activity", "SetIdle", "Backup", "PreserveBackup", "Import", "Rotate":
		return true
	}
	return false
}
