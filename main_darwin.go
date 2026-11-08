//go:build darwin && production && !server && !mcp && !devtools && !debug && !purego

package main

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"github.com/ssemakov/pass-rework/internal/desktop"
	"github.com/ssemakov/pass-rework/internal/service"
	"github.com/ssemakov/pass-rework/internal/vault"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"strconv"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	if err := run(); err != nil {
		_, _ = os.Stderr.WriteString("Pass could not start.\n")
		os.Exit(1)
	}
}
func run() error {
	// A dev-server override must never expose real vault bindings to remote assets.
	if !desktop.NavigationAllowed("wails://localhost/") {
		return service.ErrDenied
	}
	if os.Getenv("FRONTEND_DEVSERVER_URL") != "" {
		return service.ErrDenied
	}
	var app *application.App
	var window *application.WebviewWindow
	capabilityBytes := make([]byte, 32)
	if _, e := rand.Read(capabilityBytes); e != nil {
		return e
	}
	capability := hex.EncodeToString(capabilityBytes)
	manager := vault.New(desktop.Clipboard{}, func(generation string) {
		if app != nil {
			app.Event.Emit("vault-locked", generation)
		}
	})
	defer manager.Close()
	picker := func(purpose string) (string, error) {
		if smokeEnabled {
			return smokePick(purpose)
		}
		switch purpose {
		case "open", "source":
			return app.Dialog.OpenFile().SetTitle("Select encrypted vault or backup").AttachToWindow(window).PromptForSingleSelection()
		default:
			return app.Dialog.SaveFile().SetFilename("vault.pass").AttachToWindow(window).PromptForSingleSelection()
		}
	}
	authorize := func(ctx context.Context) bool {
		w, ok := ctx.Value(application.WindowKey).(application.Window)
		return ok && window != nil && w.ID() == window.ID()
	}
	shutdown := service.NewShutdown(manager.Close, func() { app.Quit() })
	facade := service.New(manager, picker, authorize)
	sub, err := fs.Sub(assets, "frontend/dist")
	if err != nil {
		return err
	}
	assetHandler := smokeHandler(application.BundledAssetFileServer(sub), capability, func() string {
		if window == nil {
			return ""
		}
		return strconv.FormatUint(uint64(window.ID()), 10)
	})
	app = application.New(application.Options{
		Name: "Pass", Description: "Offline encrypted password vault",
		Flags:    map[string]any{"vaultCapability": capability},
		Services: []application.Service{application.NewService(facade)},
		// Framework debug messages include binding args/results. Discard all logs,
		// independent of runtime log-level switches; no vault values go to disk.
		Logger:       slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})),
		PanicHandler: func(*application.PanicDetails) { os.Exit(1) },
		MarshalError: func(err error) []byte { b, _ := json.Marshal(err.Error()); return b },
		ErrorHandler: func(error) {}, WarningHandler: func(string) {},
		Assets: application.AssetOptions{Handler: assetHandler, Middleware: func(next http.Handler) http.Handler {
			return service.SecureAssets(next, capability, func() string {
				if window == nil {
					return ""
				}
				return strconv.FormatUint(uint64(window.ID()), 10)
			})
		}},
		Mac:         application.MacOptions{ApplicationShouldTerminateAfterLastWindowClosed: true},
		OnShutdown:  func() { desktop.ClearOwnedClipboard() },
		ShouldQuit:  shutdown.ShouldQuit,
		KeyBindings: map[string]func(application.Window){"CmdOrCtrl+L": func(application.Window) { go manager.Lock() }},
	})
	window = app.Window.NewWithOptions(application.WebviewWindowOptions{Title: "Pass", Name: "vault", Width: 1040, Height: 760, MinWidth: 660, MinHeight: 520, URL: "/", DevToolsEnabled: false, DefaultContextMenuDisabled: true, Mac: application.MacWindow{TabbingMode: application.MacWindowTabbingModeDisallowed}})
	for _, event := range []events.ApplicationEventType{events.Common.ScreenLocked, events.Common.SystemWillSleep, events.Mac.ApplicationWillSleep, events.Mac.ApplicationScreensDidSleep} {
		app.Event.OnApplicationEvent(event, func(*application.ApplicationEvent) { go manager.Lock() })
	}
	window.OnWindowEvent(events.Common.WindowClosing, func(*application.WindowEvent) { go manager.Lock() })
	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) { desktop.DisableRestoration() })
	installSmoke(app, window, manager)
	return app.Run()
}
