package report

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/ao/assay/internal/analyzer"
	"github.com/ao/assay/internal/model"
)

// RenderTable prints the human-readable table output.
func RenderTable(w io.Writer, path string, m *analyzer.Metrics, cost model.CostEstimate, scores model.Scores, verbose bool) {
	line := strings.Repeat("\u2500", 50)

	fmt.Fprintln(w, line)
	fmt.Fprintln(w, "  assay \u2014 Codebase Value Estimator")
	fmt.Fprintf(w, "  Path: %s\n", path)
	fmt.Fprintln(w, line)

	// SLOC with language breakdown
	langBreak := langBreakdown(m)
	fmt.Fprintf(w, "  %-20s %s  (%s)\n", "SLOC", formatNum(m.TotalSLOC), langBreak)

	// Complexity
	avgComplexity := 0.0
	if m.FileCount > 0 {
		avgComplexity = float64(m.TotalComplexity) / float64(m.FileCount)
	}
	fmt.Fprintf(w, "  %-20s avg %.1f / file\n", "Complexity", avgComplexity)

	// Test Ratio
	fmt.Fprintf(w, "  %-20s %d%%\n", "Test Ratio", int(m.TestRatio*100))

	// Duplication
	fmt.Fprintf(w, "  %-20s %.1f%%\n", "Duplication", m.DuplicationPct)

	// Dependencies
	depStr := fmt.Sprintf("%d", m.Dependencies)
	if len(m.DepFiles) > 0 {
		managers := strings.Join(m.DepFiles, ", ")
		lockIcon := "\u2717"
		if m.HasLockfile {
			lockIcon = "\u2713"
		}
		depStr = fmt.Sprintf("%d  (%s %s)", m.Dependencies, managers, lockIcon)
	}
	fmt.Fprintf(w, "  %-20s %s\n", "Dependencies", depStr)

	// Git metrics
	if m.GitAvailable {
		fmt.Fprintf(w, "  %-20s %d\n", "Contributors", m.ContributorCount)
		fmt.Fprintf(w, "  %-20s %s\n", "Repo Age", formatDays(m.RepoAgeDays))
		fmt.Fprintf(w, "  %-20s %s\n", "Last Commit", formatDaysAgo(m.LastCommitDays))
	}

	fmt.Fprintln(w)

	// Scores table
	fmt.Fprintln(w, "  \u250c\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u252c\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2510")
	fmt.Fprintln(w, "  \u2502 Dimension            \u2502  Score   \u2502")
	fmt.Fprintln(w, "  \u251c\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u253c\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2524")
	printScore(w, "Size & Effort", scores.SizeEffort)
	printScore(w, "Code Quality", scores.CodeQuality)
	printScore(w, "Test Coverage", scores.TestCoverage)
	printScore(w, "Dependency Health", scores.DepHealth)
	printScore(w, "Git Activity", scores.GitActivity)
	fmt.Fprintln(w, "  \u2514\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2534\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2518")

	fmt.Fprintln(w)
	fmt.Fprintf(w, "  %-22s %d / 100\n", "Composite Score", scores.Composite)
	fmt.Fprintf(w, "  %-22s $%s \u2013 $%s\n", "Estimated Cost", formatMoney(cost.AdjustedLow), formatMoney(cost.AdjustedHigh))

	if len(cost.Multipliers) > 0 {
		fmt.Fprintln(w)
		for _, mult := range cost.Multipliers {
			fmt.Fprintf(w, "  \u26a0 %s (x%.1f): %s\n", mult.Name, mult.Factor, mult.Reason)
		}
	}

	fmt.Fprintln(w, line)

	if verbose {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "  Per-file breakdown:")
		fmt.Fprintf(w, "  %-50s %8s %8s %s\n", "File", "SLOC", "Cmplx", "Lang")
		fmt.Fprintf(w, "  %s\n", strings.Repeat("-", 80))
		for _, f := range m.PerFile {
			fmt.Fprintf(w, "  %-50s %8d %8d %s\n", truncate(f.Path, 50), f.SLOC, f.Complexity, f.Language)
		}
	}
}

func printScore(w io.Writer, name string, score int) {
	fmt.Fprintf(w, "  \u2502 %-20s \u2502  %3d/100 \u2502\n", name, score)
}

func langBreakdown(m *analyzer.Metrics) string {
	type langPct struct {
		lang string
		pct  float64
	}
	var langs []langPct
	for lang, sloc := range m.SLOCByLang {
		if m.TotalSLOC > 0 {
			langs = append(langs, langPct{lang, float64(sloc) / float64(m.TotalSLOC) * 100})
		}
	}
	sort.Slice(langs, func(i, j int) bool { return langs[i].pct > langs[j].pct })

	var parts []string
	for i, l := range langs {
		if i >= 3 {
			break
		}
		parts = append(parts, fmt.Sprintf("%s %d%%", l.lang, int(l.pct)))
	}
	return strings.Join(parts, ", ")
}

func formatNum(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1000000 {
		return fmt.Sprintf("%d,%03d", n/1000, n%1000)
	}
	return fmt.Sprintf("%d,%03d,%03d", n/1000000, (n/1000)%1000, n%1000)
}

func formatMoney(f float64) string {
	n := int(f)
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1000000 {
		return fmt.Sprintf("%d,%03d", n/1000, n%1000)
	}
	return fmt.Sprintf("%d,%03d,%03d", n/1000000, (n/1000)%1000, n%1000)
}

func formatDays(days int) string {
	if days < 1 {
		return "< 1 day"
	}
	if days == 1 {
		return "1 day"
	}
	if days < 365 {
		return fmt.Sprintf("%d days", days)
	}
	years := days / 365
	remaining := days % 365
	if remaining == 0 {
		return fmt.Sprintf("%d years", years)
	}
	return fmt.Sprintf("%d years, %d days", years, remaining)
}

func formatDaysAgo(days int) string {
	if days == 0 {
		return "today"
	}
	if days == 1 {
		return "1 day ago"
	}
	return fmt.Sprintf("%d days ago", days)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return "..." + s[len(s)-max+3:]
}
