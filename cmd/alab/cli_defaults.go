package main

// File permissions for consistent file/directory creation across the CLI.
const (
	// DirPerm is the permission mode for created directories (rwxr-xr-x).
	DirPerm = 0755

	// FilePerm is the permission mode for created files (rw-r--r--).
	FilePerm = 0644
)

// DB URL display configuration.
const (
	// DBURLMaskLength is the max characters shown before masking with "...".
	DBURLMaskLength = 40
)

const (
	MainTitle   = "⏳ Astroladb (alab)"
	MainSummary = "★  One schema: many outputs"
)

// Default directory and file names.
const (
	// DefaultExportsDir is the default directory for export output.
	DefaultExportsDir = "exports"

	// DefaultTypesDir is the default directory for TypeScript type definitions.
	DefaultTypesDir = "types"

	// DefaultConfigFile is the default configuration filename.
	DefaultConfigFile = "alab.yaml"
)

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

// Messages for consistent CLI output.
const (
	// MsgMigrationCancelled is shown when user cancels a migration.
	MsgMigrationCancelled = "Migration cancelled"

	// MsgRollbackCancelled is shown when user cancels a rollback.
	MsgRollbackCancelled = "Rollback cancelled"

	// MsgResetCancelled is shown when user cancels a reset.
	MsgResetCancelled = "Reset cancelled"

	// MsgNoUpToDate is shown when there are no pending migrations.
	MsgNoUpToDate = "No pending migrations to apply"

	// MsgMigrationsUpToDate is the success title when migrations are current.
	MsgMigrationsUpToDate = "Migrations Up to Date"

	// MsgCommittedToGit is shown after auto-commit succeeds.
	MsgCommittedToGit = "Migration files committed to git"
)

// Panel titles for success/warning messages.
const (
	// TitleApplyMigrations is the title for the migrate command.
	TitleApplyMigrations = "Apply Migrations"

	// TitleRollbackMigrations is the title for the rollback command.
	TitleRollbackMigrations = "Rollback Migrations"

	// TitleDestructiveOperation is the title for warning panels.
	TitleDestructiveOperation = "Destructive Operation"

	// TitleDestructiveOpsDetected is the title for lint warnings.
	TitleDestructiveOpsDetected = "Destructive Operations Detected"

	// TitleMigrationsApplied is the success title after migration.
	TitleMigrationsApplied = "Migrations Applied Successfully"

	// TitleRollbackComplete is the success title after rollback.
	TitleRollbackComplete = "Rollback Complete"

	// TitleDatabaseResetComplete is the success title after reset.
	TitleDatabaseResetComplete = "Database Reset Complete"

	// TitleExportComplete is the success title after export.
	TitleExportComplete = "Export Complete"

	// TitleProjectInitialized is the success title after init.
	TitleProjectInitialized = "Project Initialized"

	// TitleSchemaFileCreated is the success title after table creation.
	TitleSchemaFileCreated = "Schema File Created"

	// TitleTypesRegenerated is the success title after types regeneration.
	TitleTypesRegenerated = "Type Definitions Updated"

	// TitleMetaExported is the success title after meta export.
	TitleMetaExported = "Schema Metadata Exported"

	// TitleHTMLFileCreated is the success title after HTML file creation.
	TitleHTMLFileCreated = "HTML File Created"

	// TitleAPIDocServer is the title for the HTTP server.
	TitleAPIDocServer = "API Documentation Server"

	// TitleLockStatus is the title for lock status.
	TitleLockStatus = "Migration Lock Status"

	// TitleReleaseLock is the title for lock release.
	TitleReleaseLock = "Release Migration Lock"

	// TitleLockAvailable is shown when no lock is held.
	TitleLockAvailable = "Lock Available"

	// TitleLockHeld is shown when a lock is held.
	TitleLockHeld = "Lock Held"

	// TitleLockReleased is shown after lock release.
	TitleLockReleased = "Lock Released"

	// TitleNoLockToRelease is shown when no lock exists to release.
	TitleNoLockToRelease = "No Lock to Release"

	// TitleLockWillBeReleased is shown before releasing a lock.
	TitleLockWillBeReleased = "Lock Will Be Released"
)

// Confirmation prompts
const (
	PromptApplyMigrations = "Apply %s?"
	PromptRollback        = "Rollback %s?"
	PromptContinueReset   = "Continue with database reset?"
	PromptReleaseLock     = "Release this lock?"
	PromptIsRename        = "Is this a rename?"
)

