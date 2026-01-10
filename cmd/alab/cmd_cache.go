package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/hlop3z/astroladb/pkg/astroladb"
	"github.com/spf13/cobra"
)

// rebuildCacheCmd rebuilds the local cache from migration files.
func rebuildCacheCmd() *cobra.Command {
	var incremental, clear, stats, quiet bool

	cmd := &cobra.Command{
		Use:     "rebuild-cache",
		Aliases: []string{"cache"},
		Short:   "Rebuild local cache from migration files",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				if handleClientError(err) {
					os.Exit(1)
				}
				return err
			}
			defer client.Close()

			// Stats only mode
			if stats {
				cacheStats, err := client.GetCacheStats()
				if err != nil {
					return fmt.Errorf("failed to get cache stats: %w", err)
				}

				fmt.Println("Cache Statistics")
				fmt.Println(strings.Repeat("=", 40))
				fmt.Printf("  Cache path:        %s\n", cacheStats.CachePath)
				fmt.Printf("  Total revisions:   %d\n", cacheStats.Revisions)
				fmt.Printf("  Schema snapshots:  %d\n", cacheStats.SchemaSnapshots)
				fmt.Printf("  Merkle hashes:     %d\n", cacheStats.MerkleHashes)

				if cacheStats.Revisions > 0 {
					coverage := float64(cacheStats.SchemaSnapshots) / float64(cacheStats.Revisions) * 100
					fmt.Printf("  Cache coverage:    %.1f%%\n", coverage)
				}

				return nil
			}

			// Clear only mode
			if clear {
				if err := client.ClearCache(); err != nil {
					return fmt.Errorf("failed to clear cache: %w", err)
				}
				fmt.Println("Cache cleared.")
				return nil
			}

			// Progress callback
			progress := func(revision string, current, total int) {
				if !quiet {
					fmt.Printf("\rCaching revision %s (%d/%d)...", revision, current, total)
				}
			}

			// Rebuild
			var cacheStats *astroladb.CacheStats
			if incremental {
				if !quiet {
					fmt.Println("Rebuilding cache (incremental)...")
				}
				cacheStats, err = client.RebuildCacheIncremental(progress)
			} else {
				if !quiet {
					fmt.Println("Rebuilding cache...")
				}
				cacheStats, err = client.RebuildCache(progress)
			}

			if err != nil {
				return err
			}

			if !quiet {
				fmt.Println() // New line after progress
				fmt.Println()
				fmt.Println("Cache rebuilt successfully!")
				fmt.Printf("  Revisions cached:  %d\n", cacheStats.SchemaSnapshots)
				fmt.Printf("  Merkle hashes:     %d\n", cacheStats.MerkleHashes)
				fmt.Printf("  Cache path:        %s\n", cacheStats.CachePath)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&incremental, "incremental", false, "Only add missing revisions (faster for updates)")
	cmd.Flags().BoolVar(&clear, "clear", false, "Clear cache without rebuilding")
	cmd.Flags().BoolVar(&stats, "stats", false, "Show cache statistics only")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "Suppress progress output")

	return cmd
}
