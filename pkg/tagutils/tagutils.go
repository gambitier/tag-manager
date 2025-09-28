package tagutils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// TagInfo represents tag information
type TagInfo struct {
	PackageName string
	Major       int
	Minor       int
	Patch       int
	Version     string
}

// FormatTag formats a tag according to the given format string
func FormatTag(format string, pkgInfo TagInfo) string {
	tag := format

	// Replace placeholders
	tag = strings.ReplaceAll(tag, "{package-name}", pkgInfo.PackageName)
	tag = strings.ReplaceAll(tag, "{major}", strconv.Itoa(pkgInfo.Major))
	tag = strings.ReplaceAll(tag, "{minor}", strconv.Itoa(pkgInfo.Minor))
	tag = strings.ReplaceAll(tag, "{patch}", strconv.Itoa(pkgInfo.Patch))
	tag = strings.ReplaceAll(tag, "{version}", pkgInfo.Version)

	return tag
}

// ParseTag parses a tag string and extracts version information
func ParseTag(tag string) (*TagInfo, error) {
	// Try to match common tag formats
	patterns := []string{
		`^(.+)/v(\d+)\.(\d+)\.(\d+)$`, // package/v1.2.3
		`^v(\d+)\.(\d+)\.(\d+)$`,      // v1.2.3
		`^(.+)-v(\d+)\.(\d+)\.(\d+)$`, // package-v1.2.3
		`^(.+)-(\d+)\.(\d+)\.(\d+)$`,  // package-1.2.3
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(tag)

		if len(matches) >= 4 {
			major, err1 := strconv.Atoi(matches[len(matches)-3])
			minor, err2 := strconv.Atoi(matches[len(matches)-2])
			patch, err3 := strconv.Atoi(matches[len(matches)-1])

			if err1 == nil && err2 == nil && err3 == nil {
				pkgName := ""
				if len(matches) > 4 {
					pkgName = matches[1]
				}

				return &TagInfo{
					PackageName: pkgName,
					Major:       major,
					Minor:       minor,
					Patch:       patch,
					Version:     fmt.Sprintf("v%d.%d.%d", major, minor, patch),
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("unable to parse tag: %s", tag)
}

// CalculateNewVersion calculates a new version based on the current version and version type
func CalculateNewVersion(current *TagInfo, versionType string) (*TagInfo, error) {
	newVersion := *current

	switch versionType {
	case "major":
		newVersion.Major++
		newVersion.Minor = 0
		newVersion.Patch = 0
	case "minor":
		newVersion.Minor++
		newVersion.Patch = 0
	case "patch":
		newVersion.Patch++
	default:
		return nil, fmt.Errorf("invalid version type: %s", versionType)
	}

	newVersion.Version = fmt.Sprintf("v%d.%d.%d", newVersion.Major, newVersion.Minor, newVersion.Patch)

	return &newVersion, nil
}

// ExtractPackageNameFromModule extracts a package name from a module path
func ExtractPackageNameFromModule(modulePath string) string {
	// Get the last part of the module path
	parts := strings.Split(modulePath, "/")
	if len(parts) == 0 {
		return modulePath
	}

	// Remove any version suffixes (e.g., v2, v3)
	lastPart := parts[len(parts)-1]
	if strings.HasPrefix(lastPart, "v") && len(lastPart) > 1 {
		// Check if it's a version suffix
		if _, err := strconv.Atoi(lastPart[1:]); err == nil {
			if len(parts) > 1 {
				return parts[len(parts)-2]
			}
		}
	}

	return lastPart
}

// ValidateTagFormat validates a tag format string
func ValidateTagFormat(format string) error {
	// Check if format contains {version} (which is valid on its own)
	if strings.Contains(format, "{version}") {
		// {version} is valid on its own, no other placeholders required
	} else {
		// If not using {version}, check for required individual placeholders
		required := []string{"{major}", "{minor}", "{patch}"}
		for _, placeholder := range required {
			if !strings.Contains(format, placeholder) {
				return fmt.Errorf("tag format must contain %s placeholder", placeholder)
			}
		}
	}

	// Check for invalid characters
	invalidChars := []string{" ", "\t", "\n", "\r"}
	for _, char := range invalidChars {
		if strings.Contains(format, char) {
			return fmt.Errorf("tag format cannot contain whitespace characters")
		}
	}

	return nil
}
