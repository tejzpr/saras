/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2026 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.AddCommand(installSkillCmd)
	installSkillCmd.Flags().Bool("cursor", false, "Install skill and rule for Cursor")
	installSkillCmd.Flags().Bool("windsurf", false, "Install skill for Windsurf")
	installSkillCmd.Flags().Bool("claude", false, "Install skill for Claude Code")
	installSkillCmd.Flags().Bool("codex", false, "Install skill for OpenAI Codex")
	installSkillCmd.Flags().Bool("copilot", false, "Install skill for GitHub Copilot")
	installSkillCmd.Flags().Bool("global", false, "Install skill globally to ~/.ide/ instead of the project directory")
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install saras integrations",
}

var installSkillCmd = &cobra.Command{
	Use:   "skill",
	Short: "Install a saras skill for an AI coding agent",
	Long: `Install a skill file that teaches an AI coding agent how to use saras
for codebase search, Q&A, symbol tracing, and architecture mapping.

The skill name and folder are derived from the current directory name so that
the skill matches the project (e.g. directory "myapp" → skill name "myapp").

Pass one or more editor flags to specify which agents to install for:

  --cursor     .cursor/skills/<project>/SKILL.md + .cursor/rules/<project>.mdc
  --windsurf   .windsurf/skills/<project>/SKILL.md
  --claude     .claude/skills/<project>/SKILL.md
  --codex      .agents/skills/<project>/SKILL.md
  --copilot    .github/skills/<project>/SKILL.md + .github/copilot-instructions.md

By default, skills are installed in the current project directory. Use --global
to install to the user's home directory (~/.cursor/, ~/.windsurf/, etc.) so the
skill is available across all projects.

Examples:
  saras install skill --claude
  saras install skill --cursor --global
  saras install skill --windsurf --codex`,
	RunE: runInstallSkill,
}

type editorSkill struct {
	name    string
	path    string
	content string
}

func runInstallSkill(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	projectName := filepath.Base(cwd)

	global, _ := cmd.Flags().GetBool("global")

	// Determine base directory: project-local (cwd) or global (~/)
	baseDir := cwd
	if global {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("get home directory: %w", err)
		}
		baseDir = home
	}

	editors := []editorSkill{
		{"claude", filepath.Join(baseDir, ".claude", "skills", projectName, "SKILL.md"), skillContentAgentSkills(projectName)},
		{"codex", filepath.Join(baseDir, ".agents", "skills", projectName, "SKILL.md"), skillContentAgentSkills(projectName)},
		{"windsurf", filepath.Join(baseDir, ".windsurf", "skills", projectName, "SKILL.md"), skillContentAgentSkills(projectName)},
		{"cursor", filepath.Join(baseDir, ".cursor", "skills", projectName, "SKILL.md"), skillContentAgentSkills(projectName)},
		{"copilot", filepath.Join(baseDir, ".github", "skills", projectName, "SKILL.md"), skillContentAgentSkills(projectName)},
	}

	// Cursor also gets a rule file (.mdc) in addition to the skill
	cursorRule := editorSkill{
		"cursor",
		filepath.Join(baseDir, ".cursor", "rules", projectName+".mdc"),
		skillContentCursor(),
	}

	// Copilot also gets a copilot-instructions.md in addition to the skill
	copilotInstructions := editorSkill{
		"copilot",
		filepath.Join(baseDir, ".github", "copilot-instructions.md"),
		"",
	}

	installed := 0
	for _, ed := range editors {
		flag, _ := cmd.Flags().GetBool(ed.name)
		if !flag {
			continue
		}

		if err := installSkillFile(cmd, ed.name, ed.path, ed.content); err != nil {
			return err
		}

		// Install the additional Cursor rule file
		if ed.name == "cursor" {
			if err := installSkillFile(cmd, "cursor (rule)", cursorRule.path, cursorRule.content); err != nil {
				return err
			}
		}

		// Install the additional Copilot instructions file
		if ed.name == "copilot" {
			content := skillContentCopilot(copilotInstructions.path)
			if err := installSkillFile(cmd, "copilot (instructions)", copilotInstructions.path, content); err != nil {
				return err
			}
		}

		installed++
	}

	if installed == 0 {
		return fmt.Errorf("specify at least one editor: --cursor, --windsurf, --claude, --codex, --copilot")
	}
	return nil
}

