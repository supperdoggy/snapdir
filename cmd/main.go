package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	version        = "1.0.0"
	defaultPerms   = 0644
	dirPerms       = 0755
	maxFileSize    = 100 * 1024 * 1024 // 100MB limit
	jsonIndent     = "  "
)

var (
	verbose        bool
	ignorePatterns []string
)

// FileInfo represents a file or directory in the snapshot
type FileInfo struct {
	Path     string `json:"path"`
	Contents string `json:"contents,omitempty"`
	IsDir    bool   `json:"is_dir"`
	Mode     uint32 `json:"mode,omitempty"`
}

// ProjectSnapshot represents the complete directory snapshot
type ProjectSnapshot struct {
	Version string     `json:"version"`
	Files   []FileInfo `json:"files"`
}

// shouldIgnore checks if a path should be ignored based on patterns
func shouldIgnore(path string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err != nil {
			logVerbose("Warning: invalid pattern %q: %v", pattern, err)
			continue
		}
		if matched {
			return true
		}

		// Check if path contains pattern as a directory component
		if strings.Contains(filepath.ToSlash(path), pattern) {
			return true
		}
	}
	return false
}

// loadGitignore loads .gitignore patterns from the source directory
func loadGitignore(source string) []string {
	patterns := []string{".git"}
	gitignorePath := filepath.Join(source, ".gitignore")

	file, err := os.Open(gitignorePath)
	if err != nil {
		logVerbose("No .gitignore found, using default patterns")
		return patterns
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}

	if err := scanner.Err(); err != nil {
		logVerbose("Warning: error reading .gitignore: %v", err)
	}

	return patterns
}

// logVerbose logs a message if verbose mode is enabled
func logVerbose(format string, args ...any) {
	if verbose {
		log.Printf(format, args...)
	}
}

// validatePath ensures a path exists and is accessible
func validatePath(path string, mustExist bool) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	if mustExist {
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("path does not exist: %s", path)
			}
			return fmt.Errorf("cannot access path %s: %w", path, err)
		}
	}

	return nil
}

// cloneProject creates a snapshot of the source directory
func cloneProject(source, outputFile string) error {
	if err := validatePath(source, true); err != nil {
		return fmt.Errorf("invalid source path: %w", err)
	}

	sourceInfo, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("failed to stat source: %w", err)
	}

	if !sourceInfo.IsDir() {
		return fmt.Errorf("source must be a directory: %s", source)
	}

	patterns := loadGitignore(source)
	if len(ignorePatterns) > 0 {
		patterns = append(patterns, ignorePatterns...)
	}

	logVerbose("Starting snapshot of %s", source)
	logVerbose("Ignore patterns: %v", patterns)

	snapshot := ProjectSnapshot{
		Version: version,
		Files:   make([]FileInfo, 0),
	}

	fileCount := 0
	err = filepath.WalkDir(source, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing %s: %w", path, err)
		}

		relPath, err := filepath.Rel(source, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %w", path, err)
		}

		if relPath == "." {
			return nil
		}

		if shouldIgnore(relPath, patterns) {
			logVerbose("Ignoring: %s", relPath)
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("failed to get file info for %s: %w", path, err)
		}

		fileInfo := FileInfo{
			Path:  filepath.ToSlash(relPath),
			IsDir: d.IsDir(),
			Mode:  uint32(info.Mode().Perm()),
		}

		if !d.IsDir() {
			if info.Size() > maxFileSize {
				logVerbose("Skipping large file: %s (size: %d bytes)", relPath, info.Size())
				return nil
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read file %s: %w", path, err)
			}
			fileInfo.Contents = string(data)
			fileCount++
		}

		snapshot.Files = append(snapshot.Files, fileInfo)
		logVerbose("Added: %s", relPath)

		return nil
	})

	if err != nil {
		return err
	}

	logVerbose("Snapshot complete: %d files, %d total entries", fileCount, len(snapshot.Files))

	jsonData, err := json.MarshalIndent(snapshot, "", jsonIndent)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(outputFile, jsonData, defaultPerms); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	logVerbose("Snapshot saved to: %s", outputFile)
	return nil
}

// restoreProject restores a directory from a snapshot file
func restoreProject(configFile, destination string) error {
	if err := validatePath(configFile, true); err != nil {
		return fmt.Errorf("invalid config file: %w", err)
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var snapshot ProjectSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	logVerbose("Restoring snapshot (version: %s) to %s", snapshot.Version, destination)

	if _, err := os.Stat(destination); err == nil {
		return fmt.Errorf("destination already exists: %s (remove it first or choose a different location)", destination)
	}

	if err := os.MkdirAll(destination, dirPerms); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	for _, file := range snapshot.Files {
		path := filepath.Join(destination, filepath.FromSlash(file.Path))

		if file.IsDir {
			if err := os.MkdirAll(path, fs.FileMode(file.Mode)); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", file.Path, err)
			}
			logVerbose("Created directory: %s", file.Path)
		} else {
			parentDir := filepath.Dir(path)
			if err := os.MkdirAll(parentDir, dirPerms); err != nil {
				return fmt.Errorf("failed to create parent directory for %s: %w", file.Path, err)
			}

			mode := fs.FileMode(file.Mode)
			if mode == 0 {
				mode = defaultPerms
			}

			if err := os.WriteFile(path, []byte(file.Contents), mode); err != nil {
				return fmt.Errorf("failed to write file %s: %w", file.Path, err)
			}
			logVerbose("Restored file: %s", file.Path)
		}
	}

	logVerbose("Restore complete: %d entries restored", len(snapshot.Files))
	return nil
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "snapdir v%s - Directory snapshot and restore tool\n\n", version)
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  %s clone <source_dir> <output.json> [flags]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s restore <config.json> <destination_dir> [flags]\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Flags:\n")
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\nExamples:\n")
	fmt.Fprintf(os.Stderr, "  %s clone ./myproject snapshot.json -v\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s restore snapshot.json ./restored -v\n", os.Args[0])
}

func main() {
	flag.BoolVar(&verbose, "v", false, "Enable verbose logging")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging (alias)")
	var ignoreFlag string
	flag.StringVar(&ignoreFlag, "ignore", "", "Additional ignore patterns (comma-separated)")
	showVersion := flag.Bool("version", false, "Show version information")

	flag.Usage = printUsage
	flag.Parse()

	if *showVersion {
		fmt.Printf("snapdir v%s\n", version)
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) < 3 {
		printUsage()
		os.Exit(1)
	}

	if ignoreFlag != "" {
		ignorePatterns = strings.Split(ignoreFlag, ",")
		for i := range ignorePatterns {
			ignorePatterns[i] = strings.TrimSpace(ignorePatterns[i])
		}
	}

	command := args[0]

	var err error
	switch command {
	case "clone":
		err = cloneProject(args[1], args[2])
		if err != nil {
			log.Fatalf("Error: failed to create snapshot: %v", err)
		}
		fmt.Println("Snapshot created successfully")

	case "restore":
		err = restoreProject(args[1], args[2])
		if err != nil {
			log.Fatalf("Error: failed to restore snapshot: %v", err)
		}
		fmt.Println("Snapshot restored successfully")

	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command %q\n\n", command)
		printUsage()
		os.Exit(1)
	}
}
