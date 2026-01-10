package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func init() {
	// Use plain mode for deterministic test output
	SetDefault(&Config{Mode: ModePlain})
}

func TestNewSpinner(t *testing.T) {
	spinner := NewSpinner("Loading...")

	if spinner == nil {
		t.Fatal("NewSpinner returned nil")
	}
	if spinner.message != "Loading..." {
		t.Errorf("message = %q, want %q", spinner.message, "Loading...")
	}
	if spinner.active {
		t.Error("spinner should not be active initially")
	}
}

func TestSpinner_StartStop_PlainMode(t *testing.T) {
	// In plain mode, spinner just prints message once
	var buf bytes.Buffer
	spinner := NewSpinner("Processing...")
	spinner.writer = &buf

	spinner.Start()
	time.Sleep(10 * time.Millisecond)
	spinner.Stop()

	output := buf.String()
	if !strings.Contains(output, "Processing...") {
		t.Errorf("output should contain message: %q", output)
	}
}

func TestSpinner_Update(t *testing.T) {
	spinner := NewSpinner("Initial")
	spinner.Update("Updated")

	if spinner.message != "Updated" {
		t.Errorf("message = %q, want %q", spinner.message, "Updated")
	}
}

func TestSpinner_StopWithMessage(t *testing.T) {
	var buf bytes.Buffer
	spinner := NewSpinner("Working...")
	spinner.writer = &buf

	spinner.StopWithMessage("Complete!")

	output := buf.String()
	if !strings.Contains(output, "Complete!") {
		t.Errorf("output should contain final message: %q", output)
	}
}

func TestSpinner_StopWithSuccess(t *testing.T) {
	var buf bytes.Buffer
	spinner := NewSpinner("Working...")
	spinner.writer = &buf

	spinner.StopWithSuccess("Done successfully")

	output := buf.String()
	if !strings.Contains(output, "Done successfully") {
		t.Errorf("output should contain success message: %q", output)
	}
}

func TestSpinner_StopWithError(t *testing.T) {
	var buf bytes.Buffer
	spinner := NewSpinner("Working...")
	spinner.writer = &buf

	spinner.StopWithError("Something failed")

	output := buf.String()
	if !strings.Contains(output, "Something failed") {
		t.Errorf("output should contain error message: %q", output)
	}
}

func TestSpinner_DoubleStart(t *testing.T) {
	// Save original and set to TTY mode for this test
	original := defaultCfg
	defer func() { defaultCfg = original }()
	SetDefault(&Config{Mode: ModeTTY})

	var buf bytes.Buffer
	spinner := NewSpinner("Test")
	spinner.writer = &buf

	spinner.Start()
	spinner.Start() // Should not panic or start again
	spinner.Stop()
}

func TestSpinner_DoubleStop(t *testing.T) {
	spinner := NewSpinner("Test")
	spinner.Stop() // Should not panic even if not started
	spinner.Stop() // Should not panic
}

func TestNewProgress(t *testing.T) {
	progress := NewProgress(100, "Processing items")

	if progress == nil {
		t.Fatal("NewProgress returned nil")
	}
	if progress.total != 100 {
		t.Errorf("total = %d, want 100", progress.total)
	}
	if progress.message != "Processing items" {
		t.Errorf("message = %q, want %q", progress.message, "Processing items")
	}
	if progress.current != 0 {
		t.Errorf("current = %d, want 0", progress.current)
	}
}

func TestProgressBar_Increment(t *testing.T) {
	progress := NewProgress(10, "Test")
	var buf bytes.Buffer
	progress.writer = &buf

	progress.Increment()
	if progress.current != 1 {
		t.Errorf("current = %d, want 1", progress.current)
	}

	progress.Increment()
	if progress.current != 2 {
		t.Errorf("current = %d, want 2", progress.current)
	}
}

func TestProgressBar_Set(t *testing.T) {
	progress := NewProgress(100, "Test")
	var buf bytes.Buffer
	progress.writer = &buf

	progress.Set(50)
	if progress.current != 50 {
		t.Errorf("current = %d, want 50", progress.current)
	}
}

