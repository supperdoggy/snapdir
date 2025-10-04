package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShouldIgnore(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		patterns []string
		want     bool
	}{
		{
			name:     "matches exact filename",
			path:     "test.log",
			patterns: []string{"*.log"},
			want:     true,
		},
		{
			name:     "matches directory name",
			path:     "node_modules/package",
			patterns: []string{"node_modules"},
			want:     true,
		},
		{
			name:     "no match",
			path:     "src/main.go",
			patterns: []string{"*.log", "node_modules"},
			want:     false,
		},
		{
			name:     "matches nested directory",
			path:     "src/.git/config",
			patterns: []string{".git"},
			want:     true,
		},
		{
			name:     "empty patterns",
			path:     "any/path",
			patterns: []string{},
			want:     false,
		},
		{
			name:     "matches wildcard pattern",
			path:     "test.tmp",
			patterns: []string{"*.tmp", "*.log"},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldIgnore(tt.path, tt.patterns); got != tt.want {
				t.Errorf("shouldIgnore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoadGitignore(t *testing.T) {
	tests := []struct {
		name            string
		gitignoreContent string
		wantPatterns    []string
	}{
		{
			name: "basic gitignore",
			gitignoreContent: `# Comment
node_modules
*.log
.env`,
			wantPatterns: []string{".git", "node_modules", "*.log", ".env"},
		},
		{
			name: "empty lines and comments",
			gitignoreContent: `
# Comment

dist

# Another comment
build
`,
			wantPatterns: []string{".git", "dist", "build"},
		},
		{
			name:            "empty gitignore",
			gitignoreContent: "",
			wantPatterns:    []string{".git"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir := t.TempDir()

			// Write .gitignore file
			gitignorePath := filepath.Join(tmpDir, ".gitignore")
			if err := os.WriteFile(gitignorePath, []byte(tt.gitignoreContent), 0644); err != nil {
				t.Fatalf("failed to write .gitignore: %v", err)
			}

			got := loadGitignore(tmpDir)

			if len(got) != len(tt.wantPatterns) {
				t.Errorf("loadGitignore() returned %d patterns, want %d", len(got), len(tt.wantPatterns))
			}

			for i, pattern := range tt.wantPatterns {
				if i >= len(got) || got[i] != pattern {
					t.Errorf("pattern[%d] = %v, want %v", i, got[i], pattern)
				}
			}
		})
	}
}

func TestLoadGitignoreNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	patterns := loadGitignore(tmpDir)

	// Should return default patterns even if .gitignore doesn't exist
	if len(patterns) != 1 || patterns[0] != ".git" {
		t.Errorf("expected default patterns [.git], got %v", patterns)
	}
}

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		mustExist bool
		setup     func(t *testing.T) string
		wantErr   bool
	}{
		{
			name:      "empty path",
			path:      "",
			mustExist: false,
			wantErr:   true,
		},
		{
			name:      "existing path",
			mustExist: true,
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			wantErr: false,
		},
		{
			name:      "non-existing path with mustExist false",
			path:      "/nonexistent/path",
			mustExist: false,
			wantErr:   false,
		},
		{
			name:      "non-existing path with mustExist true",
			path:      "/nonexistent/path",
			mustExist: true,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.path
			if tt.setup != nil {
				path = tt.setup(t)
			}

			err := validatePath(path, tt.mustExist)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCloneProject(t *testing.T) {
	// Create a test directory structure
	tmpDir := t.TempDir()

	// Create some test files and directories
	testFiles := map[string]string{
		"file1.txt":           "content1",
		"dir1/file2.txt":      "content2",
		"dir1/dir2/file3.txt": "content3",
		".gitignore":          "*.log\n",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
	}

	// Create a .log file that should be ignored
	logFile := filepath.Join(tmpDir, "test.log")
	if err := os.WriteFile(logFile, []byte("log content"), 0644); err != nil {
		t.Fatalf("failed to write log file: %v", err)
	}

	// Clone the project
	outputFile := filepath.Join(tmpDir, "snapshot.json")
	if err := cloneProject(tmpDir, outputFile); err != nil {
		t.Fatalf("cloneProject() error = %v", err)
	}

	// Read and validate the snapshot
	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read snapshot: %v", err)
	}

	var snapshot ProjectSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatalf("failed to unmarshal snapshot: %v", err)
	}

	// Verify version
	if snapshot.Version != version {
		t.Errorf("snapshot version = %v, want %v", snapshot.Version, version)
	}

	// Verify .log file is ignored
	for _, file := range snapshot.Files {
		if strings.HasSuffix(file.Path, ".log") {
			t.Errorf("log file should be ignored but found: %s", file.Path)
		}
	}

	// Verify expected files are present
	expectedFiles := []string{"file1.txt", "dir1/file2.txt", "dir1/dir2/file3.txt"}
	for _, expected := range expectedFiles {
		found := false
		for _, file := range snapshot.Files {
			if file.Path == expected {
				found = true
				if file.IsDir {
					t.Errorf("file %s marked as directory", expected)
				}
				break
			}
		}
		if !found {
			t.Errorf("expected file %s not found in snapshot", expected)
		}
	}
}

func TestCloneProjectInvalidSource(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wantErr bool
	}{
		{
			name:    "non-existent source",
			source:  "/nonexistent/path",
			wantErr: true,
		},
		{
			name:    "empty source",
			source:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpOutput := filepath.Join(t.TempDir(), "output.json")
			err := cloneProject(tt.source, tmpOutput)
			if (err != nil) != tt.wantErr {
				t.Errorf("cloneProject() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCloneProjectFileAsSource(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "file.txt")
	if err := os.WriteFile(tmpFile, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	outputFile := filepath.Join(tmpDir, "snapshot.json")
	err := cloneProject(tmpFile, outputFile)
	if err == nil {
		t.Error("cloneProject() should fail when source is a file, not directory")
	}
}

func TestRestoreProject(t *testing.T) {
	// Create a snapshot
	snapshot := ProjectSnapshot{
		Version: version,
		Files: []FileInfo{
			{Path: "file1.txt", Contents: "content1", IsDir: false, Mode: 0644},
			{Path: "dir1", IsDir: true, Mode: 0755},
			{Path: "dir1/file2.txt", Contents: "content2", IsDir: false, Mode: 0644},
			{Path: "dir1/dir2", IsDir: true, Mode: 0755},
			{Path: "dir1/dir2/file3.txt", Contents: "content3", IsDir: false, Mode: 0644},
		},
	}

	tmpDir := t.TempDir()
	snapshotFile := filepath.Join(tmpDir, "snapshot.json")

	// Write snapshot file
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal snapshot: %v", err)
	}
	if err := os.WriteFile(snapshotFile, data, 0644); err != nil {
		t.Fatalf("failed to write snapshot file: %v", err)
	}

	// Restore the project
	destDir := filepath.Join(tmpDir, "restored")
	if err := restoreProject(snapshotFile, destDir); err != nil {
		t.Fatalf("restoreProject() error = %v", err)
	}

	// Verify restored files
	for _, file := range snapshot.Files {
		restoredPath := filepath.Join(destDir, file.Path)
		info, err := os.Stat(restoredPath)
		if err != nil {
			t.Errorf("restored file/dir %s not found: %v", file.Path, err)
			continue
		}

		if info.IsDir() != file.IsDir {
			t.Errorf("file %s: IsDir = %v, want %v", file.Path, info.IsDir(), file.IsDir)
		}

		if !file.IsDir {
			content, err := os.ReadFile(restoredPath)
			if err != nil {
				t.Errorf("failed to read restored file %s: %v", file.Path, err)
				continue
			}
			if string(content) != file.Contents {
				t.Errorf("file %s: content = %q, want %q", file.Path, string(content), file.Contents)
			}
		}
	}
}

func TestRestoreProjectInvalidConfig(t *testing.T) {
	tests := []struct {
		name        string
		setupConfig func(t *testing.T) string
		wantErr     bool
	}{
		{
			name: "non-existent config",
			setupConfig: func(t *testing.T) string {
				return "/nonexistent/config.json"
			},
			wantErr: true,
		},
		{
			name: "invalid JSON",
			setupConfig: func(t *testing.T) string {
				tmpFile := filepath.Join(t.TempDir(), "invalid.json")
				os.WriteFile(tmpFile, []byte("invalid json"), 0644)
				return tmpFile
			},
			wantErr: true,
		},
		{
			name: "empty config",
			setupConfig: func(t *testing.T) string {
				return ""
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configFile := tt.setupConfig(t)
			destDir := filepath.Join(t.TempDir(), "dest")
			err := restoreProject(configFile, destDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("restoreProject() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRestoreProjectExistingDestination(t *testing.T) {
	snapshot := ProjectSnapshot{
		Version: version,
		Files:   []FileInfo{{Path: "file.txt", Contents: "content", IsDir: false, Mode: 0644}},
	}

	tmpDir := t.TempDir()
	snapshotFile := filepath.Join(tmpDir, "snapshot.json")

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal snapshot: %v", err)
	}
	if err := os.WriteFile(snapshotFile, data, 0644); err != nil {
		t.Fatalf("failed to write snapshot file: %v", err)
	}

	// Create existing destination
	destDir := filepath.Join(tmpDir, "dest")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("failed to create destination: %v", err)
	}

	// Should fail because destination exists
	err = restoreProject(snapshotFile, destDir)
	if err == nil {
		t.Error("restoreProject() should fail when destination already exists")
	}
}

func TestCloneAndRestore(t *testing.T) {
	// Integration test: clone and then restore
	originalDir := t.TempDir()

	// Create test structure
	testFiles := map[string]string{
		"file1.txt":           "content1",
		"dir1/file2.txt":      "content2",
		"dir1/dir2/file3.txt": "content3",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(originalDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
	}

	// Clone
	snapshotFile := filepath.Join(t.TempDir(), "snapshot.json")
	if err := cloneProject(originalDir, snapshotFile); err != nil {
		t.Fatalf("cloneProject() error = %v", err)
	}

	// Restore
	restoredDir := filepath.Join(t.TempDir(), "restored")
	if err := restoreProject(snapshotFile, restoredDir); err != nil {
		t.Fatalf("restoreProject() error = %v", err)
	}

	// Verify all files are restored correctly
	for path, expectedContent := range testFiles {
		restoredPath := filepath.Join(restoredDir, path)
		content, err := os.ReadFile(restoredPath)
		if err != nil {
			t.Errorf("failed to read restored file %s: %v", path, err)
			continue
		}
		if string(content) != expectedContent {
			t.Errorf("file %s: content = %q, want %q", path, string(content), expectedContent)
		}
	}
}

func TestLogVerbose(t *testing.T) {
	// Test that logVerbose doesn't panic
	verbose = false
	logVerbose("test message")

	verbose = true
	logVerbose("test message with args: %s %d", "hello", 42)

	verbose = false
}
