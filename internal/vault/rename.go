package vault

/*
#define _GNU_SOURCE
#include <stdio.h>
#include <unistd.h>
#include <errno.h>
#include <stdlib.h>
#ifdef __linux__
#include <sys/syscall.h>
#endif
static int pass_rename(int a,const char *x,int b,const char *y){
#ifdef __APPLE__
 if(renameatx_np(a,x,b,y,RENAME_EXCL)==0) return 0;
#else
 if(syscall(SYS_renameat2,a,x,b,y,1)==0) return 0;
#endif
 return errno;
}
*/
import "C"
import "unsafe"

func renameExclusive(from int, old string, to int, name string) error {
	a, b := C.CString(old), C.CString(name)
	defer C.free(unsafe.Pointer(a))
	defer C.free(unsafe.Pointer(b))
	rc := C.pass_rename(C.int(from), a, C.int(to), b)
	if rc == 0 {
		return nil
	}
	if rc == C.EEXIST {
		return ErrExists
	}
	return ErrFile
}
