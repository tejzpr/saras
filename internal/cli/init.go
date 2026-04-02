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
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/tejzpr/saras/internal/config"
	"github.com/tejzpr/saras/internal/embedder"
	"github.com/tejzpr/saras/internal/engine"
	"github.com/tejzpr/saras/internal/store"
	"github.com/tejzpr/saras/internal/tui"
)

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize saras in the current directory",
	Long: `Initialize saras in the current directory by creating a .saras/config.yaml
configuration file. An interactive wizard helps you choose your embedding
provider, model, and endpoint.

Use --yes to accept all defaults (Ollama with nomic-embed-text).`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().BoolP("yes", "y", false, "Accept all defaults without interactive wizard")
	initCmd.Flags().String("provider", "", "Provider (ollama, lmstudio, openai)")
	initCmd.Flags().String("model", "", "Embedding model name")
	initCmd.Flags().String("endpoint", "", "API endpoint URL")
	initCmd.Flags().String("api-key", "", "API key (for openai provider)")
	initCmd.Flags().String("llm-model", "", "LLM model for chat/ask (e.g. llama3.2, gpt-4o-mini)")
	initCmd.Flags().Bool("no-index", false, "Skip initial indexing after setup")
}

func runInit(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	if config.Exists(cwd) {
		fmt.Fprintf(cmd.OutOrStdout(), "saras is already initialized in this directory.\n")
		fmt.Fprintf(cmd.OutOrStdout(), "Config: %s\n", config.GetConfigPath(cwd))
		return nil
	}

	yes, _ := cmd.Flags().GetBool("yes")
	provider, _ := cmd.Flags().GetString("provider")

	var result tui.InitResult

	if yes || provider != "" {
		// Non-interactive mode
		result = buildNonInteractiveResult(cmd)
	} else {
		// Interactive TUI wizard
		model := tui.NewInitModel()
		p := tea.NewProgram(model)
		finalModel, err := p.Run()
		if err != nil {
			return fmt.Errorf("TUI error: %w", err)
		}
		result = finalModel.(tui.InitModel).GetResult()
	}

	if result.Aborted {
		return nil
	}

	cfg := buildConfigFromResult(result)

	// Probe the embedding model to auto-detect vector dimensions
	fmt.Fprintf(cmd.OutOrStdout(), "\nDetecting embedding dimensions...\n")
	dims, err := embedder.ProbeDimensions(
		context.Background(),
		cfg.Embedder.Provider, cfg.Embedder.Model,
		cfg.Embedder.Endpoint, cfg.Embedder.APIKey,
	)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "%s Could not detect dimensions: %v\n", tui.SymbolWarning, err)
		fmt.Fprintf(cmd.OutOrStdout(), "  Using default: %d\n", cfg.Embedder.GetDimensions())
	} else {
		cfg.Embedder.Dimensions = &dims
		fmt.Fprintf(cmd.OutOrStdout(), "%s Detected %d dimensions\n", tui.SymbolCheck, dims)
	}

	if err := cfg.Save(cwd); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%s Created %s\n",
		tui.SymbolCheck, config.GetConfigPath(cwd))

	// Run initial index with progress bar
	noIndex, _ := cmd.Flags().GetBool("no-index")
	if !noIndex {
		fmt.Fprintf(cmd.OutOrStdout(), "\nRunning initial index...\n")
		if err := runInitialIndex(cwd, cfg); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "\n%s Initial indexing failed: %v\n", tui.SymbolWarning, err)
			fmt.Fprintf(cmd.OutOrStdout(), "  You can index later with: saras watch\n")
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\nNext steps:\n")
	fmt.Fprintf(cmd.OutOrStdout(), "  1. Start the watcher:  saras watch\n")
	fmt.Fprintf(cmd.OutOrStdout(), "  2. Search your code:   saras search \"your query\"\n")
	fmt.Fprintf(cmd.OutOrStdout(), "  3. Ask questions:      saras ask \"how does auth work?\"\n")
	fmt.Fprintf(cmd.OutOrStdout(), "  4. Explain flow:      saras flow explain\n")

	return nil
}

