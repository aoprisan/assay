package model

import (
	"math"

	"github.com/ao/assay/internal/analyzer"
)

// CostEstimate holds the COCOMO-based cost estimation.
type CostEstimate struct {
	EffortMonths    float64
	ScheduleMonths  float64 // estimated calendar time
	TeamSize        float64 // average full-time equivalent developers
	BaseCost        float64
	AdjustedCost    float64
	AdjustedLow     float64
	AdjustedHigh    float64
	CostPerSLOC     float64
	Multiplier      float64
	Multipliers     []MultiplierDetail
	HourlyRate      float64
	ConfidencePct   float64 // confidence band percentage (e.g. 0.25 = ±25%)
	ConfidenceLevel string  // "high", "medium", "low"
}

type MultiplierDetail struct {
	Name   string
	Factor float64
	Reason string
}

// Scores holds dimension scores and the composite.
type Scores struct {
	SizeEffort   int
	CodeQuality  int
	TestCoverage int
	DepHealth    int
	GitActivity  int
	Composite    int
}

// Language effort multipliers relative to a baseline (Go = 1.0).
// Higher values mean more effort per SLOC for that language.
var languageEffort = map[string]float64{
	"Go":         1.0,
	"Python":     0.9,
	"Ruby":       0.9,
	"JavaScript": 0.95,
	"TypeScript": 1.0,
	"Java":       1.1,
	"C#":         1.1,
	"C":          1.3,
	"C++":        1.35,
	"Rust":       1.2,
	"Swift":      1.05,
	"Kotlin":     1.0,
	"Scala":      1.1,
	"PHP":        0.85,
	"Shell":      0.7,
}

