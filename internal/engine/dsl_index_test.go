//go:build integration

package engine_test

import (
	"testing"

	"github.com/hlop3z/astroladb/internal/dsl"
	"github.com/hlop3z/astroladb/internal/testutil"
)

// ---------------------------------------------------------------------------
// CreateIndex
// ---------------------------------------------------------------------------

func TestDSL_CreateIndex(t *testing.T) {
	for _, d := range setupDialects(t) {
		t.Run(d.name, func(t *testing.T) {
			ops := buildOps(func(m *dsl.MigrationBuilder) {
				m.CreateTable("test.idx_tbl", func(tb *dsl.TableBuilder) {
					addID(tb)
					tb.String("email", 255)
					tb.String("name", 100)
				})
				m.CreateIndex("test.idx_tbl", []string{"email"}, dsl.IndexName("idx_email"))
			})
			execOps(t, d.db, d.dialect, ops)

			testutil.AssertIndexExists(t, d.db, "test_idx_tbl", "idx_email")
		})
	}
}

func TestDSL_CreateUniqueIndex(t *testing.T) {
	for _, d := range setupDialects(t) {
		t.Run(d.name, func(t *testing.T) {
			ops := buildOps(func(m *dsl.MigrationBuilder) {
				m.CreateTable("test.uidx_tbl", func(tb *dsl.TableBuilder) {
					addID(tb)
					tb.String("code", 50)
					tb.String("region", 10)
				})
				m.CreateIndex("test.uidx_tbl", []string{"code", "region"},
					dsl.IndexName("idx_code_region_unique"),
					dsl.IndexUnique(),
				)
			})
			execOps(t, d.db, d.dialect, ops)

			testutil.AssertIndexExists(t, d.db, "test_uidx_tbl", "idx_code_region_unique")
		})
	}
}

// ---------------------------------------------------------------------------
// DropIndex
// ---------------------------------------------------------------------------

func TestDSL_DropIndex(t *testing.T) {
	for _, d := range setupDialects(t) {
		t.Run(d.name, func(t *testing.T) {
			create := buildOps(func(m *dsl.MigrationBuilder) {
				m.CreateTable("test.didx_tbl", func(tb *dsl.TableBuilder) {
					addID(tb)
					tb.String("val", 100)
				})
				m.CreateIndex("test.didx_tbl", []string{"val"}, dsl.IndexName("idx_drop_me"))
			})
			execOps(t, d.db, d.dialect, create)
			testutil.AssertIndexExists(t, d.db, "test_didx_tbl", "idx_drop_me")

			drop := buildOps(func(m *dsl.MigrationBuilder) {
				m.DropIndex("idx_drop_me")
			})
			execOps(t, d.db, d.dialect, drop)
		})
	}
}
