package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the tag manager configuration
type Config struct {
	Packages map[string]PackageConfig `yaml:"packages"`
	Defaults DefaultConfig            `yaml:"defaults"`
}

// PackageConfig represents configuration for a specific package
type PackageConfig struct {
	ModulePath  string `yaml:"module_path"`
	TagFormat   string `yaml:"tag_format"`
	Repository  string `yaml:"repository,omitempty"`
	UseDefault  bool   `yaml:"use_default"`
	LastUpdated string `yaml:"last_updated,omitempty"`
}

// DefaultConfig represents default configuration
type DefaultConfig struct {
	TagFormat string `yaml:"tag_format"`
}

// DefaultTagFormat is the default tag format
const DefaultTagFormat = "{package-name}/v{major}.{minor}.{patch}"

// LoadConfig loads configuration from file
func LoadConfig(configPath string) (*Config, error) {
	config := &Config{
		Packages: make(map[string]PackageConfig),
		Defaults: DefaultConfig{
			TagFormat: DefaultTagFormat,
		},
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default config file
		if err := SaveConfig(config, configPath); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
		return config, nil
	}

	// Read existing config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Ensure packages map is initialized
	if config.Packages == nil {
		config.Packages = make(map[string]PackageConfig)
	}

	// Set default tag format if not specified
	if config.Defaults.TagFormat == "" {
		config.Defaults.TagFormat = DefaultTagFormat
	}

	return config, nil
}

// SaveConfig saves configuration to file
func SaveConfig(config *Config, configPath string) error {
	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetConfigPath returns the default config file path
func GetConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ".tag-manager.yaml"
	}
	return filepath.Join(homeDir, ".tag-manager.yaml")
}

// GetPackageConfig returns configuration for a specific package
func (c *Config) GetPackageConfig(modulePath string) PackageConfig {
	if pkg, exists := c.Packages[modulePath]; exists {
		return pkg
	}

	// Return default configuration
	return PackageConfig{
		ModulePath: modulePath,
		TagFormat:  c.Defaults.TagFormat,
		UseDefault: true,
	}
}

// SetPackageConfig sets configuration for a specific package
func (c *Config) SetPackageConfig(modulePath string, pkgConfig PackageConfig) {
	c.Packages[modulePath] = pkgConfig
}
