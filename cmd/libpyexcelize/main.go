// Command libpyexcelize is built with -buildmode=c-shared and exposes the
// capi package through a minimal C ABI used by the Python bindings.
package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"unsafe"

	"github.com/adhadse/py-excelize/internal/capi"
)

var registry = capi.NewRegistry()

func ret(s string) *C.char { return C.CString(s) }

//export PexVersion
func PexVersion() *C.char { return ret(capi.Version) }

//export PexOpen
func PexOpen(path, password *C.char) *C.char {
	return ret(registry.Open(C.GoString(path), C.GoString(password)))
}

//export PexOpenBytes
func PexOpenBytes(data unsafe.Pointer, n C.size_t, password *C.char) *C.char {
	return ret(registry.OpenBytes(C.GoBytes(data, C.int(n)), C.GoString(password)))
}

//export PexClose
func PexClose(h C.longlong) *C.char { return ret(registry.Close(int64(h))) }

//export PexCall
func PexCall(h C.longlong, method, args *C.char) *C.char {
	return ret(registry.Call(int64(h), C.GoString(method), C.GoString(args)))
}

//export PexFree
func PexFree(p *C.char) { C.free(unsafe.Pointer(p)) }

func main() {}
