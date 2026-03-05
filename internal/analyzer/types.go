package analyzer

// FileInfo holds metadata about a single file for analysis.
type FileInfo struct {
	Path     string
	RelPath  string
	Language string
}

// Metrics holds all analysis results for a codebase.
type Metrics struct {
	// SLOC
	TotalSLOC    int
	SLOCByLang   map[string]int
	FileCount    int

	// Complexity
	TotalComplexity int
	FileComplexity  map[string]int // per-file complexity

	// Test coverage proxy
	TestFiles   int
	SourceFiles int
	TestLines   int
	SourceLines int
	TestRatio   float64

	// Dependency health
	Dependencies  int
	DepFiles      []string // which manifest files found
	HasLockfile   bool
	DepDetails    map[string]DepInfo

	// Git health
	CommitCount      int
	ContributorCount int
	LastCommitDays   int
	RepoAgeDays      int
	GitAvailable     bool

	// Duplication
	DuplicateLines int
	DuplicationPct float64

	// Per-file breakdown (for verbose)
	PerFile []FileStat
}

type DepInfo struct {
	Manager     string
	DepCount    int
	HasLockfile bool
}

type FileStat struct {
	Path       string
	Language   string
	SLOC       int
	Complexity int
}
