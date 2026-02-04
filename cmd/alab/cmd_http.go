package main

import (
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"

	"github.com/hlop3z/astroladb/internal/alerr"
	"github.com/hlop3z/astroladb/internal/ui"
	"github.com/hlop3z/astroladb/pkg/astroladb"
)

// SSE clients for hot reload
var (
	sseClients   = make(map[chan struct{}]struct{})
	sseClientsMu sync.Mutex
)

// Error deduplication - only show each unique error once per reload cycle
var (
	shownErrors   = make(map[string]struct{})
	shownErrorsMu sync.Mutex
)

// HTML template paths
const (
	templateSwaggerHTML = "templates/swagger.html"
	templateGraphQLHTML = "templates/graphiql.html"
)

// serveSchemaExport handles schema export endpoints with fresh client creation.
func serveSchemaExport(w http.ResponseWriter, format, endpoint, contentType string) {
	freshClient, err := newSchemaOnlyClient()
	if err != nil {
		handleSchemaError(w, err, endpoint)
		return
	}
	defer freshClient.Close()

	data, err := freshClient.SchemaExport(format)
	if err != nil {
		handleSchemaError(w, err, endpoint)
		return
	}

	w.Header().Set(HeaderContentType, contentType)
	w.Header().Set(HeaderCORS, CORSAllowAll)
	w.Write(data)
}

// handleSchemaError writes a user-friendly error response and logs to console.
func handleSchemaError(w http.ResponseWriter, err error, endpoint string) {
	// Format error with clickable file URI for IDEs
	errMsg := formatErrorWithURI(err)

	// Only print each unique error once per reload cycle
	shownErrorsMu.Lock()
	_, alreadyShown := shownErrors[errMsg]
	if !alreadyShown {
		shownErrors[errMsg] = struct{}{}
	}
	shownErrorsMu.Unlock()

	if !alreadyShown {
		// Log to console with colors
		fmt.Printf("\n  %s\n", ui.Error("Schema error"))
		fmt.Printf("  %s\n\n", errMsg)
	}

	// Return JSON error for API clients
	w.Header().Set(HeaderContentType, ContentTypeJSON)
	w.Header().Set(HeaderCORS, CORSAllowAll)
	w.WriteHeader(http.StatusBadRequest)
	fmt.Fprintf(w, `{"error": %q}`, errMsg)
}

