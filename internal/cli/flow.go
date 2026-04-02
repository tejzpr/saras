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
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/tejzpr/saras/internal/architect"
	"github.com/tejzpr/saras/internal/ask"
	"github.com/tejzpr/saras/internal/config"
	"github.com/tejzpr/saras/internal/embedder"
	"github.com/tejzpr/saras/internal/search"
	"github.com/tejzpr/saras/internal/store"
	"github.com/tejzpr/saras/internal/tui"
)

func init() {
	rootCmd.AddCommand(flowCmd)
	flowCmd.AddCommand(flowExplainCmd)
}

var flowCmd = &cobra.Command{
	Use:   "flow [function]",
	Short: "Show execution flow from entry points or a specific function",
	Long: `Generate a call-flow tree showing how execution flows through your codebase.

Without arguments (or with "full"), shows flows from all detected entry points
(main, init, CLI command handlers, HTTP handlers).

With a function name, shows the call tree from that specific function.

Examples:
  saras flow              # all entry points
  saras flow full         # all entry points (explicit)
  saras flow runInit      # flow from a specific function
  saras flow --depth 3    # limit tree depth
  saras flow -o FLOW.md   # write to file`,
	RunE: runFlow,
}

func init() {
	flowCmd.Flags().IntP("depth", "d", 8, "Maximum call tree depth")
	flowCmd.Flags().StringP("output", "o", "", "Write output to file instead of stdout")

	flowExplainCmd.Flags().IntP("depth", "d", 8, "Maximum call tree depth")
	flowExplainCmd.Flags().StringP("output", "o", "", "Write output to file instead of stdout")
	flowExplainCmd.Flags().Bool("no-tui", false, "Print response to stdout (no interactive TUI)")
	flowExplainCmd.Flags().Int("max-tokens", 4096, "Maximum response tokens")
	flowExplainCmd.Flags().Float32("temperature", 0.2, "LLM temperature")
	flowExplainCmd.Flags().String("model", "", "Override LLM model")
}

var flowExplainCmd = &cobra.Command{
	Use:   "explain [function]",
	Short: "Explain execution flow using an LLM",
	Long: `Generate a call-flow tree and send it to your configured LLM for a
natural-language explanation of how execution flows through the codebase.

Without arguments, provides a concise high-level summary of all entry points.
With "full", performs an exhaustive deep-dive: higher depth (12), detailed
per-entry-point analysis, shared infrastructure, patterns, and data flow.
With a function name, explains the call tree from that specific function.

Examples:
  saras flow explain                # concise summary of all entry points
  saras flow explain full           # exhaustive deep-dive analysis
  saras flow explain handleRequest  # explain a specific function's flow
  saras flow explain --no-tui       # plain stdout output
  saras flow explain -o EXPLAIN.md  # write to file`,
	RunE: runFlowExplain,
}

