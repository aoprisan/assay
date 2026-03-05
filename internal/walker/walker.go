package walker

import (
	"os"
	"path/filepath"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"

	"github.com/ao/assay/internal/analyzer"
)

var langMap = map[string]string{
	".go":    "Go",
	".js":    "JavaScript",
	".ts":    "TypeScript",
	".tsx":   "TypeScript",
	".jsx":   "JavaScript",
	".py":    "Python",
	".rs":    "Rust",
	".java":  "Java",
	".c":     "C",
	".cpp":   "C++",
	".cc":    "C++",
	".h":     "C",
	".hpp":   "C++",
	".rb":    "Ruby",
	".php":   "PHP",
	".swift": "Swift",
	".kt":    "Kotlin",
	".scala": "Scala",
	".cs":    "C#",
	".sh":    "Shell",
	".bash":  "Shell",
	".zsh":   "Shell",
	".lua":   "Lua",
	".r":     "R",
	".sql":   "SQL",
	".html":  "HTML",
	".css":   "CSS",
	".scss":  "SCSS",
	".vue":   "Vue",
	".svelte":"Svelte",
	".dart":  "Dart",
	".ex":    "Elixir",
	".exs":   "Elixir",
	".erl":   "Erlang",
	".hs":    "Haskell",
	".ml":    "OCaml",
	".zig":   "Zig",
	".nim":   "Nim",
}

// Walk traverses the directory tree, respecting .gitignore and exclude patterns,
// and returns a list of source files to analyze.
func Walk(root string, excludePatterns []string) ([]analyzer.FileInfo, error) {
	var gi *ignore.GitIgnore

	gitignorePath := filepath.Join(root, ".gitignore")
	if _, err := os.Stat(gitignorePath); err == nil {
		gi, _ = ignore.CompileIgnoreFile(gitignorePath)
	}

	var files []analyzer.FileInfo

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}

		relPath, _ := filepath.Rel(root, path)

		// Skip hidden directories and common non-source dirs
		if info.IsDir() {
			base := filepath.Base(path)
			if strings.HasPrefix(base, ".") && base != "." {
				return filepath.SkipDir
			}
			skip := map[string]bool{
				"node_modules": true, "vendor": true, "__pycache__": true,
				"target": true, "dist": true, "build": true, ".git": true,
			}
			if skip[base] && path != root {
				return filepath.SkipDir
			}
		}

		if info.IsDir() {
			return nil
		}

		// Check gitignore
		if gi != nil && gi.MatchesPath(relPath) {
			return nil
		}

		// Check exclude patterns
		for _, pat := range excludePatterns {
			if matched, _ := filepath.Match(pat, filepath.Base(path)); matched {
				return nil
			}
			if matched, _ := filepath.Match(pat, relPath); matched {
				return nil
			}
		}

		ext := strings.ToLower(filepath.Ext(path))
		lang, ok := langMap[ext]
		if !ok {
			return nil
		}

		files = append(files, analyzer.FileInfo{
			Path:     path,
			RelPath:  relPath,
			Language: lang,
		})

		return nil
	})

	return files, err
}
