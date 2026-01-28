// Migration: add_user_fields
//
// Evolves the auth.user table: adds phone column, renames a column,
// drops an unused column from blog.tag, and adds new indexes.

export default migration({
  up(m) {
    // Add a phone column to auth.user
    m.add_column("auth.user", (t) => {
      t.string("phone", 20).optional();
    });

    // Add a NOT NULL column with backfill for existing rows
    m.add_column("auth.user", (t) => {
      t.string("locale", 10).default("en").backfill("en");
    });

    // Index the new phone column
    m.create_index("auth.user", ["phone"], {
      name: "idx_user_phone",
    });

    // Rename a column in blog.tag
    m.rename_column("blog.tag", "slug", "url_slug");

    // Drop an unused column
    m.drop_column("blog.post", "excerpt");
  },

  down(m) {
    // Reverse in opposite order
    m.add_column("blog.post", (t) => {
      t.text("excerpt").optional();
    });

    m.rename_column("blog.tag", "url_slug", "slug");

    m.drop_index("idx_user_phone");

    m.drop_column("auth.user", "locale");
    m.drop_column("auth.user", "phone");
  },
});
