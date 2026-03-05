package analyzer

import (
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type GitResult struct {
	Available        bool
	CommitCount      int
	ContributorCount int
	LastCommitDays   int
	RepoAgeDays      int
}

// AnalyzeGit extracts git health metrics from the repository.
func AnalyzeGit(root string) GitResult {
	repo, err := git.PlainOpen(root)
	if err != nil {
		// Try parent directories
		repo, err = git.PlainOpenWithOptions(root, &git.PlainOpenOptions{DetectDotGit: true})
		if err != nil {
			return GitResult{}
		}
	}

	result := GitResult{Available: true}
	now := time.Now()

	logIter, err := repo.Log(&git.LogOptions{})
	if err != nil {
		return result
	}

	contributors := make(map[string]bool)
	var earliest, latest time.Time
	first := true

	err = logIter.ForEach(func(c *object.Commit) error {
		result.CommitCount++
		contributors[c.Author.Email] = true

		when := c.Author.When
		if first {
			latest = when
			earliest = when
			first = true
		}
		if when.After(latest) {
			latest = when
		}
		if when.Before(earliest) {
			earliest = when
		}
		first = false
		return nil
	})

	result.ContributorCount = len(contributors)

	if !latest.IsZero() {
		result.LastCommitDays = int(now.Sub(latest).Hours() / 24)
	}
	if !earliest.IsZero() {
		result.RepoAgeDays = int(now.Sub(earliest).Hours() / 24)
	}

	return result
}
