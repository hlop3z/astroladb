package main

// ============================================================================
// HTTP Constants (used 2-6 times - keep as simple constants)
// ============================================================================

const (
	HeaderContentType  = "Content-Type"
	HeaderCacheControl = "Cache-Control"
	HeaderCORS         = "Access-Control-Allow-Origin"
	ContentTypeJSON    = "application/json"
	ContentTypeHTML    = "text/html; charset=utf-8"
	CORSAllowAll       = "*"
)

// ============================================================================
// Flag Description Constants (used 2 times each)
// ============================================================================

const (
	FlagDescDryRun      = "Print SQL without executing"
	FlagDescCommit      = "Auto-commit migration files to git"
	FlagDescSkipLock    = "Skip distributed locking (use in CI)"
	FlagDescLockTimeout = "Lock acquisition timeout (default 30s)"
)

// ============================================================================
// File Permissions
// ============================================================================

const (
	DirPerm  = 0755 // rwxr-xr-x
	FilePerm = 0644 // rw-r--r--
)

// ============================================================================
// Database URL Display Configuration
// ============================================================================

const (
	DBURLMaskLength = 40 // Max characters shown before masking with "..."
)

// ============================================================================
// Structured Messages (grouped by command/feature)
// ============================================================================

// Msg is the central message hub for all user-facing text.
// Messages are organized by command/feature for easy discovery and maintenance.
//
// Usage:
//
//	ui.ShowSuccess(Msg.Migration.Apply.Title, ...)
//	if !confirm(Msg.Migration.Apply.Prompt) { ... }
var Msg = struct {
	Migration MigrationMessages
	Export    ExportMessages
	Init      InitMessages
	Table     TableMessages
	Types     TypesMessages
	Lock      LockMessages
	Server    ServerMessages
}{
	Migration: MigrationMessages{
		Apply: MigrationApplyMessages{
			Title:      "Migrations Applied Successfully",
			UpToDate:   "Migrations Up to Date",
			NoUpToDate: "No pending migrations to apply",
			Prompt:     "Apply %s?",
			Cancelled:  "Migration cancelled",
			Committed:  "Migration files committed to git",
		},
		Rollback: MigrationRollbackMessages{
			Title:     "Rollback Migrations",
			Complete:  "Rollback Complete",
			Prompt:    "Rollback %s?",
			Cancelled: "Rollback cancelled",
			DryHelp:   "Use --dry to preview SQL without executing",
		},
		Destructive: MigrationDestructiveMessages{
			Title:       "Destructive Operations Detected",
			Warning:     "These operations will permanently delete data",
			ConfirmHelp: "Run with --confirm-destroy to proceed",
		},
	},
	Export: ExportMessages{
		Complete: "Export Complete",
	},
	Init: InitMessages{
		Complete: "Project Initialized",
	},
	Table: TableMessages{
		Created: "Schema File Created",
	},
	Types: TypesMessages{
		Regenerated: "Type Definitions Updated",
	},
	Lock: LockMessages{
		Status:          "Migration Lock Status",
		Available:       "Lock Available",
		Held:            "Lock Held",
		Released:        "Lock Released",
		NoLockToRelease: "No Lock to Release",
		WillBeReleased:  "Lock Will Be Released",
		NoRunning:       "No migration is currently running",
		NotHeld:         "Migration lock is not currently held",
		ForcedRelease:   "Migration lock has been forcefully released",
		ReleaseTitle:    "Release Migration Lock",
		ReleasePrompt:   "Release this lock?",
		ReleaseHelp:     "Only release if you're certain no migration is running",
	},
	Server: ServerMessages{
		Title:         "API Documentation Server",
		HTMLCreated:   "HTML File Created",
		CustomizeHelp: "Customize this file to change the Swagger UI appearance",
	},
}

// ============================================================================
// Message Type Definitions
// ============================================================================

type MigrationMessages struct {
	Apply       MigrationApplyMessages
	Rollback    MigrationRollbackMessages
	Destructive MigrationDestructiveMessages
}

type MigrationApplyMessages struct {
	Title      string
	UpToDate   string
	NoUpToDate string
	Prompt     string
	Cancelled  string
	Committed  string
}

type MigrationRollbackMessages struct {
	Title     string
	Complete  string
	Prompt    string
	Cancelled string
	DryHelp   string
}

type MigrationDestructiveMessages struct {
	Title       string
	Warning     string
	ConfirmHelp string
}

type ExportMessages struct {
	Complete string
}

type InitMessages struct {
	Complete string
}

type TableMessages struct {
	Created string
}

type TypesMessages struct {
	Regenerated string
}

type LockMessages struct {
	Status          string
	Available       string
	Held            string
	Released        string
	NoLockToRelease string
	WillBeReleased  string
	NoRunning       string
	NotHeld         string
	ForcedRelease   string
	ReleaseTitle    string
	ReleasePrompt   string
	ReleaseHelp     string
}

type ServerMessages struct {
	Title         string
	HTMLCreated   string
	CustomizeHelp string
}

// ============================================================================
// CLI Branding & Help
// ============================================================================

const (
	MainTitle   = "⏳ Astroladb (alab)"
	MainSummary = "★  One schema: many outputs"
)

// ============================================================================
// Helper Functions
// ============================================================================

// MaskDatabaseURL truncates a database URL for display.
// If the URL is longer than DBURLMaskLength, it is truncated with "...".
func MaskDatabaseURL(url string) string {
	if len(url) > DBURLMaskLength {
		return url[:DBURLMaskLength] + "..."
	}
	return url
}

// ============================================================================
// Export Configuration
// ============================================================================

// ExportFilenames maps export format names to their output filenames.
var ExportFilenames = map[string]string{
	"openapi":    "openapi.json",
	"graphql":    "schema.graphql",
	"gql":        "schema.graphql",
	"typescript": "types.ts",
	"ts":         "types.ts",
	"go":         "types.go",
	"golang":     "types.go",
	"python":     "types.py",
	"py":         "types.py",
	"rust":       "types.rs",
	"rs":         "types.rs",
}

// AllExportFormats lists all supported export formats.
var AllExportFormats = []string{"openapi", "graphql", "typescript", "go", "python", "rust"}

// GetExportFilename returns the appropriate filename for an export format.
// If the format is not recognized, it returns format + ".json".
func GetExportFilename(format string) string {
	if filename, ok := ExportFilenames[format]; ok {
		return filename
	}
	return format + ".json"
}
