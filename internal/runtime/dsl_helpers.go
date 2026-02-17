package runtime

import (
	"fmt"

	"github.com/dop251/goja"

	"github.com/hlop3z/astroladb/internal/alerr"
)

// panicStructured panics with a structured error string.
// Format: "[XXX-NNN] cause|help"
// Goja catches the panic and creates an Exception at the JS call site,
// giving correct line numbers. The code and help are extracted later by
// wrapJSError via extractStructuredError.
func panicStructured(vm *goja.Runtime, code alerr.Code, cause, help string) {
	panic(vm.ToValue(fmt.Sprintf("[%s] %s|%s", code, cause, help)))
}

// panicPassthrough re-panics an error from a JS callback, extracting the
// exception value to strip Goja stack traces. This preserves structured
// error codes while avoiding double-wrapping.
func panicPassthrough(vm *goja.Runtime, err error) {
	if exc, ok := err.(*goja.Exception); ok {
		panic(vm.ToValue(exc.Value().String()))
	}
	panic(vm.ToValue(err.Error()))
}
