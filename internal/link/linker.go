// Package link provides functionality to create symlinks from gskills-managed
// skill directories to project .opencode/skills/ directories.
package link

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/smy-101/gskills/internal/constants"
	"github.com/smy-101/gskills/internal/registry"
	"github.com/smy-101/gskills/internal/types"
)

// Linker handles creating and managing symlinks between gskills-managed
// skill directories and project directories.
type Linker struct {
	logger Logger
}

// NewLinker creates a new Linker instance with a NoOpLogger.
func NewLinker() *Linker {
	return &Linker{
		logger: NoOpLogger{},
	}
}

// LinkSkill creates a symlink from the gskills-managed skill directory to
// the target project's .opencode/skills/<skill_name> directory.
// It updates the skills registry with linked skill metadata.
// Returns an error if the skill doesn't exist, the project path is invalid,
// or a symlink already exists at the target location.
func (l *Linker) LinkSkill(ctx context.Context, skillName, projectPath string) error {
	if skillName == "" {
		return &LinkError{
			Type:    ErrorTypeInvalidPath,
			Message: "skill name cannot be empty",
		}
	}
	if projectPath == "" {
		return &LinkError{
			Type:    ErrorTypeInvalidPath,
			Message: "project path cannot be empty",
		}
	}

	select {
	case <-ctx.Done():
		return &LinkError{
			Type:    ErrorTypeFilesystem,
			Message: "operation cancelled",
			Err:     ctx.Err(),
		}
	default:
	}

	skillPath, err := l.getSkillPath(skillName)
	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return &LinkError{
			Type:    ErrorTypeFilesystem,
			Message: "operation cancelled",
			Err:     ctx.Err(),
		}
	default:
	}

	absProjectPath, err := filepath.Abs(projectPath)
	if err != nil {
		return &LinkError{
			Type:    ErrorTypeInvalidPath,
			Message: "failed to get absolute project path",
			Err:     err,
		}
	}

	select {
	case <-ctx.Done():
		return &LinkError{
			Type:    ErrorTypeFilesystem,
			Message: "operation cancelled",
			Err:     ctx.Err(),
		}
	default:
	}

	targetDir := filepath.Join(absProjectPath, constants.OpencodeSkillsDir)
	targetPath := filepath.Join(targetDir, skillName)

	exists, err := l.checkPathExists(targetPath)
	if err != nil {
		return &LinkError{
			Type:    ErrorTypeFilesystem,
			Message: "failed to check target path existence",
			Err:     err,
		}
	}

	if exists {
		return &LinkError{
			Type:    ErrorTypeSymlinkExists,
			Message: fmt.Sprintf("skill '%s' is already linked in project '%s'", skillName, absProjectPath),
		}
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return &LinkError{
			Type:    ErrorTypeFilesystem,
			Message: "failed to create target directory",
			Err:     err,
		}
	}

	select {
	case <-ctx.Done():
		return &LinkError{
			Type:    ErrorTypeFilesystem,
			Message: "operation cancelled",
			Err:     ctx.Err(),
		}
	default:
	}

	if err := os.Symlink(skillPath, targetPath); err != nil {
		return &LinkError{
			Type:    ErrorTypeFilesystem,
			Message: "failed to create symlink",
			Err:     err,
		}
	}

	existingSkill, err := registry.FindSkillByName(skillName)
	if err != nil {
		l.logger.Error("Failed to find skill in registry", err, "skill", skillName)
		if removeErr := os.Remove(targetPath); removeErr != nil {
			l.logger.Error("Failed to clean up symlink after error", removeErr, "path", targetPath)
		}
		return fmt.Errorf("failed to find skill '%s' in registry: %w", skillName, err)
	}

	select {
	case <-ctx.Done():
		if removeErr := os.Remove(targetPath); removeErr != nil {
			l.logger.Error("Failed to clean up symlink after cancellation", removeErr, "path", targetPath)
		}
		return &LinkError{
			Type:    ErrorTypeFilesystem,
			Message: "operation cancelled",
			Err:     ctx.Err(),
		}
	default:
	}

	if existingSkill.LinkedProjects == nil {
		existingSkill.LinkedProjects = make(map[string]types.LinkedProjectInfo)
	}

	existingSkill.LinkedProjects[absProjectPath] = types.LinkedProjectInfo{
		SymlinkPath: targetPath,
		LinkedAt:    time.Now(),
	}

	existingSkill.UpdatedAt = time.Now()

	if err := registry.UpdateSkill(existingSkill); err != nil {
		l.logger.Error("Failed to update skills registry", err, "skill", skillName)
		if removeErr := os.Remove(targetPath); removeErr != nil {
			l.logger.Error("Failed to clean up symlink after error", removeErr, "path", targetPath)
		}
		return fmt.Errorf("failed to update skills registry: %w", err)
	}

	l.logger.Info("Successfully linked skill", "skill", skillName, "path", targetPath)
	return nil
}

