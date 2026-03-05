package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ao/assay/internal/analyzer"
	"github.com/ao/assay/internal/model"
	"github.com/ao/assay/internal/report"
	"github.com/ao/assay/internal/walker"
)

func main() {
	var (
		rate    float64
		format  string
		exclude string
		verbose bool
	)

	rootCmd := &cobra.Command{
		Use:   "assay [path]",
		Short: "Estimate the value of a codebase",
		Long:  "assay analyzes source code to estimate development effort and codebase value using COCOMO II.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}

			absPath, err := filepath.Abs(path)
			if err != nil {
				return fmt.Errorf("invalid path: %w", err)
			}

			info, err := os.Stat(absPath)
			if err != nil || !info.IsDir() {
				return fmt.Errorf("not a directory: %s", absPath)
			}

			var excludePatterns []string
			if exclude != "" {
				excludePatterns = strings.Split(exclude, ",")
				for i := range excludePatterns {
					excludePatterns[i] = strings.TrimSpace(excludePatterns[i])
				}
			}

			// Walk files
			files, err := walker.Walk(absPath, excludePatterns)
			if err != nil {
				return fmt.Errorf("walking directory: %w", err)
			}

			if len(files) == 0 {
				fmt.Fprintln(os.Stderr, "No source files found.")
				return nil
			}

			// Analyze
			workers := runtime.NumCPU()
			metrics := analyzer.Analyze(absPath, files, workers)

			// Cost model
			cost := model.EstimateCost(metrics, rate)
			scores := model.ComputeScores(metrics)

			// Render
			displayPath := path
			switch format {
			case "json":
				return report.RenderJSON(os.Stdout, displayPath, metrics, cost, scores, verbose)
			default:
				report.RenderTable(os.Stdout, displayPath, metrics, cost, scores, verbose)
			}

			return nil
		},
	}

	rootCmd.Flags().Float64Var(&rate, "rate", 150, "hourly developer rate in USD")
	rootCmd.Flags().StringVar(&format, "format", "table", "output format: table or json")
	rootCmd.Flags().StringVar(&exclude, "exclude", "", "comma-separated glob patterns to skip")
	rootCmd.Flags().BoolVar(&verbose, "verbose", false, "show per-file breakdown")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
