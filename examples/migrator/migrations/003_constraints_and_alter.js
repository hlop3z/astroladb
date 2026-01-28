// Migration: constraints_and_alter
//
// Demonstrates advanced operations: alter_column, add_foreign_key,
// add_check, and raw SQL.
//
// NOTE: alter_column, foreign keys, and check constraints are
//       PostgreSQL / CockroachDB only — SQLite does not support them.

export default migration({
  up(m) {
    // Widen the phone column from string(20) to string(30)
    m.alter_column("auth.user", "phone", (c) => {
      c.set_type("string", 30);
    });

    // Make phone explicitly nullable (no-op if already nullable, but shows the API)
    m.alter_column("auth.user", "phone", (c) => {
      c.set_nullable();
    });

    // Set a default on blog.comment.is_approved, then drop it
    m.alter_column("blog.comment", "is_approved", (c) => {
      c.set_default(false);
    });
    m.alter_column("blog.comment", "is_approved", (c) => {
      c.drop_default();
    });

    // Explicit foreign key (manual, not from belongs_to)
    m.add_foreign_key("blog.comment", ["post_id"], "blog.post", ["id"], {
      name: "fk_comment_post",
      on_delete: "CASCADE",
      on_update: "CASCADE",
    });

    // Check constraints
    m.add_check("auth.user", "chk_email_length", "length(email) > 0");
    m.add_check("blog.post", "chk_title_length", "length(title) > 0");

    // Raw SQL — e.g. create a Postgres extension
    m.sql("SELECT 1");
  },

  down(m) {
    m.drop_check("blog.post", "chk_title_length");
    m.drop_check("auth.user", "chk_email_length");

    m.drop_foreign_key("blog.comment", "fk_comment_post");

    // alter_column reversals
    m.alter_column("blog.comment", "is_approved", (c) => {
      c.set_default(false);
    });

    m.alter_column("auth.user", "phone", (c) => {
      c.set_not_null();
    });
    m.alter_column("auth.user", "phone", (c) => {
      c.set_type("string", 20);
    });
  },
});
