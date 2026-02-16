/**
 * Example migration demonstrating .backfill() modifier
 *
 * This migration adds a "bio" column to the users table with a default
 * backfill value for existing rows. The backfill is executed in batches
 * of 5,000 rows with 1-second sleep between batches to prevent table locks.
 *
 * Execution phases:
 * 1. DDL: ADD COLUMN bio TEXT
 * 2. Data: Batched UPDATE to backfill existing rows
 */

export default (m) => {
  // Add bio column with backfill for existing rows
  m.add_column("auth.user", (col) =>
    col.string(500, "bio")
       .backfill("No bio provided")  // Backfill existing rows
  );

  // Add optional phone number (no backfill needed for nullable columns)
  m.add_column("auth.user", (col) =>
    col.string(20, "phone")
       .nullable()
  );

  // Add notification preference with boolean backfill
  m.add_column("auth.user", (col) =>
    col.flag("notifications_enabled")
       .backfill(true)  // Default to enabled for existing users
  );

  // Add timestamp with SQL expression backfill
  m.add_column("auth.user", (col) =>
    col.timestamp("last_active")
       .backfill(sql("NOW()"))  // Set to current time for existing users
  );
};

/**
 * Generated SQL (PostgreSQL):
 *
 * Phase 1 (DDL):
 * BEGIN;
 *   SET lock_timeout = '2s';
 *   SET statement_timeout = '30s';
 *
 *   ALTER TABLE auth_user ADD COLUMN bio TEXT;
 *   ALTER TABLE auth_user ADD COLUMN phone TEXT;
 *   ALTER TABLE auth_user ADD COLUMN notifications_enabled BOOLEAN NOT NULL;
 *   ALTER TABLE auth_user ADD COLUMN last_active TIMESTAMP NOT NULL;
 * COMMIT;
 *
 * Phase 2 (Data - Batched):
 * BEGIN;
 *   SET lock_timeout = '2s';
 *   SET statement_timeout = '30s';
 *
 *   -- Backfill bio column
 *   DO $$
 *   DECLARE
 *     batch_size INT := 5000;
 *     processed INT := 0;
 *   BEGIN
 *     LOOP
 *       UPDATE auth_user
 *       SET bio = 'No bio provided'
 *       WHERE bio IS NULL AND id IN (
 *         SELECT id FROM auth_user WHERE bio IS NULL LIMIT batch_size
 *       );
 *
 *       EXIT WHEN NOT FOUND;
 *
 *       processed := processed + batch_size;
 *       RAISE NOTICE 'Progress: % rows', processed;
 *
 *       COMMIT;
 *       PERFORM pg_sleep(1);
 *     END LOOP;
 *   END $$;
 *
 *   -- Backfill notifications_enabled
 *   DO $$ ... END $$;
 *
 *   -- Backfill last_active with NOW()
 *   DO $$ ... END $$;
 * COMMIT;
 */
