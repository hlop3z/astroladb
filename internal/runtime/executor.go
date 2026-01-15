// Package runtime provides JS execution capabilities.
package runtime

import (
	"time"

	"github.com/dop251/goja"

	"github.com/hlop3z/astroladb/internal/alerr"
)

// JSExecutor handles JavaScript code execution with timeout support.
// It wraps a Goja runtime and provides consistent error handling.
type JSExecutor struct {
	vm      *goja.Runtime
	timeout time.Duration
}

// NewJSExecutor creates a new executor with the given runtime and timeout.
func NewJSExecutor(vm *goja.Runtime, timeout time.Duration) *JSExecutor {
	return &JSExecutor{
		vm:      vm,
		timeout: timeout,
	}
}

// SetTimeout updates the execution timeout.
func (e *JSExecutor) SetTimeout(d time.Duration) {
	e.timeout = d
}

// Timeout returns the current timeout duration.
func (e *JSExecutor) Timeout() time.Duration {
	return e.timeout
}

// VM returns the underlying Goja runtime.
func (e *JSExecutor) VM() *goja.Runtime {
	return e.vm
}

// Execute runs JavaScript code and returns any error.
// It applies the configured timeout and interrupts execution if exceeded.
func (e *JSExecutor) Execute(code string, errorContext *ErrorContext) error {
	// Set up timeout
	timer := time.AfterFunc(e.timeout, func() {
		e.vm.Interrupt("execution timeout")
	})
	defer timer.Stop()

	_, err := e.vm.RunString(code)
	if err != nil {
		return wrapJSError(err, alerr.ErrJSExecution, "JavaScript execution failed", errorContext)
	}
	return nil
}

// ExecuteWithResult runs JavaScript code and returns the result value.
func (e *JSExecutor) ExecuteWithResult(code string, errorContext *ErrorContext) (goja.Value, error) {
	// Set up timeout
	timer := time.AfterFunc(e.timeout, func() {
		e.vm.Interrupt("execution timeout")
	})
	defer timer.Stop()

	result, err := e.vm.RunString(code)
	if err != nil {
		return nil, wrapJSError(err, alerr.ErrJSExecution, "JavaScript execution failed", errorContext)
	}
	return result, nil
}

// ExecuteWithTimeout runs JavaScript code with a specific timeout override.
func (e *JSExecutor) ExecuteWithTimeout(code string, timeout time.Duration, errorContext *ErrorContext) error {
	// Set up timeout
	timer := time.AfterFunc(timeout, func() {
		e.vm.Interrupt("execution timeout")
	})
	defer timer.Stop()

	_, err := e.vm.RunString(code)
	if err != nil {
		// Check if it was a timeout
		if interruptErr, ok := err.(*goja.InterruptedError); ok {
			timeoutErr := alerr.New(alerr.ErrJSTimeout, "script execution timed out").
				With("timeout", timeout.String()).
				With("interrupt", interruptErr.String())
			if errorContext != nil && errorContext.FilePath != "" {
				timeoutErr.WithFile(errorContext.FilePath, 0)
			}
			return timeoutErr
		}

		return wrapJSError(err, alerr.ErrJSExecution, "JavaScript execution failed", errorContext)
	}

	// Clear any pending interrupt
	e.vm.ClearInterrupt()

	return nil
}

// SetGlobal binds a value to a global variable name.
func (e *JSExecutor) SetGlobal(name string, value any) {
	e.vm.Set(name, value)
}

// ErrorContext provides context for rich error messages.
type ErrorContext struct {
	FilePath string
	Code     string
}

// NewErrorContext creates an error context for JS execution.
func NewErrorContext(filePath, code string) *ErrorContext {
	return &ErrorContext{
		FilePath: filePath,
		Code:     code,
	}
}