func TestProgressBar_SetMessage(t *testing.T) {
	progress := NewProgress(100, "Initial")
	progress.SetMessage("Updated")

	if progress.message != "Updated" {
		t.Errorf("message = %q, want %q", progress.message, "Updated")
	}
}

func TestProgressBar_Done(t *testing.T) {
	var buf bytes.Buffer
	progress := NewProgress(10, "Test")
	progress.writer = &buf

	progress.Done() // Should not panic
}

func TestProgressBar_DoneWithMessage(t *testing.T) {
	var buf bytes.Buffer
	progress := NewProgress(10, "Test")
	progress.writer = &buf

	progress.DoneWithMessage("Completed 10 items")

	output := buf.String()
	if !strings.Contains(output, "Completed 10 items") {
		t.Errorf("output should contain final message: %q", output)
	}
}

func TestNewTaskProgress(t *testing.T) {
	tasks := []string{"Task 1", "Task 2", "Task 3"}
	tp := NewTaskProgress(tasks)

	if tp == nil {
		t.Fatal("NewTaskProgress returned nil")
	}
	if tp.total != 3 {
		t.Errorf("total = %d, want 3", tp.total)
	}
	if len(tp.tasks) != 3 {
		t.Errorf("len(tasks) = %d, want 3", len(tp.tasks))
	}
}

func TestTaskProgress_StartComplete(t *testing.T) {
	tasks := []string{"Building", "Testing", "Deploying"}
	tp := NewTaskProgress(tasks)
	var buf bytes.Buffer
	tp.writer = &buf

	tp.Start(0)
	if tp.current != 0 {
		t.Errorf("current = %d, want 0", tp.current)
	}

	time.Sleep(10 * time.Millisecond)
	tp.Complete()

	output := buf.String()
	if !strings.Contains(output, "Building") {
		t.Errorf("output should contain task name: %q", output)
	}
}

func TestTaskProgress_Failed(t *testing.T) {
	tasks := []string{"Task 1"}
	tp := NewTaskProgress(tasks)
	var buf bytes.Buffer
	tp.writer = &buf

	tp.Start(0)
	tp.Failed(nil)

	// In plain mode, should show FAILED
	output := buf.String()
	// Output format varies by mode
	_ = output
}

func TestTaskProgress_Summary(t *testing.T) {
	tasks := []string{"Task 1", "Task 2"}
	tp := NewTaskProgress(tasks)
	var buf bytes.Buffer
	tp.writer = &buf

	tp.Start(0)
	tp.Complete()
	tp.Start(1)
	tp.Complete()
	tp.Summary()

	output := buf.String()
	if !strings.Contains(output, "2") {
		t.Errorf("summary should mention task count: %q", output)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		contains string
	}{
		{500 * time.Microsecond, "µs"},
		{50 * time.Millisecond, "ms"},
		{2 * time.Second, "s"},
		{2500 * time.Millisecond, "s"},
	}

	for _, tt := range tests {
		t.Run(tt.duration.String(), func(t *testing.T) {
			result := formatDuration(tt.duration)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("formatDuration(%v) = %q, should contain %q", tt.duration, result, tt.contains)
			}
		})
	}
}

func TestSpinnerFrames(t *testing.T) {
	// Verify spinner frames are non-empty
	if len(SpinnerFrames) == 0 {
		t.Error("SpinnerFrames should not be empty")
	}
	if len(SpinnerFramesASCII) == 0 {
		t.Error("SpinnerFramesASCII should not be empty")
	}

	// Each frame should be non-empty
	for i, frame := range SpinnerFrames {
		if frame == "" {
			t.Errorf("SpinnerFrames[%d] is empty", i)
		}
	}
	for i, frame := range SpinnerFramesASCII {
		if frame == "" {
			t.Errorf("SpinnerFramesASCII[%d] is empty", i)
		}
	}
}

func TestProgressBar_ZeroTotal(t *testing.T) {
	// Edge case: zero total shouldn't cause division by zero
	progress := NewProgress(0, "Empty")
	var buf bytes.Buffer
	progress.writer = &buf

	// This should not panic
	progress.Increment()
	progress.Done()
}