// Help text patterns
const (
	HelpUseDryRun            = "Use --dry to preview SQL without executing"
	HelpUseForce             = "use --force to proceed anyway"
	HelpUseConfirmDestroy    = "Run with --confirm-destroy to proceed"
	HelpDevOnly              = "This command is intended for development only"
	HelpMakeBackup           = "Make sure you have a backup"
	HelpCannotUndo           = "This operation cannot be undone"
	HelpReleaseLockCmd       = "If this is a stuck lock, use: alab lock release"
	HelpOnlyReleaseIfSure    = "Only release if you're certain no migration is running"
	HelpCustomizeSwagger     = "Customize this file to change the Swagger UI appearance"
	HelpAutoGenerated        = "These files are auto-generated - do not edit manually"
	HelpMetaForExternalTools = "This file contains all schema metadata for external tools to generate code, ORMs, queries, etc."
)

// Warning messages
const (
	WarnDestructiveOps      = "These operations will permanently delete data"
	WarnDataPermanentDelete = "Delete ALL data permanently"
	WarnDropAllTables       = "Drop ALL tables in the database"
	WarnExecuteDownMigr     = "This will execute DOWN migrations"
	WarnRevertToRevision    = "Database state will revert to previous revision"
)

// Success messages for panels
const (
	MsgAllTablesDropped       = "All tables dropped and migrations re-applied"
	MsgNoMigrationRunning     = "No migration is currently running"
	MsgLockNotHeld            = "Migration lock is not currently held"
	MsgLockForcefullyReleased = "Migration lock has been forcefully released"
)

// HTTP header constants for consistent HTTP responses.
const (
	// HeaderContentType is the Content-Type header key.
	HeaderContentType = "Content-Type"

	// HeaderCacheControl is the Cache-Control header key.
	HeaderCacheControl = "Cache-Control"

	// HeaderCORS is the CORS header for cross-origin requests.
	HeaderCORS = "Access-Control-Allow-Origin"

	// ContentTypeJSON is the MIME type for JSON responses.
	ContentTypeJSON = "application/json"

	// ContentTypeHTML is the MIME type for HTML responses.
	ContentTypeHTML = "text/html; charset=utf-8"

	// CORSAllowAll allows all origins for CORS.
	CORSAllowAll = "*"
)

// Flag descriptions for consistent CLI flag help text.
const (
	// FlagDescDryRun is the description for --dry flag.
	FlagDescDryRun = "Print SQL without executing"

	// FlagDescForce is the description for --force flag (skip safety warnings).
	FlagDescForce = "Skip safety warnings"

	// FlagDescForceConfirm is the description for --force flag (skip confirmation).
	FlagDescForceConfirm = "Skip confirmation prompt"

	// FlagDescCommit is the description for --commit flag.
	FlagDescCommit = "Auto-commit migration files to git"

	// FlagDescSkipLock is the description for --skip-lock flag.
	FlagDescSkipLock = "Skip distributed locking (use in CI)"

	// FlagDescConfirmDestroy is the description for --confirm-destroy flag.
	FlagDescConfirmDestroy = "Confirm DROP operations"

	// FlagDescLockTimeout is the description for --lock-timeout flag.
	FlagDescLockTimeout = "Lock acquisition timeout (default 30s)"

	// FlagDescVerifySQL is the description for --verify-sql flag.
	FlagDescVerifySQL = "Verify SQL checksums before applying"

	// FlagDescJSON is the description for --json flag.
	FlagDescJSON = "Output as JSON for CI/CD"

	// FlagDescText is the description for --text flag.
	FlagDescText = "Output as plain text (non-interactive)"

	// FlagDescOverwrite is the description for --overwrite flag.
	FlagDescOverwrite = "Overwrite existing file"
)

// MaskDatabaseURL truncates a database URL for display.
// If the URL is longer than DBURLMaskLength, it is truncated with "...".
func MaskDatabaseURL(url string) string {
	if len(url) > DBURLMaskLength {
		return url[:DBURLMaskLength] + "..."
	}
	return url
}

// GetExportFilename returns the appropriate filename for an export format.
// If the format is not recognized, it returns format + ".json".
func GetExportFilename(format string) string {
	if filename, ok := ExportFilenames[format]; ok {
		return filename
	}
	return format + ".json"
}
