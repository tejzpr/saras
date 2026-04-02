/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2026 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package tui

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tejzpr/saras/internal/config"
)

// InitResult holds the user's choices from the init wizard.
type InitResult struct {
	Provider    string
	Model       string
	Endpoint    string
	APIKey      string
	LLMModel    string
	LLMEndpoint string
	Done        bool
	Aborted     bool
}

type initStep int

const (
	stepProvider initStep = iota
	stepEndpoint
	stepAPIKey
	stepModel
	stepLLMModel
	stepConfirm
)

// ollamaModelsMsg is sent when Ollama model list is fetched.
type ollamaModelsMsg struct {
	models []string
	err    error
}

// fetchOllamaModels fetches the model list from Ollama's /api/tags endpoint.
func fetchOllamaModels(endpoint string) tea.Cmd {
	return func() tea.Msg {
		endpoint = strings.TrimRight(endpoint, "/")
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(endpoint + "/api/tags")
		if err != nil {
			return ollamaModelsMsg{err: err}
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return ollamaModelsMsg{err: err}
		}

		var result struct {
			Models []struct {
				Name string `json:"name"`
			} `json:"models"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return ollamaModelsMsg{err: err}
		}

		var names []string
		for _, m := range result.Models {
			names = append(names, m.Name)
		}
		sort.Strings(names)
		return ollamaModelsMsg{models: names}
	}
}

// InitModel is the Bubble Tea model for the interactive init wizard.
type InitModel struct {
	step       initStep
	provider   int      // selected provider index
	providers  []string // provider names
	modelInput textinput.Model
	endpInput  textinput.Model
	keyInput   textinput.Model
	llmInput   textinput.Model
	result     InitResult
	width      int
	height     int
	quitting   bool

	// Ollama model selection
	ollamaModels   []string // all fetched model names
	embedModels    []string // filtered: models with "embed" in name
	llmModels      []string // filtered: models without "embed"
	embedCursor    int      // cursor for embed model selection
	llmCursor      int      // cursor for LLM model selection
	fetchingModels bool     // true while fetching
	fetchErr       error    // set if fetch failed
}

// NewInitModel creates a new init wizard model.
func NewInitModel() InitModel {
	mi := textinput.New()
	mi.Placeholder = "nomic-embed-text"
	mi.CharLimit = 100
	mi.Width = 50

	ei := textinput.New()
	ei.Placeholder = "http://localhost:11434"
	ei.CharLimit = 200
	ei.Width = 50

	ki := textinput.New()
	ki.Placeholder = "(optional, press Enter to skip)"
	ki.CharLimit = 200
	ki.Width = 50
	ki.EchoMode = textinput.EchoPassword
	ki.EchoCharacter = '•'

	li := textinput.New()
	li.Placeholder = "llama3.2"
	li.CharLimit = 100
	li.Width = 50

	return InitModel{
		step:       stepProvider,
		providers:  []string{"ollama", "lmstudio", "openai"},
		provider:   0,
		modelInput: mi,
		endpInput:  ei,
		keyInput:   ki,
		llmInput:   li,
	}
}

// GetResult returns the init wizard result.
func (m InitModel) GetResult() InitResult {
	return m.result
}

func (m InitModel) Init() tea.Cmd {
	return nil
}

func (m InitModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case ollamaModelsMsg:
		m.fetchingModels = false
		if msg.err != nil {
			m.fetchErr = msg.err
		} else {
			m.ollamaModels = msg.models
			m.embedModels = nil
			m.llmModels = nil
			for _, name := range msg.models {
				if strings.Contains(strings.ToLower(name), "embed") {
					m.embedModels = append(m.embedModels, name)
				} else {
					m.llmModels = append(m.llmModels, name)
				}
			}
		}
		// We're at stepModel now, reset cursor
		m.embedCursor = 0
		m.llmCursor = 0
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.result.Aborted = true
			m.quitting = true
			return m, tea.Quit

		case "q":
			if m.step == stepProvider || m.step == stepConfirm {
				m.result.Aborted = true
				m.quitting = true
				return m, tea.Quit
			}
		}

		switch m.step {
		case stepProvider:
			return m.updateProvider(msg)
		case stepModel:
			return m.updateModel(msg)
		case stepEndpoint:
			return m.updateEndpoint(msg)
		case stepAPIKey:
			return m.updateAPIKey(msg)
		case stepLLMModel:
			return m.updateLLMModel(msg)
		case stepConfirm:
			return m.updateConfirm(msg)
		}
	}

	return m, nil
}

func (m InitModel) updateProvider(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.provider > 0 {
			m.provider--
		}
	case "down", "j":
		if m.provider < len(m.providers)-1 {
			m.provider++
		}
	case "enter":
		m.result.Provider = m.providers[m.provider]
		embedDefaults := config.DefaultEmbedderForProvider(m.result.Provider)
		m.modelInput.SetValue(embedDefaults.Model)
		m.modelInput.Placeholder = embedDefaults.Model
		m.endpInput.SetValue(embedDefaults.Endpoint)
		m.endpInput.Placeholder = embedDefaults.Endpoint
		llmDefaults := config.DefaultLLMForProvider(m.result.Provider)
		m.llmInput.SetValue(llmDefaults.Model)
		m.llmInput.Placeholder = llmDefaults.Model
		m.step = stepEndpoint
		m.endpInput.Focus()
		return m, textinput.Blink
	}
	return m, nil
}

func (m InitModel) updateModel(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// If we have Ollama embed models, use list selection
	if len(m.embedModels) > 0 {
		return m.updateModelList(msg)
	}
	// Fallback: text input
	switch msg.String() {
	case "enter":
		val := m.modelInput.Value()
		if val == "" {
			val = m.modelInput.Placeholder
		}
		m.result.Model = val
		m.modelInput.Blur()
		m.step = stepLLMModel
		if len(m.llmModels) > 0 {
			return m, nil
		}
		m.llmInput.Focus()
		return m, textinput.Blink
	}
	var cmd tea.Cmd
	m.modelInput, cmd = m.modelInput.Update(msg)
	return m, cmd
}

func (m InitModel) updateModelList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// embedModels + "Type custom name..." option
	total := len(m.embedModels) + 1
	switch msg.String() {
	case "up", "k":
		if m.embedCursor > 0 {
			m.embedCursor--
		}
	case "down", "j":
		if m.embedCursor < total-1 {
			m.embedCursor++
		}
	case "enter":
		if m.embedCursor < len(m.embedModels) {
			m.result.Model = m.embedModels[m.embedCursor]
		} else {
			// "Custom" selected — switch to text input
			m.embedModels = nil // clear to fall back to text input
			m.modelInput.Focus()
			return m, textinput.Blink
		}
		m.step = stepLLMModel
		if len(m.llmModels) > 0 {
			return m, nil
		}
		m.llmInput.Focus()
		return m, textinput.Blink
	}
	return m, nil
}

func (m InitModel) updateEndpoint(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		val := m.endpInput.Value()
		if val == "" {
			val = m.endpInput.Placeholder
		}
		m.result.Endpoint = val
		m.endpInput.Blur()

		if m.result.Provider == "openai" {
			m.step = stepAPIKey
			m.keyInput.Focus()
			return m, textinput.Blink
		}
		// For Ollama/LMStudio: fetch models and go to embed model step
		m.step = stepModel
		if m.result.Provider == "ollama" {
			m.fetchingModels = true
			m.fetchErr = nil
			m.ollamaModels = nil
			return m, fetchOllamaModels(val)
		}
		m.modelInput.Focus()
		return m, textinput.Blink
	}
	var cmd tea.Cmd
	m.endpInput, cmd = m.endpInput.Update(msg)
	return m, cmd
}

func (m InitModel) updateAPIKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.result.APIKey = m.keyInput.Value()
		m.keyInput.Blur()
		m.step = stepModel
		m.modelInput.Focus()
		return m, textinput.Blink
	}
	var cmd tea.Cmd
	m.keyInput, cmd = m.keyInput.Update(msg)
	return m, cmd
}

func (m InitModel) updateLLMModel(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// If we have Ollama LLM models, use list selection
	if len(m.llmModels) > 0 {
		return m.updateLLMModelList(msg)
	}
	// Fallback: text input
	switch msg.String() {
	case "enter":
		val := m.llmInput.Value()
		if val == "" {
			val = m.llmInput.Placeholder
		}
		m.result.LLMModel = val
		llmDefaults := config.DefaultLLMForProvider(m.result.Provider)
		m.result.LLMEndpoint = llmDefaults.Endpoint
		m.llmInput.Blur()
		m.step = stepConfirm
		return m, nil
	}
	var cmd tea.Cmd
	m.llmInput, cmd = m.llmInput.Update(msg)
	return m, cmd
}

func (m InitModel) updateLLMModelList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// llmModels + "Type custom name..." option
	total := len(m.llmModels) + 1
	switch msg.String() {
	case "up", "k":
		if m.llmCursor > 0 {
			m.llmCursor--
		}
	case "down", "j":
		if m.llmCursor < total-1 {
			m.llmCursor++
		}
	case "enter":
		if m.llmCursor < len(m.llmModels) {
			m.result.LLMModel = m.llmModels[m.llmCursor]
		} else {
			// "Custom" selected — switch to text input
			m.llmModels = nil // clear to fall back to text input
			m.llmInput.Focus()
			return m, textinput.Blink
		}
		llmDefaults := config.DefaultLLMForProvider(m.result.Provider)
		m.result.LLMEndpoint = llmDefaults.Endpoint
		m.step = stepConfirm
		return m, nil
	}
	return m, nil
}

func (m InitModel) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		m.result.Done = true
		m.quitting = true
		return m, tea.Quit
	case "n", "N":
		m.result.Aborted = true
		m.quitting = true
		return m, tea.Quit
	case "b":
		m.step = stepProvider
		return m, nil
	}
	return m, nil
}

func (m InitModel) View() string {
	if m.quitting {
		if m.result.Done {
			return SuccessStyle.Render(SymbolCheck+" Project initialized successfully!") + "\n"
		}
		return WarningStyle.Render("Init cancelled.") + "\n"
	}

	var b strings.Builder

	// Header
	b.WriteString(LogoStyle.Render(LogoText) + "\n")
	b.WriteString(TaglineStyle.Render("  AI-native codebase intelligence") + "\n\n")

	// Progress indicator
	steps := []string{"Provider", "Endpoint", "API Key", "Embed Model", "LLM Model", "Confirm"}
	b.WriteString(m.renderProgress(steps) + "\n\n")

	switch m.step {
	case stepProvider:
		b.WriteString(m.renderProviderStep())
	case stepEndpoint:
		b.WriteString(m.renderEndpointStep())
	case stepAPIKey:
		b.WriteString(m.renderAPIKeyStep())
	case stepModel:
		b.WriteString(m.renderModelStep())
	case stepLLMModel:
		b.WriteString(m.renderLLMModelStep())
	case stepConfirm:
		b.WriteString(m.renderConfirmStep())
	}

	// Footer help
	b.WriteString("\n" + m.renderHelp())

	return b.String()
}

func (m InitModel) renderProgress(steps []string) string {
	var parts []string
	for i, name := range steps {
		if i == int(m.step) {
			parts = append(parts, FocusedStyle.Render(fmt.Sprintf("[%s]", name)))
		} else if i < int(m.step) {
			parts = append(parts, SuccessStyle.Render(fmt.Sprintf(" %s %s ", SymbolCheck, name)))
		} else {
			parts = append(parts, DimStyle.Render(fmt.Sprintf(" %s ", name)))
		}
	}
	return strings.Join(parts, DimStyle.Render(" "+SymbolArrow+" "))
}

func (m InitModel) renderProviderStep() string {
	var b strings.Builder
	b.WriteString(TitleStyle.Render("Choose your embedding provider") + "\n\n")

	descriptions := map[string]string{
		"ollama":   "Local inference, free, runs on your machine (recommended)",
		"lmstudio": "Local inference via LM Studio desktop app",
		"openai":   "OpenAI API or any OpenAI-compatible endpoint",
	}

	for i, p := range m.providers {
		cursor := "  "
		style := ItemStyle
		if i == m.provider {
			cursor = FocusedStyle.Render(SymbolArrow + " ")
			style = ActiveItemStyle
		}
		name := style.Render(p)
		desc := MutedStyle.Render(" " + SymbolDot + " " + descriptions[p])
		b.WriteString(fmt.Sprintf("%s%s%s\n", cursor, name, desc))
	}

	return b.String()
}

func (m InitModel) renderModelStep() string {
	var b strings.Builder
	b.WriteString(TitleStyle.Render("Embedding model") + "\n")
	b.WriteString(MutedStyle.Render(fmt.Sprintf("Provider: %s | Endpoint: %s", m.result.Provider, m.result.Endpoint)) + "\n\n")

	if m.fetchingModels {
		b.WriteString(MutedStyle.Render("  Fetching models from Ollama...") + "\n")
		return b.String()
	}

	if len(m.embedModels) > 0 {
		b.WriteString(MutedStyle.Render("  Select an embedding model:") + "\n\n")
		for i, name := range m.embedModels {
			cursor := "  "
			style := ItemStyle
			if i == m.embedCursor {
				cursor = FocusedStyle.Render(SymbolArrow + " ")
				style = ActiveItemStyle
			}
			b.WriteString(fmt.Sprintf("%s%s\n", cursor, style.Render(name)))
		}
		// "Custom" option
		cursor := "  "
		style := ItemStyle
		if m.embedCursor == len(m.embedModels) {
			cursor = FocusedStyle.Render(SymbolArrow + " ")
			style = ActiveItemStyle
		}
		b.WriteString(fmt.Sprintf("%s%s\n", cursor, style.Render("Type custom name...")))
		return b.String()
	}

	if m.fetchErr != nil {
		b.WriteString(WarningStyle.Render(fmt.Sprintf("  Could not fetch models: %v", m.fetchErr)) + "\n")
		b.WriteString(MutedStyle.Render("  Type model name manually:") + "\n\n")
	}

	b.WriteString(m.modelInput.View() + "\n")
	return b.String()
}

func (m InitModel) renderEndpointStep() string {
	var b strings.Builder
	b.WriteString(TitleStyle.Render("API endpoint URL") + "\n")
	b.WriteString(MutedStyle.Render(fmt.Sprintf("Provider: %s", m.result.Provider)) + "\n\n")
	b.WriteString(m.endpInput.View() + "\n")
	return b.String()
}

func (m InitModel) renderAPIKeyStep() string {
	var b strings.Builder
	b.WriteString(TitleStyle.Render("API Key") + "\n")
	b.WriteString(MutedStyle.Render("Required for OpenAI, optional for compatible APIs") + "\n\n")
	b.WriteString(m.keyInput.View() + "\n")
	return b.String()
}

func (m InitModel) renderLLMModelStep() string {
	var b strings.Builder
	b.WriteString(TitleStyle.Render("LLM model for chat / ask") + "\n")
	b.WriteString(MutedStyle.Render(fmt.Sprintf("Provider: %s (used for 'saras ask' and AGENTS.md generation)", m.result.Provider)) + "\n\n")

	if len(m.llmModels) > 0 {
		b.WriteString(MutedStyle.Render("  Select a chat/LLM model:") + "\n\n")
		for i, name := range m.llmModels {
			cursor := "  "
			style := ItemStyle
			if i == m.llmCursor {
				cursor = FocusedStyle.Render(SymbolArrow + " ")
				style = ActiveItemStyle
			}
			b.WriteString(fmt.Sprintf("%s%s\n", cursor, style.Render(name)))
		}
		// "Custom" option
		cursor := "  "
		style := ItemStyle
		if m.llmCursor == len(m.llmModels) {
			cursor = FocusedStyle.Render(SymbolArrow + " ")
			style = ActiveItemStyle
		}
		b.WriteString(fmt.Sprintf("%s%s\n", cursor, style.Render("Type custom name...")))
		return b.String()
	}

	b.WriteString(m.llmInput.View() + "\n")
	return b.String()
}

func (m InitModel) renderConfirmStep() string {
	var b strings.Builder
	b.WriteString(TitleStyle.Render("Review configuration") + "\n\n")

	labelStyle := lipgloss.NewStyle().Foreground(ColorSecondary).Width(16)

	b.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("Provider:"), m.result.Provider))
	b.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("Embed Model:"), m.result.Model))
	b.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("LLM Model:"), m.result.LLMModel))
	b.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("Endpoint:"), m.result.Endpoint))
	if m.result.APIKey != "" {
		b.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("API Key:"), "••••••••"))
	}

	b.WriteString("\n")
	b.WriteString(MutedStyle.Render("  This will create .saras/config.yaml in the current directory.") + "\n\n")
	b.WriteString(FocusedStyle.Render("  Proceed? ") + MutedStyle.Render("[y]es / [n]o / [b]ack") + "\n")

	return b.String()
}

func (m InitModel) renderHelp() string {
	switch m.step {
	case stepProvider:
		return HelpKeyStyle.Render("↑↓") + HelpDescStyle.Render(" select  ") +
			HelpKeyStyle.Render("enter") + HelpDescStyle.Render(" confirm  ") +
			HelpKeyStyle.Render("esc") + HelpDescStyle.Render(" quit")
	case stepModel:
		if len(m.embedModels) > 0 {
			return HelpKeyStyle.Render("↑↓") + HelpDescStyle.Render(" select  ") +
				HelpKeyStyle.Render("enter") + HelpDescStyle.Render(" confirm  ") +
				HelpKeyStyle.Render("esc") + HelpDescStyle.Render(" quit")
		}
		return HelpKeyStyle.Render("enter") + HelpDescStyle.Render(" next  ") +
			HelpKeyStyle.Render("esc") + HelpDescStyle.Render(" quit")
	case stepLLMModel:
		if len(m.llmModels) > 0 {
			return HelpKeyStyle.Render("↑↓") + HelpDescStyle.Render(" select  ") +
				HelpKeyStyle.Render("enter") + HelpDescStyle.Render(" confirm  ") +
				HelpKeyStyle.Render("esc") + HelpDescStyle.Render(" quit")
		}
		return HelpKeyStyle.Render("enter") + HelpDescStyle.Render(" next  ") +
			HelpKeyStyle.Render("esc") + HelpDescStyle.Render(" quit")
	case stepConfirm:
		return HelpKeyStyle.Render("y") + HelpDescStyle.Render(" confirm  ") +
			HelpKeyStyle.Render("n") + HelpDescStyle.Render(" cancel  ") +
			HelpKeyStyle.Render("b") + HelpDescStyle.Render(" back")
	default:
		return HelpKeyStyle.Render("enter") + HelpDescStyle.Render(" next  ") +
			HelpKeyStyle.Render("esc") + HelpDescStyle.Render(" quit")
	}
}
