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
	"github.com/hlop3z/astroladb/internal/alerr"
	"github.com/hlop3z/astroladb/internal/ui"
	"github.com/hlop3z/astroladb/pkg/astroladb"
	"github.com/spf13/cobra"
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

// defaultSwaggerHTML is the embedded Swagger UI HTML template.
const defaultSwaggerHTML = `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <title>Alab API Documentation</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css" />
    <style>
      body { margin: 0; padding: 0; }
      .swagger-ui .topbar { display: none; }
      .copy-btn {
        position: absolute;
        top: 5px;
        right: 5px;
        padding: 4px 8px;
        font-size: 11px;
        cursor: pointer;
        background: #4990e2;
        color: white;
        border: none;
        border-radius: 3px;
        opacity: 0.8;
      }
      .copy-btn:hover { opacity: 1; }
      .copy-btn.copied { background: #49cc90; }
      .swagger-ui .example-value-wrapper,
      .swagger-ui .highlight-code { position: relative; }
    </style>
  </head>
  <body>
    <div id="swagger-ui"></div>
    <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-standalone-preset.js"></script>
    <script>
      window.onload = () => {
        SwaggerUIBundle({
          url: "/openapi.json",
          dom_id: "#swagger-ui",
          presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
          layout: "StandaloneLayout",
          deepLinking: true,
          defaultModelsExpandDepth: 1,
          defaultModelExpandDepth: 1,
          tryItOutEnabled: false,
          showExtensions: true,
          showCommonExtensions: true,
        });

        // Add copy buttons to code blocks
        const addCopyButtons = () => {
          document.querySelectorAll('.highlight-code, .example-value-wrapper').forEach(block => {
            if (block.querySelector('.copy-btn')) return;
            const btn = document.createElement('button');
            btn.className = 'copy-btn';
            btn.textContent = 'Copy';
            btn.onclick = () => {
              const code = block.querySelector('code, pre')?.textContent || block.textContent;
              navigator.clipboard.writeText(code).then(() => {
                btn.textContent = 'Copied!';
                btn.classList.add('copied');
                setTimeout(() => { btn.textContent = 'Copy'; btn.classList.remove('copied'); }, 1500);
              });
            };
            block.style.position = 'relative';
            block.appendChild(btn);
          });
        };

        // Run periodically to catch dynamically loaded content
        setInterval(addCopyButtons, 1000);

        // Hot reload: listen for schema changes
        const evtSource = new EventSource("/_reload");
        evtSource.onmessage = () => location.reload();
      };
    </script>
  </body>
</html>
`

// defaultGraphQLHTML is the embedded GraphiQL HTML template with SDL support.
const defaultGraphQLHTML = `<!DOCTYPE html>
<html>
  <head>
    <meta charset="utf-8" />
    <title>Alab GraphQL Explorer</title>
    <link rel="stylesheet" href="https://unpkg.com/graphiql@3/graphiql.min.css" />
    <script type="importmap">
      { "imports": { "graphql": "https://esm.sh/graphql@16?bundle" } }
    </script>
  </head>
  <body style="margin:0;">
    <div id="graphiql" style="height:100vh;"></div>
    <script src="https://unpkg.com/react@18/umd/react.production.min.js" crossorigin></script>
    <script src="https://unpkg.com/react-dom@18/umd/react-dom.production.min.js" crossorigin></script>
    <script src="https://unpkg.com/graphiql@3/graphiql.min.js" crossorigin></script>
    <script type="module">
      import { buildSchema, graphql } from "graphql";

      // Load SDL and examples
      const [sdl, examples] = await Promise.all([
        fetch("/graphql").then(r => r.text()),
        fetch("/graphql/examples").then(r => r.json())
      ]);
      const schema = buildSchema(sdl);

      // Create root resolver that returns examples
      const rootValue = {};
      for (const [key, value] of Object.entries(examples)) {
        rootValue[key] = () => value;
      }

      const fetcher = async (params) => {
        return graphql({ schema, source: params.query, variableValues: params.variables, rootValue });
      };

      const root = ReactDOM.createRoot(document.getElementById("graphiql"));
      // Hot reload: listen for schema changes
      const evtSource = new EventSource("/_reload");
      evtSource.onmessage = () => location.reload();

      root.render(React.createElement(GraphiQL, {
        fetcher,
        defaultQuery: ` + "`" + `# Alab GraphQL Schema Explorer
#
# This is a local mock server with example data.
# Use the Docs panel (top-right) to browse types.
#
# Try running this query:
query {
  authUser(id: "1") {
    id
    email
    username
  }
}
` + "`" + `,
      }));
    </script>
  </body>
</html>
`

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
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
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
		Use:   "http",
		Short: "Start local server for live API documentation",
		Long: `Starts a local HTTP server that serves your API documentation.

The server provides:
  /             - Swagger UI interface
  /openapi.json - Live OpenAPI spec (regenerated on each request)
  /graphiql     - GraphiQL interface
  /graphql      - GraphQL schema

Changes to your schema files are reflected immediately - just refresh the page.`,
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

	return cmd
}