// getSkillPath retrieves the absolute path to a gskills-managed skill directory.
// Returns an error if the skill doesn't exist in ~/.gskills/skills/.
func (l *Linker) getSkillPath(skillName string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", &LinkError{
			Type:    ErrorTypeFilesystem,
			Message: "failed to get home directory",
			Err:     err,
		}
	}

	skillsDir := filepath.Join(homeDir, ".gskills", "skills", skillName)
	exists, err := l.checkPathExists(skillsDir)
	if err != nil {
		return "", &LinkError{
			Type:    ErrorTypeFilesystem,
			Message: "failed to check skill directory",
			Err:     err,
		}
	}

	if !exists {
		return "", &LinkError{
			Type:    ErrorTypeSkillNotFound,
			Message: fmt.Sprintf("skill '%s' not found in ~/.gskills/skills/", skillName),
		}
	}

	absPath, err := filepath.Abs(skillsDir)
	if err != nil {
		return "", &LinkError{
			Type:    ErrorTypeFilesystem,
			Message: "failed to get absolute skill path",
			Err:     err,
		}
	}

	return absPath, nil
}

// validateProjectPath validates that the project path exists and is a directory.
// Returns an error if the path doesn't exist or is not a directory.
func (l *Linker) validateProjectPath(projectPath string) error {
	info, err := os.Stat(projectPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &LinkError{
				Type:    ErrorTypeInvalidPath,
				Message: fmt.Sprintf("project path '%s' does not exist", projectPath),
			}
		}
		return &LinkError{
			Type:    ErrorTypeInvalidPath,
			Message: "failed to validate project path",
			Err:     err,
		}
	}

	if !info.IsDir() {
		return &LinkError{
			Type:    ErrorTypeInvalidPath,
			Message: fmt.Sprintf("project path '%s' is not a directory", projectPath),
		}
	}

	return nil
}

// checkPathExists checks if a path exists using os.Lstat to handle symlinks.
// Returns true if the path exists, false otherwise.
// Returns an error if the stat operation fails for reasons other than non-existence.
func (l *Linker) checkPathExists(path string) (bool, error) {
	_, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// UnlinkSkill removes a symlink from a project and updates the registry.
// Returns an error if the skill is not found, not linked to the project,
// or if the symlink removal fails.
func (l *Linker) UnlinkSkill(skillName, projectPath string) error {
	if skillName == "" {
		return &LinkError{
			Type:    ErrorTypeInvalidPath,
			Message: "skill name cannot be empty",
		}
	}
	if projectPath == "" {
		return &LinkError{
			Type:    ErrorTypeInvalidPath,
			Message: "project path cannot be empty",
		}
	}

	skill, err := registry.FindSkillByName(skillName)
	if err != nil {
		return &LinkError{
			Type:    ErrorTypeSkillNotFound,
			Message: fmt.Sprintf("skill '%s' not found in registry", skillName),
			Err:     err,
		}
	}

	absProjectPath, err := filepath.Abs(projectPath)
	if err != nil {
		return &LinkError{
			Type:    ErrorTypeInvalidPath,
			Message: "failed to get absolute project path",
			Err:     err,
		}
	}

	if skill.LinkedProjects == nil {
		return &LinkError{
			Type:    ErrorTypeInvalidPath,
			Message: fmt.Sprintf("skill '%s' is not linked to any projects", skillName),
		}
	}

	linkInfo, linked := skill.LinkedProjects[absProjectPath]
	if !linked {
		return &LinkError{
			Type:    ErrorTypeInvalidPath,
			Message: fmt.Sprintf("skill '%s' is not linked to project '%s'", skillName, absProjectPath),
		}
	}

	if err := os.Remove(linkInfo.SymlinkPath); err != nil {
		return &LinkError{
			Type:    ErrorTypeFilesystem,
			Message: "failed to remove symlink",
			Err:     err,
		}
	}

	delete(skill.LinkedProjects, absProjectPath)

	if len(skill.LinkedProjects) == 0 {
		skill.LinkedProjects = nil
	}

	skill.UpdatedAt = time.Now()

	if err := registry.UpdateSkill(skill); err != nil {
		return fmt.Errorf("failed to update skills registry: %w", err)
	}

	return nil
}
