/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2026 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewInitModel(t *testing.T) {
	m := NewInitModel()

	if len(m.providers) != 3 {
		t.Errorf("expected 3 providers, got %d", len(m.providers))
	}
	if m.step != stepProvider {
		t.Errorf("expected initial step to be stepProvider")
	}
	if m.provider != 0 {
		t.Errorf("expected initial provider index 0, got %d", m.provider)
	}
}

func TestInitModelProviderNavigation(t *testing.T) {
	m := NewInitModel()

	// Down arrow moves selection
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(InitModel)
	if m.provider != 1 {
		t.Errorf("expected provider 1 after down, got %d", m.provider)
	}

	// Down again
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(InitModel)
	if m.provider != 2 {
		t.Errorf("expected provider 2 after second down, got %d", m.provider)
	}

	// Down at bottom should stay
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(InitModel)
	if m.provider != 2 {
		t.Errorf("expected provider to stay at 2, got %d", m.provider)
	}

	// Up arrow
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(InitModel)
	if m.provider != 1 {
		t.Errorf("expected provider 1 after up, got %d", m.provider)
	}

	// Up at top should stay
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(InitModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(InitModel)
	if m.provider != 0 {
		t.Errorf("expected provider to stay at 0, got %d", m.provider)
	}
}

func TestInitModelProviderSelectionOllama(t *testing.T) {
	m := NewInitModel()

	// Select ollama (index 0) with enter
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(InitModel)

	if m.step != stepEndpoint {
		t.Errorf("expected step to advance to stepEndpoint, got %d", m.step)
	}
	if m.result.Provider != "ollama" {
		t.Errorf("expected provider ollama, got %s", m.result.Provider)
	}
}

func TestInitModelProviderSelectionLMStudio(t *testing.T) {
	m := NewInitModel()

	// Navigate to lmstudio (index 1)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(InitModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(InitModel)

	if m.result.Provider != "lmstudio" {
		t.Errorf("expected provider lmstudio, got %s", m.result.Provider)
	}
}

func TestInitModelProviderSelectionOpenAI(t *testing.T) {
	m := NewInitModel()

	// Navigate to openai (index 2)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(InitModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(InitModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(InitModel)

	if m.result.Provider != "openai" {
		t.Errorf("expected provider openai, got %s", m.result.Provider)
	}
}

func TestInitModelEscapeAborts(t *testing.T) {
	m := NewInitModel()

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(InitModel)

	if !m.result.Aborted {
		t.Error("expected Aborted to be true after Esc")
	}
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestInitModelCtrlCAborts(t *testing.T) {
	m := NewInitModel()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(InitModel)

	if !m.result.Aborted {
		t.Error("expected Aborted to be true after Ctrl+C")
	}
}

func TestInitModelViewRendersLogo(t *testing.T) {
	m := NewInitModel()
	view := m.View()

	if !strings.Contains(view, "║ S ║") {
		t.Error("expected logo in view")
	}
}

func TestInitModelViewProviderStep(t *testing.T) {
	m := NewInitModel()
	view := m.View()

	if !strings.Contains(view, "ollama") {
		t.Error("expected ollama in provider view")
	}
	if !strings.Contains(view, "lmstudio") {
		t.Error("expected lmstudio in provider view")
	}
	if !strings.Contains(view, "openai") {
		t.Error("expected openai in provider view")
	}
	if !strings.Contains(view, "Choose your embedding provider") {
		t.Error("expected provider selection title")
	}
}

func TestInitModelViewConfirmStep(t *testing.T) {
	m := NewInitModel()

	// Walk through to confirm
	m.result.Provider = "ollama"
	m.result.Model = "nomic-embed-text"
	m.result.Endpoint = "http://localhost:11434"
	m.step = stepConfirm

	view := m.View()
	if !strings.Contains(view, "Review configuration") {
		t.Error("expected review title")
	}
	if !strings.Contains(view, "ollama") {
		t.Error("expected provider in confirm view")
	}
	if !strings.Contains(view, "nomic-embed-text") {
		t.Error("expected model in confirm view")
	}
}

func TestInitModelConfirmYes(t *testing.T) {
	m := NewInitModel()
	m.result.Provider = "ollama"
	m.result.Model = "nomic-embed-text"
	m.result.Endpoint = "http://localhost:11434"
	m.step = stepConfirm

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = updated.(InitModel)

	if !m.result.Done {
		t.Error("expected Done to be true after 'y'")
	}
	if m.result.Aborted {
		t.Error("expected Aborted to be false")
	}
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestInitModelConfirmNo(t *testing.T) {
	m := NewInitModel()
	m.step = stepConfirm

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = updated.(InitModel)

	if m.result.Done {
		t.Error("expected Done to be false after 'n'")
	}
	if !m.result.Aborted {
		t.Error("expected Aborted to be true after 'n'")
	}
}

func TestInitModelConfirmBack(t *testing.T) {
	m := NewInitModel()
	m.step = stepConfirm

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	m = updated.(InitModel)

	if m.step != stepProvider {
		t.Errorf("expected step to go back to provider, got %d", m.step)
	}
}

func TestInitModelQuitView(t *testing.T) {
	m := NewInitModel()
	m.quitting = true
	m.result.Done = true

	view := m.View()
	if !strings.Contains(view, "initialized successfully") {
		t.Errorf("expected success message in quit view, got: %s", view)
	}
}

func TestInitModelAbortedView(t *testing.T) {
	m := NewInitModel()
	m.quitting = true
	m.result.Aborted = true

	view := m.View()
	if !strings.Contains(view, "cancelled") {
		t.Errorf("expected cancelled message in quit view, got: %s", view)
	}
}

func TestInitModelWindowSize(t *testing.T) {
	m := NewInitModel()

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(InitModel)

	if m.width != 120 {
		t.Errorf("expected width 120, got %d", m.width)
	}
	if m.height != 40 {
		t.Errorf("expected height 40, got %d", m.height)
	}
}

func TestInitModelVimNavigation(t *testing.T) {
	m := NewInitModel()

	// 'j' should move down
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(InitModel)
	if m.provider != 1 {
		t.Errorf("expected provider 1 after 'j', got %d", m.provider)
	}

	// 'k' should move up
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = updated.(InitModel)
	if m.provider != 0 {
		t.Errorf("expected provider 0 after 'k', got %d", m.provider)
	}
}

func TestInitModelGetResult(t *testing.T) {
	m := NewInitModel()
	m.result = InitResult{
		Provider: "ollama",
		Model:    "test-model",
		Endpoint: "http://test:1234",
		Done:     true,
	}

	r := m.GetResult()
	if r.Provider != "ollama" {
		t.Errorf("expected provider ollama, got %s", r.Provider)
	}
	if r.Model != "test-model" {
		t.Errorf("expected model test-model, got %s", r.Model)
	}
	if !r.Done {
		t.Error("expected Done to be true")
	}
}

func TestInitModelInit(t *testing.T) {
	m := NewInitModel()
	cmd := m.Init()
	if cmd != nil {
		t.Error("expected nil Init command")
	}
}

func TestInitModelOpenAIShowsAPIKeyStep(t *testing.T) {
	m := NewInitModel()

	// Select openai
	m.result.Provider = "openai"
	m.result.Model = "text-embedding-3-small"
	m.result.Endpoint = "https://api.openai.com/v1"
	m.step = stepEndpoint

	// Press enter to advance from endpoint - should go to API key step for openai
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(InitModel)

	if m.step != stepAPIKey {
		t.Errorf("expected API key step for openai, got step %d", m.step)
	}
}

func TestInitModelNonOpenAISkipsAPIKeyStep(t *testing.T) {
	m := NewInitModel()

	// Select ollama
	m.result.Provider = "ollama"
	m.step = stepEndpoint
	m.endpInput.SetValue("http://localhost:11434")

	// Press enter to advance from endpoint - should skip API key for ollama and go to Embed Model
	// Also triggers async fetch
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(InitModel)

	if m.step != stepModel {
		t.Errorf("expected embed model step for ollama (skip API key), got step %d", m.step)
	}
	if !m.fetchingModels {
		t.Error("expected fetchingModels to be true for ollama")
	}
	if cmd == nil {
		t.Error("expected a fetch command for ollama")
	}

	// Simulate fetch error (Ollama not running) — falls back to text input
	updated, _ = m.Update(ollamaModelsMsg{err: fmt.Errorf("connection refused")})
	m = updated.(InitModel)

	if m.fetchingModels {
		t.Error("expected fetchingModels to be false after msg")
	}
	if m.fetchErr == nil {
		t.Error("expected fetchErr to be set")
	}

	// Now text input should work as before
	m.modelInput.SetValue("nomic-embed-text")
	m.modelInput.Focus()
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(InitModel)

	if m.step != stepLLMModel {
		t.Errorf("expected LLM model step after embed model, got step %d", m.step)
	}

	// Press enter to advance from LLM model to confirm
	m.llmInput.SetValue("llama3.2")
	m.llmInput.Focus()
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(InitModel)

	if m.step != stepConfirm {
		t.Errorf("expected confirm step after LLM model, got step %d", m.step)
	}
	if m.result.LLMModel != "llama3.2" {
		t.Errorf("expected LLM model llama3.2, got %s", m.result.LLMModel)
	}
}

func TestInitModelOllamaModelListSelection(t *testing.T) {
	m := NewInitModel()
	m.result.Provider = "ollama"
	m.result.Endpoint = "http://localhost:11434"
	m.step = stepModel
	m.fetchingModels = true

	// Simulate successful fetch with embed + LLM models
	updated, _ := m.Update(ollamaModelsMsg{
		models: []string{"llama3.2:latest", "nomic-embed-text:latest", "qwen3.5:2b"},
	})
	m = updated.(InitModel)

	if len(m.embedModels) != 1 {
		t.Errorf("expected 1 embed model, got %d", len(m.embedModels))
	}
	if len(m.llmModels) != 2 {
		t.Errorf("expected 2 LLM models, got %d", len(m.llmModels))
	}

	// Embed model list: cursor at 0 (first embed model)
	if m.embedCursor != 0 {
		t.Errorf("expected embedCursor 0, got %d", m.embedCursor)
	}

	// Select the first embed model
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(InitModel)

	if m.result.Model != "nomic-embed-text:latest" {
		t.Errorf("expected nomic-embed-text:latest, got %s", m.result.Model)
	}
	if m.step != stepLLMModel {
		t.Errorf("expected stepLLMModel, got step %d", m.step)
	}

	// Navigate LLM model list: down to second model
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(InitModel)
	if m.llmCursor != 1 {
		t.Errorf("expected llmCursor 1, got %d", m.llmCursor)
	}

	// Select second LLM model
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(InitModel)

	if m.result.LLMModel != "qwen3.5:2b" {
		t.Errorf("expected qwen3.5:2b, got %s", m.result.LLMModel)
	}
	if m.step != stepConfirm {
		t.Errorf("expected stepConfirm, got step %d", m.step)
	}
}

func TestInitModelOllamaCustomModelFallback(t *testing.T) {
	m := NewInitModel()
	m.result.Provider = "ollama"
	m.result.Endpoint = "http://localhost:11434"
	m.step = stepModel

	// Simulate fetch with one embed model
	updated, _ := m.Update(ollamaModelsMsg{
		models: []string{"nomic-embed-text:latest", "llama3.2:latest"},
	})
	m = updated.(InitModel)

	// Navigate to "Type custom name..." (index 1 = after the 1 embed model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(InitModel)
	if m.embedCursor != 1 {
		t.Errorf("expected embedCursor 1, got %d", m.embedCursor)
	}

	// Select "Custom" — should switch to text input
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(InitModel)

	if len(m.embedModels) != 0 {
		t.Error("expected embedModels cleared for custom fallback")
	}
	if m.step != stepModel {
		t.Errorf("expected to stay on stepModel for text input, got step %d", m.step)
	}
	if cmd == nil {
		t.Error("expected blink command for text input focus")
	}
}

func TestInitModelOllamaFetchingView(t *testing.T) {
	m := NewInitModel()
	m.result.Provider = "ollama"
	m.result.Endpoint = "http://localhost:11434"
	m.step = stepModel
	m.fetchingModels = true

	view := m.View()
	if !strings.Contains(view, "Fetching models from Ollama") {
		t.Error("expected fetching message in view")
	}
}

func TestInitModelOllamaModelListView(t *testing.T) {
	m := NewInitModel()
	m.result.Provider = "ollama"
	m.result.Endpoint = "http://localhost:11434"
	m.step = stepModel
	m.embedModels = []string{"nomic-embed-text:latest", "mxbai-embed-large:latest"}
	m.embedCursor = 0

	view := m.View()
	if !strings.Contains(view, "nomic-embed-text:latest") {
		t.Error("expected embed model in view")
	}
	if !strings.Contains(view, "mxbai-embed-large:latest") {
		t.Error("expected second embed model in view")
	}
	if !strings.Contains(view, "Type custom name") {
		t.Error("expected custom option in view")
	}
}

func TestInitModelLMStudioNoFetch(t *testing.T) {
	m := NewInitModel()
	m.result.Provider = "lmstudio"
	m.step = stepEndpoint
	m.endpInput.SetValue("http://localhost:1234")

	// Press enter — should NOT trigger fetch for lmstudio
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(InitModel)

	if m.step != stepModel {
		t.Errorf("expected stepModel, got step %d", m.step)
	}
	if m.fetchingModels {
		t.Error("expected fetchingModels to be false for lmstudio")
	}
}

func TestInitModelHelpText(t *testing.T) {
	m := NewInitModel()

	// Provider step help
	view := m.View()
	if !strings.Contains(view, "select") {
		t.Error("expected 'select' in provider help")
	}

	// Confirm step help
	m.step = stepConfirm
	view = m.View()
	if !strings.Contains(view, "confirm") {
		t.Error("expected 'confirm' in confirm help")
	}
}