// createHTMLFile creates the alab.html file in the current directory.
func createHTMLFile() error {
	filename := "alab.html"
	if _, err := os.Stat(filename); err == nil {
		return fmt.Errorf("%s already exists", filename)
	}

	if err := os.WriteFile(filename, []byte(defaultSwaggerHTML), 0644); err != nil {
		return fmt.Errorf("failed to create %s: %w", filename, err)
	}

	fmt.Printf("Created %s - customize as needed\n", filename)
	return nil
}

// startServer starts the HTTP server.
func startServer(port int) error {
	// Start file watcher for hot reload
	go watchSchemas()

	// SSE endpoint for hot reload
	http.HandleFunc("/_reload", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

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
		freshClient, err := newSchemaOnlyClient()
		if err != nil {
			handleSchemaError(w, err, "/openapi.json")
			return
		}
		defer freshClient.Close()

		data, err := freshClient.SchemaExport("openapi")
		if err != nil {
			handleSchemaError(w, err, "/openapi.json")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Write(data)
	})

	// Serve GraphQL schema (regenerated on each request)
	http.HandleFunc("/graphql", func(w http.ResponseWriter, r *http.Request) {
		freshClient, err := newSchemaOnlyClient()
		if err != nil {
			handleSchemaError(w, err, "/graphql")
			return
		}
		defer freshClient.Close()

		data, err := freshClient.SchemaExport("graphql")
		if err != nil {
			handleSchemaError(w, err, "/graphql")
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Write(data)
	})

	// Serve GraphQL examples (for mock responses)
	http.HandleFunc("/graphql/examples", func(w http.ResponseWriter, r *http.Request) {
		freshClient, err := newSchemaOnlyClient()
		if err != nil {
			handleSchemaError(w, err, "/graphql/examples")
			return
		}
		defer freshClient.Close()

		data, err := freshClient.SchemaExport("graphql-examples")
		if err != nil {
			handleSchemaError(w, err, "/graphql/examples")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Write(data)
	})

	// Serve GraphQL schema viewer
	http.HandleFunc("/graphiql", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(defaultGraphQLHTML))
	})

	// Serve Swagger UI
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		html := getHTML()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(html))
	})

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("Starting API documentation server at http://localhost%s\n", addr)
	fmt.Printf("Press Ctrl+C to stop\n\n")
	fmt.Printf("  Swagger:  http://localhost%s/\n", addr)
	fmt.Printf("  OpenAPI:  http://localhost%s/openapi.json\n", addr)
	fmt.Printf("  GraphiQL: http://localhost%s/graphiql\n", addr)

	return http.ListenAndServe(addr, nil)
}

// getHTML returns the HTML content, preferring local alab.html if it exists.
func getHTML() string {
	// Check for local alab.html
	localFile := filepath.Join(".", "alab.html")
	if data, err := os.ReadFile(localFile); err == nil {
		return string(data)
	}

	return defaultSwaggerHTML
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

	filepath.Walk(watchDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			watcher.Add(path)
		}
		return nil
	})

	fmt.Printf("  Watching: %s (hot reload enabled)\n", watchDir)

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
