package astroladb

import (
	"strings"
	"testing"
	"time"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/engine"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{"less than 1 second", 500 * time.Millisecond, "<1s"},
		{"exactly 1 second", 1 * time.Second, "1s"},
		{"30 seconds", 30 * time.Second, "30s"},
		{"1 minute", 60 * time.Second, "1m"},
		{"1 minute 30 seconds", 90 * time.Second, "1m 30s"},
		{"5 minutes", 5 * time.Minute, "5m"},
		{"1 hour", 1 * time.Hour, "1h 0m"},
		{"1 hour 30 minutes", 90 * time.Minute, "1h 30m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDuration(tt.duration)
			if got != tt.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, got, tt.want)
			}
		})
	}
}

func TestFormatRowCount(t *testing.T) {
	tests := []struct {
		name  string
		count int64
		want  string
	}{
		{"zero", 0, "0"},
		{"small number", 42, "42"},
		{"hundreds", 999, "999"},
		{"thousands", 1000, "1,000"},
		{"ten thousand", 10000, "10,000"},
		{"hundred thousand", 100000, "100,000"},
		{"million", 1000000, "1,000,000"},
		{"large number", 1523441, "1,523,441"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatRowCount(tt.count)
			if got != tt.want {
				t.Errorf("formatRowCount(%d) = %q, want %q", tt.count, got, tt.want)
			}
		})
	}
}

func TestEstimatePhaseTime(t *testing.T) {
	c := &Client{} // Just need a client instance for the method

	tests := []struct {
		name      string
		phaseType engine.PhaseType
		rowCount  int64
		dialect   string
		wantParts []string // Parts that should be in the result
	}{
		{
			name:      "DDL phase is instant",
			phaseType: engine.DDL,
			rowCount:  0,
			dialect:   "postgres",
			wantParts: []string{"instant"},
		},
		{
			name:      "Index on empty table",
			phaseType: engine.Index,
			rowCount:  0,
			dialect:   "postgres",
			wantParts: []string{"<1s", "empty"},
		},
		{
			name:      "Index on small table",
			phaseType: engine.Index,
			rowCount:  50000,
			dialect:   "postgres",
			wantParts: []string{"10-30s", "server load"},
		},
		{
			name:      "Index on medium table",
			phaseType: engine.Index,
			rowCount:  500000,
			dialect:   "postgres",
			wantParts: []string{"1-3 minutes", "server load"},
		},
		{
			name:      "Index on large table",
			phaseType: engine.Index,
			rowCount:  5000000,
			dialect:   "postgres",
			wantParts: []string{"3-10 minutes", "server load"},
		},
		{
			name:      "Backfill on empty table",
			phaseType: engine.Data,
			rowCount:  0,
			dialect:   "postgres",
			wantParts: []string{"<1s", "no rows"},
		},
		{
			name:      "Backfill 10,000 rows (2 batches)",
			phaseType: engine.Data,
			rowCount:  10000,
			dialect:   "postgres",
			wantParts: []string{"4s"}, // 2 batches × 2s = 4s
		},
		{
			name:      "Backfill 1,000,000 rows (200 batches)",
			phaseType: engine.Data,
			rowCount:  1000000,
			dialect:   "postgres",
			wantParts: []string{"6m 40s"}, // 200 batches × 2s = 400s = 6m 40s
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			phase := engine.Phase{Type: tt.phaseType}
			got := c.estimatePhaseTime(phase, tt.rowCount, tt.dialect)

			for _, part := range tt.wantParts {
				if !strings.Contains(got, part) {
					t.Errorf("estimatePhaseTime() = %q, want to contain %q", got, part)
				}
			}
		})
	}
}

func TestExtractTableName(t *testing.T) {
	c := &Client{}

	tests := []struct {
		name  string
		phase engine.Phase
		want  string
	}{
		{
			name: "Empty phase",
			phase: engine.Phase{
				Type: engine.DDL,
				Ops:  []ast.Operation{},
			},
			want: "",
		},
		{
			name: "AddColumn operation",
			phase: engine.Phase{
				Type: engine.DDL,
				Ops: []ast.Operation{
					&ast.AddColumn{
						TableRef: ast.TableRef{Table_: "users"},
						Column:   &ast.ColumnDef{Name: "bio", Type: "string"},
					},
				},
			},
			want: "users",
		},
		{
			name: "CreateTable operation",
			phase: engine.Phase{
				Type: engine.DDL,
				Ops: []ast.Operation{
					&ast.CreateTable{
						TableOp: ast.TableOp{Name: "posts"},
					},
				},
			},
			want: "posts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.extractTableName(tt.phase)
			if got != tt.want {
				t.Errorf("extractTableName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCalculateMigrationTime(t *testing.T) {
	c := &Client{}

	tests := []struct {
		name   string
		phases []PhaseEstimate
		want   string
	}{
		{
			name:   "No phases",
			phases: []PhaseEstimate{},
			want:   "<1s (DDL only, instant)",
		},
		{
			name: "DDL only (instant)",
			phases: []PhaseEstimate{
				{Phase: engine.Phase{Type: engine.DDL}, RowCount: 0},
			},
			want: "<1s (DDL only, instant)",
		},
		{
			name: "Index only (variable)",
			phases: []PhaseEstimate{
				{Phase: engine.Phase{Type: engine.Index}, RowCount: 100000},
			},
			want: "depends on index creation time (variable)",
		},
		{
			name: "Backfill only (calculated)",
			phases: []PhaseEstimate{
				{Phase: engine.Phase{Type: engine.Data}, RowCount: 10000}, // 2 batches × 2s = 4s
			},
			want: "~4s",
		},
		{
			name: "Index + Backfill",
			phases: []PhaseEstimate{
				{Phase: engine.Phase{Type: engine.Index}, RowCount: 100000},
				{Phase: engine.Phase{Type: engine.Data}, RowCount: 10000}, // 2 batches × 2s = 4s
			},
			want: "~4s + index creation time (variable)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.calculateMigrationTime(tt.phases)
			if got != tt.want {
				t.Errorf("calculateMigrationTime() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetOperations(t *testing.T) {
	c := &Client{}

	upOps := []ast.Operation{
		&ast.CreateTable{TableOp: ast.TableOp{Name: "users"}},
	}
	downOps := []ast.Operation{
		&ast.DropTable{TableOp: ast.TableOp{Name: "users"}},
	}

	tests := []struct {
		name      string
		migration engine.Migration
		direction engine.Direction
		want      int // Number of operations expected
	}{
		{
			name: "UP direction",
			migration: engine.Migration{
				Operations: upOps,
			},
			direction: engine.Up,
			want:      1,
		},
		{
			name: "DOWN direction with DownOps",
			migration: engine.Migration{
				Operations: upOps,
				DownOps:    downOps,
			},
			direction: engine.Down,
			want:      1,
		},
		{
			name: "DOWN direction without DownOps",
			migration: engine.Migration{
				Operations: upOps,
			},
			direction: engine.Down,
			want:      0, // Returns nil when no DownOps
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.getOperations(tt.migration, tt.direction)
			if len(got) != tt.want {
				t.Errorf("getOperations() returned %d ops, want %d", len(got), tt.want)
			}
		})
	}
}

// TestDryRunEnhancedIntegration tests the full dry-run flow
func TestDryRunEnhancedIntegration(t *testing.T) {
	// This test would require a full Client setup with database connection
	// For now, we'll skip it and test the individual components
	t.Skip("Integration test requires full client setup - tested manually")
}
