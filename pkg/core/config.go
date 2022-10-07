package core

import (
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

type Config struct {
	Logger      bool                    `yaml:"logger"`
	SCPSLConfig SCPSLConfig             `yaml:"scpslConfig"`
	Servers     map[string]ServerConfig `yaml:"servers"`
}

type SCPSLConfig struct {
	AccountID *int    `yaml:"accountID"`
	APIKey    *string `yaml:"APIKey"`
}

type ServerConfig struct {
	Name     string                 `yaml:"name"`
	Game     string                 `yaml:"game"`
	BotToken string                 `yaml:"botToken"`
	BotID    string                 `yaml:"botID"`
	Info     map[string]interface{} `yaml:"info"`
}

func ParseConfig(path string) (*Config, error) {
	filename, _ := filepath.Abs(path)
	yamlFile, err := os.ReadFile(filename)

	if err != nil {
		return nil, err
	}

	var config Config

	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