func runFlow(cmd *cobra.Command, args []string) error {
	depth, _ := cmd.Flags().GetInt("depth")
	outputFile, _ := cmd.Flags().GetString("output")

	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("not a saras project (run 'saras init' first): %w", err)
	}

	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	fm := architect.NewFlowMapper(projectRoot, cfg.Ignore, depth)
	ctx := context.Background()

	var output string

	if len(args) == 0 || args[0] == "full" {
		trees, err := fm.GenerateFullFlow(ctx)
		if err != nil {
			return fmt.Errorf("generate flow: %w", err)
		}
		if len(trees) == 0 {
			return fmt.Errorf("no entry points detected")
		}
		output = architect.FormatFlowTrees(trees)
	} else {
		tree, err := fm.GenerateFunctionFlow(ctx, args[0])
		if err != nil {
			return fmt.Errorf("generate flow: %w", err)
		}
		output = architect.FormatFlowTree(tree)
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

const flowExplainSystemPrompt = `You are Saras, a code-flow analyst. You receive a call-flow tree generated from a codebase and explain it in clear, concise natural language.

Instructions:
- Describe the execution flow step by step.
- Explain what each major branch of the call tree does at a high level.
- Note any interesting patterns: cycles, deeply nested paths, fan-out.
- Keep explanations factual — do not invent functionality not shown in the tree.
- Reference function names exactly as shown.
- Use markdown formatting with headers and bullet points.
- Be concise. Aim for a useful summary, not an exhaustive line-by-line walkthrough.`

const flowExplainFullSystemPrompt = `You are Saras, a senior code-flow analyst performing a comprehensive deep-dive analysis. You receive a call-flow tree generated from a codebase and produce an exhaustive, detailed explanation.

Instructions:
- Analyze EVERY entry point and its complete call chain in detail.
- For each entry point, explain:
  - What triggers it (CLI command, HTTP request, lifecycle event, etc.)
  - The step-by-step sequence of function calls it makes
  - What each called function does and why it is called at that point
  - How data flows between the functions (setup, transformation, persistence, I/O)
  - Error handling patterns and cleanup/defer behavior if visible
- Dedicate a full section (## heading) to each entry point.
- After analyzing all entry points, provide:
  - A "Shared Infrastructure" section identifying functions called by multiple entry points
  - A "Key Patterns" section covering: initialization patterns, dependency injection, fan-out hotspots, recursive structures, and cycle analysis
  - A "Data Flow Summary" describing how data moves through the system end-to-end
- Reference function names and file paths exactly as shown in the tree.
- Use markdown with ## headings per entry point, ### subheadings for phases, and bullet points for details.
- Be thorough — this is a complete architectural walkthrough, not a summary.`

func runFlowExplain(cmd *cobra.Command, args []string) error {
	depth, _ := cmd.Flags().GetInt("depth")
	outputFile, _ := cmd.Flags().GetString("output")
	noTUI, _ := cmd.Flags().GetBool("no-tui")
	maxTokens, _ := cmd.Flags().GetInt("max-tokens")
	temperature, _ := cmd.Flags().GetFloat32("temperature")
	model, _ := cmd.Flags().GetString("model")

	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("not a saras project (run 'saras init' first): %w", err)
	}

	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Step 1: Generate the flow tree
	fm := architect.NewFlowMapper(projectRoot, cfg.Ignore, depth)
	ctx := context.Background()

	var flowOutput string
	var question string
	var systemPrompt string
	fullMode := len(args) > 0 && args[0] == "full"

	if fullMode {
		// Deep-dive mode: use higher depth and more detailed prompts
		if !cmd.Flags().Changed("depth") {
			depth = 12
		}
		if !cmd.Flags().Changed("max-tokens") {
			maxTokens = 8192
		}
		fm = architect.NewFlowMapper(projectRoot, cfg.Ignore, depth)
		trees, err := fm.GenerateFullFlow(ctx)
		if err != nil {
			return fmt.Errorf("generate flow: %w", err)
		}
		if len(trees) == 0 {
			return fmt.Errorf("no entry points detected")
		}
		flowOutput = architect.FormatFlowTrees(trees)
		systemPrompt = flowExplainFullSystemPrompt
		question = "Perform a comprehensive deep-dive analysis of this codebase's execution flow. Analyze every entry point exhaustively, explain each function's role in the call chain, identify shared infrastructure, and describe the overall architecture and data flow patterns."
	} else if len(args) == 0 {
		trees, err := fm.GenerateFullFlow(ctx)
		if err != nil {
			return fmt.Errorf("generate flow: %w", err)
		}
		if len(trees) == 0 {
			return fmt.Errorf("no entry points detected")
		}
		flowOutput = architect.FormatFlowTrees(trees)
		systemPrompt = flowExplainSystemPrompt
		question = "Explain the execution flow of this codebase starting from all entry points. Describe what each entry point does and how the call chains fan out."
	} else {
		tree, err := fm.GenerateFunctionFlow(ctx, args[0])
		if err != nil {
			return fmt.Errorf("generate flow: %w", err)
		}
		flowOutput = architect.FormatFlowTree(tree)
		systemPrompt = flowExplainSystemPrompt
		question = fmt.Sprintf("Explain the execution flow starting from the function %q. Describe what it does and how its call chain fans out.", args[0])
	}

	// Step 2: Set up LLM pipeline
	storePath := filepath.Join(config.GetConfigDir(projectRoot), "index.gob")
	st := store.NewGobStore(storePath)
	if err := st.Load(ctx); err != nil {
		_ = err
	}
	defer st.Close()

	emb, err := embedder.NewFromConfig(cfg)
	if err != nil {
		return fmt.Errorf("create embedder: %w", err)
	}
	defer emb.Close()

	searcher := search.NewSearcher(st, emb, cfg.Search)
	chatEndpoint := buildChatEndpoint(cfg)

	pipelineOpts := []ask.PipelineOption{}
	if cfg.LLM.APIKey != "" {
		pipelineOpts = append(pipelineOpts, ask.WithAPIKey(cfg.LLM.APIKey))
	}

	chatModel := cfg.LLM.Model
	if model != "" {
		chatModel = model
	}

	pipeline := ask.NewPipeline(searcher, chatEndpoint, chatModel, pipelineOpts...)

	askOpts := ask.AskOptions{
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}

	// Step 3: Send flow tree to LLM
	contextStr := fmt.Sprintf("Call-flow tree:\n```\n%s```", flowOutput)
	ch, err := pipeline.AskWithContext(ctx, systemPrompt, contextStr, question, askOpts)
	if err != nil {
		return fmt.Errorf("explain flow: %w", err)
	}

	// Step 4: Output the response
	if outputFile != "" {
		return flowExplainToFile(ch, outputFile, cmd)
	}

	if noTUI {
		return flowExplainPlain(ch, cmd)
	}

	return flowExplainTUI(ch, question)
}

func flowExplainPlain(ch <-chan ask.StreamChunk, cmd *cobra.Command) error {
	out := cmd.OutOrStdout()
	for chunk := range ch {
		if chunk.Err != nil {
			return fmt.Errorf("explain: %w", chunk.Err)
		}
		fmt.Fprint(out, chunk.Content)
	}
	fmt.Fprintln(out)
	return nil
}

func flowExplainToFile(ch <-chan ask.StreamChunk, path string, cmd *cobra.Command) error {
	var buf strings.Builder
	for chunk := range ch {
		if chunk.Err != nil {
			return fmt.Errorf("explain: %w", chunk.Err)
		}
		buf.WriteString(chunk.Content)
	}
	if err := os.WriteFile(path, []byte(buf.String()), 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Written to %s\n", path)
	return nil
}

func flowExplainTUI(ch <-chan ask.StreamChunk, question string) error {
	model := tui.NewAskModel("flow explain: " + question)
	p := tea.NewProgram(model, tea.WithAltScreen())

	go func() {
		for chunk := range ch {
			p.Send(tui.AskStreamChunkMsg{
				Content: chunk.Content,
				Done:    chunk.Done,
				Err:     chunk.Err,
			})
		}
	}()

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}
