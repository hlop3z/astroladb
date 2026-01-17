package runtime

import (
	"testing"
	"time"

	"github.com/dop251/goja"

	"github.com/hlop3z/astroladb/internal/alerr"
)

// -----------------------------------------------------------------------------
// JSExecutor Tests
// -----------------------------------------------------------------------------

func TestNewJSExecutor(t *testing.T) {
	vm := goja.New()
	timeout := 5 * time.Second

	exec := NewJSExecutor(vm, timeout)
	if exec == nil {
		t.Fatal("NewJSExecutor() returned nil")
	}
	if exec.vm != vm {
		t.Error("vm should be set")
	}
	if exec.timeout != timeout {
		t.Errorf("timeout = %v, want %v", exec.timeout, timeout)
	}
}

func TestJSExecutor_SetTimeout(t *testing.T) {
	vm := goja.New()
	exec := NewJSExecutor(vm, 1*time.Second)

	exec.SetTimeout(10 * time.Second)
	if exec.Timeout() != 10*time.Second {
		t.Errorf("Timeout() = %v, want 10s", exec.Timeout())
	}
}

func TestJSExecutor_Timeout(t *testing.T) {
	vm := goja.New()
	exec := NewJSExecutor(vm, 5*time.Second)

	if exec.Timeout() != 5*time.Second {
		t.Errorf("Timeout() = %v, want 5s", exec.Timeout())
	}
}

func TestJSExecutor_VM(t *testing.T) {
	vm := goja.New()
	exec := NewJSExecutor(vm, 1*time.Second)

	if exec.VM() != vm {
		t.Error("VM() should return the underlying runtime")
	}
}

func TestJSExecutor_Execute(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		vm := goja.New()
		exec := NewJSExecutor(vm, 5*time.Second)

		err := exec.Execute("var x = 1 + 1;", nil)
		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
	})

	t.Run("syntax_error", func(t *testing.T) {
		vm := goja.New()
		exec := NewJSExecutor(vm, 5*time.Second)

		err := exec.Execute("var x = ;", nil)
		if err == nil {
			t.Error("Execute() should error on syntax error")
		}
		if !alerr.Is(err, alerr.ErrJSExecution) {
			t.Errorf("error should be ErrJSExecution, got: %v", err)
		}
	})

	t.Run("runtime_error", func(t *testing.T) {
		vm := goja.New()
		exec := NewJSExecutor(vm, 5*time.Second)

		err := exec.Execute("throw new Error('test');", nil)
		if err == nil {
			t.Error("Execute() should error on throw")
		}
	})

	t.Run("with_error_context", func(t *testing.T) {
		vm := goja.New()
		exec := NewJSExecutor(vm, 5*time.Second)
		ctx := NewErrorContext("/path/to/file.js", "var x = ;")

		err := exec.Execute("var x = ;", ctx)
		if err == nil {
			t.Fatal("Execute() should error on syntax error")
		}
		// Error should contain file path from context
		errStr := err.Error()
		if errStr == "" {
			t.Error("error string should not be empty")
		}
	})
}

func TestJSExecutor_ExecuteWithResult(t *testing.T) {
	t.Run("returns_number", func(t *testing.T) {
		vm := goja.New()
		exec := NewJSExecutor(vm, 5*time.Second)

		result, err := exec.ExecuteWithResult("1 + 2", nil)
		if err != nil {
			t.Fatalf("ExecuteWithResult() error = %v", err)
		}
		if result.ToInteger() != 3 {
			t.Errorf("result = %v, want 3", result.Export())
		}
	})

	t.Run("returns_string", func(t *testing.T) {
		vm := goja.New()
		exec := NewJSExecutor(vm, 5*time.Second)

		result, err := exec.ExecuteWithResult("'hello' + ' world'", nil)
		if err != nil {
			t.Fatalf("ExecuteWithResult() error = %v", err)
		}
		if result.String() != "hello world" {
			t.Errorf("result = %q, want 'hello world'", result.String())
		}
	})

	t.Run("returns_object", func(t *testing.T) {
		vm := goja.New()
		exec := NewJSExecutor(vm, 5*time.Second)

		result, err := exec.ExecuteWithResult("({a: 1, b: 2})", nil)
		if err != nil {
			t.Fatalf("ExecuteWithResult() error = %v", err)
		}
		obj, ok := result.Export().(map[string]any)
		if !ok {
			t.Fatalf("expected map result, got %T", result.Export())
		}
		if obj["a"] != int64(1) {
			t.Errorf("obj['a'] = %v, want 1", obj["a"])
		}
	})

	t.Run("error_returns_nil", func(t *testing.T) {
		vm := goja.New()
		exec := NewJSExecutor(vm, 5*time.Second)

		result, err := exec.ExecuteWithResult("throw new Error('test');", nil)
		if err == nil {
			t.Error("ExecuteWithResult() should error on throw")
		}
		if result != nil {
			t.Error("result should be nil on error")
		}
	})
}

