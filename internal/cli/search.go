/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2026 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/tejzpr/saras/internal/config"
	"github.com/tejzpr/saras/internal/embedder"
	searchpkg "github.com/tejzpr/saras/internal/search"
	"github.com/tejzpr/saras/internal/store"
	"github.com/tejzpr/saras/internal/tui"
)

func init() {
	rootCmd.AddCommand(searchCmd)
}

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Semantic search across your codebase",
	Long: `Search your indexed codebase using natural language queries.
Combines vector similarity with optional text search and boost scoring.

Examples:
  saras search "authentication login flow"
  saras search "database connection pool" --limit 20
  saras search "error handling" --json`,
	Args: cobra.MinimumNArgs(1),
	RunE: runSearch,
}

func init() {
	searchCmd.Flags().IntP("limit", "n", 10, "Maximum number of results")
	searchCmd.Flags().Bool("json", false, "Output results as JSON")
	searchCmd.Flags().Bool("no-tui", false, "Print results to stdout (no interactive TUI)")
}

func runSearch(cmd *cobra.Command, args []string) error {
	query := strings.Join(args, " ")
	limit, _ := cmd.Flags().GetInt("limit")
	jsonOutput, _ := cmd.Flags().GetBool("json")
	noTUI, _ := cmd.Flags().GetBool("no-tui")

	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("not a saras project (run 'saras init' first): %w", err)
	}

	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Open store
	storePath := filepath.Join(config.GetConfigDir(projectRoot), "index.gob")
	st := store.NewGobStore(storePath)
	if err := st.Load(context.Background()); err != nil {
		// Fresh store is okay, just won't have results
		_ = err
	}
	defer st.Close()

	// Create embedder
	emb, err := embedder.NewFromConfig(cfg)
	if err != nil {
		return fmt.Errorf("create embedder: %w", err)
	}
	defer emb.Close()

	// Create searcher
	searcher := searchpkg.NewSearcher(st, emb, cfg.Search)

	// Run search
	ctx := context.Background()
	results, err := searcher.Search(ctx, query, limit)
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}

	if jsonOutput {
		return printSearchJSON(cmd, results)
	}

	if noTUI {
		return printSearchPlain(cmd, query, results)
	}

	// Interactive TUI
	model := tui.NewSearchModelWithResults(query, results)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}

func printSearchJSON(cmd *cobra.Command, results []searchpkg.Result) error {
	fmt.Fprintln(cmd.OutOrStdout(), "[")
	for i, r := range results {
		comma := ","
		if i == len(results)-1 {
			comma = ""
		}
		fmt.Fprintf(cmd.OutOrStdout(), `  {"file":"%s","start_line":%d,"end_line":%d,"score":%.4f}%s`+"\n",
			r.FilePath, r.StartLine, r.EndLine, r.Score, comma)
	}
	fmt.Fprintln(cmd.OutOrStdout(), "]")
	return nil
}

func printSearchPlain(cmd *cobra.Command, query string, results []searchpkg.Result) error {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Search: %q (%d results)\n\n", query, len(results))

	for i, r := range results {
		fmt.Fprintf(out, "%d. [%.2f] %s:%d-%d\n", i+1, r.Score, r.FilePath, r.StartLine, r.EndLine)

		// Show first 2 lines of content
		lines := strings.SplitN(strings.TrimSpace(r.Content), "\n", 3)
		for _, line := range lines[:min(len(lines), 2)] {
			if len(line) > 100 {
				line = line[:97] + "..."
			}
			fmt.Fprintf(out, "   %s\n", line)
		}
		fmt.Fprintln(out)
	}

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