func installSkillFile(cmd *cobra.Command, agent, targetPath, content string) error {
	// Create parent directories
	dir := filepath.Dir(targetPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}

	// For copilot, append if file already exists
	if agent == "copilot" {
		if existing, err := os.ReadFile(targetPath); err == nil {
			if strings.Contains(string(existing), "# Saras Codebase Intelligence") {
				fmt.Fprintf(cmd.OutOrStdout(), "Saras skill already exists in %s\n", targetPath)
				return nil
			}
			content = string(existing) + "\n\n" + content
		}
	} else {
		if _, err := os.Stat(targetPath); err == nil {
			fmt.Fprintf(cmd.OutOrStdout(), "Overwriting existing skill at %s\n", targetPath)
		}
	}

	if err := os.WriteFile(targetPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write skill file: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Installed saras skill for %s at %s\n", agent, targetPath)
	return nil
}

// skillContentAgentSkills returns the SKILL.md content following the Agent Skills spec
// (used by Claude Code, OpenAI Codex, and Windsurf).
func skillContentAgentSkills(projectName string) string {
	return `---
name: ` + projectName + `
description: Uses saras CLI to search, ask questions about, trace symbols in, map the
  architecture of, and visualize execution flow in a codebase. Use when user asks to
  "search the code", "find where this is defined", "explain how this works", "trace this
  function", "show me the architecture", "what calls this", "understand this codebase",
  "how does this feature work", "generate an architecture map", "show the execution flow",
  or "what does main call". Requires saras to be initialized in the project.
---
## Searching Code

Use semantic search to find code.

` + "```bash" + `
saras search "authentication middleware" --limit 10 --json
` + "```" + `

- Always use --json for machine-readable output you can parse
- Use --limit to control how many results (default 5)
- Phrase queries as natural language for best results

## Asking Questions

For questions that need explanation.
Ask short individual questions per ask to yield faster answers.

` + "```bash" + `
saras ask --no-tui "how does the payment flow work?"
` + "```" + `

## Tracing Symbols

Find where a symbol is defined, who calls it, and what it calls:

` + "```bash" + `
saras trace HandleRequest
saras trace HandleRequest --callers
saras trace HandleRequest --callees
` + "```" + `

- Use exact symbol names (case-sensitive)

## Architecture Maps

Generate a structural overview of the project:

` + "```bash" + `
saras map --format summary
saras map --format markdown
saras map --format tree
` + "```" + `

## Execution Flow

Visualize call trees from entry points (main, CLI handlers, HTTP handlers):

` + "```bash" + `
saras flow                    # all entry points
saras flow full               # same as above
saras flow HandleRequest      # from a specific function
saras flow --depth 3          # limit depth (default: 8)
saras flow explain            # LLM-powered explanation of the flow
saras flow explain runSearch  # explain a specific function's flow
` + "```" + `

- Works across all supported languages
- Markers: (cycle), (↩) already expanded, (...) depth limit

## Important
- Do not run saras watch (blocking)
- No results: run saras init or saras watch
- Outputs include paths, lines, relevance
- Ask is stateless
- Runs locally
`
}

// skillContentCursor returns the .mdc content for Cursor rules.
func skillContentCursor() string {
	return `---
description: Uses saras CLI for codebase search, Q&A, symbol tracing, architecture
  mapping, and execution flow visualization. Use when user asks to search code, explain
  how something works, trace a function, show architecture, visualize execution flow,
  or understand the codebase. Requires saras to be initialized.
alwaysApply: false
---
## Searching Code

Use semantic search to find code.

` + "```bash" + `
saras search "authentication middleware" --limit 10 --json
` + "```" + `

- Always use --json for machine-readable output you can parse
- Use --limit to control how many results (default 5)
- Phrase queries as natural language for best results

## Asking Questions

For questions that need explanation.
Ask short individual questions per ask to yield faster answers.

` + "```bash" + `
saras ask --no-tui "how does the payment flow work?"
` + "```" + `

## Tracing Symbols

Find where a symbol is defined, who calls it, and what it calls:

` + "```bash" + `
saras trace HandleRequest
saras trace HandleRequest --callers
saras trace HandleRequest --callees
` + "```" + `

- Use exact symbol names (case-sensitive)

## Architecture Maps

Generate a structural overview of the project:

` + "```bash" + `
saras map --format summary
saras map --format markdown
saras map --format tree
` + "```" + `

## Execution Flow

Visualize call trees from entry points (main, CLI handlers, HTTP handlers):

` + "```bash" + `
saras flow                    # all entry points
saras flow full               # same as above
saras flow HandleRequest      # from a specific function
saras flow --depth 3          # limit depth (default: 8)
saras flow explain            # LLM-powered explanation of the flow
saras flow explain runSearch  # explain a specific function's flow
` + "```" + `

- Works across all supported languages
- Markers: (cycle), (↩) already expanded, (...) depth limit

## Important
- Do not run saras watch (blocking)
- No results: run saras init or saras watch
- Outputs include paths, lines, relevance
- Ask is stateless
- Runs locally
`
}

// skillContentCopilot returns plain markdown for GitHub Copilot instructions.
func skillContentCopilot(path string) string {
	return `# Saras Codebase Intelligence

## Searching Code

Use semantic search to find code.

` + "```bash" + `
saras search "authentication middleware" --limit 10 --json
` + "```" + `

- Always use --json for machine-readable output you can parse
- Use --limit to control how many results (default 5)
- Phrase queries as natural language for best results

## Asking Questions

For questions that need explanation.
Ask short individual questions per ask to yield faster answers.

` + "```bash" + `
saras ask --no-tui "how does the payment flow work?"
` + "```" + `

## Tracing Symbols

Find where a symbol is defined, who calls it, and what it calls:

` + "```bash" + `
saras trace HandleRequest
saras trace HandleRequest --callers
saras trace HandleRequest --callees
` + "```" + `

- Use exact symbol names (case-sensitive)

## Architecture Maps

Generate a structural overview of the project:

` + "```bash" + `
saras map --format summary
saras map --format markdown
saras map --format tree
` + "```" + `

## Execution Flow

Visualize call trees from entry points (main, CLI handlers, HTTP handlers):

` + "```bash" + `
saras flow                    # all entry points
saras flow full               # same as above
saras flow HandleRequest      # from a specific function
saras flow --depth 3          # limit depth (default: 8)
saras flow explain            # LLM-powered explanation of the flow
saras flow explain runSearch  # explain a specific function's flow
` + "```" + `

- Works across all supported languages
- Markers: (cycle), (↩) already expanded, (...) depth limit

## Important
- Do not run saras watch (blocking)
- No results: run saras init or saras watch
- Outputs include paths, lines, relevance
- Ask is stateless
- Runs locally
`
}
