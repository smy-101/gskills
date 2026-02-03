package remove

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/smy-101/gskills/internal/add"
	"github.com/smy-101/gskills/internal/types"
)

// findSkillByName searches for a skill by its name in the registry.
// Returns the skill metadata if found, or an error if not found.
func findSkillByName(name string) (*types.SkillMetadata, error) {
	skills, err := add.LoadRegistry()
	if err != nil {
		return nil, fmt.Errorf("failed to load registry: %w", err)
	}

	for _, skill := range skills {
		if skill.Name == name {
			return &skill, nil
		}
	}

	return nil, fmt.Errorf("skill '%s' not found", name)
}

// promptForConfirmation asks the user to confirm before removing a skill.
// Returns true if the user confirms (y/yes), false otherwise.
func promptForConfirmation(name string) (bool, error) {
	fmt.Printf("Are you sure you want to remove skill '%s'? [y/N]: ", name)

	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		if err == io.EOF {
			return false, nil
		}
		return false, nil
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
func RemoveSkillByName(name string) error {
	skill, err := findSkillByName(name)
	if err != nil {
		return err
	}

	confirmed, err := promptForConfirmation(name)
	if err != nil {
		return err
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
