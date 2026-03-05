package analyzer

import (
	"bufio"
	"os"
	"strings"
)

// decision-point keywords per language family
var decisionKeywords = map[string][]string{
	"Go":         {"if ", "for ", "switch ", "case ", "&&", "||"},
	"JavaScript": {"if ", "for ", "while ", "switch ", "case ", "&&", "||", "? "},
	"TypeScript": {"if ", "for ", "while ", "switch ", "case ", "&&", "||", "? "},
	"Python":     {"if ", "for ", "while ", "elif ", "and ", "or ", "except "},
	"Rust":       {"if ", "for ", "while ", "match ", "&&", "||"},
	"Java":       {"if ", "for ", "while ", "switch ", "case ", "&&", "||", "? "},
	"C":          {"if ", "for ", "while ", "switch ", "case ", "&&", "||", "? "},
	"C++":        {"if ", "for ", "while ", "switch ", "case ", "&&", "||", "? "},
	"C#":         {"if ", "for ", "while ", "switch ", "case ", "&&", "||", "? "},
	"Ruby":       {"if ", "unless ", "while ", "until ", "when ", "&&", "||"},
	"PHP":        {"if ", "for ", "while ", "switch ", "case ", "&&", "||", "? "},
	"Swift":      {"if ", "for ", "while ", "switch ", "case ", "&&", "||", "guard "},
	"Kotlin":     {"if ", "for ", "while ", "when ", "&&", "||"},
	"Scala":      {"if ", "for ", "while ", "match ", "case ", "&&", "||"},
	"Shell":      {"if ", "for ", "while ", "case ", "&&", "||"},
}

// EstimateComplexity approximates cyclomatic complexity by counting decision points.
func EstimateComplexity(path string, lang string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	keywords, ok := decisionKeywords[lang]
	if !ok {
		keywords = decisionKeywords["Go"] // fallback
	}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	complexity := 1 // base complexity

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// skip comments
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "--") {
			continue
		}
		for _, kw := range keywords {
			complexity += strings.Count(line, kw)
		}
	}

	return complexity, scanner.Err()
}
