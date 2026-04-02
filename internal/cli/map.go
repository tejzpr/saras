/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2026 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tejzpr/saras/internal/architect"
	"github.com/tejzpr/saras/internal/config"
)

func init() {
	rootCmd.AddCommand(mapCmd)
}

var mapCmd = &cobra.Command{
	Use:   "map",
	Short: "Generate a codebase architecture map",
	Long: `Generate a high-level map of your codebase showing packages, types,
functions, dependencies, and project structure.

Output formats:
  - tree: directory tree (default)
  - markdown: detailed markdown report
  - summary: compact overview

Examples:
  saras map
  saras map --format markdown
  saras map --format markdown --output ARCHITECTURE.md`,
	RunE: runMap,
}

func init() {
	mapCmd.Flags().StringP("format", "f", "tree", "Output format: tree, markdown, summary")
	mapCmd.Flags().StringP("output", "o", "", "Write output to file instead of stdout")
}

func runMap(cmd *cobra.Command, args []string) error {
	format, _ := cmd.Flags().GetString("format")
	outputFile, _ := cmd.Flags().GetString("output")

	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("not a saras project (run 'saras init' first): %w", err)
	}

	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	mapper := architect.NewMapper(projectRoot, cfg.Ignore)
	ctx := context.Background()

	var output string

	switch format {
	case "tree":
		output, err = mapper.GenerateTree(ctx)
		if err != nil {
			return fmt.Errorf("generate tree: %w", err)
		}

	case "markdown", "md":
		output, err = mapper.GenerateMarkdown(ctx)
		if err != nil {
			return fmt.Errorf("generate markdown: %w", err)
		}

	case "summary":
		cmap, err := mapper.GenerateMap(ctx)
		if err != nil {
			return fmt.Errorf("generate map: %w", err)
		}
		output = formatSummary(cmap)

	default:
		return fmt.Errorf("unknown format %q (use: tree, markdown, summary)", format)
	}

	if outputFile != "" {
		if err := os.WriteFile(outputFile, []byte(output), 0644); err != nil {
			return fmt.Errorf("write file: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Written to %s\n", outputFile)
		return nil
	}

	fmt.Fprint(cmd.OutOrStdout(), output)
	return nil
}

func formatSummary(cmap *architect.CodebaseMap) string {
	var s string
	s += fmt.Sprintf("Project: %s\n", cmap.ProjectRoot)
	s += fmt.Sprintf("Files:   %d\n", cmap.TotalFiles)
	s += fmt.Sprintf("Lines:   %d\n", cmap.TotalLines)
	s += fmt.Sprintf("Packages: %d\n\n", len(cmap.Packages))

	for _, pkg := range cmap.Packages {
		s += fmt.Sprintf("  %-30s %d files, %d funcs, %d types, %d ifaces\n",
			pkg.Path, len(pkg.Files), pkg.Functions, pkg.Types, pkg.Interfaces)
	}

	if len(cmap.Dependencies) > 0 {
		s += fmt.Sprintf("\nDependencies: %d\n", len(cmap.Dependencies))
		for _, d := range cmap.Dependencies {
			s += fmt.Sprintf("  %s → %s\n", d.From, d.To)
		}
	}

	return s
}
