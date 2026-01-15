package ui

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Spinner provides a simple text-based spinner for operations.
type Spinner struct {
	message string
	frames  []string
	mu      sync.Mutex
	stop    chan struct{}
	done    bool
	writer  io.Writer
}

// NewSpinner creates a new spinner with a message.
func NewSpinner(message string) *Spinner {
	return &Spinner{
		message: message,
		frames:  []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		stop:    make(chan struct{}),
		writer:  os.Stderr,
	}
}

// Start starts the spinner animation.
func (s *Spinner) Start() {
	go func() {
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()

		i := 0
		for {
			select {
			case <-s.stop:
				return
			case <-ticker.C:
				s.mu.Lock()
				frame := s.frames[i%len(s.frames)]
				fmt.Fprintf(s.writer, "\r%s %s", Primary(frame), s.message)
				i++
				s.mu.Unlock()
			}
		}
	}()
}

// Update updates the spinner message.
func (s *Spinner) Update(message string) {
	s.mu.Lock()
	s.message = message
	s.mu.Unlock()
}

// Stop stops the spinner and clears the line.
func (s *Spinner) Stop() {
	if s.done {
		return
	}
	s.done = true
	close(s.stop)

	// Clear the spinner line
	fmt.Fprintf(s.writer, "\r%s\r", clearLine())
}

// Success stops the spinner and shows a success message.
func (s *Spinner) Success(message string) {
	s.Stop()
	fmt.Fprintln(s.writer, Success(message))
}

// Error stops the spinner and shows an error message.
func (s *Spinner) Error(message string) {
	s.Stop()
	fmt.Fprintln(s.writer, Error(message))
}

// clearLine returns a string of spaces to clear the current line.
func clearLine() string {
	return "                                                                                "
}
