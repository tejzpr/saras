/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2026 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package lang

import (
	"path/filepath"
	"strings"
	"sync"
)

// SymbolKind classifies extracted symbols.
type SymbolKind int

const (
	KindFunction SymbolKind = iota
	KindMethod
	KindClass
	KindType
	KindInterface
	KindStruct
	KindEnum
	KindVariable
	KindConstant
	KindImport
	KindPackage
	KindModule
	KindTrait
	KindProperty
)

func (k SymbolKind) String() string {
	switch k {
	case KindFunction:
		return "function"
	case KindMethod:
		return "method"
	case KindClass:
		return "class"
	case KindType:
		return "type"
	case KindInterface:
		return "interface"
	case KindStruct:
		return "struct"
	case KindEnum:
		return "enum"
	case KindVariable:
		return "variable"
	case KindConstant:
		return "constant"
	case KindImport:
		return "import"
	case KindPackage:
		return "package"
	case KindModule:
		return "module"
	case KindTrait:
		return "trait"
	case KindProperty:
		return "property"
	default:
		return "unknown"
	}
}

// Symbol represents a code symbol extracted from source.
type Symbol struct {
	Name      string     `json:"name"`
	Kind      SymbolKind `json:"kind"`
	StartLine int        `json:"start_line"` // 1-indexed
	EndLine   int        `json:"end_line"`   // 1-indexed, inclusive
	Signature string     `json:"signature"`
	Parent    string     `json:"parent,omitempty"`
}

// LanguageParser extracts symbols from source code of a specific language.
// Community parsers implement this interface and call Register().
type LanguageParser interface {
	// Name returns the language name (e.g. "go", "python", "rust").
	Name() string

	// Extensions returns file extensions this parser handles (e.g. [".go"], [".py"]).
	Extensions() []string

	// ExtractSymbols parses source content and returns discovered symbols.
	ExtractSymbols(content string) []Symbol

	// IsTestFile returns true if the file path is a test file for this language.
	IsTestFile(path string) bool
}

// ---------------------------------------------------------------------------
// Global registry
// ---------------------------------------------------------------------------

var (
	registryMu sync.RWMutex
	parsers    = make(map[string]LanguageParser) // extension → parser
	byName     = make(map[string]LanguageParser) // name → parser
)

// Register adds a language parser to the global registry.
// It maps each of the parser's extensions to the parser.
// Registering a parser with an already-claimed extension overwrites silently.
func Register(p LanguageParser) {
	registryMu.Lock()
	defer registryMu.Unlock()

	byName[p.Name()] = p
	for _, ext := range p.Extensions() {
		ext = normalizeExt(ext)
		parsers[ext] = p
	}
}

// ParserForFile returns the registered parser for a file path, or nil.
func ParserForFile(path string) LanguageParser {
	ext := normalizeExt(filepath.Ext(path))
	registryMu.RLock()
	defer registryMu.RUnlock()
	return parsers[ext]
}

// ParserByName returns a parser by language name, or nil.
func ParserByName(name string) LanguageParser {
	registryMu.RLock()
	defer registryMu.RUnlock()
	return byName[strings.ToLower(name)]
}

// RegisteredLanguages returns the names of all registered languages.
func RegisteredLanguages() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()

	names := make([]string, 0, len(byName))
	for n := range byName {
		names = append(names, n)
	}
	return names
}

// SupportedExtensions returns all registered file extensions.
func SupportedExtensions() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()

	exts := make([]string, 0, len(parsers))
	for e := range parsers {
		exts = append(exts, e)
	}
	return exts
}

// IsSupported returns true if the file extension has a registered parser.
func IsSupported(path string) bool {
	return ParserForFile(path) != nil
}

func normalizeExt(ext string) string {
	ext = strings.ToLower(ext)
	if ext != "" && ext[0] != '.' {
		ext = "." + ext
	}
	return ext
}
