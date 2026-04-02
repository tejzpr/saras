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
	"github.com/tejzpr/saras/internal/ask"
	"github.com/tejzpr/saras/internal/config"
	"github.com/tejzpr/saras/internal/embedder"
	"github.com/tejzpr/saras/internal/search"
	"github.com/tejzpr/saras/internal/store"
	"github.com/tejzpr/saras/internal/tui"
)

func init() {
	rootCmd.AddCommand(askCmd)
}

var askCmd = &cobra.Command{
	Use:   "ask [question]",
	Short: "Ask a question about your codebase using RAG",
	Long: `Ask a natural language question about your codebase. Saras will search
for relevant code, build context, and stream an AI-generated answer.

The response is powered by your configured LLM provider.

Examples:
  saras ask "how does the authentication flow work?"
  saras ask "what database connections are used?" --limit 10
  saras ask "explain the error handling strategy" --no-tui`,
	Args: cobra.MinimumNArgs(1),
	RunE: runAsk,
}

func init() {
	askCmd.Flags().IntP("limit", "n", 5, "Number of code snippets for context")
	askCmd.Flags().Int("max-tokens", 2048, "Maximum response tokens")
	askCmd.Flags().Float32("temperature", 0.1, "LLM temperature")
	askCmd.Flags().String("model", "", "Override LLM model")
	askCmd.Flags().Bool("no-tui", false, "Print response to stdout (no interactive TUI)")
}

func runAsk(cmd *cobra.Command, args []string) error {
	question := strings.Join(args, " ")
	limit, _ := cmd.Flags().GetInt("limit")
	maxTokens, _ := cmd.Flags().GetInt("max-tokens")
	temperature, _ := cmd.Flags().GetFloat32("temperature")
	model, _ := cmd.Flags().GetString("model")
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
	searcher := search.NewSearcher(st, emb, cfg.Search)

	// Build chat endpoint from LLM config
	chatEndpoint := buildChatEndpoint(cfg)

	// Create pipeline
	pipelineOpts := []ask.PipelineOption{}
	if cfg.LLM.APIKey != "" {
		pipelineOpts = append(pipelineOpts, ask.WithAPIKey(cfg.LLM.APIKey))
	}

	chatModel := cfg.LLM.Model
	if model != "" {
		chatModel = model
	}

	pipeline := ask.NewPipeline(searcher, chatEndpoint, chatModel, pipelineOpts...)

	opts := ask.AskOptions{
		Query:       question,
		Limit:       limit,
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}

	if noTUI {
		return runAskPlain(cmd, pipeline, opts)
	}

	return runAskTUI(cmd, pipeline, opts)
}

func runAskPlain(cmd *cobra.Command, pipeline *ask.Pipeline, opts ask.AskOptions) error {
	ch, err := pipeline.Ask(context.Background(), opts)
	if err != nil {
		return fmt.Errorf("ask: %w", err)
	}

	out := cmd.OutOrStdout()
	for chunk := range ch {
		if chunk.Err != nil {
			return fmt.Errorf("ask: %w", chunk.Err)
		}
		fmt.Fprint(out, chunk.Content)
	}
	fmt.Fprintln(out)
	return nil
}

func runAskTUI(cmd *cobra.Command, pipeline *ask.Pipeline, opts ask.AskOptions) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := pipeline.Ask(ctx, opts)
	if err != nil {
		return fmt.Errorf("ask: %w", err)
	}

	model := tui.NewAskModel(opts.Query)
	p := tea.NewProgram(model, tea.WithAltScreen())

	// Feed stream chunks to TUI in background
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
		cancel()
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}

func buildChatEndpoint(cfg *config.Config) string {
	endpoint := cfg.LLM.Endpoint

	switch cfg.LLM.Provider {
	case "ollama":
		endpoint = strings.TrimRight(endpoint, "/")
		if !strings.HasSuffix(endpoint, "/v1") {
			endpoint += "/v1"
		}
	case "lmstudio":
		endpoint = strings.TrimRight(endpoint, "/")
		if !strings.HasSuffix(endpoint, "/v1") {
			endpoint += "/v1"
		}
	case "openai":
		endpoint = strings.TrimRight(endpoint, "/")
	}

	return endpoint
}
