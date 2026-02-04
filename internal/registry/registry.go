package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/smy-101/gskills/internal/types"
)

const skillsRegistryFile = "skills.json"

var (
	registryMutexes sync.Map
)

func getRegistryPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	return filepath.Join(homeDir, ".gskills", skillsRegistryFile), nil
}

func LoadRegistry() ([]types.SkillMetadata, error) {
	registryPath, err := getRegistryPath()
	if err != nil {
		return nil, err
	}

	return loadRegistryWithPath(registryPath)
}

func loadRegistryWithPath(registryPath string) ([]types.SkillMetadata, error) {
	data, err := os.ReadFile(registryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []types.SkillMetadata{}, nil
		}
		return nil, fmt.Errorf("failed to read registry file: %w", err)
	}

	var skills []types.SkillMetadata
	if err := json.Unmarshal(data, &skills); err != nil {
		return nil, fmt.Errorf("failed to unmarshal registry: %w", err)
	}

	return skills, nil
}

func SaveRegistry(skills []types.SkillMetadata) error {
	registryPath, err := getRegistryPath()
	if err != nil {
		return err
	}

	return SaveRegistryWithPath(registryPath, skills)
}

func SaveRegistryWithPath(registryPath string, skills []types.SkillMetadata) error {
	registryDir := filepath.Dir(registryPath)
	if err := os.MkdirAll(registryDir, 0755); err != nil {
		return fmt.Errorf("failed to create registry directory: %w", err)
	}

	data, err := json.MarshalIndent(skills, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	tmpPath := registryPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary registry file: %w", err)
	}

	if err := os.Rename(tmpPath, registryPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename registry file: %w", err)
	}

	return nil
}

func validateSkillMetadata(skill *types.SkillMetadata) error {
	if skill == nil {
		return fmt.Errorf("skill metadata cannot be nil")
	}
	if skill.ID == "" {
		return fmt.Errorf("skill ID cannot be empty")
	}
	if skill.Name == "" {
		return fmt.Errorf("skill name cannot be empty")
	}
	if skill.Version == "" {
		return fmt.Errorf("skill version cannot be empty")
	}
	if skill.SourceURL == "" {
		return fmt.Errorf("skill source URL cannot be empty")
	}
	if skill.StorePath == "" {
		return fmt.Errorf("skill store path cannot be empty")
	}
	return nil
}

func AddOrUpdateSkill(skill *types.SkillMetadata) error {
	if err := validateSkillMetadata(skill); err != nil {
		return err
	}

	registryPath, err := getRegistryPath()
	if err != nil {
		return err
	}

	return addOrUpdateSkillWithPath(registryPath, skill)
}

func addOrUpdateSkillWithPath(registryPath string, skill *types.SkillMetadata) error {
	if err := validateSkillMetadata(skill); err != nil {
		return err
	}

	muIface, _ := registryMutexes.LoadOrStore(registryPath, &sync.Mutex{})
	mu, ok := muIface.(*sync.Mutex)
	if !ok {
		return fmt.Errorf("failed to get mutex for registry path")
	}
	mu.Lock()
	defer mu.Unlock()

	skills, err := loadRegistryWithPath(registryPath)
	if err != nil {
		return err
	}

	indexMap := make(map[string]int, len(skills))
	for i, s := range skills {
		indexMap[s.ID] = i
	}

	if idx, exists := indexMap[skill.ID]; exists {
		skills[idx] = *skill
	} else {
		skills = append(skills, *skill)
	}

	return SaveRegistryWithPath(registryPath, skills)
}

func RemoveSkill(skillID string) error {
	if skillID == "" {
		return fmt.Errorf("skill ID cannot be empty")
	}

	registryPath, err := getRegistryPath()
	if err != nil {
		return err
	}

	return removeSkillWithPath(registryPath, skillID)
}

func removeSkillWithPath(registryPath string, skillID string) error {
	if skillID == "" {
		return fmt.Errorf("skill ID cannot be empty")
	}

	muIface, _ := registryMutexes.LoadOrStore(registryPath, &sync.Mutex{})
	mu, ok := muIface.(*sync.Mutex)
	if !ok {
		return fmt.Errorf("failed to get mutex for registry path")
	}
	mu.Lock()
	defer mu.Unlock()

	skills, err := loadRegistryWithPath(registryPath)
	if err != nil {
		return err
	}

	newSkills := make([]types.SkillMetadata, 0, len(skills))
	for _, s := range skills {
		if s.ID != skillID {
			newSkills = append(newSkills, s)
		}
	}

	return SaveRegistryWithPath(registryPath, newSkills)
}

func FindSkillByName(name string) (*types.SkillMetadata, error) {
	if name == "" {
		return nil, fmt.Errorf("skill name cannot be empty")
	}

	skills, err := LoadRegistry()
	if err != nil {
		return nil, fmt.Errorf("failed to load registry: %w", err)
	}

	for i := range skills {
		if skills[i].Name == name {
			return &skills[i], nil
		}
	}

	return nil, fmt.Errorf("skill '%s' not found in registry", name)
}

func UpdateSkill(skill *types.SkillMetadata) error {
	if skill == nil {
		return fmt.Errorf("skill cannot be nil")
	}
	if skill.ID == "" {
		return fmt.Errorf("skill ID cannot be empty")
	}

	registryPath, err := getRegistryPath()
	if err != nil {
		return err
	}

	muIface, _ := registryMutexes.LoadOrStore(registryPath, &sync.Mutex{})
	mu, ok := muIface.(*sync.Mutex)
	if !ok {
		return fmt.Errorf("failed to get mutex for registry path")
	}
	mu.Lock()
	defer mu.Unlock()

	skills, err := loadRegistryWithPath(registryPath)
	if err != nil {
		return err
	}

	found := false
	for i := range skills {
		if skills[i].ID == skill.ID {
			skills[i] = *skill
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("skill with ID '%s' not found", skill.ID)
	}

	return SaveRegistryWithPath(registryPath, skills)
}
