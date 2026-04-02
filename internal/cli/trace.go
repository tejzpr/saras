/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2026 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tejzpr/saras/internal/config"
	"github.com/tejzpr/saras/internal/trace"
)

func init() {
	rootCmd.AddCommand(traceCmd)
}

var traceCmd = &cobra.Command{
	Use:   "trace [symbol]",
	Short: "Trace a symbol: definition, references, callers, callees",
	Long: `Trace a code symbol across your project. Shows the symbol definition,
all references, caller functions, and callee functions.

Examples:
  saras trace Login
  saras trace handleRequest --callers
  saras trace NewDB --callees`,
	Args: cobra.ExactArgs(1),
	RunE: runTrace,
}

func init() {
	traceCmd.Flags().Bool("callers", false, "Show only callers")
	traceCmd.Flags().Bool("callees", false, "Show only callees")
	traceCmd.Flags().Bool("refs", false, "Show only references")
	traceCmd.Flags().Bool("json", false, "Output as JSON")
}

func runTrace(cmd *cobra.Command, args []string) error {
	symbolName := args[0]
	showCallers, _ := cmd.Flags().GetBool("callers")
	showCallees, _ := cmd.Flags().GetBool("callees")
	showRefs, _ := cmd.Flags().GetBool("refs")

	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("not a saras project (run 'saras init' first): %w", err)
	}

	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	tracer := trace.NewTracer(projectRoot, cfg.Ignore)
	ctx := context.Background()

	out := cmd.OutOrStdout()

	// If a specific view is requested
	if showCallers {
		callers, err := tracer.FindCallers(ctx, symbolName)
		if err != nil {
			return fmt.Errorf("find callers: %w", err)
		}
		printCallers(cmd, symbolName, callers)
		return nil
	}

	if showCallees {
		callees, err := tracer.FindCallees(ctx, symbolName)
		if err != nil {
			return fmt.Errorf("find callees: %w", err)
		}
		printCallees(cmd, symbolName, callees)
		return nil
	}

	if showRefs {
		refs, err := tracer.FindReferences(ctx, symbolName)
		if err != nil {
			return fmt.Errorf("find references: %w", err)
		}
		printRefs(cmd, symbolName, refs)
		return nil
	}

	// Full trace
	result, err := tracer.Trace(ctx, symbolName)
	if err != nil {
		return fmt.Errorf("trace: %w", err)
	}

	// Symbol definition
	if result.Symbol != nil {
		s := result.Symbol
		fmt.Fprintf(out, "Symbol: %s (%s)\n", s.Name, s.Kind)
		fmt.Fprintf(out, "  File: %s:%d-%d\n", s.FilePath, s.Line, s.EndLine)
		if s.Signature != "" {
			fmt.Fprintf(out, "  Sig:  %s\n", s.Signature)
		}
		if s.Parent != "" {
			fmt.Fprintf(out, "  Type: %s\n", s.Parent)
		}
		fmt.Fprintln(out)
	} else {
		fmt.Fprintf(out, "Symbol %q not found as a definition\n\n", symbolName)
	}

	// References
	if len(result.References) > 0 {
		printRefs(cmd, symbolName, result.References)
		fmt.Fprintln(out)
	}

	// Callers
	if len(result.Callers) > 0 {
		printCallers(cmd, symbolName, result.Callers)
		fmt.Fprintln(out)
	}

	// Callees
	if len(result.Callees) > 0 {
		printCallees(cmd, symbolName, result.Callees)
	}

	return nil
}

func printRefs(cmd *cobra.Command, name string, refs []trace.Reference) {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "References to %q (%d):\n", name, len(refs))
	for _, r := range refs {
		fmt.Fprintf(out, "  %s:%d  %s\n", r.FilePath, r.Line, truncate(r.Context, 80))
	}
}

func printCallers(cmd *cobra.Command, name string, callers []trace.CallEdge) {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Callers of %q (%d):\n", name, len(callers))
	for _, c := range callers {
		fmt.Fprintf(out, "  %s (%s:%d)\n", c.Caller, c.CallerFile, c.CallerLine)
	}
}

func printCallees(cmd *cobra.Command, name string, callees []trace.CallEdge) {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Callees of %q (%d):\n", name, len(callees))
	for _, c := range callees {
		fmt.Fprintf(out, "  %s\n", c.Callee)
	}
}

func truncate(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}
