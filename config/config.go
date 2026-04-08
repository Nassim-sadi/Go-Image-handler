package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"

	"imagehandler/models"
)

type Config struct {
	Projects     map[string]*models.Project
	Presets      map[string]*models.Preset
	LastProject  string `json:"lastProject"`
}

var AppConfig *Config

func init() {
	AppConfig = &Config{
		Projects: make(map[string]*models.Project),
		Presets:  make(map[string]*models.Preset),
	}
}

func GetConfigPath() string {
	var configDir string
	if runtime.GOOS == "windows" {
		configDir = filepath.Join(os.Getenv("APPDATA"), "ImageHandler")
	} else {
		configDir = filepath.Join(os.Getenv("HOME"), ".imagehandler")
	}
	return configDir
}

func GetConfigFilePath() string {
	return filepath.Join(GetConfigPath(), "config.json")
}

func Load() error {
	configPath := GetConfigFilePath()
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return json.Unmarshal(data, AppConfig)
}

func Save() error {
	configPath := GetConfigFilePath()
	
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(AppConfig, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

func AddProject(project *models.Project) {
	AppConfig.Projects[project.ID] = project
	AppConfig.LastProject = project.ID
	Save()
}

func AddPreset(preset *models.Preset) {
	AppConfig.Presets[preset.ID] = preset
	Save()
}

func DeleteProject(id string) {
	delete(AppConfig.Projects, id)
	Save()
}

func DeletePreset(id string) {
	delete(AppConfig.Presets, id)
	Save()
}

func GetProject(id string) *models.Project {
	return AppConfig.Projects[id]
}

func GetPreset(id string) *models.Preset {
	return AppConfig.Presets[id]
}

func GetAllProjects() []*models.Project {
	projects := make([]*models.Project, 0, len(AppConfig.Projects))
	for _, p := range AppConfig.Projects {
		projects = append(projects, p)
	}
	return projects
}

func GetProjectPresets(projectID string) []*models.Preset {
	project := AppConfig.Projects[projectID]
	if project == nil {
		return []*models.Preset{}
	}

	presets := make([]*models.Preset, 0, len(project.PresetIDs))
	for _, pid := range project.PresetIDs {
		if preset := AppConfig.Presets[pid]; preset != nil {
			presets = append(presets, preset)
		}
	}
	return presets
}

func AddPresetToProject(projectID, presetID string) {
	project := AppConfig.Projects[projectID]
	if project == nil {
		return
	}

	for _, id := range project.PresetIDs {
		if id == presetID {
			return
		}
	}

	project.PresetIDs = append(project.PresetIDs, presetID)
	Save()
}

func RemovePresetFromProject(projectID, presetID string) {
	project := AppConfig.Projects[projectID]
	if project == nil {
		return
	}

	newIDs := make([]string, 0)
	for _, id := range project.PresetIDs {
		if id != presetID {
			newIDs = append(newIDs, id)
		}
	}
	project.PresetIDs = newIDs
	Save()
}