// EstimateCost computes COCOMO II Basic cost with quality multipliers,
// language-weighted effort, schedule estimation, and dynamic confidence bands.
func EstimateCost(m *analyzer.Metrics, hourlyRate float64) CostEstimate {
	ksloc := float64(m.TotalSLOC) / 1000.0
	if ksloc < 0.01 {
		ksloc = 0.01
	}

	// COCOMO II adjusted for code-reproduction cost estimation.
	// Standard COCOMO II uses a=2.94, b=1.10 but that models the full SDLC
	// (requirements, design, testing, documentation, management overhead).
	// For estimating the cost to reproduce existing code, we use the organic
	// model coefficients which better reflect pure development effort.
	effortMonths := 2.0 * math.Pow(ksloc, 1.05)

	// Apply language weighting: adjust effort based on language mix
	langMultiplier := computeLanguageMultiplier(m)
	effortMonths *= langMultiplier

	monthlyCost := hourlyRate * 160 // 160 working hours per month
	baseCost := effortMonths * monthlyCost

	// --- Quality & risk multipliers ---
	multiplier := 1.0
	var details []MultiplierDetail

	// Test coverage: graduated thresholds
	if m.TestRatio < 0.05 {
		multiplier *= 1.15
		details = append(details, MultiplierDetail{"Very Low Test Coverage", 1.15, "test ratio < 5%"})
	} else if m.TestRatio < 0.1 {
		multiplier *= 1.08
		details = append(details, MultiplierDetail{"Low Test Coverage", 1.08, "test ratio < 10%"})
	} else if m.TestRatio >= 0.4 {
		multiplier *= 0.92
		details = append(details, MultiplierDetail{"Strong Test Coverage", 0.92, "test ratio >= 40%"})
	}

	// Code duplication: graduated
	if m.DuplicationPct > 30 {
		multiplier *= 1.15
		details = append(details, MultiplierDetail{"Very High Duplication", 1.15, "duplication > 30%"})
	} else if m.DuplicationPct > 15 {
		multiplier *= 1.08
		details = append(details, MultiplierDetail{"High Duplication", 1.08, "duplication > 15%"})
	}

	// Complexity: high average complexity per file increases cost
	avgComplexity := 0.0
	if m.FileCount > 0 {
		avgComplexity = float64(m.TotalComplexity) / float64(m.FileCount)
	}
	if avgComplexity > 30 {
		multiplier *= 1.15
		details = append(details, MultiplierDetail{"Very High Complexity", 1.15, "avg complexity > 30 per file"})
	} else if avgComplexity > 15 {
		multiplier *= 1.08
		details = append(details, MultiplierDetail{"High Complexity", 1.08, "avg complexity > 15 per file"})
	}

	// Dependency management
	if len(m.DepFiles) > 0 && !m.HasLockfile {
		multiplier *= 1.05
		details = append(details, MultiplierDetail{"Missing Lockfile", 1.05, "no lockfile found"})
	}

	// Heavy dependency count
	if m.Dependencies > 100 {
		multiplier *= 1.08
		details = append(details, MultiplierDetail{"Heavy Dependencies", 1.08, "over 100 dependencies"})
	} else if m.Dependencies > 50 {
		multiplier *= 1.03
		details = append(details, MultiplierDetail{"Moderate Dependencies", 1.03, "over 50 dependencies"})
	}

	// Repository staleness
	if m.GitAvailable && m.LastCommitDays > 365 {
		multiplier *= 0.8
		details = append(details, MultiplierDetail{"Stale Repository", 0.8, "last commit > 365 days"})
	}

	// Multi-contributor maturity bonus
	if m.GitAvailable && m.ContributorCount >= 5 && m.CommitCount > 200 {
		multiplier *= 0.95
		details = append(details, MultiplierDetail{"Mature Project", 0.95, "5+ contributors, 200+ commits"})
	}

	// Cap the compound multiplier to prevent runaway estimates
	if multiplier > 1.5 {
		multiplier = 1.5
	}

	adjusted := baseCost * multiplier

	// Dynamic confidence band based on data quality
	confidencePct, confidenceLevel := computeConfidence(m)
	adjustedLow := adjusted * (1.0 - confidencePct)
	adjustedHigh := adjusted * (1.0 + confidencePct)

	// Schedule estimate using COCOMO II: T = 3.67 * (E)^0.28
	adjustedEffort := effortMonths * multiplier
	scheduleMonths := 3.67 * math.Pow(adjustedEffort, 0.28)

	// Team size = effort / schedule
	teamSize := 0.0
	if scheduleMonths > 0 {
		teamSize = adjustedEffort / scheduleMonths
	}

	// Cost per SLOC
	costPerSLOC := 0.0
	if m.TotalSLOC > 0 {
		costPerSLOC = adjusted / float64(m.TotalSLOC)
	}

	return CostEstimate{
		EffortMonths:    adjustedEffort,
		ScheduleMonths:  scheduleMonths,
		TeamSize:        teamSize,
		BaseCost:        baseCost,
		AdjustedCost:    adjusted,
		AdjustedLow:     adjustedLow,
		AdjustedHigh:    adjustedHigh,
		CostPerSLOC:     costPerSLOC,
		Multiplier:      multiplier,
		Multipliers:     details,
		HourlyRate:      hourlyRate,
		ConfidencePct:   confidencePct,
		ConfidenceLevel: confidenceLevel,
	}
}

// computeLanguageMultiplier calculates a weighted effort multiplier based on the
// language distribution in the codebase.
func computeLanguageMultiplier(m *analyzer.Metrics) float64 {
	if m.TotalSLOC == 0 || len(m.SLOCByLang) == 0 {
		return 1.0
	}

	weighted := 0.0
	for lang, sloc := range m.SLOCByLang {
		factor, ok := languageEffort[lang]
		if !ok {
			factor = 1.0
		}
		weighted += factor * float64(sloc)
	}
	return weighted / float64(m.TotalSLOC)
}

// computeConfidence determines the confidence band and level based on how much
// data we have. More signals = tighter band = higher confidence.
func computeConfidence(m *analyzer.Metrics) (float64, string) {
	// Start with wide band and narrow it as we get more signals
	signals := 0
	totalSignals := 5

	// Have meaningful SLOC count?
	if m.TotalSLOC > 100 {
		signals++
	}

	// Have git data?
	if m.GitAvailable {
		signals++
	}

	// Have dependency info?
	if len(m.DepFiles) > 0 {
		signals++
	}

	// Have test info?
	if m.TestFiles > 0 || m.SourceFiles > 0 {
		signals++
	}

	// Have duplication analysis?
	if m.FileCount > 5 {
		signals++
	}

	ratio := float64(signals) / float64(totalSignals)

	switch {
	case ratio >= 0.8:
		return 0.20, "high"
	case ratio >= 0.5:
		return 0.35, "medium"
	default:
		return 0.50, "low"
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
