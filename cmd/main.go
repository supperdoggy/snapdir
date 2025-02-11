package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type FileInfo struct {
	Path     string `json:"path"`
	Contents string `json:"contents,omitempty"`
	IsDir    bool   `json:"is_dir"`
}

type ProjectClone struct {
	Files []FileInfo `json:"files"`
}

// Clone the project recursively
func cloneProject(source string, outputFile string) error {
	var project ProjectClone

	err := filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(source, path)
		if relPath == "." {
			return nil // Skip root folder
		}

		file := FileInfo{
			Path:  relPath,
			IsDir: info.IsDir(),
		}

		if !info.IsDir() {
			data, err := os.ReadFile(path)
			if err == nil {
				file.Contents = string(data)
			}
		}

		project.Files = append(project.Files, file)
		return nil
	})

	if err != nil {
		return err
	}

	jsonData, err := json.MarshalIndent(project, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(outputFile, jsonData, 0644)
}

// Restore the project from JSON config
func restoreProject(configFile string, destination string) error {
	var project ProjectClone
	data, err := os.ReadFile(configFile)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &project); err != nil {
		return err
	}

	for _, file := range project.Files {
		path := filepath.Join(destination, file.Path)
		if file.IsDir {
			if err := os.MkdirAll(path, os.ModePerm); err != nil {
				return err
			}
		} else {
			if err := os.WriteFile(path, []byte(file.Contents), 0644); err != nil {
				return err
			}
		}
	}

	return nil
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage:")
		fmt.Println("  clone <source_path> <output.json>")
		fmt.Println("  restore <config.json> <destination_path>")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "clone":
		if err := cloneProject(os.Args[2], os.Args[3]); err != nil {
			fmt.Println("Error cloning project:", err)
		}
	case "restore":
		if err := restoreProject(os.Args[2], os.Args[3]); err != nil {
			fmt.Println("Error restoring project:", err)
		}
	default:
		fmt.Println("Unknown command.")
	}
}
