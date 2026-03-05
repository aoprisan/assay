package analyzer

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

type DepResult struct {
	Details    map[string]DepInfo
	TotalDeps  int
	HasLockfile bool
	DepFiles   []string
}

type depSpec struct {
	manifest string
	lockfile string
	manager  string
	counter  func(path string) int
}

var depSpecs = []depSpec{
	{"go.mod", "go.sum", "go.mod", countGoModDeps},
	{"package.json", "package-lock.json", "package.json", countPackageJSONDeps},
	{"package.json", "yarn.lock", "package.json", nil},
	{"requirements.txt", "requirements.txt", "requirements.txt", countLineBasedDeps},
	{"Cargo.toml", "Cargo.lock", "Cargo.toml", countCargoTOMLDeps},
}

// AnalyzeDeps checks for dependency manifests and lockfiles.
func AnalyzeDeps(root string) DepResult {
	result := DepResult{
		Details: make(map[string]DepInfo),
	}

	seen := make(map[string]bool)

	for _, spec := range depSpecs {
		manifestPath := filepath.Join(root, spec.manifest)
		if _, err := os.Stat(manifestPath); err != nil {
			continue
		}
		if seen[spec.manager] {
			// check lockfile variant (e.g., yarn.lock vs package-lock.json)
			lockPath := filepath.Join(root, spec.lockfile)
			if _, err := os.Stat(lockPath); err == nil {
				info := result.Details[spec.manager]
				info.HasLockfile = true
				result.Details[spec.manager] = info
				result.HasLockfile = true
			}
			continue
		}
		seen[spec.manager] = true

		count := 0
		if spec.counter != nil {
			count = spec.counter(manifestPath)
		}

		lockPath := filepath.Join(root, spec.lockfile)
		hasLock := false
		if spec.manifest != spec.lockfile {
			if _, err := os.Stat(lockPath); err == nil {
				hasLock = true
			}
		} else {
			hasLock = true // requirements.txt is its own "lock"
		}

		result.Details[spec.manager] = DepInfo{
			Manager:     spec.manager,
			DepCount:    count,
			HasLockfile: hasLock,
		}
		result.TotalDeps += count
		result.DepFiles = append(result.DepFiles, spec.manager)
		if hasLock {
			result.HasLockfile = true
		}
	}

	return result
}

func countGoModDeps(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()

	count := 0
	inRequire := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "require (") || line == "require (" {
			inRequire = true
			continue
		}
		if inRequire {
			if line == ")" {
				inRequire = false
				continue
			}
			if line != "" && !strings.HasPrefix(line, "//") {
				count++
			}
		}
		if strings.HasPrefix(line, "require ") && !strings.Contains(line, "(") {
			count++
		}
	}
	return count
}

func countPackageJSONDeps(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()

	count := 0
	inDeps := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.Contains(line, `"dependencies"`) || strings.Contains(line, `"devDependencies"`) {
			inDeps = true
			continue
		}
		if inDeps {
			if strings.HasPrefix(line, "}") {
				inDeps = false
				continue
			}
			if strings.Contains(line, `"`) {
				count++
			}
		}
	}
	return count
}

func countLineBasedDeps(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			count++
		}
	}
	return count
}

func countCargoTOMLDeps(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()

	count := 0
	inDeps := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "[dependencies]" || line == "[dev-dependencies]" || line == "[build-dependencies]" {
			inDeps = true
			continue
		}
		if strings.HasPrefix(line, "[") {
			inDeps = false
			continue
		}
		if inDeps && line != "" && !strings.HasPrefix(line, "#") && strings.Contains(line, "=") {
			count++
		}
	}
	return count
}
