package tidy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/smy-101/gskills/internal/constants"
	"github.com/smy-101/gskills/internal/registry"
	"github.com/smy-101/gskills/internal/types"
)

const (
	// maxWorkers limits the number of concurrent goroutines during cleanup operations.
	maxWorkers = 10
)

// CleanupReport summarizes the results of a tidy operation.
// It provides statistics about the cleanup process including the number of
// stale registry entries removed and orphaned symlinks deleted.
type CleanupReport struct {
	// StaleRegistryEntries is the count of invalid project links removed from the registry.
	StaleRegistryEntries int
	// OrphanedSymlinks is the count of symlinks removed from project directories.
	OrphanedSymlinks int
	// SkillsChecked is the total number of skills processed.
	SkillsChecked int
	// ProjectsScanned is the number of unique project directories examined.
	ProjectsScanned int
}

// Field represents a key-value pair for structured logging.
type Field struct {
	Key   string
	Value interface{}
}

// Logger defines the structured logging interface used by Tidier.
// It provides methods for logging at different severity levels with structured fields.
type Logger interface {
	// Debug logs a debug message with optional structured fields.
	Debug(msg string, fields ...Field)
	// Info logs an informational message with optional structured fields.
	Info(msg string, fields ...Field)
	// Warn logs a warning message with optional structured fields.
	Warn(msg string, fields ...Field)
	// Error logs an error message with the error object and optional structured fields.
	Error(msg string, err error, fields ...Field)
}

// NoOpLogger is a logger that discards all log messages.
// It is used as the default logger when no custom logger is provided.
type NoOpLogger struct{}

func (NoOpLogger) Debug(msg string, fields ...Field) {}

func (NoOpLogger) Info(msg string, fields ...Field) {}

func (NoOpLogger) Warn(msg string, fields ...Field) {}

func (NoOpLogger) Error(msg string, err error, fields ...Field) {}

// Tidier handles cleanup of stale registry entries and orphaned symlinks.
// It performs two main operations:
// 1. Removes registry entries for symlinks that no longer exist on disk
// 2. Deletes orphaned symlinks that point to non-existent skills
type Tidier struct {
	logger Logger
}

// NewTidier creates a new Tidier instance with a no-op logger.
func NewTidier() *Tidier {
	return &Tidier{
		logger: NoOpLogger{},
	}
}

// NewTidierWithLogger creates a new Tidier with a custom logger for observability.
func NewTidierWithLogger(logger Logger) *Tidier {
	return &Tidier{
		logger: logger,
	}
}

// Tidy performs cleanup of stale registry entries and orphaned symlinks.
// It uses a worker pool pattern to limit concurrent goroutines to maxWorkers.
// The operation can be cancelled via the provided context.
//
// Returns a CleanupReport with statistics about what was cleaned up.
// If the context is cancelled, a partial report may be returned with an error.
func (t *Tidier) Tidy(ctx context.Context) (*CleanupReport, error) {
	report := &CleanupReport{}
	var wg sync.WaitGroup
	var mu sync.Mutex

	skills, err := registry.LoadRegistry()
	if err != nil {
		return nil, &TidyError{
			Type:    ErrorTypeRegistry,
			Message: "failed to load skills registry",
			Err:     err,
		}
	}

	report.SkillsChecked = len(skills)

	uniqueProjectPaths := make(map[string]struct{})
	for _, skill := range skills {
		for projectPath := range skill.LinkedProjects {
			uniqueProjectPaths[projectPath] = struct{}{}
		}
	}

	report.ProjectsScanned = len(uniqueProjectPaths)

	type pendingUpdate struct {
		skillID       string
		staleProjects []string
		skill         types.SkillMetadata
	}

	updateChan := make(chan pendingUpdate, len(skills))
	sem := make(chan struct{}, maxWorkers)

	for _, skill := range skills {
		select {
		case <-ctx.Done():
			return report, &TidyError{
				Type:    ErrorTypeRegistry,
				Message: "operation cancelled",
				Err:     ctx.Err(),
			}
		default:
		}

		if len(skill.LinkedProjects) == 0 {
			continue
		}

		wg.Add(1)
		sem <- struct{}{}
		go func(s types.SkillMetadata) {
			defer func() { <-sem; wg.Done() }()

			staleEntries := t.findStaleLinks(s)

			if len(staleEntries) > 0 {
				mu.Lock()
				report.StaleRegistryEntries += len(staleEntries)
				mu.Unlock()

				updateChan <- pendingUpdate{
					skillID:       s.ID,
					staleProjects: staleEntries,
					skill:         s,
				}
			}
		}(skill)
	}

	go func() {
		wg.Wait()
		close(updateChan)
	}()

	pendingUpdates := make([]pendingUpdate, 0)
	for update := range updateChan {
		pendingUpdates = append(pendingUpdates, update)
	}

	for _, update := range pendingUpdates {
		for _, projectPath := range update.staleProjects {
			delete(update.skill.LinkedProjects, projectPath)
		}

		if len(update.skill.LinkedProjects) == 0 {
			update.skill.LinkedProjects = nil
		}

		if err := registry.UpdateSkill(&update.skill); err != nil {
			t.logger.Error("Failed to remove stale links from registry", err,
				Field{Key: "skill", Value: update.skill.Name})
		} else {
			t.logger.Info("Removed stale links",
				Field{Key: "skill", Value: update.skill.Name},
				Field{Key: "count", Value: len(update.staleProjects)})
		}
	}

	select {
	case <-ctx.Done():
		return report, &TidyError{
			Type:    ErrorTypeRegistry,
			Message: "operation cancelled",
			Err:     ctx.Err(),
		}
	default:
	}

	orphanedSymlinks, err := t.findAndRemoveOrphanedSymlinks(ctx, uniqueProjectPaths)
	if err != nil {
		return report, &TidyError{
			Type:    ErrorTypeFilesystem,
			Message: "failed to remove orphaned symlinks",
			Err:     err,
		}
	}

	report.OrphanedSymlinks = orphanedSymlinks

	return report, nil
}

