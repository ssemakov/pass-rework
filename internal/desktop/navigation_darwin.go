package desktop

/*
#cgo LDFLAGS: -framework WebKit -framework AppKit -ObjC
#include <stdlib.h>
int pass_navigation_allowed(const void*,long long);
*/
import "C"

func NavigationAllowed(url string) bool {
	b := C.CBytes([]byte(url))
	defer C.free(b)
	return C.pass_navigation_allowed(b, C.longlong(len(url))) != 0
}
