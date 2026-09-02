package config

import (
	_ "embed"
	"fmt"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

//go:embed default.toml
var defaultTOML []byte

// Default returns a new Config initialized with standard compiled-in defaults
// loaded from the embedded default.toml specification.
// If baseDir is provided, the project name defaults to the base directory's name.
func Default(baseDir string) *Config {
	var cfg Config
	if err := toml.Unmarshal(defaultTOML, &cfg); err != nil {
		panic(fmt.Sprintf("failed to parse embedded default.toml: %v", err))
	}

	if baseDir != "" && baseDir != "." {
		clean := filepath.Clean(baseDir)
		base := filepath.Base(clean)
		if base != "" && base != "." && base != "/" {
			cfg.Project.Name = base
		}
	}

	return &cfg
}
