package discovery

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Package represents a discovered Go package
type Package struct {
	ModulePath  string
	GoVersion   string
	Path        string
	PackageName string
	GitHubRepo  string
	LatestTag   string
}

// DiscoverPackages scans for Go modules and returns discovered packages
func DiscoverPackages(searchPaths []string) ([]Package, error) {
	var packages []Package
	seen := make(map[string]bool)

	for _, searchPath := range searchPaths {
		pkgs, err := scanDirectory(searchPath)
		if err != nil {
			return nil, fmt.Errorf("failed to scan directory %s: %w", searchPath, err)
		}

		for _, pkg := range pkgs {
			// Avoid duplicates
			if !seen[pkg.ModulePath] {
				packages = append(packages, pkg)
				seen[pkg.ModulePath] = true
			}
		}
	}

	// Sort packages by module path
	sort.Slice(packages, func(i, j int) bool {
		return packages[i].ModulePath < packages[j].ModulePath
	})

	return packages, nil
}

// scanDirectory recursively scans a directory for go.mod files
func scanDirectory(rootPath string) ([]Package, error) {
	var packages []Package

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories and common build/cache directories
		if info.IsDir() && (strings.HasPrefix(info.Name(), ".") ||
			info.Name() == "node_modules" ||
			info.Name() == "vendor" ||
			info.Name() == "build" ||
			info.Name() == "dist") {
			return filepath.SkipDir
		}

		// Check if this is a go.mod file
		if info.Name() == "go.mod" {
			pkg, err := parseGoMod(path)
			if err != nil {
				// Log error but continue scanning
				fmt.Printf("Warning: failed to parse %s: %v\n", path, err)
				return nil
			}

			packages = append(packages, pkg)
		}

		return nil
	})

	return packages, err
}

// parseGoMod parses a go.mod file and extracts module information
func parseGoMod(filePath string) (Package, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return Package{}, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var modulePath, goVersion string

	// Read file to find module declaration and go version
	for scanner.Scan() && len(scanner.Text()) < 1000 { // Prevent reading huge files
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "module ") {
			modulePath = strings.TrimSpace(strings.TrimPrefix(line, "module"))
		} else if strings.HasPrefix(line, "go ") {
			goVersion = strings.TrimSpace(strings.TrimPrefix(line, "go"))
		}
	}

	if err := scanner.Err(); err != nil {
		return Package{}, err
	}

	if modulePath == "" {
		return Package{}, fmt.Errorf("no module declaration found in %s", filePath)
	}

	// Extract package name from module path
	packageName := extractPackageName(modulePath)

	// Get GitHub repository from git config
	githubRepo := getGitHubRepo(filepath.Dir(filePath))

	// Get latest tag from git
	latestTag := getLatestTag(filepath.Dir(filePath))

	return Package{
		ModulePath:  modulePath,
		GoVersion:   goVersion,
		Path:        filepath.Dir(filePath),
		PackageName: packageName,
		GitHubRepo:  githubRepo,
		LatestTag:   latestTag,
	}, nil
}

// extractPackageName extracts a clean package name from module path
func extractPackageName(modulePath string) string {
	// Get the last part of the module path
	parts := strings.Split(modulePath, "/")
	if len(parts) == 0 {
		return modulePath
	}

	lastPart := parts[len(parts)-1]

	// Remove any version suffixes (e.g., v2, v3)
	if strings.HasPrefix(lastPart, "v") && len(lastPart) > 1 {
		if _, err := strconv.Atoi(lastPart[1:]); err == nil {
			if len(parts) > 1 {
				return parts[len(parts)-2]
			}
		}
	}

	return lastPart
}

// getGitHubRepo gets the GitHub repository URL from git config
func getGitHubRepo(path string) string {
	// Try to get remote origin URL
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	url := strings.TrimSpace(string(output))
	if url == "" {
		return ""
	}

	// Convert git URL to GitHub URL format
	// Handle both SSH and HTTPS formats
	if strings.HasPrefix(url, "git@github.com:") {
		// SSH format: git@github.com:owner/repo.git
		repo := strings.TrimPrefix(url, "git@github.com:")
		repo = strings.TrimSuffix(repo, ".git")
		return fmt.Sprintf("github.com/%s", repo)
	} else if strings.HasPrefix(url, "https://github.com/") {
		// HTTPS format: https://github.com/owner/repo.git
		repo := strings.TrimPrefix(url, "https://github.com/")
		repo = strings.TrimSuffix(repo, ".git")
		return fmt.Sprintf("github.com/%s", repo)
	}

	return ""
}

// getLatestTag gets the latest tag for a specific package from git
func getLatestTag(path string) string {
	// Get the directory name to use as package name
	packageName := filepath.Base(path)

	// Try to get package-specific tags first (format: package-name/v*)
	cmd := exec.Command("git", "tag", "--sort=-version:refname", "--list", fmt.Sprintf("%s/v*", packageName))
	cmd.Dir = path
	output, err := cmd.Output()
	if err == nil {
		tags := strings.Split(strings.TrimSpace(string(output)), "\n")
		if len(tags) > 0 && tags[0] != "" {
			return tags[0]
		}
	}

	// If no package-specific tags found, try alternative patterns
	patterns := []string{
		fmt.Sprintf("%s-*", packageName),  // package-name-v1.2.3
		fmt.Sprintf("v*%s*", packageName), // v1.2.3-package-name
		"v*",                              // any v* tags
	}

	for _, pattern := range patterns {
		cmd := exec.Command("git", "tag", "--sort=-version:refname", "--list", pattern)
		cmd.Dir = path
		output, err := cmd.Output()
		if err == nil {
			tags := strings.Split(strings.TrimSpace(string(output)), "\n")
			if len(tags) > 0 && tags[0] != "" {
				return tags[0]
			}
		}
	}

	return ""
}

// GetDefaultSearchPaths returns default search paths for package discovery
func GetDefaultSearchPaths() []string {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return []string{"."}
	}

	// Only search current directory and its children
	// This prevents scanning parent directories
	return []string{cwd}
}