// formatErrorWithURI formats errors in a Rust-style format with source context.
func formatErrorWithURI(err error) string {
	var b strings.Builder

	// Try to extract alerr.Error first (has rich context)
	var aErr *alerr.Error
	if errors.As(err, &aErr) {
		ctx := aErr.GetContext()
		file, _ := ctx["file"].(string)
		line, _ := ctx["line"].(int)
		col, _ := ctx["column"].(int)
		source, _ := ctx["source"].(string)
		spanStart, _ := ctx["span_start"].(int)
		spanEnd, _ := ctx["span_end"].(int)

		// Get the actual error message (from cause if available)
		msg := aErr.GetMessage()
		if cause := aErr.GetCause(); cause != nil {
			// Extract just the meaningful part of the cause
			causeMsg := cause.Error()
			// Clean up Goja stack traces and prefixes
			causeMsg = strings.TrimPrefix(causeMsg, "GoError: ")
			// Remove " at github.com/..." stack trace suffix
			if idx := strings.Index(causeMsg, " at github.com/"); idx > 0 {
				causeMsg = causeMsg[:idx]
			}
			// Remove " at <eval>..." suffix
			if idx := strings.Index(causeMsg, " at <eval>"); idx > 0 {
				causeMsg = causeMsg[:idx]
			}
			if causeMsg != "" && causeMsg != msg {
				msg = causeMsg
			}
		}

		if file != "" {
			absPath := file
			if !filepath.IsAbs(absPath) {
				if abs, err := filepath.Abs(absPath); err == nil {
					absPath = abs
				}
			}
			if line == 0 {
				line = 1
			}
			if col == 0 {
				col = 1
			}

			// Rust-style header
			b.WriteString(fmt.Sprintf("%s:%d:%d: %s\n", absPath, line, col, msg))

			// Show source line with caret if available
			if source != "" {
				lineNumWidth := len(fmt.Sprintf("%d", line))
				padding := strings.Repeat(" ", lineNumWidth)

				b.WriteString(fmt.Sprintf("%s |\n", padding))
				b.WriteString(fmt.Sprintf("%d | %s\n", line, source))

				// Draw the caret line
				if col > 0 {
					caretPadding := strings.Repeat(" ", col-1)
					caretLen := 1
					if spanEnd > spanStart && spanStart > 0 {
						caretLen = spanEnd - spanStart
					}
					carets := strings.Repeat("^", caretLen)
					b.WriteString(fmt.Sprintf("%s | %s%s\n", padding, caretPadding, carets))
				}
			}

			return b.String()
		}
	}

	// Fall back to SchemaError
	var schemaErr *astroladb.SchemaError
	if errors.As(err, &schemaErr) && schemaErr.File != "" {
		absPath := schemaErr.File
		if !filepath.IsAbs(absPath) {
			if abs, err := filepath.Abs(absPath); err == nil {
				absPath = abs
			}
		}

		// Check if it's a file (not a directory)
		if info, statErr := os.Stat(absPath); statErr == nil && !info.IsDir() {
			if schemaErr.Line > 0 {
				return fmt.Sprintf("%s:%d:1: %s", absPath, schemaErr.Line, schemaErr.Message)
			}
			return fmt.Sprintf("%s:1:1: %s", absPath, schemaErr.Message)
		}

		// Directory - check if cause has better info
		if schemaErr.Cause != nil {
			return formatErrorWithURI(schemaErr.Cause)
		}

		return fmt.Sprintf("%s: %s", absPath, schemaErr.Message)
	}

	return err.Error()
}