// findStaleLinks identifies project links where the symlink no longer exists.
// It checks each linked project and returns a list of project paths where
// the recorded symlink path does not exist on disk.
func (t *Tidier) findStaleLinks(skill types.SkillMetadata) []string {
	var staleEntries []string

	for projectPath, linkInfo := range skill.LinkedProjects {
		exists, err := t.checkSymlinkExists(linkInfo.SymlinkPath)
		if err != nil {
			t.logger.Warn("Failed to check symlink",
				Field{Key: "path", Value: linkInfo.SymlinkPath},
				Field{Key: "error", Value: err})
			continue
		}

		if !exists {
			staleEntries = append(staleEntries, projectPath)
			t.logger.Debug("Found stale link",
				Field{Key: "skill", Value: skill.Name},
				Field{Key: "project", Value: projectPath})
		}
	}

	return staleEntries
}

// checkSymlinkExists checks if a symlink exists at the given path.
func (t *Tidier) checkSymlinkExists(symlinkPath string) (bool, error) {
	_, err := os.Lstat(symlinkPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

// findAndRemoveOrphanedSymlinks scans project directories for symlinks pointing
// to non-existent skills and removes them.
func (t *Tidier) findAndRemoveOrphanedSymlinks(ctx context.Context, projectPaths map[string]struct{}) (int, error) {
	skills, err := registry.LoadRegistry()
	if err != nil {
		return 0, fmt.Errorf("failed to load registry: %w", err)
	}

	validSkillStorePaths := make(map[string]string)
	for _, skill := range skills {
		validSkillStorePaths[skill.StorePath] = skill.Name
	}

	var orphanedCount int
	var mu sync.Mutex
	var wg sync.WaitGroup

	sem := make(chan struct{}, maxWorkers)

	for projectPath := range projectPaths {
		select {
		case <-ctx.Done():
			return orphanedCount, ctx.Err()
		default:
		}

		wg.Add(1)
		sem <- struct{}{}
		go func(ppath string) {
			defer func() { <-sem; wg.Done() }()

			skillsDirPath := filepath.Join(ppath, constants.OpencodeSkillsDir)
			entries, err := os.ReadDir(skillsDirPath)
			if err != nil {
				if os.IsNotExist(err) {
					return
				}
				t.logger.Warn("Failed to read project skills directory",
					Field{Key: "path", Value: skillsDirPath},
					Field{Key: "error", Value: err})
				return
			}

			localOrphaned := 0

			for _, entry := range entries {
				symlinkPath := filepath.Join(skillsDirPath, entry.Name())

				info, err := os.Lstat(symlinkPath)
				if err != nil {
					continue
				}

				if info.Mode()&os.ModeSymlink == 0 {
					continue
				}

				target, err := os.Readlink(symlinkPath)
				if err != nil {
					t.logger.Warn("Failed to read symlink",
						Field{Key: "path", Value: symlinkPath},
						Field{Key: "error", Value: err})
					continue
				}

				target = filepath.Clean(target)

				isValid := false

				var absTarget string
				if filepath.IsAbs(target) {
					absTarget = target
				} else {
					absTarget, err = filepath.Abs(filepath.Join(filepath.Dir(symlinkPath), target))
					if err != nil {
						t.logger.Warn("Failed to resolve absolute path",
							Field{Key: "path", Value: symlinkPath},
							Field{Key: "target", Value: target},
							Field{Key: "error", Value: err})
						continue
					}
				}

				if skillName, ok := validSkillStorePaths[absTarget]; ok {
					if skillName == entry.Name() {
						isValid = true
					}
				}

				if !isValid {
					if err := os.Remove(symlinkPath); err != nil {
						t.logger.Error("Failed to remove orphaned symlink", err,
							Field{Key: "path", Value: symlinkPath})
					} else {
						t.logger.Info("Removed orphaned symlink",
							Field{Key: "path", Value: symlinkPath})
						localOrphaned++
					}
				}
			}

			mu.Lock()
			orphanedCount += localOrphaned
			mu.Unlock()
		}(projectPath)
	}

	wg.Wait()

	return orphanedCount, nil
}
