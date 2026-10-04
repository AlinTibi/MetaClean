// Package settings persists MetaClean's user preferences to a small local
// JSON file. Nothing here ever leaves the machine.
package settings

import (
	"encoding/json"
	"os"
	"path/filepath"

	"MetaClean/internal/model"
)

// Path returns the settings file location: %APPDATA%\MetaClean\settings.json
// on Windows (falling back to the user config dir elsewhere, since the
// backend is kept cross-platform where practical).
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "MetaClean", "settings.json"), nil
}

// Load reads settings from disk, returning model.DefaultSettings() if no
// settings file exists yet (first run).
func Load() (model.Settings, error) {
	path, err := Path()
	if err != nil {
		return model.DefaultSettings(), err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return model.DefaultSettings(), nil
		}
		return model.DefaultSettings(), err
	}

	settings := model.DefaultSettings()
	if err := json.Unmarshal(data, &settings); err != nil {
		return model.DefaultSettings(), err
	}
	return settings, nil
}

// Save writes settings to disk, creating the containing directory if
// needed.
func Save(s model.Settings) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