func runInitialIndex(projectRoot string, cfg *config.Config) error {
	ctx := context.Background()

	// Open store
	storePath := filepath.Join(config.GetConfigDir(projectRoot), "index.gob")
	st := store.NewGobStore(storePath)
	if err := st.Load(ctx); err != nil {
		_ = err
	}
	defer func() {
		st.Persist(ctx)
		st.Close()
	}()

	// Create embedder
	emb, err := embedder.NewFromConfig(cfg)
	if err != nil {
		return fmt.Errorf("create embedder: %w", err)
	}
	defer emb.Close()

	// Create indexer
	scanner := engine.NewScanner(projectRoot, cfg.Ignore)
	chunker := engine.NewChunker(cfg.Chunking.Size, cfg.Chunking.Overlap)
	indexer := engine.NewIndexer(projectRoot, st, emb, chunker, scanner)

	// Set up progress bar TUI
	model := tui.NewProgressModel()
	p := tea.NewProgram(model)

	// Run indexer in background, sending progress to TUI
	go func() {
		start := time.Now()
		stats, err := indexer.IndexAllWithProgress(ctx, func(info engine.ProgressInfo) {
			p.Send(tui.ProgressUpdateMsg{
				Current:     info.Current,
				Total:       info.Total,
				CurrentFile: info.CurrentFile,
			})
		})

		result := tui.ProgressDoneMsg{
			Duration: time.Since(start),
			Err:      err,
		}
		if stats != nil {
			result.FilesIndexed = stats.FilesIndexed
			result.ChunksCreated = stats.ChunksCreated
		}
		p.Send(result)
	}()

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("progress TUI: %w", err)
	}

	result := finalModel.(tui.ProgressModel).GetResult()
	if result.Err != nil {
		return result.Err
	}

	return nil
}

func buildNonInteractiveResult(cmd *cobra.Command) tui.InitResult {
	provider, _ := cmd.Flags().GetString("provider")
	model, _ := cmd.Flags().GetString("model")
	endpoint, _ := cmd.Flags().GetString("endpoint")
	apiKey, _ := cmd.Flags().GetString("api-key")
	llmModel, _ := cmd.Flags().GetString("llm-model")

	if provider == "" {
		provider = "ollama"
	}

	embedDefaults := config.DefaultEmbedderForProvider(provider)
	llmDefaults := config.DefaultLLMForProvider(provider)

	if model == "" {
		model = embedDefaults.Model
	}
	if endpoint == "" {
		endpoint = embedDefaults.Endpoint
	}
	if llmModel == "" {
		llmModel = llmDefaults.Model
	}

	// Use the user-provided endpoint for LLM too; fall back to default only
	// when the user didn't supply one (i.e. endpoint came from embedder defaults).
	llmEndpoint := endpoint
	if llmEndpoint == embedDefaults.Endpoint {
		llmEndpoint = llmDefaults.Endpoint
	}

	return tui.InitResult{
		Provider:    provider,
		Model:       model,
		Endpoint:    endpoint,
		APIKey:      apiKey,
		LLMModel:    llmModel,
		LLMEndpoint: llmEndpoint,
		Done:        true,
	}
}

func buildConfigFromResult(result tui.InitResult) *config.Config {
	cfg := config.DefaultConfig()
	cfg.Embedder.Provider = result.Provider
	cfg.Embedder.Model = result.Model
	cfg.Embedder.Endpoint = result.Endpoint
	cfg.Embedder.APIKey = result.APIKey

	providerDefaults := config.DefaultEmbedderForProvider(result.Provider)
	if providerDefaults.Dimensions != nil {
		dim := *providerDefaults.Dimensions
		cfg.Embedder.Dimensions = &dim
	}

	// LLM config for ask/explain
	cfg.LLM.Provider = result.Provider
	cfg.LLM.Model = result.LLMModel
	if result.LLMEndpoint != "" {
		cfg.LLM.Endpoint = result.LLMEndpoint
	}
	if result.Provider == "openai" {
		cfg.LLM.APIKey = result.APIKey
	}

	return cfg
}
