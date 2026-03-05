package report

import (
	"encoding/json"
	"io"

	"github.com/ao/assay/internal/analyzer"
	"github.com/ao/assay/internal/model"
)

type jsonOutput struct {
	Path    string            `json:"path"`
	Metrics jsonMetrics       `json:"metrics"`
	Scores  jsonScores        `json:"scores"`
	Cost    jsonCost          `json:"cost"`
}

type jsonMetrics struct {
	SLOC           int                `json:"sloc"`
	SLOCByLang     map[string]int     `json:"sloc_by_language"`
	FileCount      int                `json:"file_count"`
	AvgComplexity  float64            `json:"avg_complexity"`
	TestRatio      float64            `json:"test_ratio"`
	TestFiles      int                `json:"test_files"`
	SourceFiles    int                `json:"source_files"`
	DuplicationPct float64            `json:"duplication_pct"`
	Dependencies   int                `json:"dependencies"`
	DepFiles       []string           `json:"dep_files"`
	HasLockfile    bool               `json:"has_lockfile"`
	Contributors   int                `json:"contributors"`
	CommitCount    int                `json:"commit_count"`
	RepoAgeDays    int                `json:"repo_age_days"`
	LastCommitDays int                `json:"last_commit_days"`
	GitAvailable   bool               `json:"git_available"`
	PerFile        []analyzer.FileStat `json:"per_file,omitempty"`
}

type jsonScores struct {
	SizeEffort   int `json:"size_effort"`
	CodeQuality  int `json:"code_quality"`
	TestCoverage int `json:"test_coverage"`
	DepHealth    int `json:"dep_health"`
	GitActivity  int `json:"git_activity"`
	Composite    int `json:"composite"`
}

type jsonCost struct {
	HourlyRate      float64                  `json:"hourly_rate"`
	EffortMonths    float64                  `json:"effort_months"`
	ScheduleMonths  float64                  `json:"schedule_months"`
	TeamSize        float64                  `json:"team_size"`
	BaseCost        float64                  `json:"base_cost"`
	AdjustedCost    float64                  `json:"adjusted_cost"`
	CostLow         float64                  `json:"cost_low"`
	CostHigh        float64                  `json:"cost_high"`
	CostPerSLOC     float64                  `json:"cost_per_sloc"`
	Multiplier      float64                  `json:"multiplier"`
	Multipliers     []model.MultiplierDetail `json:"multipliers,omitempty"`
	ConfidencePct   float64                  `json:"confidence_pct"`
	ConfidenceLevel string                   `json:"confidence_level"`
}

// RenderJSON writes JSON output.
func RenderJSON(w io.Writer, path string, m *analyzer.Metrics, cost model.CostEstimate, scores model.Scores, verbose bool) error {
	avgComplexity := 0.0
	if m.FileCount > 0 {
		avgComplexity = float64(m.TotalComplexity) / float64(m.FileCount)
	}

	out := jsonOutput{
		Path: path,
		Metrics: jsonMetrics{
			SLOC:           m.TotalSLOC,
			SLOCByLang:     m.SLOCByLang,
			FileCount:      m.FileCount,
			AvgComplexity:  avgComplexity,
			TestRatio:      m.TestRatio,
			TestFiles:      m.TestFiles,
			SourceFiles:    m.SourceFiles,
			DuplicationPct: m.DuplicationPct,
			Dependencies:   m.Dependencies,
			DepFiles:       m.DepFiles,
			HasLockfile:    m.HasLockfile,
			Contributors:   m.ContributorCount,
			CommitCount:    m.CommitCount,
			RepoAgeDays:    m.RepoAgeDays,
			LastCommitDays: m.LastCommitDays,
			GitAvailable:   m.GitAvailable,
		},
		Scores: jsonScores{
			SizeEffort:   scores.SizeEffort,
			CodeQuality:  scores.CodeQuality,
			TestCoverage: scores.TestCoverage,
			DepHealth:    scores.DepHealth,
			GitActivity:  scores.GitActivity,
			Composite:    scores.Composite,
		},
		Cost: jsonCost{
			HourlyRate:      cost.HourlyRate,
			EffortMonths:    cost.EffortMonths,
			ScheduleMonths:  cost.ScheduleMonths,
			TeamSize:        cost.TeamSize,
			BaseCost:        cost.BaseCost,
			AdjustedCost:    cost.AdjustedCost,
			CostLow:         cost.AdjustedLow,
			CostHigh:        cost.AdjustedHigh,
			CostPerSLOC:     cost.CostPerSLOC,
			Multiplier:      cost.Multiplier,
			Multipliers:     cost.Multipliers,
			ConfidencePct:   cost.ConfidencePct,
			ConfidenceLevel: cost.ConfidenceLevel,
		},
	}

	if verbose {
		out.Metrics.PerFile = m.PerFile
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
