package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/reddec/editstruct/internal/config"
	"github.com/reddec/editstruct/internal/editor"
)

func main() {
	configPath := flag.String("config", "edit.yaml", "path to configuration file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(os.Stderr, "config file not found: %s\n", *configPath)
		} else {
			fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		}
		os.Exit(1)
	}

	if len(cfg) == 0 {
		return
	}

	files, err := findGoFiles()
	if err != nil {
		fmt.Fprintf(os.Stderr, "find go files: %v\n", err)
		os.Exit(1)
	}

	for _, file := range files {
		if err := processFile(file, cfg); err != nil {
			fmt.Fprintf(os.Stderr, "process %s: %v\n", file, err)
			os.Exit(1)
		}
	}
}

func findGoFiles() ([]string, error) {
	entries, err := os.ReadDir(".")
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
			files = append(files, name)
		}
	}
	return files, nil
}

func processFile(path string, configs []config.TypeConfig) error {
	ed, err := editor.ParseFile(path)
	if err != nil {
		return err
	}

	structNames := ed.StructNames()
	configMap := make(map[string]config.TypeConfig)
	for _, c := range configs {
		configMap[c.Type] = c
	}

	var anyModified bool
	for _, name := range structNames {
		tc, ok := configMap[name]
		if !ok {
			continue
		}

		modified, err := ed.EditStruct(name, tc.Fields)
		if err != nil {
			return fmt.Errorf("edit struct %s: %w", name, err)
		}
		if modified {
			anyModified = true
		}
	}

	if anyModified {
		ed.Apply()

		requiredImports := make(map[string]string)
		for _, tc := range configs {
			for alias, pkg := range tc.Imports() {
				requiredImports[alias] = pkg
			}
		}

		if len(requiredImports) > 0 {
			if err := ed.AddImports(requiredImports); err != nil {
				return fmt.Errorf("add imports: %w", err)
			}
		}
	}

	if anyModified {
		return ed.WriteTo(path)
	}

	return nil
}
