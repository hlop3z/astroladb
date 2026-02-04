package ui

// TView color tags - semantic naming for dynamic color formatting
const (
	// Text styles
	TagLabel   = "[yellow]" // For labels like "Revision:", "Status:"
	TagValue   = "[white]"  // For values/content
	TagSuccess = "[green]"  // For success indicators
	TagError   = "[red]"    // For error indicators
	TagMuted   = "[gray]"   // For dimmed/secondary text
	TagInverse = "[black]"  // For inverted text (on light bg)
	TagReset   = "[-]"      // Reset foreground color
	TagEnd     = "[-:-]"    // Reset both foreground and background

	// Selection style
	TagSelected = "[black:white]" // Inverted: black text on white background
)

// Tab identifiers
const (
	TabBrowse  = "browse"
	TabHistory = "history"
	TabVerify  = "verify"
	TabDrift   = "drift"
)

// Tab labels with keyboard shortcuts (unified status view)
const (
	TabLabelBrowse  = "1 Browse"
	TabLabelHistory = "2 History"
	TabLabelVerify  = "3 Verify"
	TabLabelDrift   = "4 Drift"
)

// Panel titles (with padding for borders)
const (
	PanelDetails           = " Details "
	PanelTables            = " Tables "
	PanelColumns           = " Columns "
	PanelChainIntegrity    = " Chain Integrity "
	PanelLockFileIntegrity = " Lock File "
	PanelGitStatus         = " Git Status "
	PanelDriftDetails      = " Drift Details "
	PanelSQLPreview        = " SQL Preview "
	PanelConfirm           = " Confirm "
	PanelInput             = " Input "
	PanelSelect            = " Select "
)

// View headers
const (
	HeaderStatus = "Status"
)

// Status bar hints
const (
	HintsStatus = " q quit  1/2/3/4 tabs  h/l panels  j/k navigate  g/G top/bottom "
	HintsSelect = "↑↓ navigate  Enter select  Esc cancel"
)

// Migration status
const (
	StatusApplied  = "applied"
	StatusPending  = "pending"
	StatusMissing  = "missing"
	StatusModified = "modified"
)

// Status icons for migration list
const (
	IconApplied = "✓"
	IconPending = "○"
	IconIssue   = "✗"
)

// Unicode status indicators
const (
	SymbolCheck  = "✓"
	SymbolCross  = "✗"
	SymbolCircle = "○"
)

// Table column headers
const (
	ColRevision  = "Revision"
	ColName      = "Name"
	ColStatus    = "Status"
	ColApplied   = "Applied"
	ColAppliedAt = "Applied At"
	ColExecTime  = "Exec Time"
	ColType      = "Type"
	ColNull      = "Null"
	ColKey       = "Key"
	ColObject    = "Object"
	ColOperation = "Operation"
	ColTable     = "Table"
	ColDetails   = "Details"
)

// Drift types
const (
	DriftMissing  = "missing"
	DriftExtra    = "extra"
	DriftModified = "modified"
)

// Drift type icons
const (
	DriftIconMissing  = "-"
	DriftIconExtra    = "+"
	DriftIconModified = "~"
)

// Drift detail messages
const (
	DriftMsgTableMissing   = "Table exists in migrations but not in database"
	DriftMsgTableExtra     = "Table exists in database but not in migrations"
	DriftMsgColumnMissing  = "Column exists in migrations but not in database"
	DriftMsgColumnExtra    = "Column exists in database but not in migrations"
	DriftMsgColumnModified = "Column definition differs between migrations and database"
	DriftMsgIndexMissing   = "Index exists in migrations but not in database"
	DriftMsgIndexExtra     = "Index exists in database but not in migrations"
	DriftMsgNoDrift        = "No drift detected"
	DriftMsgSchemaMatches  = "Schema matches database state."
)

// Internal table names (used for filtering)
var InternalTables = []string{
	"alab.migrations",
	"alab.migrations_lock",
	"alab_migrations",
	"alab_migrations_lock",
}

// Empty state messages
const (
	MsgNoTables          = "No tables defined"
	MsgNoMigrationsYet   = "No migrations applied yet"
	MsgNoDriftDetected   = "(none detected)"
	MsgNone              = "(none)"
	MsgNotGitRepo        = "Not a git repository"
	MsgAllCommitted      = "All migrations committed"
	MsgAllChecksumsValid = "All checksums valid"
	MsgRunNewMigration   = "Run 'alab new <name>' to create your first migration"
	MsgRunMigrate        = "Run 'alab migrate' to apply pending migrations"
	MsgRunGitAdd         = "Run 'git add' to stage changes"
	MsgGitVersionControl = "Migration files should be version controlled for team collaboration."
)

// Section titles for text output
const (
	TitleMigrationHistory     = "Migration History"
	TitleMigrationVerify      = "Migration Verification"
	TitleVerification         = "Verification"
	TitleSchemaDrift          = "Schema Drift"
	TitleMigrationStatus      = "Migration Status"
	TitleMigrationChainStatus = "Migration Chain Status"
	TitleLockFileStatus       = "Lock File Status"
)

// Common format strings
const (
	FmtLabelValue = TagLabel + "%s:" + TagValue + " %s\n" // "[yellow]Label:[white] value\n"
)

// Detail panel labels
const (
	LabelRevision    = "Revision:"
	LabelName        = "Name:"
	LabelStatus      = "Status:"
	LabelApplied     = "Applied:"
	LabelPending     = "Pending:"
	LabelIssues      = "Issues:"
	LabelDuration    = "Duration:"
	LabelChecksum    = "Checksum:"
	LabelChecksums   = "Checksums"
	LabelDescription = "Description:"
	LabelFile        = "File:"
	LabelBranch      = "Branch:"
	LabelLock        = "Lock:"
	LabelType        = "Type:"
	LabelObject      = "Object:"
	LabelDetails     = "Details:"
	LabelIndexes     = "Indexes:"
	LabelForeignKeys = "Foreign Keys:"
	LabelSQLPreview  = "SQL Preview:"
	LabelUncommitted = "Uncommitted files:"
	LabelOnDelete    = "ON DELETE:"
	LabelOnUpdate    = "ON UPDATE:"
)

// Column header arrays for tables
var (
	ColsHistory = []string{ColRevision, ColName, ColAppliedAt, ColExecTime}
	ColsDrift   = []string{ColType, ColObject, ColDetails}
	ColsColumns = []string{ColName, ColType, ColNull, ColKey}
)

// Key indicators for column tables
const (
	KeyYes    = "YES"
	KeyNo     = "NO"
	KeyPK     = "PK"
	KeyUQ     = "UQ"
	KeyUnique = "(UNIQUE)"
)

// Drift help text for detail panel
const (
	DriftHelpMissing  = "This object exists in migrations but not in the database.\nRun 'alab migrate' to apply pending migrations."
	DriftHelpExtra    = "This object exists in the database but not in migrations.\nRun 'alab new' to create a migration that handles this."
	DriftHelpModified = "This object differs between migrations and database.\nReview the differences and create a new migration if needed."
)

// Header format strings
const (
	FmtSchemaAtRev = "Schema at revision %s"
	FmtIssuesFound = "%d issues found"
	FmtTotalMigr   = " Total: %d migrations (%s)"
	FmtLockHeldBy  = "Held by %s"
)
