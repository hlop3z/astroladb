package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/hlop3z/astroladb/internal/chain"
	"github.com/hlop3z/astroladb/internal/git"
	"github.com/hlop3z/astroladb/internal/ui"
	"github.com/spf13/cobra"
)

// verifyCmd verifies migration chain integrity and git status.
func verifyCmd() *cobra.Command {
	var branch string

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify migration chain integrity and git status",
		Long: `Verify migration chain integrity and git status.

Checks checksums, cryptographic integrity, git status, and optionally compares with a branch.
Ensures migration history hasn't been tampered with and is tracked in version control.`,
		Example: `  # Verify local migration chain and git status
  alab verify

  # Compare local migrations with a specific branch
  alab verify --branch main

  # Verify and check differences from production branch
  alab verify --branch production`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			fmt.Println(ui.RenderTitle("Migration Verification"))
			fmt.Println()

			// Compute and display chain status
			migrationChain, err := chain.ComputeFromDir(cfg.MigrationsDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error computing chain: %v\n", err)
			} else {
				fmt.Println(chain.FormatChainStatus(migrationChain))
			}

			fmt.Println()

			// Check git status
			info, err := git.GetBranchInfo(cfg.MigrationsDir)
			if err != nil {
				fmt.Println("Git Status: Not in a git repository")
				fmt.Println()
				fmt.Println(git.FormatNotGitRepoWarning())
			} else {
				uncommitted, _ := git.VerifyMigrationsCommitted(cfg.MigrationsDir)
				fmt.Println(git.FormatGitStatusSummary(info, uncommitted))
			}

			fmt.Println()

			// Compare with branch if specified
			if branch != "" {
				fmt.Println(ui.RenderSubtitle(fmt.Sprintf("Comparing with %s", branch)))
				fmt.Println()

				repo, err := git.Open(cfg.MigrationsDir)
				if err != nil {
					return fmt.Errorf("not in a git repository")
				}

				// Get files at specified branch
				branchFiles, err := repo.GetFilesAtCommit(cfg.MigrationsDir, branch)
				if err != nil {
					return fmt.Errorf("failed to read migrations from %s: %v", branch, err)
				}

				// Get local files
				localEntries, err := os.ReadDir(cfg.MigrationsDir)
				if err != nil {
					return fmt.Errorf("failed to read local migrations: %v", err)
				}

				localFiles := make(map[string]bool)
				for _, e := range localEntries {
					if !e.IsDir() && strings.HasSuffix(e.Name(), ".js") {
						localFiles[e.Name()] = true
					}
				}

				// Find differences
				var onlyInBranch, onlyLocal, common []string
				for name := range branchFiles {
					if localFiles[name] {
						common = append(common, name)
					} else {
						onlyInBranch = append(onlyInBranch, name)
					}
				}
				for name := range localFiles {
					if _, exists := branchFiles[name]; !exists {
						onlyLocal = append(onlyLocal, name)
					}
				}

				if len(onlyInBranch) == 0 && len(onlyLocal) == 0 {
					fmt.Println(ui.RenderSuccessPanel(
						"Branches match",
						fmt.Sprintf("Local migrations match %s", branch),
					))
				} else {
					list := ui.NewList()

					if len(onlyInBranch) > 0 {
						for _, name := range onlyInBranch {
							list.AddError(fmt.Sprintf("Missing locally: %s", name))
						}
					}

					if len(onlyLocal) > 0 {
						for _, name := range onlyLocal {
							list.AddSuccess(fmt.Sprintf("Only local: %s", name))
						}
					}

					fmt.Println(ui.RenderWarningPanel("Branch differences detected", list.String()))
				}
				fmt.Println()
			}

			// Show chain verification from database if possible
			client, err := newClient()
			if err == nil {
				defer client.Close()

				// Verify chain against database
				verifyResult, err := client.VerifyChain()
				if err == nil {
					fmt.Println(chain.FormatVerificationResult(verifyResult))
				} else {
					// Still show basic status
					statuses, err := client.MigrationStatus()
					if err == nil && len(statuses) > 0 {
						fmt.Println("Database Status")
						fmt.Println(strings.Repeat("-", 50))

						var applied, pending int
						for _, s := range statuses {
							if s.Status == "applied" {
								applied++
							} else {
								pending++
							}
						}

						fmt.Printf("  Applied: %d\n", applied)
						fmt.Printf("  Pending: %d\n", pending)

						if pending > 0 {
							fmt.Println("\n  Pending migrations:")
							for _, s := range statuses {
								if s.Status == "pending" {
									fmt.Printf("    - %s_%s\n", s.Revision, s.Name)
								}
							}
						}
					}
				}
			}

			fmt.Println()
			fmt.Println(ui.Done("Verification complete"))
			return nil
		},
	}

	cmd.Flags().StringVar(&branch, "branch", "", "Compare with a git branch")

	setupCommandHelp(cmd)
	return cmd
}
