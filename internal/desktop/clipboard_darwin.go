package desktop

/*
#cgo LDFLAGS: -framework AppKit
#include <stdlib.h>
long long pass_clip_write(const void*,long long);
void pass_clip_clear(long long);
void pass_clip_clear_last(void);
void pass_disable_restoration(void);
static void pass_clip_wipe(void *p,size_t n){volatile unsigned char *q=p;while(n--)*q++=0;}
*/
import "C"
import (
	"github.com/wailsapp/wails/v3/pkg/application"
)

type Clipboard struct{}

func (Clipboard) Write(text string) (token int64, ok bool) {
	b := C.CBytes([]byte(text))
	defer func() { C.pass_clip_wipe(b, C.size_t(len(text))); C.free(b) }()
	application.InvokeSync(func() { token = int64(C.pass_clip_write(b, C.longlong(len(text)))); ok = token > 0 })
	return
}
func (Clipboard) ClearIfOwned(token int64) {
	application.InvokeSync(func() { C.pass_clip_clear(C.longlong(token)) })
}

func ClearOwnedClipboard() { C.pass_clip_clear_last() }

func DisableRestoration() { application.InvokeSync(func() { C.pass_disable_restoration() }) }
