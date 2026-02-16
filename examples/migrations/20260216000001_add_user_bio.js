// Example migration demonstrating multi-phase execution with dry-run estimation
// This migration will be split into 3 phases:
// - Phase 1 (DDL): Add bio column
// - Phase 2 (Index): Create UNIQUE index on bio
// - Phase 3 (Data): Backfill existing rows with default value

export default migration({
  description: "Add bio column to users with backfill",

  up(m) {
    // This single operation generates all 3 phases automatically:
    // 1. DDL: ALTER TABLE users ADD COLUMN bio TEXT
    // 2. Index: CREATE UNIQUE INDEX CONCURRENTLY idx_users_bio ON users(bio)
    // 3. Data: Batched UPDATE users SET bio = 'No bio provided' WHERE bio IS NULL
    m.add_column("auth.user", (col) =>
      col.text("bio")
        .unique()                    // Generates Phase 2 (Index)
        .backfill("No bio provided")  // Generates Phase 3 (Data)
    );
  },

  down(m) {
    m.drop_column("auth.user", "bio");
  }
});
