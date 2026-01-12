package testutil

import "fmt"

// MigrationBuilder helps build test migrations with consistent formatting.
type MigrationBuilder struct {
	upBody   string
	downBody string
}

// NewMigration creates a new migration builder.
func NewMigration() *MigrationBuilder {
	return &MigrationBuilder{}
}

// Up sets the up() function body.
func (m *MigrationBuilder) Up(body string) *MigrationBuilder {
	m.upBody = body
	return m
}

// Down sets the down() function body.
func (m *MigrationBuilder) Down(body string) *MigrationBuilder {
	m.downBody = body
	return m
}

// Build returns the complete migration file content.
func (m *MigrationBuilder) Build() string {
	return fmt.Sprintf(`export default migration({
  up(m) {
%s
  },

  down(m) {
%s
  }
})
`, m.upBody, m.downBody)
}

// SimpleMigration creates a migration with just the up/down bodies.
// Bodies should be pre-indented (4 spaces).
func SimpleMigration(upBody, downBody string) string {
	return NewMigration().Up(upBody).Down(downBody).Build()
}

// CreateTable is a helper for creating a single table migration.
func CreateTable(namespace, table, columns string) string {
	up := fmt.Sprintf(`    m.create_table("%s.%s", t => {
%s
    })`, namespace, table, columns)

	down := fmt.Sprintf(`    m.drop_table("%s.%s")`, namespace, table)

	return SimpleMigration(up, down)
}
