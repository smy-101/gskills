package remove

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/smy-101/gskills/internal/add"
)

func promptForConfirmation(name string) (bool, error) {
	fmt.Printf("Are you sure you want to remove skill '%s'? [y/N]: ", name)

	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		if err == io.EOF {
			return false, nil
		}
		if err.Error() == "unexpected newline" {
			return false, nil
		}
		return false, fmt.Errorf("failed to read user input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes", nil
}

// removeSkillDirectory deletes the skill directory at the given path.
func removeSkillDirectory(storePath string) error {
	if err := os.RemoveAll(storePath); err != nil {
		return fmt.Errorf("failed to remove skill directory '%s': %w", storePath, err)
	}
	return nil
}

// RemoveSkillByName removes a skill by its name from the registry and deletes its directory.
// It prompts the user for confirmation before performing the removal.
// If the skill is linked to any projects, it will also remove all symlinks.
func RemoveSkillByName(name string) error {
	skill, err := add.FindSkillByName(name)
	if err != nil {
		return err
	}

	var confirmed bool

	if skill.LinkedProjects != nil && len(skill.LinkedProjects) > 0 {
		fmt.Printf("Warning: Skill '%s' is linked to %d project(s):\n", name, len(skill.LinkedProjects))
		for projectPath, linkInfo := range skill.LinkedProjects {
			fmt.Printf("  • %s (linked at %s)\n", projectPath, linkInfo.LinkedAt.Format("2006-01-02 15:04"))
		}

		confirmed, err = promptForConfirmationWithLinks(name, len(skill.LinkedProjects))
		if err != nil {
			return err
		}

		if confirmed {
			for _, linkInfo := range skill.LinkedProjects {
				if err := os.Remove(linkInfo.SymlinkPath); err != nil {
					fmt.Printf("Warning: Failed to remove symlink %s: %v\n", linkInfo.SymlinkPath, err)
				}
			}
		}
	} else {
		confirmed, err = promptForConfirmation(name)
		if err != nil {
			return err
		}
	}

	if !confirmed {
		return fmt.Errorf("operation cancelled")
	}

	if err := removeSkillDirectory(skill.StorePath); err != nil {
		return err
	}

	if err := add.RemoveSkill(skill.ID); err != nil {
		return fmt.Errorf("failed to remove skill from registry: %w", err)
	}

	return nil
}

// promptForConfirmationWithLinks asks the user to confirm before removing a skill with links.
// Returns true if the user confirms (y/yes), false otherwise.
func promptForConfirmationWithLinks(name string, linkCount int) (bool, error) {
	fmt.Printf("Remove skill '%s' and all %d symlink(s)? [y/N]: ", name, linkCount)

	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		if err == io.EOF {
			return false, nil
		}
		if err.Error() == "unexpected newline" {
			return false, nil
		}
		return false, fmt.Errorf("failed to read user input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes", nil
}
