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

	"github.com/spf13/cobra"
	"github.com/tejzpr/saras/internal/architect"
	"github.com/tejzpr/saras/internal/ask"
	"github.com/tejzpr/saras/internal/config"
	"github.com/tejzpr/saras/internal/embedder"
	"github.com/tejzpr/saras/internal/mcp"
	"github.com/tejzpr/saras/internal/search"
	"github.com/tejzpr/saras/internal/store"
	"github.com/tejzpr/saras/internal/trace"
)

func init() {
	rootCmd.AddCommand(serveCmd)
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the MCP server for AI agent integration",
	Long: `Start an MCP server with SSE transport that exposes saras tools.
AI agents and editors (e.g. Windsurf, Cursor, VS Code) can connect to this
server to search, ask, trace, and map your codebase.

SSE endpoint:     GET  /sse
Message endpoint: POST /message

Examples:
  saras serve
  saras serve --addr 0.0.0.0:9420`,
	RunE: runServe,
}

func init() {
	serveCmd.Flags().String("addr", "127.0.0.1:9420", "Listen address (host:port)")
}

func runServe(cmd *cobra.Command, args []string) error {
	addr, _ := cmd.Flags().GetString("addr")

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

	// Create ask pipeline (optional - may not have chat endpoint)
	var pipeline *ask.Pipeline
	chatEndpoint := buildChatEndpoint(cfg)
	if chatEndpoint != "" {
		pipeline = ask.NewPipeline(searcher, chatEndpoint, cfg.Embedder.Model)
		if cfg.Embedder.APIKey != "" {
			pipeline = ask.NewPipeline(searcher, chatEndpoint, cfg.Embedder.Model,
				ask.WithAPIKey(cfg.Embedder.APIKey))
		}
	}

	// Create tracer and mapper
	tracer := trace.NewTracer(projectRoot, cfg.Ignore)
	mapper := architect.NewMapper(projectRoot, cfg.Ignore)

	// Create and start server
	serverName := filepath.Base(projectRoot)
	server := mcp.NewServer(searcher, pipeline, tracer, mapper, cfg, mcp.WithAddr(addr), mcp.WithName(serverName))

	fmt.Fprintf(cmd.OutOrStdout(), "Saras MCP server listening on %s\n", addr)
	fmt.Fprintf(cmd.OutOrStdout(), "Tools: search, ask, trace, map, symbols\n")

	return server.Serve()
}
