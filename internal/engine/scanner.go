/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2026 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package engine

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileMeta holds metadata about a discovered source file.
type FileMeta struct {
	Path    string    // relative path from project root
	AbsPath string    // absolute path on disk
	ModTime time.Time // last modification time
	Size    int64     // file size in bytes
}

// Scanner walks a project directory and discovers indexable source files,
// respecting .gitignore patterns and a configurable ignore list.
type Scanner struct {
	root        string
	ignoreList  []string
	gitPatterns []ignorePattern
}

// ignorePattern is a parsed gitignore rule.
type ignorePattern struct {
	pattern  string
	negated  bool
	dirOnly  bool
	anchored bool // contains a slash (matches from root)
}

// NewScanner creates a new file scanner rooted at the given directory.
// ignoreList is the set of directory/file names to always skip (e.g. node_modules, .git).
func NewScanner(root string, ignoreList []string) *Scanner {
	s := &Scanner{
		root:       root,
		ignoreList: ignoreList,
	}
	s.gitPatterns = s.loadGitignore()
	return s
}

// ScanAll returns metadata for every indexable file under the project root.
func (s *Scanner) ScanAll() ([]FileMeta, error) {
	var files []FileMeta
	err := filepath.Walk(s.root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}

		relPath, _ := filepath.Rel(s.root, path)
		if relPath == "." {
			return nil
		}

		name := info.Name()

		// Skip hidden files/dirs (except .gitignore itself)
		if strings.HasPrefix(name, ".") && name != ".gitignore" {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Check hard-coded ignore list
		if s.isInIgnoreList(name) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Check gitignore patterns
		if s.matchesGitignore(relPath, info.IsDir()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			return nil
		}

		// Skip binary / non-text files by extension
		if !isTextFile(name) {
			return nil
		}

		// Skip very large files (>1MB)
		if info.Size() > 1*1024*1024 {
			return nil
		}

		files = append(files, FileMeta{
			Path:    relPath,
			AbsPath: path,
			ModTime: info.ModTime(),
			Size:    info.Size(),
		})
		return nil
	})
	return files, err
}

// ScanChanged returns files modified after the given time.
func (s *Scanner) ScanChanged(since time.Time) ([]FileMeta, error) {
	all, err := s.ScanAll()
	if err != nil {
		return nil, err
	}
	var changed []FileMeta
	for _, f := range all {
		if f.ModTime.After(since) {
			changed = append(changed, f)
		}
	}
	return changed, nil
}

func (s *Scanner) isInIgnoreList(name string) bool {
	for _, ig := range s.ignoreList {
		if name == ig {
			return true
		}
	}
	return false
}

func (s *Scanner) loadGitignore() []ignorePattern {
	path := filepath.Join(s.root, ".gitignore")
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var patterns []ignorePattern
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		p := parseGitignoreLine(line)
		patterns = append(patterns, p)
	}
	return patterns
}

func parseGitignoreLine(line string) ignorePattern {
	p := ignorePattern{}

	if strings.HasPrefix(line, "!") {
		p.negated = true
		line = line[1:]
	}

	if strings.HasSuffix(line, "/") {
		p.dirOnly = true
		line = strings.TrimSuffix(line, "/")
	}

	// A pattern with a slash (not just trailing) is anchored to root
	if strings.Contains(line, "/") {
		p.anchored = true
	}

	p.pattern = line
	return p
}

func (s *Scanner) matchesGitignore(relPath string, isDir bool) bool {
	matched := false
	for _, p := range s.gitPatterns {
		if p.dirOnly && !isDir {
			continue
		}

		doesMatch := false
		if p.anchored {
			// Match against full relative path
			doesMatch, _ = filepath.Match(p.pattern, relPath)
		} else {
			// Match against basename
			doesMatch, _ = filepath.Match(p.pattern, filepath.Base(relPath))
			if !doesMatch {
				// Also try matching against full path for patterns like "dir/file"
				doesMatch, _ = filepath.Match(p.pattern, relPath)
			}
		}

		if doesMatch {
			if p.negated {
				matched = false
			} else {
				matched = true
			}
		}
	}
	return matched
}

// isTextFile returns true if the file extension indicates a text/source file.
var textExtensions = map[string]bool{
	// Programming languages
	".go": true, ".js": true, ".ts": true, ".jsx": true, ".tsx": true,
	".py": true, ".rb": true, ".rs": true, ".java": true, ".kt": true,
	".c": true, ".h": true, ".cpp": true, ".hpp": true, ".cc": true,
	".cs": true, ".swift": true, ".m": true, ".mm": true,
	".php": true, ".lua": true, ".zig": true, ".nim": true,
	".scala": true, ".clj": true, ".ex": true, ".exs": true,
	".erl": true, ".hs": true, ".ml": true, ".fs": true, ".fsx": true,
	".r": true, ".jl": true, ".dart": true, ".v": true, ".vhdl": true,
	".pl": true, ".pm": true, ".sh": true, ".bash": true, ".zsh": true,
	".fish": true, ".ps1": true, ".bat": true, ".cmd": true,
	// Web
	".html": true, ".htm": true, ".css": true, ".scss": true, ".sass": true,
	".less": true, ".vue": true, ".svelte": true, ".astro": true,
	// Data / Config
	".json": true, ".yaml": true, ".yml": true, ".toml": true, ".xml": true,
	".ini": true, ".cfg": true, ".conf": true, ".env": true, ".properties": true,
	// Documentation
	".md": true, ".rst": true, ".txt": true, ".adoc": true,
	// Templates
	".tmpl": true, ".tpl": true, ".ejs": true, ".hbs": true, ".pug": true,
	// Build / CI
	".mk": true, ".cmake": true, ".gradle": true,
	// SQL
	".sql": true,
	// Proto
	".proto": true,
	// Docker
	".dockerfile": true,
	// GraphQL
	".graphql": true, ".gql": true,
}

func isTextFile(name string) bool {
	// Special filenames without extensions
	lower := strings.ToLower(name)
	switch lower {
	case "makefile", "dockerfile", "rakefile", "gemfile",
		"cmakelists.txt", "go.mod", "go.sum", "cargo.toml", "cargo.lock",
		"package.json", "tsconfig.json", ".gitignore", ".dockerignore":
		return true
	}
	ext := strings.ToLower(filepath.Ext(name))
	return textExtensions[ext]
}
