// Copyright 2025 Antenore Gatta
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package files

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type FileLoader struct {
	MaxFileSize     int64
	MaxFiles        int
	gitignoreFilter *GitignoreFilter
}

func NewFileLoader() *FileLoader {
	return NewFileLoaderWithOptions(true) // Respect .gitignore by default
}

// NewFileLoaderWithOptions creates a FileLoader with gitignore options
func NewFileLoaderWithOptions(respectGitignore bool) *FileLoader {
	return &FileLoader{
		MaxFileSize:     10 * 1024 * 1024, // 10MB default
		MaxFiles:        100,
		gitignoreFilter: NewGitignoreFilter(respectGitignore),
	}
}

type LoadedFile struct {
	Path     string
	RelPath  string
	Content  string
	Size     int64
	Language string
}

func (fl *FileLoader) LoadFiles(patterns []string) ([]LoadedFile, error) {
	// Validate patterns first
	for _, pattern := range patterns {
		if err := fl.validatePattern(pattern); err != nil {
			return nil, err
		}
	}

	// First, expand all patterns and collect unique paths
	allPaths := make(map[string]bool)
	for _, pattern := range patterns {
		matches, err := fl.expandPattern(pattern)
		if err != nil {
			return nil, fmt.Errorf("error expanding pattern %s: %w", pattern, err)
		}
		for _, path := range matches {
			absPath, err := filepath.Abs(path)
			if err != nil {
				continue
			}
			allPaths[absPath] = true
		}
	}

	// Check if we would exceed the file limit
	if len(allPaths) > fl.MaxFiles {
		return nil, fmt.Errorf("pattern matches %d files, exceeds maximum limit of %d. Use more specific patterns like '*.go' instead of '*'", len(allPaths), fl.MaxFiles)
	}

	// Now load all the files
	var files []LoadedFile
	for absPath := range allPaths {
		file, err := fl.loadSingleFile(absPath)
		if err != nil {
			return nil, fmt.Errorf("error loading %s: %w", absPath, err)
		}
		files = append(files, file)
	}

	return files, nil
}

func (fl *FileLoader) LoadFile(path string) (LoadedFile, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return LoadedFile{}, fmt.Errorf("error resolving path: %w", err)
	}

	return fl.loadSingleFile(absPath)
}

func (fl *FileLoader) loadSingleFile(absPath string) (LoadedFile, error) {
	info, err := os.Stat(absPath)
	if err != nil {
		relPath, _ := filepath.Rel(".", absPath)
		return LoadedFile{}, fmt.Errorf("file not found: %s. Try: /load *.go or /list to see available files", relPath)
	}

	if info.IsDir() {
		relPath, _ := filepath.Rel(".", absPath)
		return LoadedFile{}, fmt.Errorf("'%s' is a directory, not a file. Try: /load %s/* or /load %s/**/*", relPath, relPath, relPath)
	}

	if info.Size() > fl.MaxFileSize {
		relPath, _ := filepath.Rel(".", absPath)
		sizeMB := float64(info.Size()) / (1024 * 1024)
		maxMB := float64(fl.MaxFileSize) / (1024 * 1024)
		return LoadedFile{}, fmt.Errorf("file too large: %s (%.1fMB, max: %.0fMB). Use a text editor to view large files", relPath, sizeMB, maxMB)
	}

	if fl.isBinaryFile(absPath) {
		relPath, _ := filepath.Rel(".", absPath)
		return LoadedFile{}, fmt.Errorf("'%s' appears to be a binary file, skipping. Use /load <text_files> instead", relPath)
	}

	file, err := os.Open(absPath)
	if err != nil {
		return LoadedFile{}, fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return LoadedFile{}, fmt.Errorf("error reading file: %w", err)
	}

	// Calculate relative path from current working directory
	cwd, cwdErr := os.Getwd()
	var relPath string

	if cwdErr == nil {
		relPath, err = filepath.Rel(cwd, absPath)
		// If the file is in current directory or subdirectory, use the relative path
		// Otherwise, keep the absolute path for clarity
		if err != nil || strings.HasPrefix(relPath, "..") {
			relPath = absPath
		}
	} else {
		// Fallback if we can't get cwd
		relPath, err = filepath.Rel(".", absPath)
		if err != nil {
			relPath = absPath
		}
	}

	return LoadedFile{
		Path:     absPath,
		RelPath:  relPath,
		Content:  string(content),
		Size:     info.Size(),
		Language: fl.detectLanguage(absPath),
	}, nil
}

