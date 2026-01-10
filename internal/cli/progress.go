package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// Spinner provides an animated spinner for indeterminate operations.
type Spinner struct {
	message string
	writer  io.Writer
	active  bool
	done    chan struct{}
	mu      sync.Mutex
	frames  []string
	current int
}

// SpinnerFrames are the animation frames for the spinner.
var SpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// SpinnerFramesASCII are ASCII fallback frames for non-Unicode terminals.
var SpinnerFramesASCII = []string{"|", "/", "-", "\\"}

// NewSpinner creates a new spinner with the given message.
func NewSpinner(message string) *Spinner {
	frames := SpinnerFrames
	// Use ASCII frames if not in TTY mode (simpler for logs)
	if !EnableColors() {
		frames = SpinnerFramesASCII
	}
	return &Spinner{
		message: message,
		writer:  os.Stderr,
		frames:  frames,
	}
}

// Start begins the spinner animation.
func (s *Spinner) Start() {
	if !EnableColors() {
		// In non-TTY mode, just print the message once
		fmt.Fprintf(s.writer, "%s...\n", s.message)
		return
	}

	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return
	}
	s.active = true
	s.done = make(chan struct{})
	s.mu.Unlock()

	go s.spin()
}

// spin runs the animation loop.
func (s *Spinner) spin() {
	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.mu.Lock()
			frame := Progress(s.frames[s.current])
			msg := s.message
			s.current = (s.current + 1) % len(s.frames)
			s.mu.Unlock()

			// Clear line and write new frame
			fmt.Fprintf(s.writer, "\r%s %s", frame, msg)
		}
	}
}

// Update changes the spinner message.
func (s *Spinner) Update(message string) {
	s.mu.Lock()
	s.message = message
	s.mu.Unlock()
}

// Stop stops the spinner and clears the line.
func (s *Spinner) Stop() {
	if !EnableColors() {
		return
	}

	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return
	}
	s.active = false
	close(s.done)
	s.mu.Unlock()

	// Clear the spinner line
	fmt.Fprintf(s.writer, "\r%s\r", strings.Repeat(" ", len(s.message)+10))
}

// StopWithMessage stops the spinner and prints a final message.
func (s *Spinner) StopWithMessage(message string) {
	s.Stop()
	fmt.Fprintln(s.writer, message)
}

// StopWithSuccess stops the spinner with a success message.
func (s *Spinner) StopWithSuccess(message string) {
	s.Stop()
	fmt.Fprintln(s.writer, Success("✓")+" "+message)
}

// StopWithError stops the spinner with an error message.
func (s *Spinner) StopWithError(message string) {
	s.Stop()
	fmt.Fprintln(s.writer, Failed("✗")+" "+message)
}

// Progress represents a progress bar for determinate operations.
type ProgressBar struct {
	total    int
	current  int
	message  string
	writer   io.Writer
	width    int
	mu       sync.Mutex
	lastDraw time.Time
}

// NewProgress creates a new progress bar.
func NewProgress(total int, message string) *ProgressBar {
	return &ProgressBar{
		total:   total,
		message: message,
		writer:  os.Stderr,
		width:   40,
	}
}

// Increment advances the progress by one.
func (p *ProgressBar) Increment() {
	p.mu.Lock()
	p.current++
	p.mu.Unlock()
	p.draw()
}

// Set sets the current progress value.
func (p *ProgressBar) Set(value int) {
	p.mu.Lock()
	p.current = value
	p.mu.Unlock()
	p.draw()
}

// SetMessage updates the progress message.
func (p *ProgressBar) SetMessage(message string) {
	p.mu.Lock()
	p.message = message
	p.mu.Unlock()
	p.draw()
}

// draw renders the progress bar.
func (p *ProgressBar) draw() {
	if !EnableColors() {
		// In non-TTY mode, throttle output
		p.mu.Lock()
		if time.Since(p.lastDraw) < 500*time.Millisecond && p.current < p.total {
			p.mu.Unlock()
			return
		}
		p.lastDraw = time.Now()
		current := p.current
		total := p.total
		message := p.message
		p.mu.Unlock()

		fmt.Fprintf(p.writer, "[%d/%d] %s\n", current, total, message)
		return
	}

	p.mu.Lock()
	current := p.current
	total := p.total
	message := p.message
	p.mu.Unlock()

	// Calculate progress
	percent := float64(current) / float64(total)
	filled := int(percent * float64(p.width))
	empty := p.width - filled

	// Build progress bar
	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

	// Clear line and draw
	fmt.Fprintf(p.writer, "\r%s [%s] %d/%d %s",
		Progress(fmt.Sprintf("%3.0f%%", percent*100)),
		bar,
		current,
		total,
		message)
}

// Done completes the progress bar.
func (p *ProgressBar) Done() {
	if EnableColors() {
		// Clear the progress line
		fmt.Fprintf(p.writer, "\r%s\r", strings.Repeat(" ", p.width+50))
	}
}

// DoneWithMessage completes the progress bar with a message.
func (p *ProgressBar) DoneWithMessage(message string) {
	p.Done()
	fmt.Fprintln(p.writer, message)
}

// TaskProgress provides step-by-step progress output for a list of tasks.
type TaskProgress struct {
	tasks   []string
	current int
	total   int
	writer  io.Writer
	times   []time.Duration
	start   time.Time
}

// NewTaskProgress creates a new task progress tracker.
func NewTaskProgress(tasks []string) *TaskProgress {
	return &TaskProgress{
		tasks:  tasks,
		total:  len(tasks),
		writer: os.Stderr,
		times:  make([]time.Duration, len(tasks)),
	}
}

// Start starts tracking the next task.
func (t *TaskProgress) Start(index int) {
	t.current = index
	t.start = time.Now()

	if EnableColors() {
		fmt.Fprintf(t.writer, "  [%d/%d] %s ",
			index+1, t.total, t.tasks[index])
	} else {
		fmt.Fprintf(t.writer, "  [%d/%d] %s...\n",
			index+1, t.total, t.tasks[index])
	}
}

// Complete marks the current task as complete.
func (t *TaskProgress) Complete() {
	elapsed := time.Since(t.start)
	t.times[t.current] = elapsed

	if EnableColors() {
		// Fill with dots and show result
		taskLen := len(t.tasks[t.current])
		dots := strings.Repeat(".", 40-taskLen)
		fmt.Fprintf(t.writer, "%s %s (%s)\n",
			dots, Done("done"), formatDuration(elapsed))
	}
}

// Failed marks the current task as failed.
func (t *TaskProgress) Failed(err error) {
	elapsed := time.Since(t.start)
	t.times[t.current] = elapsed

	if EnableColors() {
		taskLen := len(t.tasks[t.current])
		dots := strings.Repeat(".", 40-taskLen)
		fmt.Fprintf(t.writer, "%s %s (%s)\n",
			dots, Failed("failed"), formatDuration(elapsed))
	} else {
		fmt.Fprintf(t.writer, "    FAILED: %v\n", err)
	}
}

// Summary prints a summary of all tasks.
func (t *TaskProgress) Summary() {
	var total time.Duration
	for _, d := range t.times {
		total += d
	}
	fmt.Fprintf(t.writer, "\nCompleted %d tasks in %s\n", t.total, formatDuration(total))
}

// formatDuration formats a duration for display.
func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}
