package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const configFileName = ".gatorconfig.json"

func Read() (Config, error) {
	filePath, err := getConfigFilePath()
	if err != nil {
		return Config{}, err
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg.Config); err != nil {
		return Config{}, err
	}

	return cfg, err
}
func getConfigFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".gatorconfig.json"), nil
}
func write(cfg Config) error {
	filePath, err := getConfigFilePath()
	if err != nil {
		return err
	}
	data, err := json.Marshal(cfg.Config)
	if err != nil {
		return err
	}
	err = os.WriteFile(filePath, data, 0666)
	return err
}