func (fl *FileLoader) expandPattern(pattern string) ([]string, error) {
	if strings.Contains(pattern, "**") {
		return fl.expandDoubleStarPattern(pattern)
	}

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	if len(matches) == 0 && !strings.ContainsAny(pattern, "*?[") {
		if _, err := os.Stat(pattern); err == nil {
			return []string{pattern}, nil
		}
		return nil, fmt.Errorf("no files matching: %s. Try: /load *.* for all files, or /load *.go for Go files", pattern)
	}

	var files []string
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}
		if !info.IsDir() && !fl.gitignoreFilter.ShouldIgnore(match) {
			files = append(files, match)
		}
	}

	return files, nil
}

func (fl *FileLoader) expandDoubleStarPattern(pattern string) ([]string, error) {
	parts := strings.Split(pattern, "**")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid ** pattern: %s. Use format like 'src/**/*.go' for recursive search", pattern)
	}

	baseDir := strings.TrimSuffix(parts[0], string(filepath.Separator))
	if baseDir == "" {
		baseDir = "."
	}

	suffix := strings.TrimPrefix(parts[1], string(filepath.Separator))

	var matches []string
	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			// Skip hidden directories and directories ignored by gitignore
			if strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
				return filepath.SkipDir
			}
			if fl.gitignoreFilter.ShouldIgnore(path) {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip files ignored by gitignore
		if fl.gitignoreFilter.ShouldIgnore(path) {
			return nil
		}

		relPath, _ := filepath.Rel(baseDir, path)
		matched, _ := filepath.Match(suffix, filepath.Base(path))
		if matched || strings.HasSuffix(relPath, suffix) {
			matches = append(matches, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return matches, nil
}

func (fl *FileLoader) isBinaryFile(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return true
	}
	defer file.Close()

	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return true
	}

	for i := 0; i < n; i++ {
		if buf[i] == 0 {
			return true
		}
	}

	return false
}

func (fl *FileLoader) detectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	
	languageMap := map[string]string{
		".go":     "go",
		".js":     "javascript",
		".jsx":    "javascript",
		".ts":     "typescript",
		".tsx":    "typescript",
		".py":     "python",
		".rb":     "ruby",
		".java":   "java",
		".c":      "c",
		".cpp":    "cpp",
		".cc":     "cpp",
		".cxx":    "cpp",
		".h":      "c",
		".hpp":    "cpp",
		".cs":     "csharp",
		".php":    "php",
		".swift":  "swift",
		".kt":     "kotlin",
		".rs":     "rust",
		".sh":     "bash",
		".bash":   "bash",
		".zsh":    "zsh",
		".fish":   "fish",
		".ps1":    "powershell",
		".r":      "r",
		".R":      "r",
		".scala":  "scala",
		".clj":    "clojure",
		".cljs":   "clojure",
		".ex":     "elixir",
		".exs":    "elixir",
		".erl":    "erlang",
		".hrl":    "erlang",
		".lua":    "lua",
		".pl":     "perl",
		".pm":     "perl",
		".vim":    "vim",
		".sql":    "sql",
		".html":   "html",
		".htm":    "html",
		".xml":    "xml",
		".css":    "css",
		".scss":   "scss",
		".sass":   "sass",
		".less":   "less",
		".json":   "json",
		".yaml":   "yaml",
		".yml":    "yaml",
		".toml":   "toml",
		".ini":    "ini",
		".cfg":    "ini",
		".conf":   "conf",
		".md":     "markdown",
		".rst":    "rst",
		".tex":    "latex",
		".dart":   "dart",
		".zig":    "zig",
		".nim":    "nim",
		".v":      "v",
		".jl":     "julia",
		".ml":     "ocaml",
		".mli":    "ocaml",
		".fs":     "fsharp",
		".fsx":    "fsharp",
		".fsi":    "fsharp",
		".elm":    "elm",
		".purs":   "purescript",
		".hs":     "haskell",
		".lhs":    "haskell",
		".vue":    "vue",
		".svelte": "svelte",
	}

	if lang, ok := languageMap[ext]; ok {
		return lang
	}

	if base := filepath.Base(path); base == "Makefile" || base == "makefile" {
		return "makefile"
	}
	if base := filepath.Base(path); base == "Dockerfile" {
		return "dockerfile"
	}

	return "text"
}

