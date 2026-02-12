package runtime

import (
	"testing"

	"github.com/hlop3z/astroladb/internal/metadata"
)

func TestJunctionMarker(t *testing.T) {
	tests := []struct {
		name          string
		schema        string
		wantM2MCount  int
		wantSourceRef string
		wantTargetRef string
		wantSourceFK  string
		wantTargetFK  string
		expectError   bool
	}{
		{
			name: "explicit junction with 2 params",
			schema: `
export default table({
  post_id: col.belongs_to("blog.post"),
  tag_id: col.belongs_to("blog.tag"),
}).junction("blog.post", "blog.tag");
			`,
			wantM2MCount:  1,
			wantSourceRef: "blog.post",
			wantTargetRef: "blog.tag",
			wantSourceFK:  "post_id",
			wantTargetFK:  "tag_id",
		},
		{
			name: "auto-detect junction with 0 params",
			schema: `
export default table({
  user_id: col.belongs_to("auth.user"),
  role_id: col.belongs_to("auth.role"),
}).junction();
			`,
			wantM2MCount:  1,
			wantSourceRef: "auth.user",
			wantTargetRef: "auth.role",
			wantSourceFK:  "user_id",
			wantTargetFK:  "role_id",
		},
		{
			name: "junction with extra metadata columns",
			schema: `
export default table({
  post_id: col.belongs_to("blog.post"),
  tag_id: col.belongs_to("blog.tag"),
  position: col.integer(),
  created_at: col.datetime(),
}).junction("blog.post", "blog.tag");
			`,
			wantM2MCount:  1,
			wantSourceRef: "blog.post",
			wantTargetRef: "blog.tag",
			wantSourceFK:  "post_id",
			wantTargetFK:  "tag_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sb := NewSandbox(nil)

			// Parse the junction table schema
			_, err := sb.EvalSchema(tt.schema, "test", "junction_table")
			if tt.expectError {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify metadata was registered
			meta := sb.Metadata()
			if len(meta.ManyToMany) != tt.wantM2MCount {
				t.Fatalf("expected %d M2M relationships, got %d", tt.wantM2MCount, len(meta.ManyToMany))
			}

			if tt.wantM2MCount > 0 {
				m2m := meta.ManyToMany[0]

				// Note: Source and Target may be swapped due to alphabetical sorting
				// in AddExplicitJunction, so we check both orderings
				sourceMatches := m2m.Source == tt.wantSourceRef && m2m.Target == tt.wantTargetRef
				targetMatches := m2m.Source == tt.wantTargetRef && m2m.Target == tt.wantSourceRef

				if !sourceMatches && !targetMatches {
					t.Errorf("expected source/target %q/%q, got %q/%q",
						tt.wantSourceRef, tt.wantTargetRef, m2m.Source, m2m.Target)
				}

				// Check FK names (order may be swapped)
				fkMatches := (m2m.SourceFK == tt.wantSourceFK && m2m.TargetFK == tt.wantTargetFK) ||
					(m2m.SourceFK == tt.wantTargetFK && m2m.TargetFK == tt.wantSourceFK)

				if !fkMatches {
					t.Errorf("expected FKs %q/%q, got %q/%q",
						tt.wantSourceFK, tt.wantTargetFK, m2m.SourceFK, m2m.TargetFK)
				}

				// Verify join table name is not empty
				if m2m.JoinTable == "" {
					t.Errorf("join table name should not be empty")
				}
			}
		})
	}
}

func TestJunctionMetadataEquivalence(t *testing.T) {
	// Verify that magic .many_to_many() and explicit .junction()
	// produce identical metadata entries

	magicSchema := `
export default table({
  id: col.id(),
  title: col.string(200),
}).many_to_many("blog.tag");
	`

	explicitSchema := `
export default table({
  post_id: col.belongs_to("blog.post"),
  tag_id: col.belongs_to("blog.tag"),
}).junction("blog.post", "blog.tag");
	`

	// Parse magic many_to_many
	sb1 := NewSandbox(nil)
	_, err := sb1.EvalSchema(magicSchema, "blog", "post")
	if err != nil {
		t.Fatalf("magic schema error: %v", err)
	}
	magicMeta := sb1.Metadata()

	// Parse explicit junction
	sb2 := NewSandbox(nil)
	_, err = sb2.EvalSchema(explicitSchema, "blog", "posts_tags")
	if err != nil {
		t.Fatalf("explicit schema error: %v", err)
	}
	explicitMeta := sb2.Metadata()

	// Both should have exactly 1 M2M relationship
	if len(magicMeta.ManyToMany) != 1 {
		t.Fatalf("magic: expected 1 M2M, got %d", len(magicMeta.ManyToMany))
	}
	if len(explicitMeta.ManyToMany) != 1 {
		t.Fatalf("explicit: expected 1 M2M, got %d", len(explicitMeta.ManyToMany))
	}

	magic := magicMeta.ManyToMany[0]
	explicit := explicitMeta.ManyToMany[0]

	// Compare key fields (they should be structurally equivalent)
	compareM2M := func(a, b *metadata.ManyToManyMeta) bool {
		return a.Source == b.Source &&
			a.Target == b.Target &&
			a.JoinTable == b.JoinTable &&
			a.SourceFK == b.SourceFK &&
			a.TargetFK == b.TargetFK
	}

	if !compareM2M(magic, explicit) {
		t.Errorf("metadata mismatch:\nmagic:    %+v\nexplicit: %+v", magic, explicit)
	}
}
