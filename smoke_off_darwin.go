//go:build darwin && production && !native_smoke

package main

import (
	"github.com/ssemakov/pass-rework/internal/service"
	"github.com/ssemakov/pass-rework/internal/vault"
	"github.com/wailsapp/wails/v3/pkg/application"
	"net/http"
)

const smokeEnabled = false

func smokePick(string) (string, error)                                          { return "", service.ErrDenied }
func smokeSettingsPath() (string, error)                                        { return "", service.ErrDenied }
func smokeHandler(h http.Handler, _ string, _ func() string) http.Handler       { return h }
func installSmoke(*application.App, *application.WebviewWindow, *vault.Manager) {}
func smokePostShutdown()                                                        {}
