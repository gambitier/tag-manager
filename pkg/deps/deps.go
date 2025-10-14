package deps

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gambitier/tag-manager/pkg/discovery"
)

// Package represents a Go package with its dependencies
type Package struct {
	ModulePath   string
	PackageName  string
	Directory    string
	Dependencies []string
	Dependents   []string
	IsFoundation bool
	Level        int
}

// DependencyGraph represents the complete dependency graph
type DependencyGraph struct {
	Packages map[string]*Package
	Levels   map[int][]*Package
}

// AnalyzeDependencies analyzes dependencies for all discovered packages
func AnalyzeDependencies(searchPaths []string) (*DependencyGraph, error) {
	// First discover all packages
	packages, err := discovery.DiscoverPackages(searchPaths)
	if err != nil {
		return nil, fmt.Errorf("failed to discover packages: %w", err)
	}

	graph := &DependencyGraph{
		Packages: make(map[string]*Package),
		Levels:   make(map[int][]*Package),
	}

	// Initialize packages in graph
	for _, pkg := range packages {
		graph.Packages[pkg.ModulePath] = &Package{
			ModulePath:   pkg.ModulePath,
			PackageName:  pkg.PackageName,
			Directory:    pkg.Path,
			Dependencies: []string{},
			Dependents:   []string{},
			IsFoundation: true,
			Level:        0,
		}
	}

	// Analyze dependencies for each package
	for _, pkg := range packages {
		deps, err := getPackageDependencies(pkg.Path)
		if err != nil {
			// Continue even if we can't analyze one package
			continue
		}

		// Filter to only include internal dependencies (same repo)
		internalDeps := filterInternalDependencies(deps, graph.Packages)
		graph.Packages[pkg.ModulePath].Dependencies = internalDeps
		graph.Packages[pkg.ModulePath].IsFoundation = len(internalDeps) == 0

		// Update dependents for each dependency
		for _, dep := range internalDeps {
			if dependentPkg, exists := graph.Packages[dep]; exists {
				dependentPkg.Dependents = append(dependentPkg.Dependents, pkg.ModulePath)
			}
		}
	}

	// Calculate dependency levels
	calculateLevels(graph)

	return graph, nil
}

// getPackageDependencies extracts dependencies from go.mod file
func getPackageDependencies(packageDir string) ([]string, error) {
	goModPath := filepath.Join(packageDir, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("go.mod not found in %s", packageDir)
	}

	file, err := os.Open(goModPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open go.mod: %w", err)
	}
	defer file.Close()

	var dependencies []string
	scanner := bufio.NewScanner(file)
	inRequire := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "require (" {
			inRequire = true
			continue
		}

		if inRequire && line == ")" {
			break
		}

		if inRequire && line != "" && !strings.HasPrefix(line, "//") {
			// Extract module path (everything before the version)
			parts := strings.Fields(line)
			if len(parts) > 0 {
				dependencies = append(dependencies, parts[0])
			}
		}

		// Handle single-line require statements
		if strings.HasPrefix(line, "require ") && !strings.Contains(line, "(") {
			parts := strings.Fields(line)
			if len(parts) > 1 {
				dependencies = append(dependencies, parts[1])
			}
		}
	}

	return dependencies, scanner.Err()
}

// filterInternalDependencies filters dependencies to only include internal packages
func filterInternalDependencies(deps []string, packages map[string]*Package) []string {
	var internalDeps []string
	for _, dep := range deps {
		if _, exists := packages[dep]; exists {
			internalDeps = append(internalDeps, dep)
		}
	}
	return internalDeps
}

