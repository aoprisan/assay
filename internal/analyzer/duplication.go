package analyzer

import (
	"bufio"
	"crypto/sha256"
	"os"
	"strings"
)

// DuplicationResult holds deduplication analysis results.
type DuplicationResult struct {
	DuplicateLines int
	TotalLines     int
	Percentage     float64
}

// AnalyzeDuplication hashes normalized lines and flags those appearing 3+ times.
func AnalyzeDuplication(files []FileInfo) DuplicationResult {
	lineCounts := make(map[[32]byte]int)
	totalLines := 0

	for _, fi := range files {
		f, err := os.Open(fi.Path)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			// Skip blank and trivial lines
			if len(line) < 4 {
				continue
			}
			// Normalize: strip all whitespace
			normalized := strings.Join(strings.Fields(line), "")
			hash := sha256.Sum256([]byte(normalized))
			lineCounts[hash]++
			totalLines++
		}
		f.Close()
	}

	duplicateLines := 0
	for _, count := range lineCounts {
		if count >= 3 {
			duplicateLines += count
		}
	}

	pct := 0.0
	if totalLines > 0 {
		pct = float64(duplicateLines) / float64(totalLines) * 100
	}

	return DuplicationResult{
		DuplicateLines: duplicateLines,
		TotalLines:     totalLines,
		Percentage:     pct,
	}
}
