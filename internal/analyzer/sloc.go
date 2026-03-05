package analyzer

import (
	"bufio"
	"os"
	"strings"
)

type lineClassifier struct {
	singleLine    string
	blockStart    string
	blockEnd      string
}

var commentStyles = map[string]lineClassifier{
	"Go":         {"//", "/*", "*/"},
	"JavaScript": {"//", "/*", "*/"},
	"TypeScript": {"//", "/*", "*/"},
	"Java":       {"//", "/*", "*/"},
	"C":          {"//", "/*", "*/"},
	"C++":        {"//", "/*", "*/"},
	"C#":         {"//", "/*", "*/"},
	"Rust":       {"//", "/*", "*/"},
	"Swift":      {"//", "/*", "*/"},
	"Kotlin":     {"//", "/*", "*/"},
	"Scala":      {"//", "/*", "*/"},
	"Dart":       {"//", "/*", "*/"},
	"PHP":        {"//", "/*", "*/"},
	"Python":     {"#", `"""`, `"""`},
	"Ruby":       {"#", "=begin", "=end"},
	"Shell":      {"#", "", ""},
	"Lua":        {"--", "--[[", "]]"},
	"R":          {"#", "", ""},
	"SQL":        {"--", "/*", "*/"},
	"HTML":       {"", "<!--", "-->"},
	"CSS":        {"", "/*", "*/"},
	"SCSS":       {"//", "/*", "*/"},
	"Vue":        {"//", "<!--", "-->"},
	"Svelte":     {"//", "<!--", "-->"},
	"Elixir":     {"#", "", ""},
	"Erlang":     {"%", "", ""},
	"Haskell":    {"--", "{-", "-}"},
	"OCaml":      {"", "(*", "*)"},
	"Zig":        {"//", "", ""},
	"Nim":        {"#", "", ""},
}

// CountSLOC counts non-blank, non-comment source lines in a file.
func CountSLOC(path string, lang string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	cls, ok := commentStyles[lang]
	if !ok {
		cls = lineClassifier{"//", "/*", "*/"}
	}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	sloc := 0
	inBlock := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			continue
		}

		if inBlock {
			if cls.blockEnd != "" && strings.Contains(line, cls.blockEnd) {
				inBlock = false
			}
			continue
		}

		if cls.singleLine != "" && strings.HasPrefix(line, cls.singleLine) {
			continue
		}

		if cls.blockStart != "" && strings.HasPrefix(line, cls.blockStart) {
			if cls.blockEnd != "" && !strings.Contains(line, cls.blockEnd) {
				inBlock = true
			}
			continue
		}

		sloc++
	}

	return sloc, scanner.Err()
}
