package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type TypeConfig struct {
	Type   string            `yaml:"type"`
	Fields map[string]string `yaml:"fields"`
}

func Load(path string) ([]TypeConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var configs []TypeConfig
	decoder := yaml.NewDecoder(strings.NewReader(string(data)))

	for {
		var cfg TypeConfig
		err := decoder.Decode(&cfg)
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, fmt.Errorf("parse config: %w", err)
		}
		if cfg.Type != "" && len(cfg.Fields) > 0 {
			configs = append(configs, cfg)
		}
	}

	return configs, nil
}

func (tc TypeConfig) Imports() map[string]string {
	imports := make(map[string]string)
	for _, fieldType := range tc.Fields {
		if pkg, alias, ok := parseQualifiedType(fieldType); ok {
			imports[alias] = pkg
		}
	}
	return imports
}

func parseQualifiedType(typeStr string) (pkg string, alias string, ok bool) {
	typeStr = strings.TrimPrefix(typeStr, "*")
	parts := strings.SplitN(typeStr, ".", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[0], true
}
