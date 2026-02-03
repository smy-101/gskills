# AGENTS.md

Instructions for agentic coding assistants working in this repository.

## Required Skill

**This project uses the `golang-pro` skill throughout development.** When working on this repository, load the `golang-pro` skill to get specialized guidance for Go applications requiring concurrent programming, microservices architecture, or high-performance systems. Invoke for goroutines, channels, Go generics, gRPC integration, etc.

## Build Commands

```bash
# Build and run
go build -o bin/gskills ./cmd/gskills
go run ./cmd/gskills [command]

# Tests
go test ./...
go test -v ./...
go test ./internal/add
go test -race ./...
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run single test
go test ./internal/add -run TestParseGitHubURL
go test ./internal/add -v -run TestParseGitHubURL/valid_URL_with_branch_and_path

# Benchmarks
go test -bench=. ./...
go test -bench=BenchmarkParseGitHubURL ./internal/add
```

## Project Structure

- `cmd/gskills/` - Main entry point
- `pkg/cmd/` - Cobra CLI commands (add, remove, list, etc.)
- `internal/` - Internal packages (add, remove, types)
- `internal/types/` - Shared type definitions
- `.gskills/` - Runtime configuration directory (in user home)

## Code Style Guidelines

### Imports

```go
// Group 1: Standard library, Group 2: Third-party (alphabetical), Group 3: Local
import (
    "context"
    "fmt"
    "os"
    "time"
    "github.com/go-resty/resty/v2"
    "github.com/spf13/cobra"
    "github.com/smy-101/gskills/internal/types"
)
```

### Naming Conventions

- **Package names**: Single lowercase words (`add`, `remove`, `types`)
- **Constants**: `CamelCase` exported, `lowerCamelCase` unexported
- **Types/Interfaces**: `PascalCase` exported, `lowerCamelCase` unexported
- **Functions**: `PascalCase` exported, `lowerCamelCase` unexported
- **Variables**: `PascalCase` exported, `lowerCamelCase` unexported

### Error Handling

```go
// Use fmt.Errorf with %w for wrapping errors
return fmt.Errorf("failed to parse URL: %w", err)

// Create custom error types with Is() and Unwrap()
type DownloadError struct {
    Type    ErrorType
    Message string
    Err     error
}
func (e *DownloadError) Error() string { return fmt.Sprintf("%s: %v", e.Message, e.Err) }
func (e *DownloadError) Unwrap() error { return e.Err }
func (e *DownloadError) Is(target error) bool { ... }
```

### Struct Tags

```go
// Use JSON tags with snake_case
type SkillMetadata struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    SourceURL   string    `json:"source_url"`
    UpdatedAt   time.Time `json:"updated_at"`
    Version     string    `json:"version,omitempty"`
}
```

### File Permissions

- Directories: `0755` (rwxr-xr-x)
- Files: `0644` (rw-r--r--)

### Testing

```go
// Use table-driven tests with t.Run() for subtests
func TestParseGitHubURL(t *testing.T) {
    tests := []struct {
        name    string
        rawURL  string
        want    *GitHubRepoInfo
        wantErr bool
    }{...}
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := parseGitHubURL(tt.rawURL)
            // assertions
        })
    }
}
// Use t.TempDir() and httptest for HTTP mocking
tmpDir := t.TempDir()
server := httptest.NewServer(http.HandlerFunc(...))
defer server.Close()
```

### Concurrency

```go
// Use sync.Mutex for locking
var mu sync.Mutex
mu.Lock()
defer mu.Unlock()

// Use sync.Map for concurrent map access
var registryMutexes sync.Map
mu, _ := registryMutexes.LoadOrStore(key, &sync.Mutex{})

// Use context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
```

### File I/O Patterns

```go
// Atomic writes: write to temp file then rename
tmpPath := registryPath + ".tmp"
os.WriteFile(tmpPath, data, 0644)
os.Rename(tmpPath, registryPath)

// JSON marshaling with indentation
data, err := json.MarshalIndent(obj, "", "  ")
```

### Logging

```go
type Logger interface {
    Debug(msg string, fields ...interface{})
    Info(msg string, fields ...interface{})
    Warn(msg string, fields ...interface{})
    Error(msg string, err error, fields ...interface{})
}
```

## CLI Commands (Cobra)

Commands are defined in `pkg/cmd/`. Each command file uses `init()` to add itself to `rootCmd`, defines a `&cobra.Command` with `Use`, `Short`, and `Args` validation, and uses `RunE` for error handling.

## Configuration

Configuration is stored in `~/.gskills/config.json` with Viper:
- `github_token`: GitHub API token (optional)
- `proxy`: HTTP proxy URL (optional)
