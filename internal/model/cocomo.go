package model

import (
	"math"

	"github.com/ao/assay/internal/analyzer"
)

// CostEstimate holds the COCOMO-based cost estimation.
type CostEstimate struct {
	EffortMonths  float64
	BaseCost      float64
	AdjustedLow   float64
	AdjustedHigh  float64
	Multiplier    float64
	Multipliers   []MultiplierDetail
}

type MultiplierDetail struct {
	Name   string
	Factor float64
	Reason string
}

// Scores holds dimension scores and the composite.
type Scores struct {
	SizeEffort    int
	CodeQuality   int
	TestCoverage  int
	DepHealth     int
	GitActivity   int
	Composite     int
}

// EstimateCost computes COCOMO II Basic cost with multipliers.
func EstimateCost(m *analyzer.Metrics, hourlyRate float64) CostEstimate {
	ksloc := float64(m.TotalSLOC) / 1000.0
	if ksloc < 0.01 {
		ksloc = 0.01
	}

	effortMonths := 2.94 * math.Pow(ksloc, 1.10)
	monthlyCost := hourlyRate * 160
	baseCost := effortMonths * monthlyCost

	multiplier := 1.0
	var details []MultiplierDetail

	if m.TestRatio < 0.1 {
		multiplier *= 1.3
		details = append(details, MultiplierDetail{"Low Test Coverage", 1.3, "test_ratio < 10%"})
	}

	if m.DuplicationPct > 15 {
		multiplier *= 1.2
		details = append(details, MultiplierDetail{"High Duplication", 1.2, "duplication > 15%"})
	}

	if len(m.DepFiles) > 0 && !m.HasLockfile {
		multiplier *= 1.1
		details = append(details, MultiplierDetail{"Missing Lockfile", 1.1, "no lockfile found"})
	}

	if m.GitAvailable && m.LastCommitDays > 365 {
		multiplier *= 0.8
		details = append(details, MultiplierDetail{"Stale Repository", 0.8, "last commit > 365 days"})
	}

	adjusted := baseCost * multiplier

	return CostEstimate{
		EffortMonths: effortMonths,
		BaseCost:     baseCost,
		AdjustedLow:  adjusted * 0.8,
		AdjustedHigh: adjusted * 1.2,
		Multiplier:   multiplier,
		Multipliers:  details,
	}
}

// ComputeScores calculates dimension scores and composite score.
func ComputeScores(m *analyzer.Metrics) Scores {
	s := Scores{}

	// Size & Effort: based on SLOC (more code = higher score, capped at 100)
	s.SizeEffort = clamp(int(math.Log10(float64(m.TotalSLOC+1))*25), 0, 100)

	// Code Quality: inverse of complexity and duplication
	avgComplexity := 0.0
	if m.FileCount > 0 {
		avgComplexity = float64(m.TotalComplexity) / float64(m.FileCount)
	}
	complexityScore := clamp(100-int(avgComplexity*3), 0, 100)
	dupScore := clamp(100-int(m.DuplicationPct*4), 0, 100)
	s.CodeQuality = (complexityScore + dupScore) / 2

	// Test Coverage
	s.TestCoverage = clamp(int(m.TestRatio*200), 0, 100) // 50% ratio = 100

	// Dependency Health
	depScore := 50
	if len(m.DepFiles) > 0 {
		depScore = 70
		if m.HasLockfile {
			depScore = 90
		}
		if m.Dependencies > 50 {
			depScore -= 15
		} else if m.Dependencies > 20 {
			depScore -= 5
		}
	}
	s.DepHealth = clamp(depScore, 0, 100)

	// Git Activity
	if m.GitAvailable {
		gitScore := 50
		if m.CommitCount > 100 {
			gitScore += 15
		} else if m.CommitCount > 20 {
			gitScore += 10
		}
		if m.ContributorCount > 3 {
			gitScore += 15
		} else if m.ContributorCount > 1 {
			gitScore += 10
		}
		if m.LastCommitDays < 30 {
			gitScore += 20
		} else if m.LastCommitDays < 90 {
			gitScore += 10
		} else if m.LastCommitDays > 365 {
			gitScore -= 20
		}
		s.GitActivity = clamp(gitScore, 0, 100)
	} else {
		s.GitActivity = 0
	}

	// Composite: weighted average
	s.Composite = (s.SizeEffort*25 + s.CodeQuality*25 + s.TestCoverage*20 + s.DepHealth*15 + s.GitActivity*15) / 100

	return s
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