func (fl *FileLoader) GetFilesInfo(files []LoadedFile) string {
	if len(files) == 0 {
		return "No files loaded"
	}

	var info strings.Builder
	
	// Header with file count
	if len(files) == 1 {
		info.WriteString("Loaded 1 file:\n\n")
	} else {
		info.WriteString(fmt.Sprintf("Loaded %d files:\n\n", len(files)))
	}
	
	totalSize := int64(0)
	for i, f := range files {
		// Get file type icon
		icon := fl.getFileTypeIcon(f.Language)
		
		// Format file size in human-readable format
		sizeStr := fl.formatFileSize(f.Size)
		
		// Enhanced file info with icon and better formatting
		info.WriteString(fmt.Sprintf("  %s %s\n", icon, f.RelPath))
		info.WriteString(fmt.Sprintf("    %s • %s\n", f.Language, sizeStr))
		
		if i < len(files)-1 {
			info.WriteString("\n")
		}
		
		totalSize += f.Size
	}
	
	// Footer with total context size
	totalSizeStr := fl.formatFileSize(totalSize)
	info.WriteString(fmt.Sprintf("\nTotal context: %s", totalSizeStr))
	
	return info.String()
}

// getFileTypeIcon returns an appropriate icon for the file type
func (fl *FileLoader) getFileTypeIcon(language string) string {
	iconMap := map[string]string{
		"go":         "🐹",
		"javascript": "🟨",
		"typescript": "🔷",
		"python":     "🐍",
		"rust":       "🦀",
		"java":       "☕",
		"c":          "⚡",
		"cpp":        "⚡",
		"html":       "🌐",
		"css":        "🎨",
		"json":       "📋",
		"yaml":       "📝",
		"markdown":   "📖",
		"sql":        "🗃️",
		"dockerfile": "🐳",
		"makefile":   "🔨",
		"bash":       "🖥️",
		"text":       "📄",
	}
	
	if icon, ok := iconMap[language]; ok {
		return icon
	}
	return "📄" // default file icon
}

// formatFileSize formats bytes in human-readable format
func (fl *FileLoader) formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d bytes", bytes)
	}
	
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"KB", "MB", "GB", "TB"}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}

// validatePattern validates file patterns and returns helpful error messages
func (fl *FileLoader) validatePattern(pattern string) error {
	if pattern == "" {
		return fmt.Errorf("empty pattern not allowed. Try: /load *.go or /load main.go")
	}

	// Check for common problematic patterns
	if pattern == "*" {
		return fmt.Errorf("pattern '*' matches all files, which may be too broad. Try: /load *.go for Go files, or /load *.* for all files")
	}

	// Check for patterns that might match too many files
	if strings.HasSuffix(pattern, "/**/*") && !strings.Contains(pattern, ".") {
		return fmt.Errorf("pattern '%s' may match too many files. Try: /load %s*.go or add file extension", pattern, pattern)
	}

	// Check for invalid ** patterns
	if strings.Contains(pattern, "**") {
		parts := strings.Split(pattern, "**")
		if len(parts) != 2 {
			return fmt.Errorf("invalid ** pattern: %s. Use format like 'src/**/*.go' for recursive search", pattern)
		}
	}

	// Check for absolute paths that might be problematic
	if filepath.IsAbs(pattern) && strings.Contains(pattern, "*") {
		return fmt.Errorf("absolute path patterns may be too broad: %s. Try relative patterns like '*.go' or 'src/**/*.go'", pattern)
	}

	// Warn about patterns that include common directories to avoid
	problematicDirs := []string{"node_modules", "target", "dist", "build", ".git", "vendor"}
	for _, dir := range problematicDirs {
		if strings.Contains(pattern, dir) {
			return fmt.Errorf("pattern '%s' includes '%s' directory, which may contain many files. Use more specific patterns", pattern, dir)
		}
	}

	return nil
}