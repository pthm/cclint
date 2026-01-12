package agent

import (
	"embed"
	"fmt"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

//go:embed configs/*.yaml
var configFS embed.FS

// builtinConfigs maps agent names to their configurations
var builtinConfigs = map[string]*Config{}

func init() {
	// Load builtin configurations
	entries, err := configFS.ReadDir("configs")
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		data, err := configFS.ReadFile(filepath.Join("configs", entry.Name()))
		if err != nil {
			continue
		}

		var cfg Config
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			continue
		}

		// Compile reference patterns
		for i := range cfg.ReferencePatterns {
			if err := cfg.ReferencePatterns[i].Compile(); err != nil {
				continue
			}
		}

		builtinConfigs[cfg.Name] = &cfg
	}
}

// Load loads an agent configuration by name
func Load(name string) (*Config, error) {
	if cfg, ok := builtinConfigs[name]; ok {
		return cfg, nil
	}
	return nil, fmt.Errorf("unknown agent: %s", name)
}

// Available returns the names of all available agent configurations
func Available() []string {
	names := make([]string, 0, len(builtinConfigs))
	for name := range builtinConfigs {
		names = append(names, name)
	}
	return names
}

// LoadFromFile loads an agent configuration from a YAML file
func LoadFromFile(path string) (*Config, error) {
	// This allows users to define custom agent configurations
	// Implementation would read from filesystem instead of embed.FS
	return nil, fmt.Errorf("custom agent configs not yet implemented")
}
