/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2026 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package embedder

import (
	"context"
	"fmt"
	"time"

	"github.com/tejzpr/saras/internal/config"
)

// ProbeDimensions makes a single embedding call with sample text to detect the
// model's actual output dimensions. Returns the vector length or an error.
func ProbeDimensions(ctx context.Context, provider, model, endpoint, apiKey string) (int, error) {
	tmpCfg := &config.Config{
		Embedder: config.EmbedderConfig{
			Provider: provider,
			Model:    model,
			Endpoint: endpoint,
			APIKey:   apiKey,
		},
	}

	emb, err := NewFromConfig(tmpCfg)
	if err != nil {
		return 0, fmt.Errorf("create embedder: %w", err)
	}
	defer emb.Close()

	probeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	vec, err := emb.Embed(probeCtx, "dimension probe")
	if err != nil {
		return 0, fmt.Errorf("probe embed call: %w", err)
	}
	if len(vec) == 0 {
		return 0, fmt.Errorf("probe returned empty vector")
	}
	return len(vec), nil
}

// NewFromConfig creates an Embedder from the project configuration.
func NewFromConfig(cfg *config.Config) (Embedder, error) {
	switch cfg.Embedder.Provider {
	case "ollama":
		opts := []OllamaOption{
			WithOllamaEndpoint(cfg.Embedder.Endpoint),
			WithOllamaModel(cfg.Embedder.Model),
		}
		if cfg.Embedder.Dimensions != nil {
			opts = append(opts, WithOllamaDimensions(*cfg.Embedder.Dimensions))
		}
		return NewOllamaEmbedder(opts...), nil

	case "lmstudio":
		opts := []LMStudioOption{
			WithLMStudioEndpoint(cfg.Embedder.Endpoint),
			WithLMStudioModel(cfg.Embedder.Model),
		}
		if cfg.Embedder.Dimensions != nil {
			opts = append(opts, WithLMStudioDimensions(*cfg.Embedder.Dimensions))
		}
		return NewLMStudioEmbedder(opts...), nil

	case "openai":
		opts := []OpenAIOption{
			WithOpenAIEndpoint(cfg.Embedder.Endpoint),
			WithOpenAIModel(cfg.Embedder.Model),
			WithOpenAIKey(cfg.Embedder.APIKey),
		}
		if cfg.Embedder.Dimensions != nil {
			opts = append(opts, WithOpenAIDimensions(*cfg.Embedder.Dimensions))
		}
		return NewOpenAIEmbedder(opts...)

	default:
		return nil, fmt.Errorf("unknown embedding provider: %q", cfg.Embedder.Provider)
	}
}
