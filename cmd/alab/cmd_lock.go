package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/hlop3z/astroladb/internal/lockfile"
	"github.com/hlop3z/astroladb/internal/ui"
)

// lockCmd manages migration locks.
func lockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lock",
		Short: "Manage migration locks",
		Long: `Manage distributed migration locks.

The migration system uses a lock table to prevent concurrent migrations.
Use these commands to check lock status or release stuck locks.`,
	}

	cmd.AddCommand(lockStatusCmd())
	cmd.AddCommand(lockReleaseCmd())
	cmd.AddCommand(lockVerifyCmd())
	cmd.AddCommand(lockRepairCmd())

	setupCommandHelp(cmd)
	return cmd
}

// lockStatusCmd shows the current lock status.
func lockStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show migration lock status",
		Long: `Show the current state of the migration lock.

Displays whether a lock is held, and if so, by which process and when it was acquired.
This is useful for debugging stuck migrations or verifying no other process is running.`,
		Example: `  # Check if migration lock is held
  alab lock status`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				if handleClientError(err) {
					os.Exit(1)
				}
				return err
			}
			defer client.Close()

			info, err := client.MigrationLockStatus()
			if err != nil {
				return err
			}

			fmt.Println(ui.RenderTitle(TitleLockStatus))
			fmt.Println()

			if !info.Locked {
				fmt.Println(ui.RenderSuccessPanel(
					TitleLockAvailable,
					MsgNoMigrationRunning,
				))
			} else {
				lockedAt := "unknown"
				if info.LockedAt != nil {
					lockedAt = info.LockedAt.Format(TimeFull)
				}

				list := ui.NewList()
				list.AddInfo(fmt.Sprintf("Locked by: %s", info.LockedBy))
				list.AddInfo(fmt.Sprintf("Locked at: %s", lockedAt))

				fmt.Println(ui.RenderWarningPanel(
					TitleLockHeld,
					list.String()+"\n"+
						ui.Help(HelpReleaseLockCmd),
				))
			}

			return nil
		},
	}

	setupCommandHelp(cmd)
	return cmd
}

// lockReleaseCmd forcefully releases a stuck lock.
func lockReleaseCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "release",
		Short: "Release a stuck migration lock",
		Long: `Forcefully release a stuck migration lock.

Use this command to release a lock that was not properly released due to
a process crash or other failure. Only use this when you are certain no
other migration is currently running.`,
		Example: `  # Release a stuck lock (with confirmation)
  alab lock release

  # Force release without confirmation
  alab lock release --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				if handleClientError(err) {
					os.Exit(1)
				}
				return err
			}
			defer client.Close()

			// Check current lock status first
			info, err := client.MigrationLockStatus()
			if err != nil {
				return err
			}

			if !info.Locked {
				fmt.Println(ui.RenderSuccessPanel(
					TitleNoLockToRelease,
					MsgLockNotHeld,
				))
				return nil
			}

			// Show warning
			fmt.Println(ui.RenderTitle(TitleReleaseLock))
			fmt.Println()

			lockedAt := "unknown"
			if info.LockedAt != nil {
				lockedAt = info.LockedAt.Format(TimeFull)
			}

			list := ui.NewList()
			list.AddInfo(fmt.Sprintf("Locked by: %s", info.LockedBy))
			list.AddInfo(fmt.Sprintf("Locked at: %s", lockedAt))

			fmt.Println(ui.RenderWarningPanel(
				TitleLockWillBeReleased,
				list.String()+"\n"+
					ui.Warning(HelpOnlyReleaseIfSure),
			))
			fmt.Println()

			// Confirm unless --force
			if !force {
				if !ui.Confirm(PromptReleaseLock, false) {
					fmt.Println(ui.Dim("Release cancelled"))
					return nil
				}
				fmt.Println()
			}

			if err := client.MigrationReleaseLock(); err != nil {
				return err
			}

			fmt.Println(ui.RenderSuccessPanel(
				TitleLockReleased,
				MsgLockForcefullyReleased,
			))

			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")

	setupCommandHelp(cmd)
	return cmd
}

// lockVerifyCmd verifies the schema lock file against current migration files.
func lockVerifyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify migration lock file integrity",
		Long: `Verify that the schema lock file matches the current migration files.

Checks that no migration files have been added, removed, or modified
since the lock file was last generated.`,
		Example: `  # Verify lock file integrity
  alab lock verify`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			lockPath := lockfile.DefaultPath()
			if err := lockfile.Verify(cfg.MigrationsDir, lockPath); err != nil {
				fmt.Fprintln(os.Stderr, ui.Error("Lock verification failed")+": "+err.Error())
				fmt.Fprintln(os.Stderr, ui.Help("Run 'alab lock repair' to regenerate the lock file"))
				os.Exit(1)
			}

			fmt.Println(ui.Success("Lock file verified successfully"))
			return nil
		},
	}

	setupCommandHelp(cmd)
	return cmd
}

// lockRepairCmd regenerates the schema lock file from current migration files.
func lockRepairCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repair",
		Short: "Regenerate the migration lock file",
		Long: `Regenerate the schema lock file from the current migration files.

Use this after resolving any lock file verification failures.`,
		Example: `  # Regenerate lock file
  alab lock repair`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			lockPath := lockfile.DefaultPath()
			if err := lockfile.Repair(cfg.MigrationsDir, lockPath); err != nil {
				return fmt.Errorf("failed to repair lock file: %w", err)
			}

			fmt.Println(ui.Success("Lock file regenerated: " + ui.FilePath(lockPath)))
			return nil
		},
	}

	setupCommandHelp(cmd)
	return cmd
}
