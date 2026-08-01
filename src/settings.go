package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// uiAccountSettings are per-account dashboard credentials entered through the
// management UI. They are persisted with 0600 permissions inside the plugin
// state directory and are never returned by the management API.
type uiAccountSettings struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	Cookie      string `json:"cookie,omitempty"`
}

type persistedSettings struct {
	// Accounts is keyed by the account identity key (the stable auth ID
	// derived from the API key), so entries survive renames but reset when a
	// key is rotated.
	Accounts map[string]uiAccountSettings `json:"accounts"`
}

// settingsPath returns the path of the persisted UI settings file. The
// filename deliberately avoids a .json suffix: CPA's git-backed auth store
// indexes every *.json file under the auth directory recursively, so a
// settings.json inside the state directory would surface as a bogus auth file.
func settingsPath(stateDir string) string {
	return filepath.Join(stateDir, "settings.data")
}

// legacySettingsPath is the pre-0.3.3 filename that must be migrated away.
func legacySettingsPath(stateDir string) string {
	return filepath.Join(stateDir, "settings.json")
}

func loadUISettings(stateDir string) map[string]uiAccountSettings {
	if stateDir == "" {
		return nil
	}
	raw, errRead := os.ReadFile(settingsPath(stateDir))
	if errRead != nil {
		// Fall back to the legacy .json filename so settings saved before the
		// rename are not lost; saveUISettings removes the legacy file once the
		// new one is written.
		raw, errRead = os.ReadFile(legacySettingsPath(stateDir))
	}
	if errRead != nil {
		return nil
	}
	var persisted persistedSettings
	if errUnmarshal := json.Unmarshal(raw, &persisted); errUnmarshal != nil {
		return nil
	}
	return persisted.Accounts
}

func saveUISettings(stateDir string, accounts map[string]uiAccountSettings) error {
	if stateDir == "" {
		return nil
	}
	if errMkdir := os.MkdirAll(stateDir, 0o700); errMkdir != nil {
		return errMkdir
	}
	raw, errMarshal := json.MarshalIndent(persistedSettings{Accounts: accounts}, "", "  ")
	if errMarshal != nil {
		return errMarshal
	}
	tmp := settingsPath(stateDir) + ".tmp"
	if errWrite := os.WriteFile(tmp, raw, 0o600); errWrite != nil {
		return errWrite
	}
	if errRename := os.Rename(tmp, settingsPath(stateDir)); errRename != nil {
		return errRename
	}
	// Remove the legacy .json file so the git-backed auth store stops
	// indexing it as a bogus auth file.
	if errRemove := os.Remove(legacySettingsPath(stateDir)); errRemove != nil && !os.IsNotExist(errRemove) {
		return errRemove
	}
	return nil
}
