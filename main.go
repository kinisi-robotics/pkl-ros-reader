package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/apple/pkl-go/pkl"
)

func main() {
	client, err := pkl.NewExternalReaderClient(
		pkl.WithExternalClientModuleReader(&rospkgModuleReader{}),
	)
	if err != nil {
		log.Fatalln(err)
	}
	if err := client.Run(); err != nil {
		log.Fatalln(err)
	}
}

type rospkgModuleReader struct{}

var _ pkl.ModuleReader = &rospkgModuleReader{}

// Scheme returns "rospkg" - the URI scheme this reader handles
func (r *rospkgModuleReader) Scheme() string {
	return "rospkg"
}

// HasHierarchicalUris returns true - rospkg: URIs support hierarchy (rospkg:///path/to/file.pkl)
func (r *rospkgModuleReader) HasHierarchicalUris() bool {
	return true
}

// IsGlobbable returns false - we don't support glob imports
func (r *rospkgModuleReader) IsGlobbable() bool {
	return false
}

// IsLocal returns true - files are local to the filesystem
func (r *rospkgModuleReader) IsLocal() bool {
	return true
}

// ListElements is not implemented (globbing not supported)
func (r *rospkgModuleReader) ListElements(baseURI url.URL) ([]pkl.PathElement, error) {
	return nil, nil
}

// Read resolves a rospkg: URI to a ROS package file
//
// URI format:
//
//	rospkg:///package_name/path/to/file.pkl
//
// Resolution order:
//  1. Source directory via git repo root: <git_root>/ros/package_name/path/to/file.pkl
//  2. ROS package discovery via `ros2 pkg prefix <package_name>`
//
// Example:
//
//	rospkg:///my_package/config/my_config.pkl
//	-> <git_root>/ros/my_package/config/my_config.pkl (source)
//	-> $(ros2 pkg prefix my_package)/share/my_package/config/my_config.pkl (installed)
func (r *rospkgModuleReader) Read(uri url.URL) (string, error) {
	// For hierarchical URIs: rospkg:///package_name/path/to/file.pkl
	// The path is in uri.Path (starts with /)
	relativePath := uri.Path
	if len(relativePath) > 0 && relativePath[0] == '/' {
		relativePath = relativePath[1:] // Remove leading /
	}

	// Extract package name (first component of path)
	// e.g., "my_package/config/system/file.pkl" -> package="my_package", rest="config/system/file.pkl"
	parts := splitPath(relativePath)
	if len(parts) == 0 {
		return "", &os.PathError{
			Op:   "read",
			Path: relativePath,
			Err:  os.ErrInvalid,
		}
	}

	packageName := parts[0]
	packageRelPath := ""
	if len(parts) > 1 {
		packageRelPath = filepath.Join(parts[1:]...)
	}

	// Try source directory via git repo root
	if repoRoot := findGitRoot(); repoRoot != "" {
		sourcePath := filepath.Join(repoRoot, "ros", relativePath)
		if contents, err := os.ReadFile(sourcePath); err == nil {
			return string(contents), nil
		}
	}

	// Try ROS package discovery via `ros2 pkg prefix`
	if packagePath := findROSPackage(packageName, packageRelPath); packagePath != "" {
		if contents, err := os.ReadFile(packagePath); err == nil {
			return string(contents), nil
		}
	}

	return "", fmt.Errorf("read %s: could not resolve rospkg:///%s (tried git root source and ros2 pkg prefix)", relativePath, relativePath)
}

// findGitRoot returns the git repository root directory, or empty string if not in a git repo
func findGitRoot() string {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// findROSPackage searches for a ROS package using `ros2 pkg prefix`
// Returns the full path to the requested file in the package, or empty string if not found
func findROSPackage(packageName, relPath string) string {
	// Use ros2 pkg prefix to find the package installation prefix
	cmd := exec.Command("ros2", "pkg", "prefix", packageName)
	output, err := cmd.Output()
	if err != nil {
		// Package not found or ros2 not available
		return ""
	}

	// Parse the output (trim whitespace/newlines)
	prefix := strings.TrimSpace(string(output))
	if prefix == "" {
		return ""
	}

	// Construct path: <prefix>/share/<package_name>/<relPath>
	packagePath := filepath.Join(prefix, "share", packageName, relPath)

	// Verify the file exists
	if _, err := os.Stat(packagePath); err != nil {
		return ""
	}

	return packagePath
}

// splitPath splits a path into components using / as separator
func splitPath(path string) []string {
	normalized := filepath.ToSlash(path)
	if normalized == "" {
		return []string{}
	}
	return splitOnSlash(normalized)
}

func splitOnSlash(path string) []string {
	var result []string
	current := ""
	for _, ch := range path {
		if ch == '/' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}
