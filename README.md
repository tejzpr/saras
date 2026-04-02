<h1 align="center">Saras</h1>

<p align="center">
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go" alt="Go Version"></a>
  <a href="https://opensource.org/licenses/MPL-2.0"><img src="https://img.shields.io/badge/License-MPL%202.0-brightgreen.svg" alt="License: MPL 2.0"></a>
  <a href="https://github.com/tejzpr/saras/actions"><img src="https://github.com/tejzpr/saras/actions/workflows/codeql-analysis.yml/badge.svg" alt="CodeQL"></a>
  <a href="https://github.com/tejzpr/saras/releases"><img src="https://img.shields.io/github/v/release/tejzpr/saras?style=flat" alt="Release"></a>
</p>

**Codebase intelligence for AI agents and developers** — reduce LLM context length by giving your agent precise, relevant code instead of entire files.

Saras indexes your codebase into vector embeddings and exposes semantic search, RAG-powered Q&A, symbol tracing, and architecture mapping — all accessible via CLI or [MCP](https://modelcontextprotocol.io/). Instead of stuffing thousands of lines into an LLM's context window, agents can query saras to retrieve only the symbols, definitions, and code paths they actually need.

## Why Saras?

AI coding agents today hit context window limits fast. A single "explain how auth works" prompt can pull in dozens of files. Saras solves this by acting as a **local codebase knowledge layer**:

- **Smaller context, better answers** — Agents retrieve targeted code chunks via semantic search instead of dumping whole files into the prompt
- **Symbol-level precision** — Trace a function's callers/callees across your entire codebase without loading everything into context
- **Architecture-aware** — Agents can get a structural overview of the project in a few hundred tokens instead of browsing every directory
- **Works with any LLM** — Ollama, LM Studio, OpenAI, or any OpenAI-compatible endpoint

## Features

- **Multi-Language** — Symbol-aware indexing for Go, Python, JavaScript, TypeScript, Java, C, C++, C#, Rust, Kotlin, Ruby, PHP, Zig, CSS, HTML, XML (pluggable for more)
- **Semantic Search** — Find code by meaning, not just keywords
- **Ask** — Ask natural language questions about your codebase (RAG pipeline)
- **Trace** — Track symbol definitions, references, callers, and callees across all supported languages
- **Map** — Generate architecture maps with package structure and dependencies
- **Watch** — Live file watcher with automatic re-indexing
- **MCP Server** — Expose all tools to AI agents via Model Context Protocol

## Prerequisites

Saras requires an **embedding model** (to index your code) and an **LLM** (for Ask and AGENTS.md generation). You can use any of these providers:

| Provider | Setup |
|----------|-------|
| [Ollama](https://ollama.com/) (recommended) | `ollama pull qwen3-embedding:0.6b && ollama pull qwen3.5:2b` |
| [LM Studio](https://lmstudio.ai/) | Download an embedding model and a chat model |
| OpenAI-compatible | Any endpoint that speaks the OpenAI API (requires API key) |

## Quick Start

### Install

```bash
# From source
go install github.com/tejzpr/saras/cmd/saras@latest

# Or download binary
curl -sSfL https://raw.githubusercontent.com/tejzpr/saras/main/install.sh | bash
```

### Update

```bash
saras update
```

Checks GitHub for the latest release and updates the binary in-place. No update is performed if you're already on the latest version.

### Initialize

```bash
cd your-project
saras init
```

This creates a `.saras/` directory with configuration. The interactive setup lets you choose your embedding provider (Ollama, LM Studio, or OpenAI-compatible). When Ollama is selected, saras auto-fetches your installed models and presents them as selectable lists for both embedding and LLM models.

### Search

```bash
saras search "authentication flow"
saras search "database connection" --limit 20
saras search "error handling" --json
```

### Ask

```bash
saras ask "how does the login flow work?"
saras ask "what database connections are used?" --no-tui
```

With `--no-tui`, responses stream to stdout in real-time as the LLM generates them.

### Trace

```bash
saras trace Login
saras trace handleRequest --callers
saras trace NewDB --callees
```

### Map

```bash
saras map                          # directory tree
saras map --format markdown        # full architecture report
saras map --format summary         # compact overview
saras map -f markdown -o ARCH.md   # write to file
```

### Watch

```bash
saras watch                    # live TUI dashboard
saras watch --no-tui           # log mode
saras watch --index-first=false  # skip initial index
```

### MCP Server

Expose saras tools to AI agents via the Model Context Protocol (SSE transport).

```bash
saras serve                      # start on 127.0.0.1:9420
saras serve --addr 0.0.0.0:8080 # custom address
```

**SSE endpoints:**

| Method | Path | Description |
|--------|------|-------------|
| GET | `/sse` | SSE connection for MCP clients |
| POST | `/message` | Send JSON-RPC messages to the server |

**Available tools:** `search`, `ask`, `trace`, `map`, `symbols`

The server name advertised to MCP clients matches the project directory name.

Connect from Windsurf, Cursor, or any MCP-compatible client:
```
URL: http://127.0.0.1:9420/sse
```

### Install Skills

Install skill files that teach AI coding agents how to use saras:

```bash
saras install skill --windsurf   # .windsurf/skills/<project>/SKILL.md
saras install skill --cursor     # .cursor/skills/<project>/SKILL.md + .cursor/rules/<project>.mdc
saras install skill --claude      # .claude/skills/<project>/SKILL.md
saras install skill --codex       # .agents/skills/<project>/SKILL.md
saras install skill --copilot     # .github/skills/<project>/SKILL.md + .github/copilot-instructions.md
```

The skill name and folder are derived from the current directory name so they match the project.

Use `--global` to install the skill to your home directory (`~/.cursor/`, `~/.windsurf/`, etc.) so it's available across all projects:

```bash
saras install skill --windsurf --global   # ~/.windsurf/skills/<project>/SKILL.md
saras install skill --cursor --global     # ~/.cursor/skills/<project>/SKILL.md + rules
```

### Install AGENTS.md

Generate LLM-powered per-directory `AGENTS.md` files that describe each package's purpose, symbols, and conventions:

```bash
saras install agentsmd                     # generate AGENTS.md files
saras install agentsmd --with-claudemd     # also create CLAUDE.md with @AGENTS.md import
saras install agentsmd --min-files 3       # only for packages with 3+ files (default: 2)
```

## Configuration

Saras stores configuration in `.saras/config.yaml`. Key settings:

```yaml
embedder:
  provider: ollama          # ollama, lmstudio, openai
  endpoint: http://localhost:11434
  model: nomic-embed-text
  # api_key: sk-...        # required for openai

llm:
  provider: ollama          # ollama, lmstudio, openai
  endpoint: http://localhost:11434
  model: llama3.2
  # api_key: sk-...        # required for openai

chunking:
  size: 1500
  overlap: 200

search:
  hybrid:
    enabled: true
    k: 60
  boost:
    enabled: true
  dedup:
    enabled: true

watch:
  debounce_ms: 500

ignore:
  - node_modules
  - vendor
  - .git
```

## Architecture

```
saras/
├── cmd/saras/          # CLI entrypoint
├── internal/
│   ├── architect/      # Codebase map generator
│   ├── ask/            # RAG pipeline (search + LLM)
│   ├── cli/            # Cobra commands
│   ├── config/         # YAML configuration
│   ├── embedder/       # Embedding providers (Ollama, LMStudio, OpenAI)
│   ├── engine/         # Indexer, chunker, scanner, watcher
│   ├── lang/           # Pluggable language parsers (symbol extraction)
│   ├── mcp/            # MCP server
│   ├── search/         # Vector, text, hybrid search with RRF
│   ├── store/          # Vector store (gob backend)
│   ├── trace/          # Multi-language symbol tracing, call graph
│   └── tui/            # Bubble Tea interactive UIs
```

## Development

```bash
make build              # build binary to bin/saras
make install            # go install
make test               # run tests
make test-verbose       # run tests with -v
make test-coverage      # tests + coverage report
make fmt                # gofmt + goimports
make vet                # go vet
make lint               # golangci-lint
make clean              # remove build artifacts
make release            # goreleaser release
make release-snapshot   # goreleaser snapshot (no publish)
make all                # fmt + vet + lint + test + build
```

## Embedding Providers

| Provider | Endpoint | Model Example |
|----------|----------|---------------|
| Ollama | `http://localhost:11434` | `nomic-embed-text` |
| LM Studio | `http://localhost:1234` | `text-embedding-nomic-embed-text-v1.5` |
| OpenAI | `https://api.openai.com` | `text-embedding-3-small` |

## Supported Languages

Built-in symbol-aware parsing for:

| Language | Extensions | Symbols Extracted |
|----------|------------|-------------------|
| Go | `.go` | functions, methods, structs, interfaces, vars, consts |
| Python | `.py`, `.pyi` | functions, methods, classes, async functions, constants |
| JavaScript | `.js`, `.jsx`, `.mjs`, `.cjs` | functions, arrow functions, classes, methods, consts |
| TypeScript | `.ts`, `.tsx` | functions, classes, interfaces, types, enums, methods |
| Java | `.java` | classes, interfaces, enums, methods, constants |
| C | `.c`, `.h` | functions, structs, enums, typedefs, #defines |
| C++ | `.cpp`, `.cc`, `.cxx`, `.hpp`, `.hxx`, `.hh` | classes, structs, namespaces, enums, methods, functions |
| Rust | `.rs` | functions, methods, structs, enums, traits, impl blocks |
| Kotlin | `.kt`, `.kts` | classes, interfaces, enums, functions, methods, type aliases |
| Ruby | `.rb`, `.rake`, `.gemspec` | modules, classes, methods, constants, attributes |
| PHP | `.php`, `.phtml` | namespaces, classes, interfaces, traits, enums, methods, constants |
| C# | `.cs`, `.csx` | namespaces, classes, structs, interfaces, enums, records, methods, properties, delegates |
| CSS | `.css`, `.scss`, `.less`, `.sass` | selectors, variables, keyframes, mixins |
| HTML | `.html`, `.htm` | elements, ids, classes, scripts, styles |
| XML | `.xml`, `.xsl`, `.xsd`, `.svg`, `.plist` | elements, attributes, namespaces |
| Zig | `.zig` | functions, structs, enums, unions, constants |
| Python 2 | `.py2`, `.pyw` | functions, classes, methods, constants |

Unsupported file types still get line-based chunking for search and embedding.

### Adding a New Language

Implement the `lang.LanguageParser` interface and call `lang.Register()`:

```go
package myplugin

import "github.com/tejzpr/saras/internal/lang"

func init() { lang.Register(&MyLangParser{}) }

type MyLangParser struct{}

func (p *MyLangParser) Name() string                         { return "mylang" }
func (p *MyLangParser) Extensions() []string                 { return []string{".ml"} }
func (p *MyLangParser) IsTestFile(path string) bool           { return false }
func (p *MyLangParser) ExtractSymbols(content string) []lang.Symbol {
    // your parsing logic here
    return nil
}
```

## License

[MPL-2.0](LICENCE)
