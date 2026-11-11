//go:build darwin && production && !server && !mcp && !devtools && !debug && !purego

package main

import (
	"encoding/json"
	"github.com/ssemakov/pass-rework/internal/vault"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Keep storage actions in the native menu so the legacy document stays minimal.
// Commands carry no paths or secrets and still use the same authorized facade.
func installMenu(app *application.App, window *application.WebviewWindow, manager *vault.Manager) {
	menu := app.NewMenu()
	menu.AddRole(application.AppMenu)
	file := menu.AddSubmenu("File")
	command := func(name string) func(*application.Context) {
		return func(*application.Context) {
			payload, _ := json.Marshal(name)
			window.ExecJS("window.dispatchEvent(new CustomEvent('pass-command',{detail:" + string(payload) + "}))")
		}
	}
	file.Add("Open Storage…").SetAccelerator("CmdOrCtrl+O").OnClick(command("open"))
	file.Add("New Storage…").SetAccelerator("CmdOrCtrl+Shift+N").OnClick(command("create"))
	file.Add("Import Storage…").OnClick(command("import"))
	file.AddSeparator()
	file.Add("Add new secret").SetAccelerator("CmdOrCtrl+N").OnClick(command("new"))
	file.Add("Save Encrypted Backup…").OnClick(command("backup"))
	file.Add("Change Master Password…").OnClick(command("rotate"))
	file.Add("Storage settings…").SetAccelerator("CmdOrCtrl+,").OnClick(command("settings"))
	file.AddSeparator()
	file.Add("Lock Storage").SetAccelerator("CmdOrCtrl+~").OnClick(func(*application.Context) { manager.Lock() })
	menu.AddRole(application.EditMenu)
	menu.AddSubmenu("View").AddRole(application.ToggleFullscreen)
	menu.AddRole(application.WindowMenu)
	help := menu.AddSubmenu("Help")
	help.Add("Learn More").OnClick(func(*application.Context) { _ = app.Browser.OpenURL("https://github.com/ssemakov/pass-rework") })
	help.Add("Search Issues").OnClick(func(*application.Context) { _ = app.Browser.OpenURL("https://github.com/ssemakov/pass-rework/issues") })
	app.Menu.Set(menu)
}
