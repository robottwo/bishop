package analytics

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"mvdan.cc/sh/v3/interp"
)

const (
	defaultMaxWidth = 40 // Default max width for truncated columns
)

func NewAnalyticsCommandHandler(analyticsManager *AnalyticsManager) func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
	return func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
		return func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return next(ctx, args)
			}

			if args[0] != "bish_analytics" {
				return next(ctx, args)
			}

			// Parse flags and arguments
			if len(args) > 1 {
				switch args[1] {
				case "-c", "--clear":
					// Clear the analytics
					return analyticsManager.ResetAnalytics()

				case "-d", "--delete":
					// Delete a specific entry
					if len(args) < 3 {
						return fmt.Errorf("analytics -d requires an entry ID")
					}
					id, err := strconv.Atoi(args[2])
					if err != nil {
						return fmt.Errorf("invalid analytics entry ID: %s", args[2])
					}
					if err := analyticsManager.DeleteEntry(uint(id)); err != nil {
						return fmt.Errorf("failed to delete analytics entry %d: %v", id, err)
					}
					return nil

				case "-h", "--help":
					printAnalyticsHelp()
					return nil

				case "-n", "--count":
					// Show total count of entries
					count, err := analyticsManager.GetTotalCount()
					if err != nil {
						return fmt.Errorf("failed to get analytics count: %v", err)
					}
					fmt.Printf("Total analytics entries: %d\n", count)
					return nil
				}
			}

			// Default limit is 20 entries, or use provided number
			limit := 20
			if len(args) > 1 {
				providedLimit, err := strconv.Atoi(args[1])
				if err == nil && providedLimit > 0 {
					limit = providedLimit
				}
			}

			// Get recent entries
			entries, err := analyticsManager.GetRecentEntries(limit)
			if err != nil {
				return err
			}

			if len(entries) == 0 {
				fmt.Println("No analytics entries found.")
				return nil
			}

			// Print entries in table format
			printEntriesTable(entries)

			return nil
		}
	}
}

func printAnalyticsHelp() {
	help := []string{
		"Usage: bish_analytics [option] [n]",
		"Display or manipulate the analytics data.",
		"",
		"Options:",
		"  -c, --clear    clear all analytics data",
		"  -d, --delete   delete analytics entry by ID",
		"  -h, --help     display this help message",
		"  -n, --count    display total number of entries",
		"",
		"If n is given, display only the last n entries.",
		"If no options are given, display the analytics list in table format.",
	}
	fmt.Println(strings.Join(help, "\n"))
}

// printEntriesTable prints analytics entries in a formatted table
func printEntriesTable(entries []AnalyticsEntry) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Print header
	fmt.Fprintln(w, "ID\tTIME\tINPUT\tPREDICTION\tACTUAL")
	fmt.Fprintln(w, "──\t────\t─────\t──────────\t──────")

	// Print each entry
	for _, entry := range entries {
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n",
			entry.ID,
			entry.CreatedAt.Format("01/02 15:04"),
			truncate(entry.Input, defaultMaxWidth),
			truncate(entry.Prediction, defaultMaxWidth),
			truncate(entry.Actual, defaultMaxWidth),
		)
	}

	w.Flush()
}

// truncate shortens a string to maxLen characters, adding ellipsis if truncated
func truncate(s string, maxLen int) string {
	// Replace newlines with spaces for single-line display
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.TrimSpace(s)

	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
