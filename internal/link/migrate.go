package link

import (
	"fmt"
	"strings"

	"github.com/smy-101/gskills/internal/add"
	"github.com/smy-101/gskills/internal/types"
)

// MigrateLegacyLinks converts old-style linked skill entries (with "linked-" prefix)
// to the new format where links are tracked in the original skill's LinkedProjects field.
func MigrateLegacyLinks() error {
	skills, err := add.LoadRegistry()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	var toUpdate []types.SkillMetadata
	var toRemove []string

	for _, skill := range skills {
		if strings.HasPrefix(skill.ID, "linked-") {
			parts := strings.SplitN(strings.TrimPrefix(skill.ID, "linked-"), "@", 2)
			if len(parts) != 2 {
				fmt.Printf("Warning: Invalid legacy link ID format: %s\n", skill.ID)
				continue
			}

			originalSkillName := parts[0]
			projectPath := strings.TrimPrefix(skill.SourceURL, "linked:")

			var originalSkill *types.SkillMetadata
			for i := range skills {
				if skills[i].Name == originalSkillName && !strings.HasPrefix(skills[i].ID, "linked-") {
					originalSkill = &skills[i]
					break
				}
			}

			if originalSkill == nil {
				fmt.Printf("Warning: Could not find original skill '%s'\n", originalSkillName)
				continue
			}

			if originalSkill.LinkedProjects == nil {
				originalSkill.LinkedProjects = make(map[string]types.LinkedProjectInfo)
			}

			originalSkill.LinkedProjects[projectPath] = types.LinkedProjectInfo{
				SymlinkPath: skill.StorePath,
				LinkedAt:    skill.UpdatedAt,
			}

			toUpdate = append(toUpdate, *originalSkill)
			toRemove = append(toRemove, skill.ID)
		}
	}

	for _, skill := range toUpdate {
		if err := add.UpdateSkill(&skill); err != nil {
			return fmt.Errorf("failed to update skill '%s': %w", skill.Name, err)
		}
	}

	for _, id := range toRemove {
		if err := add.RemoveSkill(id); err != nil {
			fmt.Printf("Warning: Failed to remove legacy entry '%s': %v\n", id, err)
		}
	}

	if len(toUpdate) > 0 {
		fmt.Printf("Migrated %d legacy link(s)\n", len(toUpdate))
	} else {
		fmt.Println("No legacy links found to migrate")
	}

	return nil
}

// CheckLegacyLinks checks if there are any legacy linked skills in the registry
// and returns the count. Does not perform migration.
func CheckLegacyLinks() (int, error) {
	skills, err := add.LoadRegistry()
	if err != nil {
		return 0, fmt.Errorf("failed to load registry: %w", err)
	}

	count := 0
	for _, skill := range skills {
		if strings.HasPrefix(skill.ID, "linked-") {
			count++
		}
	}

	return count, nil
}
