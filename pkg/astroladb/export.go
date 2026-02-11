package astroladb

// Export format constants.
const (
	FormatOpenAPI    = "openapi"
	FormatTypeScript = "typescript"
	FormatGo         = "go"
	FormatPython     = "python"
	FormatRust       = "rust"
)

// Note: All export implementations have been moved to language-specific files:
// - export_converters.go: Base types, interfaces, and type converters
// - export_typescript.go: TypeScript export
// - export_golang.go: Go export
// - export_python.go: Python export
// - export_rust.go: Rust export
// - export_openapi.go: OpenAPI export
// - export_graphql.go: GraphQL export
// - export_helpers.go: Shared helper functions for WithRelations variants
//
// The routing logic is in schema.go's SchemaExport() method.
