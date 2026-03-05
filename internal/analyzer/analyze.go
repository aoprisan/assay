package analyzer

import (
	"strings"
	"sync"
)

// Analyze runs all analyzers concurrently and returns aggregated metrics.
func Analyze(root string, files []FileInfo, workers int) *Metrics {
	m := &Metrics{
		SLOCByLang:     make(map[string]int),
		FileComplexity: make(map[string]int),
	}

	if workers <= 0 {
		workers = 8
	}

	// Per-file analysis with worker pool
	type fileResult struct {
		fi         FileInfo
		sloc       int
		complexity int
		isTest     bool
	}

	results := make([]fileResult, len(files))
	ch := make(chan int, len(files))
	for i := range files {
		ch <- i
	}
	close(ch)

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range ch {
				fi := files[idx]
				sloc, _ := CountSLOC(fi.Path, fi.Language)
				complexity, _ := EstimateComplexity(fi.Path, fi.Language)
				isTest := isTestFile(fi.Path, fi.Language)
				results[idx] = fileResult{fi, sloc, complexity, isTest}
			}
		}()
	}
	wg.Wait()

	// Aggregate
	for _, r := range results {
		m.TotalSLOC += r.sloc
		m.SLOCByLang[r.fi.Language] += r.sloc
		m.TotalComplexity += r.complexity
		m.FileComplexity[r.fi.RelPath] = r.complexity
		m.FileCount++

		if r.isTest {
			m.TestFiles++
			m.TestLines += r.sloc
		} else {
			m.SourceFiles++
			m.SourceLines += r.sloc
		}

		m.PerFile = append(m.PerFile, FileStat{
			Path:       r.fi.RelPath,
			Language:   r.fi.Language,
			SLOC:       r.sloc,
			Complexity: r.complexity,
		})
	}

	// Test ratio
	if m.SourceFiles > 0 {
		m.TestRatio = float64(m.TestLines) / float64(m.SourceLines+m.TestLines)
	}

	// Dependencies (run in parallel with git and duplication)
	var depResult DepResult
	var gitResult GitResult
	var dupResult DuplicationResult

	var wg2 sync.WaitGroup
	wg2.Add(3)
	go func() {
		defer wg2.Done()
		depResult = AnalyzeDeps(root)
	}()
	go func() {
		defer wg2.Done()
		gitResult = AnalyzeGit(root)
	}()
	go func() {
		defer wg2.Done()
		dupResult = AnalyzeDuplication(files)
	}()
	wg2.Wait()

	m.Dependencies = depResult.TotalDeps
	m.DepFiles = depResult.DepFiles
	m.HasLockfile = depResult.HasLockfile
	m.DepDetails = depResult.Details

	m.GitAvailable = gitResult.Available
	m.CommitCount = gitResult.CommitCount
	m.ContributorCount = gitResult.ContributorCount
	m.LastCommitDays = gitResult.LastCommitDays
	m.RepoAgeDays = gitResult.RepoAgeDays

	m.DuplicateLines = dupResult.DuplicateLines
	m.DuplicationPct = dupResult.Percentage

	return m
}

func isTestFile(path string, lang string) bool {
	lower := strings.ToLower(path)
	switch lang {
	case "Go":
		return strings.HasSuffix(lower, "_test.go")
	case "Python":
		return strings.HasSuffix(lower, "_test.py") || strings.HasPrefix(strings.ToLower(strings.Replace(path, "\\", "/", -1)), "test")
	case "JavaScript", "TypeScript":
		return strings.Contains(lower, ".test.") || strings.Contains(lower, ".spec.") || strings.Contains(lower, "__tests__")
	case "Rust":
		return strings.Contains(lower, "/tests/")
	case "Java":
		return strings.Contains(lower, "test") && strings.HasSuffix(lower, ".java")
	default:
		return strings.Contains(lower, "test")
	}
}
