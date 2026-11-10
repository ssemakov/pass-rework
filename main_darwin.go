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
	"github.com/ssemakov/pass-rework/internal/settings"
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
		_, _ = os.Stderr.WriteString("PassV2 could not start.\n")
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
			return app.Dialog.OpenFile().SetTitle("Select storage path").AttachToWindow(window).PromptForSingleSelection()
		default:
			return app.Dialog.SaveFile().SetFilename("storage.pass").AttachToWindow(window).PromptForSingleSelection()
		}
	}
	authorize := func(ctx context.Context) bool {
		w, ok := ctx.Value(application.WindowKey).(application.Window)
		return ok && window != nil && w.ID() == window.ID()
	}
	shutdown := service.NewShutdown(manager.Close, func() { app.Quit() })
	settingsPath, err := settings.DefaultPath()
	if smokeEnabled {
		settingsPath, err = smokeSettingsPath()
	}
	if err != nil {
		return err
	}
	facade := service.NewWithSettings(manager, picker, authorize, settings.New(settingsPath))

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
		Name: "PassV2", Description: "Another password manager.",
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
		Mac:          application.MacOptions{ApplicationShouldTerminateAfterLastWindowClosed: true},
		OnShutdown:   func() { desktop.ClearOwnedClipboard() },
		PostShutdown: smokePostShutdown,
		ShouldQuit:   shutdown.ShouldQuit,
		// Wails names the physical tilde key as backquote on macOS, including
		// when Shift produces '~'. The native menu displays the requested ⌘~.
		KeyBindings: map[string]func(application.Window){
			"CmdOrCtrl+`":       func(application.Window) { go manager.Lock() },
			"CmdOrCtrl+Shift+`": func(application.Window) { go manager.Lock() },
		},
	})
	window = app.Window.NewWithOptions(application.WebviewWindowOptions{Title: "PassV2", Name: "vault", Width: 904, Height: 534, Hidden: true, URL: "/", DevToolsEnabled: false, DefaultContextMenuDisabled: true, Mac: application.MacWindow{TabbingMode: application.MacWindowTabbingModeDisallowed}})
	installMenu(app, window, manager)
	for _, event := range []events.ApplicationEventType{events.Common.ScreenLocked, events.Common.SystemWillSleep, events.Mac.ApplicationWillSleep, events.Mac.ApplicationScreensDidSleep} {
		app.Event.OnApplicationEvent(event, func(*application.ApplicationEvent) { go manager.Lock() })
	}
	window.OnWindowEvent(events.Mac.WebViewDidStartProvisionalNavigation, func(*application.WindowEvent) { go manager.Lock() })
	window.OnWindowEvent(events.Common.WindowClosing, func(*application.WindowEvent) { go manager.Lock() })
	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		// Electron's legacy 904×534 dimensions describe the outer window.
		// Wails constructs a content rect; SetSize explicitly sizes the frame.
		window.SetSize(904, 534)
		window.Show()
		desktop.DisableRestoration()
	})
	installSmoke(app, window, manager)
	return app.Run()
}
