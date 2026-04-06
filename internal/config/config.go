package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Settings struct {
	Theme         string `json:"theme"`
	DateFormat    string `json:"date_format"`
	ConfirmDelete bool   `json:"confirm_delete"`
}

func Defaults() Settings {
	return Settings{
		Theme:         "auto",
		DateFormat:    "Mon 2006-01-02 15:04",
		ConfirmDelete: true,
	}
}

func path() (string, error) {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfg, "cronk", "settings.json"), nil
}

func Load() (Settings, error) {
	p, err := path()
	if err != nil {
		return Defaults(), err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			s := Defaults()
			_ = Save(s)
			return s, nil
		}
		return Defaults(), err
	}
	s := Defaults()
	if err := json.Unmarshal(data, &s); err != nil {
		return Defaults(), err
	}
	return s, nil
}

func Save(s Settings) error {
	p, err := path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o600)
}
