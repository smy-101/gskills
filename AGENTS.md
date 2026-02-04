# AGENTS.md

Instructions for agentic coding assistants working in this repository.

## Required Skill

**This project uses the `golang-pro` skill throughout development.** When working on this repository, load the `golang-pro` skill to get specialized guidance for Go applications requiring concurrent programming, microservices architecture, or high-performance systems. Invoke for goroutines, channels, Go generics, gRPC integration, etc.

## Essential Commands

```bash
# Build
go build -o bin/gskills ./cmd/gskills
go run ./cmd/gskills [command]

# Testing
go test ./...                          # All tests
go test -v ./...                       # Verbose
go test -race ./...                    # Race detection (REQUIRED before committing)
go test -cover ./...                   # Coverage
go test -coverprofile=coverage.out ./...

# Single Test (IMPORTANT)
go test ./internal/add -run TestParseGitHubURL
go test ./internal/add -v -run TestParseGitHubURL/valid_URL
go test ./internal/add -run TestDownload -count=1  # Disable cache

# Benchmarks
go test -bench=. ./...

# Linting
gofmt -w .                             # Format code (REQUIRED)
go vet ./...                            # Static analysis
golangci-lint run                       # Full lint
```

## Project Structure

- `cmd/gskills/` - Main entry point, config initialization
- `pkg/cmd/` - Cobra CLI commands (add, link, remove, list, update)
- `internal/` - Internal packages (add, link, registry, remove, update, types, constants)
- `.gskills/` - Runtime directory (in user home: `~/.gskills/`)

## Code Style Guidelines

### Imports (3 groups: stdlib, third-party, local)

```go
import (
    "context"
    "fmt"
    "os"

    "github.com/go-resty/resty/v2"
    "github.com/spf13/cobra"

    "github.com/smy-101/gskills/internal/types"
)
```

### Naming

- **Packages**: lowercase (`add`, `remove`, `types`)
- **Constants**: `CamelCase` (exported), `lowerCamelCase` (unexported)
- **Types/Interfaces**: `PascalCase` (exported), `lowerCamelCase` (unexported)
- **Functions**: `PascalCase` (exported), `lowerCamelCase` (unexported)
- **Acronyms**: Keep uppercase (`HTTPServer`, `GitHubAPI`)

### Error Handling

```go
// Wrap with %w for error chains
return fmt.Errorf("failed to parse URL: %w", err)

// Custom errors with Is() and Unwrap()
type DownloadError struct {
    Type    ErrorType
    Message string
    Err     error
}
func (e *DownloadError) Error() string { return fmt.Sprintf("%s: %v", e.Message, e.Err) }
func (e *DownloadError) Unwrap() error { return e.Err }
func (e *DownloadError) Is(target error) bool { ... }

// In Cobra commands, return errors from RunE, NEVER call os.Exit()
RunE: func(cmd *cobra.Command, args []string) error {
    return executeCommand(args)  // Let Cobra handle it
}
```

### Struct Tags (snake_case JSON)

```go
type SkillMetadata struct {
    ID             string                       `json:"id"`
    Name           string                       `json:"name"`
    SourceURL      string                       `json:"source_url"`
    StorePath      string                       `json:"store_path"`
    UpdatedAt      time.Time                    `json:"updated_at"`
    Version        string                       `json:"version,omitempty"`
}
```

### File Permissions

- **Directories**: `0755`
- **Files**: `0644`
- **Executables**: `0755`

### Testing

```go
// Table-driven tests with t.Run() for subtests
func TestParseGitHubURL(t *testing.T) {
    tests := []struct {
        name    string
        rawURL  string
        want    *GitHubRepoInfo
        wantErr bool
    }{
        {name: "valid URL", rawURL: "...", want: ..., wantErr: false},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseGitHubURL(tt.rawURL)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("= %v, want %v", got, tt.want)
            }
        })
    }
}

// Use t.TempDir() and httptest.Server
tmpDir := t.TempDir()
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"data": "value"}`))
}))
defer server.Close()
```

### Concurrency

```go
// Always use context.Context for blocking operations
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

// Use sync.Mutex for critical sections
var mu sync.Mutex
mu.Lock()
defer mu.Unlock()

// Use sync.Map for concurrent map access
var registryMutexes sync.Map
mu, _ := registryMutexes.LoadOrStore(key, &sync.Mutex{})

// Use buffered channels as semaphores
sem := make(chan struct{}, maxConcurrentDownloads)
sem <- struct{}{}        // Acquire
defer func() { <-sem }()  // Release

// ALWAYS run with -race flag before committing
go test -race ./...
```

### File I/O

```go
// Atomic writes: write to temp file then rename
tmpPath := registryPath + ".tmp"
if err := os.WriteFile(tmpPath, data, 0644); err != nil {
    return err
}
if err := os.Rename(tmpPath, registryPath); err != nil {
    os.Remove(tmpPath)
    return err
}

// JSON with indentation
data, err := json.MarshalIndent(obj, "", "  ")
```

### CLI Commands (Cobra)

```go
func init() {
    rootCmd.AddCommand(myCmd)
}

var myCmd = &cobra.Command{
    Use:   "command <arg> [optional]",
    Short: "Brief description",
    Args: func(cmd *cobra.Command, args []string) error {
        if len(args) < 1 {
            return errors.New("requires arg")
        }
        return nil
    },
    RunE: func(cmd *cobra.Command, args []string) error {
        return executeCommand(args)
    },
}

func executeCommand(args []string) error {
    ctx := context.Background()
    // ... work with ctx
}
```

## Configuration

Config stored in `~/.gskills/config.json` via Viper:
- `github_token`: GitHub API token (optional, increases rate limits)
- `proxy`: HTTP proxy URL (optional)

**IMPORTANT**: Always validate config exists, create with defaults if missing.

## Quality Gates

Before committing:
1. ✅ All tests pass: `go test ./...`
2. ✅ No race conditions: `go test -race ./...`
3. ✅ Code formatted: `gofmt -w .`
4. ✅ Vet passes: `go vet ./...`
5. ✅ Test coverage not decreased

Target coverage: **60%+** (current: 61.5%)