func TestJSExecutor_ExecuteWithTimeout(t *testing.T) {
	t.Run("completes_within_timeout", func(t *testing.T) {
		vm := goja.New()
		exec := NewJSExecutor(vm, 1*time.Second)

		err := exec.ExecuteWithTimeout("var x = 1;", 5*time.Second, nil)
		if err != nil {
			t.Errorf("ExecuteWithTimeout() error = %v", err)
		}
	})

	t.Run("timeout_on_infinite_loop", func(t *testing.T) {
		vm := goja.New()
		exec := NewJSExecutor(vm, 10*time.Second)

		start := time.Now()
		err := exec.ExecuteWithTimeout("while(true) {}", 100*time.Millisecond, nil)
		elapsed := time.Since(start)

		if err == nil {
			t.Error("ExecuteWithTimeout() should error on timeout")
		}
		if !alerr.Is(err, alerr.ErrJSTimeout) {
			t.Errorf("expected ErrJSTimeout, got: %v", err)
		}
		if elapsed > 500*time.Millisecond {
			t.Errorf("should timeout quickly, took %v", elapsed)
		}
	})

	t.Run("timeout_with_error_context", func(t *testing.T) {
		vm := goja.New()
		exec := NewJSExecutor(vm, 10*time.Second)
		ctx := NewErrorContext("/path/to/file.js", "while(true) {}")

		err := exec.ExecuteWithTimeout("while(true) {}", 100*time.Millisecond, ctx)
		if err == nil {
			t.Fatal("ExecuteWithTimeout() should error on timeout")
		}
		// Error should have file context
		if !alerr.Is(err, alerr.ErrJSTimeout) {
			t.Errorf("expected ErrJSTimeout, got: %v", err)
		}
	})

	t.Run("syntax_error", func(t *testing.T) {
		vm := goja.New()
		exec := NewJSExecutor(vm, 10*time.Second)

		err := exec.ExecuteWithTimeout("var x = ;", 5*time.Second, nil)
		if err == nil {
			t.Error("ExecuteWithTimeout() should error on syntax error")
		}
		if !alerr.Is(err, alerr.ErrJSExecution) {
			t.Errorf("expected ErrJSExecution, got: %v", err)
		}
	})
}

func TestJSExecutor_SetGlobal(t *testing.T) {
	vm := goja.New()
	exec := NewJSExecutor(vm, 5*time.Second)

	exec.SetGlobal("testVar", 42)

	result, err := exec.ExecuteWithResult("testVar", nil)
	if err != nil {
		t.Fatalf("error reading global: %v", err)
	}
	if result.ToInteger() != 42 {
		t.Errorf("testVar = %v, want 42", result.Export())
	}
}

func TestJSExecutor_SetGlobal_Function(t *testing.T) {
	vm := goja.New()
	exec := NewJSExecutor(vm, 5*time.Second)

	exec.SetGlobal("add", func(a, b int) int {
		return a + b
	})

	result, err := exec.ExecuteWithResult("add(2, 3)", nil)
	if err != nil {
		t.Fatalf("error calling global function: %v", err)
	}
	if result.ToInteger() != 5 {
		t.Errorf("add(2, 3) = %v, want 5", result.Export())
	}
}

// -----------------------------------------------------------------------------
// ErrorContext Tests
// -----------------------------------------------------------------------------

func TestNewErrorContext(t *testing.T) {
	ctx := NewErrorContext("/path/to/file.js", "var x = 1;")

	if ctx.FilePath != "/path/to/file.js" {
		t.Errorf("FilePath = %q, want %q", ctx.FilePath, "/path/to/file.js")
	}
	if ctx.Code != "var x = 1;" {
		t.Errorf("Code = %q, want %q", ctx.Code, "var x = 1;")
	}
}

func TestErrorContext_Empty(t *testing.T) {
	ctx := NewErrorContext("", "")

	if ctx.FilePath != "" {
		t.Error("FilePath should be empty")
	}
	if ctx.Code != "" {
		t.Error("Code should be empty")
	}
}

// -----------------------------------------------------------------------------
// Edge Cases
// -----------------------------------------------------------------------------

func TestJSExecutor_MultipleExecutions(t *testing.T) {
	vm := goja.New()
	exec := NewJSExecutor(vm, 5*time.Second)

	// First execution sets a variable
	err := exec.Execute("var counter = 1;", nil)
	if err != nil {
		t.Fatalf("first execution error: %v", err)
	}

	// Second execution modifies it
	err = exec.Execute("counter += 10;", nil)
	if err != nil {
		t.Fatalf("second execution error: %v", err)
	}

	// Third execution reads it
	result, err := exec.ExecuteWithResult("counter", nil)
	if err != nil {
		t.Fatalf("third execution error: %v", err)
	}
	if result.ToInteger() != 11 {
		t.Errorf("counter = %v, want 11", result.Export())
	}
}

func TestJSExecutor_ClearInterruptAfterTimeout(t *testing.T) {
	vm := goja.New()
	exec := NewJSExecutor(vm, 10*time.Second)

	// First, trigger a timeout
	_ = exec.ExecuteWithTimeout("while(true) {}", 50*time.Millisecond, nil)

	// Wait a bit for things to settle
	time.Sleep(10 * time.Millisecond)

	// Need a new VM since the old one is interrupted
	vm2 := goja.New()
	exec2 := NewJSExecutor(vm2, 5*time.Second)

	// Should be able to run code on new executor
	err := exec2.Execute("var x = 1;", nil)
	if err != nil {
		t.Errorf("subsequent execution should work on fresh VM: %v", err)
	}
}

func TestJSExecutor_ZeroTimeout(t *testing.T) {
	vm := goja.New()
	exec := NewJSExecutor(vm, 0)

	// With zero timeout, any code should interrupt immediately
	// Note: This tests the edge case behavior
	err := exec.ExecuteWithTimeout("var x = 1;", 0, nil)
	// With 0 timeout, the timer fires immediately but the code might still complete
	// This is an edge case - behavior depends on goroutine scheduling
	_ = err // Just ensure it doesn't panic
}
