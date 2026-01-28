// Migration: rename_and_drop_table
//
// Demonstrates table-level operations: rename_table and drop_table.

export default migration({
  up(m) {
    // Rename blog.tag → blog.label
    m.rename_table("blog.tag", "label");

    // Create a temporary table, then drop it
    m.create_table("temp.scratch", (t) => {
      t.id();
      t.string("notes", 500).optional();
      t.timestamps();
    });
    m.drop_table("temp.scratch");
  },

  down(m) {
    // Recreate the dropped table
    m.create_table("temp.scratch", (t) => {
      t.id();
      t.string("notes", 500).optional();
      t.timestamps();
    });
    m.drop_table("temp.scratch");

    // Reverse the rename
    m.rename_table("blog.label", "tag");
  },
});
