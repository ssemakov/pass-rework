//go:build darwin && production && native_smoke

package main

/*
#cgo LDFLAGS: -framework AppKit
void pass_smoke_key(int tilde);
*/
import "C"
import "github.com/wailsapp/wails/v3/pkg/application"

// Synthetic AppKit input, compiled only into the separate native test binary.
func nativeSmokeKey(tilde bool) {
	value := C.int(0)
	if tilde {
		value = 1
	}
	application.InvokeSync(func() { C.pass_smoke_key(value) })
}
