package ui

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Spinner provides an animated spinner for indeterminate operations.
type Spinner struct {
	message string
	writer  io.Writer
	program *tea.Program
	model   spinnerModel
}

// spinnerModel wraps bubbles spinner for bubbletea.
type spinnerModel struct {
	spinner spinner.Model
	message string
	quitting bool
}

func (m spinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			m.quitting = true
			return m, tea.Quit
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m spinnerModel) View() string {
	if m.quitting {
		return ""
	}
	return fmt.Sprintf("%s %s", m.spinner.View(), m.message)
}

// NewSpinner creates a new spinner with the given message.
func NewSpinner(message string) *Spinner {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(theme.Info.GetForeground())

	return &Spinner{
		message: message,
		writer:  os.Stderr,
		model: spinnerModel{
			spinner: s,
			message: message,
		},
	}
}

// Start begins the spinner animation.
func (s *Spinner) Start() {
	// In non-TTY mode, just print the message once
	if !isTTY() {
		fmt.Fprintf(s.writer, "%s...\n", s.message)
		return
	}

	s.program = tea.NewProgram(s.model, tea.WithOutput(s.writer))
	go s.program.Run()
}

// Update changes the spinner message.
func (s *Spinner) Update(message string) {
	s.message = message
	s.model.message = message
	if s.program != nil {
		s.program.Send(s.model)
	}
}

// Stop stops the spinner and clears the line.
func (s *Spinner) Stop() {
	if s.program != nil {
		s.program.Quit()
		s.program = nil
	}
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
	fmt.Fprintln(s.writer, Error("✗")+" "+message)
}

// ProgressBar represents a progress bar for determinate operations.
type ProgressBar struct {
	total    int
	current  int
	message  string
	writer   io.Writer
	model    progress.Model
	lastDraw time.Time
}

// NewProgress creates a new progress bar.
func NewProgress(total int, message string) *ProgressBar {
	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
	)

	return &ProgressBar{
		total:   total,
		current: 0,
		message: message,
		writer:  os.Stderr,
		model:   p,
	}
}

// Increment advances the progress by one.
func (p *ProgressBar) Increment() {
	p.current++
	p.draw()
}

// Set sets the current progress value.
func (p *ProgressBar) Set(value int) {
	p.current = value
	p.draw()
}

// SetMessage updates the progress message.
func (p *ProgressBar) SetMessage(message string) {
	p.message = message
	p.draw()
}

// draw renders the progress bar.
func (p *ProgressBar) draw() {
	// In non-TTY mode, throttle output
	if !isTTY() {
		if time.Since(p.lastDraw) < 500*time.Millisecond && p.current < p.total {
			return
		}
		p.lastDraw = time.Now()
		fmt.Fprintf(p.writer, "[%d/%d] %s\n", p.current, p.total, p.message)
		return
	}

	// Calculate progress percentage
	percent := float64(p.current) / float64(p.total)

	// Render progress bar
	bar := p.model.ViewAs(percent)

	// Clear line and draw
	fmt.Fprintf(p.writer, "\r%s %d/%d %s",
		bar,
		p.current,
		p.total,
		p.message)
}

// Done completes the progress bar.
func (p *ProgressBar) Done() {
	if isTTY() {
		// Clear the progress line
		fmt.Fprintf(p.writer, "\r%s\r", strings.Repeat(" ", 100))
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

	if isTTY() {
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

	if isTTY() {
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

	if isTTY() {
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

// isTTY checks if we're running in a TTY.
func isTTY() bool {
	// Simple check: if NO_COLOR is set or stdout is not a terminal
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	// Check if running in CI
	if os.Getenv("CI") != "" {
		return false
	}
	return true
}
