package main

import (
	"fmt"
	"os"

	"github.com/hlop3z/astroladb/internal/ui"
	"github.com/spf13/cobra"
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

			fmt.Println(ui.RenderTitle("Migration Lock Status"))
			fmt.Println()

			if !info.Locked {
				fmt.Println(ui.RenderSuccessPanel(
					"Lock Available",
					"No migration is currently running",
				))
			} else {
				lockedAt := "unknown"
				if info.LockedAt != nil {
					lockedAt = info.LockedAt.Format("2006-01-02 15:04:05")
				}

				list := ui.NewList()
				list.AddInfo(fmt.Sprintf("Locked by: %s", info.LockedBy))
				list.AddInfo(fmt.Sprintf("Locked at: %s", lockedAt))

				fmt.Println(ui.RenderWarningPanel(
					"Lock Held",
					list.String()+"\n"+
						ui.Help("If this is a stuck lock, use: alab lock release"),
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
					"No Lock to Release",
					"Migration lock is not currently held",
				))
				return nil
			}

			// Show warning
			fmt.Println(ui.RenderTitle("Release Migration Lock"))
			fmt.Println()

			lockedAt := "unknown"
			if info.LockedAt != nil {
				lockedAt = info.LockedAt.Format("2006-01-02 15:04:05")
			}

			list := ui.NewList()
			list.AddInfo(fmt.Sprintf("Locked by: %s", info.LockedBy))
			list.AddInfo(fmt.Sprintf("Locked at: %s", lockedAt))

			fmt.Println(ui.RenderWarningPanel(
				"Lock Will Be Released",
				list.String()+"\n"+
					ui.Warning("Only release if you're certain no migration is running"),
			))
			fmt.Println()

			// Confirm unless --force
			if !force {
				if !ui.Confirm("Release this lock?", false) {
					fmt.Println(ui.Dim("Release cancelled"))
					return nil
				}
				fmt.Println()
			}

			if err := client.MigrationReleaseLock(); err != nil {
				return err
			}

			fmt.Println(ui.RenderSuccessPanel(
				"Lock Released",
				"Migration lock has been forcefully released",
			))

			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")

	setupCommandHelp(cmd)
	return cmd
}