// httpCmd starts a local HTTP server for API documentation.
func httpCmd() *cobra.Command {
	var port int
	var create bool

	cmd := &cobra.Command{
		Use:   "live",
		Short: "Start local server for live API documentation",
		Long: `Start local HTTP server with interactive API documentation and hot reload.

Endpoints: / (Swagger UI), /openapi.json, /graphiql, /graphql, /graphql/examples.
Watches schema files and auto-reloads browser on changes. Use --create to customize UI.`,
		Example: `  # Start server on default port 8080
  alab live

  # Start server on custom port
  alab live -p 3000

  # Create customizable HTML file for Swagger UI
  alab live --create`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create alab.html if requested
			if create {
				return createHTMLFile()
			}

			return startServer(port)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to listen on")
	cmd.Flags().BoolVar(&create, "create", false, "Create alab.html file for customization")

	setupCommandHelp(cmd)
	return cmd
}

// createHTMLFile creates the alab.html file in the current directory.
func createHTMLFile() error {
	filename := "alab.html"
	if _, err := os.Stat(filename); err == nil {
		return fmt.Errorf("%s already exists", filename)
	}

	if err := os.WriteFile(filename, []byte(mustReadTemplate(templateSwaggerHTML)), FilePerm); err != nil {
		return fmt.Errorf("failed to create %s: %w", filename, err)
	}

	ui.ShowSuccess(
		TitleHTMLFileCreated,
		fmt.Sprintf("Created %s\n%s",
			ui.FilePath(filename),
			ui.Help(HelpCustomizeSwagger),
		),
	)
	return nil
}

// startServer starts the HTTP server.
func startServer(port int) error {
	// Start file watcher for hot reload
	go watchSchemas()

	// SSE endpoint for hot reload
	http.HandleFunc("/_reload", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(HeaderContentType, "text/event-stream")
		w.Header().Set(HeaderCacheControl, "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set(HeaderCORS, CORSAllowAll)

		ch := make(chan struct{}, 1)
		sseClientsMu.Lock()
		sseClients[ch] = struct{}{}
		sseClientsMu.Unlock()

		defer func() {
			sseClientsMu.Lock()
			delete(sseClients, ch)
			sseClientsMu.Unlock()
		}()

		flusher, _ := w.(http.Flusher)
		for {
			select {
			case <-ch:
				fmt.Fprintf(w, "data: reload\n\n")
				if flusher != nil {
					flusher.Flush()
				}
			case <-r.Context().Done():
				return
			}
		}
	})

	// Serve OpenAPI spec (regenerated on each request)
	http.HandleFunc("/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		serveSchemaExport(w, "openapi", "/openapi.json", ContentTypeJSON)
	})

	// Serve GraphQL schema (regenerated on each request)
	http.HandleFunc("/graphql", func(w http.ResponseWriter, r *http.Request) {
		serveSchemaExport(w, "graphql", "/graphql", "text/plain; charset=utf-8")
	})

	// Serve GraphQL examples (for mock responses)
	http.HandleFunc("/graphql/examples", func(w http.ResponseWriter, r *http.Request) {
		serveSchemaExport(w, "graphql-examples", "/graphql/examples", ContentTypeJSON)
	})

	// Serve GraphQL schema viewer
	http.HandleFunc("/graphiql", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(HeaderContentType, ContentTypeHTML)
		w.Write([]byte(mustReadTemplate(templateGraphQLHTML)))
	})

	// Serve logo
	http.HandleFunc("/logo.png", func(w http.ResponseWriter, r *http.Request) {
		data, err := templates.ReadFile("templates/logo.png")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set(HeaderContentType, "image/png")
		w.Header().Set(HeaderCacheControl, "public, max-age=86400")
		w.Write(data)
	})

	// Serve Swagger UI
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		html := getHTML()
		w.Header().Set(HeaderContentType, ContentTypeHTML)
		w.Write([]byte(html))
	})

	addr := fmt.Sprintf(":%d", port)
	baseURL := fmt.Sprintf("http://localhost%s", addr)

	// Show server info panel
	fmt.Println(ui.RenderTitle(TitleAPIDocServer))
	fmt.Println()

	list := ui.NewList()
	list.AddInfo(fmt.Sprintf("Swagger UI:  %s/", ui.Primary(baseURL)))
	list.AddInfo(fmt.Sprintf("OpenAPI:     %s/openapi.json", ui.Primary(baseURL)))
	list.AddInfo(fmt.Sprintf("GraphiQL:    %s/graphiql", ui.Primary(baseURL)))
	list.AddInfo(fmt.Sprintf("GraphQL:     %s/graphql", ui.Primary(baseURL)))

	fmt.Println(list.String())
	fmt.Println()
	fmt.Println(ui.Help("Press Ctrl+C to stop"))
	fmt.Println()

	return http.ListenAndServe(addr, nil)
}

// getHTML returns the HTML content, preferring local alab.html if it exists.
func getHTML() string {
	// Check for local alab.html
	localFile := filepath.Join(".", "alab.html")
	if data, err := os.ReadFile(localFile); err == nil {
		return string(data)
	}

	return mustReadTemplate(templateSwaggerHTML)
}

// watchSchemas watches the schemas directory for changes and notifies SSE clients.
func watchSchemas() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Printf("Warning: file watcher failed: %v\n", err)
		return
	}
	defer watcher.Close()

	// Watch schemas directory recursively
	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Warning: could not load config: %v\n", err)
		return
	}
	watchDir := cfg.SchemasDir

	_ = filepath.Walk(watchDir, func(path string, info os.FileInfo, err error) error { //nolint:errcheck
		if err != nil {
			return nil
		}
		if info.IsDir() {
			watcher.Add(path)
		}
		return nil
	})

	fmt.Printf("  %s Watching %s %s\n",
		ui.Success(""),
		ui.Primary(watchDir),
		ui.Dim("(hot reload enabled)"))

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove) != 0 {
				// Clear error cache so errors show again on reload
				shownErrorsMu.Lock()
				shownErrors = make(map[string]struct{})
				shownErrorsMu.Unlock()

				// Notify all SSE clients
				sseClientsMu.Lock()
				for ch := range sseClients {
					select {
					case ch <- struct{}{}:
					default:
					}
				}
				sseClientsMu.Unlock()
			}
		case _, ok := <-watcher.Errors:
			if !ok {
				return
			}
		}
	}
}
