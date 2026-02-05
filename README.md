# gskills

> A powerful CLI tool for managing and linking skill packages from GitHub repositories

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Test Coverage](https://img.shields.io/badge/coverage-57.6%25-brightgreen)](README.md#testing)

## 🚀 Features

- **Download & Install**: Fetch skill packages from GitHub with automatic dependency resolution
- **Version Management**: Track commits and update skills to latest versions
- **Smart Linking**: Symlink skills to multiple projects without duplication
- **Concurrent Downloads**: Optimized parallel file downloading with configurable limits
- **Atomic Operations**: Safe file operations with automatic rollback on errors
- **Rate Limit Handling**: Intelligent retry with exponential backoff for GitHub API limits
- **Registry Management**: Centralized skill metadata storage with JSON persistence
- **Binary Initialization**: First-time setup with automatic PATH configuration and shell detection

## 📦 Installation

### From Source

```bash
git clone https://github.com/smy-101/gskills.git
cd gskills
go build -o bin/gskills ./cmd/gskills

# Add to PATH (optional)
export PATH=$PATH:$(pwd)/bin
```

### Using Go Install

```bash
go install github.com/smy-101/gskills/cmd/gskills@latest
```

## 🎯 Quick Start

### 0. Initialize (First Time Setup)

Install gskills binary to `~/.gskills/bin` and add to PATH:

```bash
gskills init
```

The tool will:
- Detect your shell (bash/zsh/fish)
- Copy binary to `~/.gskills/bin`
- Add export statement to shell config
- Provide source command to apply changes

Example output:
```
✓ 检测到源路径: /usr/local/bin/gskills
✓ 复制二进制文件: /home/user/.gskills/bin/gskills
✓ 检测到 shell: zsh
✓ 更新配置文件: /home/user/.zshrc

gskills 已成功初始化！

请执行以下命令使配置生效:
  source ~/.zshrc

或重新打开终端窗口。
```

### 1. Add a Skill

Download a skill package from GitHub:

```bash
gskills add https://github.com/owner/repo/tree/branch/skills/prompt-engineer
```

The tool will:
- Validate that `SKILL.md` exists in the target directory
- Download all files recursively to `~/.gskills/skills/<skill-name>`
- Register the skill in the local registry
- Display download statistics

### 2. List Installed Skills

```bash
gskills list
```

Output:
```
Installed Skills:

Name:             prompt-engineer
Version:          main
Commit:           abc1234
Source:           https://github.com/owner/repo/tree/main/skills/prompt-engineer
Location:         /home/user/.gskills/skills/prompt-engineer
Last Updated:     2024-02-04 15:30:00
Linked Projects:  2
  • /home/user/project1
  • /home/user/project2
```

### 3. Link to a Project

Link a skill to your project using symbolic links:

```bash
# Link to current directory
gskills link prompt-engineer

# Link to specific project
gskills link prompt-engineer /path/to/project
```

This creates a symlink at `<project>/.opencode/skills/<skill-name>` pointing to `~/.gskills/skills/<skill-name>`.

### 4. Update Skills

```bash
# Update a specific skill
gskills update prompt-engineer

# Update all skills
gskills update
```

### 5. Remove a Skill

```bash
gskills remove prompt-engineer
```

## 📚 Command Reference

### `gskills add <url>`

Download and add a skill from a GitHub repository.

**URL Format**: `https://github.com/<owner>/<repo>/tree/<branch>/<path>`

**Example**:
```bash
gskills add https://github.com/example/skills/tree/main/skills/golang-pro
```

### `gskills list`

List all installed skills with detailed information.

**Flags**: None

### `gskills link <skill-name> [project-path]`

Link a skill to a project directory.

**Arguments**:
- `skill-name`: Name of the skill to link
- `project-path`: Project directory (defaults to current directory)

**Example**:
```bash
gskills link golang-pro ~/myproject
```

### `gskills unlink <skill-name> [project-path]`

Remove a skill link from a project.

**Example**:
```bash
gskills unlink golang-pro ~/myproject
```

### `gskills info <skill-name>`

Display detailed information about a skill including all linked projects.

**Example**:
```bash
gskills info golang-pro
```

### `gskills update [skill-name]`

Update installed skills to their latest commits.

**Examples**:
```bash
# Update specific skill
gskills update golang-pro

# Update all skills
gskills update
```

### `gskills remove <skill-name>`

Remove a skill from the local registry and filesystem.

**Warning**: This will delete the skill directory and all its links.

### `gskills init`

Initialize gskills by installing the binary to `~/.gskills/bin` and adding it to PATH.

**This command will:**
- Detect your current shell (bash, zsh, or fish)
- Copy the gskills binary to `~/.gskills/bin`
- Add the appropriate export statement to your shell configuration file
- Display the source command needed to apply the changes

**Example**:
```bash
gskills init
```

**Supported Shells**:
- `bash` - Updates `~/.bashrc` or `~/.bash_profile` (macOS)
- `zsh` - Updates `~/.zshrc`
- `fish` - Updates `~/.config/fish/config.fish`

### `gskills config`

Display current configuration settings.

**Example**:
```bash
gskills config
```

### `gskills tidy`

Clean up stale registry entries and orphaned symlinks.

**This command performs two cleanup operations:**
1. Removes registry entries pointing to non-existent symlinks
2. Deletes orphaned symlinks pointing to deleted skills

**Features**:
- Uses worker pool pattern with semaphore-controlled concurrency (max 10 workers)
- Context cancellation support for safe interruption
- Generates detailed cleanup report

**Example**:
```bash
gskills tidy
```

**Output**:
```
正在清理无用的技能链接...

清理完成！
• 移除了 3 个无效的注册表项
• 删除了 2 个孤立的符号链接

已检查 5 个技能，扫描了 4 个项目目录
```

### `gskills install`

Install a new project (for project initialization).

### `gskills migrate`

Migrate legacy link data to the new format.

### `gskills prune`

Clean up unused or orphaned projects.

## ⚙️ Configuration

Configuration is stored in `~/.gskills/config.json`:

```json
{
  "github_token": "your_github_token_here",
  "proxy": "http://proxy.example.com:8080"
}
```

### Settings

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `github_token` | string | No | GitHub personal access token for API authentication (increases rate limits) |
| `proxy` | string | No | HTTP proxy URL for downloading files |

### Setting Configuration

Edit the config file directly or use environment variables:

```bash
export GSKILLS_GITHUB_TOKEN="your_token"
export GSKILLS_PROXY="http://proxy:8080"
```

## 🏗️ Project Structure

```
gskills/
├── cmd/
│   └── gskills/           # Main application entry point
│       ├── main.go
│       └── main_test.go
├── pkg/
│   └── cmd/               # Cobra CLI command definitions
│       ├── root.go
│       ├── add.go
│       ├── link.go
│       ├── list.go
│       ├── remove.go
│       ├── update.go
│       ├── init.go        # Initialization command
│       ├── tidy.go        # Cleanup command
│       ├── install.go     # Project installation
│       └── ...
├── internal/
│   ├── add/               # Skill download and installation
│   ├── initializer/       # Binary installation and PATH setup
│   ├── link/              # Symlink management
│   ├── registry/          # Skill registry persistence
│   ├── remove/            # Skill removal logic
│   ├── tidy/              # Cleanup operations
│   ├── update/            # Update checking and application
│   ├── types/             # Shared type definitions
│   └── constants/         # Application constants
├── .gskills/              # Runtime directory (created in user home)
│   ├── config.json        # Configuration file
│   ├── skills.json        # Skills registry
│   └── skills/            # Downloaded skill packages
├── AGENTS.md              # Development guidelines for AI assistants
├── go.mod
└── README.md
```

## 🧪 Testing

### Run All Tests

```bash
go test ./...
```

### Run with Race Detection

```bash
go test -race ./...
```

### Test Coverage

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# View coverage in terminal
go tool cover -func=coverage.out
```

**Current Coverage**: 57.6%

### Run Specific Tests

```bash
# Test a specific package
go test ./internal/add

# Run a specific test
go test ./internal/add -run TestParseGitHubURL

# Run with verbose output
go test -v ./internal/add
```

### Benchmarks

```bash
go test -bench=. ./...
go test -bench=BenchmarkParseGitHubURL ./internal/add
```

## 🔧 Development

### Build from Source

```bash
# Build the binary
go build -o bin/gskills ./cmd/gskills

# Run directly
go run ./cmd/gskills [command]
```

### Code Quality

```bash
# Format code
gofmt -w .

# Check for issues
go vet ./...

# Run linters (requires golangci-lint)
golangci-lint run
```

### Architecture Highlights

- **Concurrent Downloads**: Uses worker pools with semaphore-controlled concurrency (maxWorkers=10)
- **Context Propagation**: Proper context cancellation throughout the call stack
- **Atomic File Operations**: All registry writes use atomic rename patterns
- **Error Wrapping**: Comprehensive error chains with `%w` verb
- **Custom Error Types**: Typed errors with `Is()` and `Unwrap()` support
- **Table-Driven Tests**: Comprehensive test coverage with subtests
- **HTTP Mocking**: Uses `httptest.Server` for integration testing
- **Worker Pool Pattern**: Semaphore-controlled concurrency for cleanup operations (max 10 workers)
- **Structured Logging**: Logger interface with Debug/Info/Warn/Error levels for observability
- **Shell Detection**: Auto-detects bash/zsh/fish with appropriate config file handling (.bashrc, .zshrc, config.fish)
- **Context Cancellation**: Proper cleanup support in concurrent tidy operations

## 🤝 Contributing

Contributions are welcome! Please follow these guidelines:

1. **Code Style**: Follow the conventions in [AGENTS.md](AGENTS.md)
2. **Testing**: Maintain test coverage above 60%
3. **Commits**: Use clear commit messages with conventional format
4. **Pull Requests**: Include tests for new features and update documentation

### Development Workflow

```bash
# Fork and clone the repository
git clone https://github.com/your-username/gskills.git
cd gskills

# Create a feature branch
git checkout -b feature/your-feature

# Make changes and test
go test ./...
go test -race ./...

# Commit and push
git add .
git commit -m "feat: add your feature"
git push origin feature/your-feature

# Open a pull request
```

## 📊 Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| [cobra](https://github.com/spf13/cobra) | v1.10.2 | CLI framework |
| [viper](https://github.com/spf13/viper) | v1.21.0 | Configuration management |
| [resty](https://github.com/go-resty/resty) | v2.17.1 | HTTP client with retry logic |
| [tablewriter](https://github.com/olekukonko/tablewriter) | v1.1.3 | Terminal table formatting |

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built with [Cobra](https://github.com/spf13/cobra) and [Viper](https://github.com/spf13/viper)
- Inspired by modern package managers and CLI tools
- Follows Go best practices for concurrent programming

## 📧 Support

- **Issues**: [GitHub Issues](https://github.com/smy-101/gskills/issues)
- **Documentation**: See [AGENTS.md](AGENTS.md) for development guidelines

## 🔗 Related Projects

- [opencode](https://github.com/anomalyco/opencode) - AI-powered coding assistant
- [golang-pro skill](https://github.com/smy-101/golang-pro) - Go development skill package

---

**Made with ❤️ using Go 1.21+**