// calculateLevels calculates the dependency level for each package
func calculateLevels(graph *DependencyGraph) {
	// Reset levels
	for _, pkg := range graph.Packages {
		pkg.Level = -1
	}

	// Calculate levels using topological sort
	visited := make(map[string]bool)
	var calculateLevel func(pkg *Package) int

	calculateLevel = func(pkg *Package) int {
		if visited[pkg.ModulePath] {
			return pkg.Level
		}

		visited[pkg.ModulePath] = true
		maxDepLevel := -1

		for _, dep := range pkg.Dependencies {
			if depPkg, exists := graph.Packages[dep]; exists {
				depLevel := calculateLevel(depPkg)
				if depLevel > maxDepLevel {
					maxDepLevel = depLevel
				}
			}
		}

		pkg.Level = maxDepLevel + 1
		return pkg.Level
	}

	// Calculate levels for all packages
	for _, pkg := range graph.Packages {
		calculateLevel(pkg)
	}

	// Group packages by level
	graph.Levels = make(map[int][]*Package)
	for _, pkg := range graph.Packages {
		graph.Levels[pkg.Level] = append(graph.Levels[pkg.Level], pkg)
	}

	// Sort packages within each level
	for level := range graph.Levels {
		sort.Slice(graph.Levels[level], func(i, j int) bool {
			return graph.Levels[level][i].PackageName < graph.Levels[level][j].PackageName
		})
	}
}

// GetUpdateImpact returns packages that need to be updated when a given package is updated
func (g *DependencyGraph) GetUpdateImpact(packagePath string) []string {
	var impact []string
	visited := make(map[string]bool)

	var collectDependents func(pkgPath string)
	collectDependents = func(pkgPath string) {
		if visited[pkgPath] {
			return
		}
		visited[pkgPath] = true

		if pkg, exists := g.Packages[pkgPath]; exists {
			for _, dependent := range pkg.Dependents {
				impact = append(impact, dependent)
				collectDependents(dependent)
			}
		}
	}

	collectDependents(packagePath)
	return impact
}

// GetUpdateOrder returns packages in the order they should be updated (bottom-up)
func (g *DependencyGraph) GetUpdateOrder() []*Package {
	var order []*Package
	maxLevel := 0

	// Find max level
	for level := range g.Levels {
		if level > maxLevel {
			maxLevel = level
		}
	}

	// Add packages from bottom to top
	for level := 0; level <= maxLevel; level++ {
		if packages, exists := g.Levels[level]; exists {
			order = append(order, packages...)
		}
	}

	return order
}

// GetFoundationPackages returns packages with no internal dependencies
func (g *DependencyGraph) GetFoundationPackages() []*Package {
	var foundation []*Package
	for _, pkg := range g.Packages {
		if pkg.IsFoundation {
			foundation = append(foundation, pkg)
		}
	}
	sort.Slice(foundation, func(i, j int) bool {
		return foundation[i].PackageName < foundation[j].PackageName
	})
	return foundation
}

// GetTopLevelPackages returns packages with no dependents
func (g *DependencyGraph) GetTopLevelPackages() []*Package {
	var topLevel []*Package
	for _, pkg := range g.Packages {
		if len(pkg.Dependents) == 0 {
			topLevel = append(topLevel, pkg)
		}
	}
	sort.Slice(topLevel, func(i, j int) bool {
		return topLevel[i].PackageName < topLevel[j].PackageName
	})
	return topLevel
}

// ValidateGraph checks for circular dependencies
func (g *DependencyGraph) ValidateGraph() error {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var hasCycle func(pkgPath string) bool
	hasCycle = func(pkgPath string) bool {
		visited[pkgPath] = true
		recStack[pkgPath] = true

		if pkg, exists := g.Packages[pkgPath]; exists {
			for _, dep := range pkg.Dependencies {
				if !visited[dep] {
					if hasCycle(dep) {
						return true
					}
				} else if recStack[dep] {
					return true
				}
			}
		}

		recStack[pkgPath] = false
		return false
	}

	for pkgPath := range g.Packages {
		if !visited[pkgPath] {
			if hasCycle(pkgPath) {
				return fmt.Errorf("circular dependency detected")
			}
		}
	}

	return nil
}

// GetPackageByPath returns a package by its module path
func (g *DependencyGraph) GetPackageByPath(modulePath string) (*Package, bool) {
	pkg, exists := g.Packages[modulePath]
	return pkg, exists
}

// GetPackageByName returns a package by its package name
func (g *DependencyGraph) GetPackageByName(packageName string) (*Package, bool) {
	for _, pkg := range g.Packages {
		if pkg.PackageName == packageName {
			return pkg, true
		}
	}
	return nil, false
}
