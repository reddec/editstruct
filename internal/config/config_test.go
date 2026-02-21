package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	t.Run("single document", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "edit.yaml")
		err := os.WriteFile(configPath, []byte(`type: Example
fields:
  Total: uint64
`), 0644)
		require.NoError(t, err)

		configs, err := Load(configPath)
		require.NoError(t, err)
		require.Len(t, configs, 1)
		assert.Equal(t, "Example", configs[0].Type)
		assert.Equal(t, map[string]string{"Total": "uint64"}, configs[0].Fields)
	})

	t.Run("multiple documents", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "edit.yaml")
		err := os.WriteFile(configPath, []byte(`type: Example
fields:
  Total: uint64
---
type: Order
fields:
  CreatedAt: time.Time
`), 0644)
		require.NoError(t, err)

		configs, err := Load(configPath)
		require.NoError(t, err)
		require.Len(t, configs, 2)
		assert.Equal(t, "Example", configs[0].Type)
		assert.Equal(t, "Order", configs[1].Type)
	})

	t.Run("empty fields ignored", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "edit.yaml")
		err := os.WriteFile(configPath, []byte(`type: OnlyType
---
type: WithFields
fields:
  Foo: string
`), 0644)
		require.NoError(t, err)

		configs, err := Load(configPath)
		require.NoError(t, err)
		require.Len(t, configs, 1)
		assert.Equal(t, "WithFields", configs[0].Type)
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := Load("/nonexistent/path.yaml")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "read config")
	})

	t.Run("invalid yaml", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "edit.yaml")
		err := os.WriteFile(configPath, []byte(`type: Example
fields: [invalid`), 0644)
		require.NoError(t, err)

		_, err = Load(configPath)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parse config")
	})
}

func TestTypeConfig_Imports(t *testing.T) {
	t.Run("no qualified types", func(t *testing.T) {
		tc := TypeConfig{
			Type:   "Example",
			Fields: map[string]string{"Total": "uint64", "Name": "string"},
		}
		imports := tc.Imports()
		assert.Empty(t, imports)
	})

	t.Run("with qualified types", func(t *testing.T) {
		tc := TypeConfig{
			Type: "Example",
			Fields: map[string]string{
				"CreatedAt": "time.Time",
				"ID":        "uuid.UUID",
			},
		}
		imports := tc.Imports()
		assert.Equal(t, map[string]string{"time": "time", "uuid": "uuid"}, imports)
	})

	t.Run("pointer to qualified type", func(t *testing.T) {
		tc := TypeConfig{
			Type:   "Example",
			Fields: map[string]string{"CreatedAt": "*time.Time"},
		}
		imports := tc.Imports()
		assert.Equal(t, map[string]string{"time": "time"}, imports)
	})

	t.Run("mixed types", func(t *testing.T) {
		tc := TypeConfig{
			Type: "Example",
			Fields: map[string]string{
				"Name":      "string",
				"CreatedAt": "time.Time",
				"Count":     "int64",
			},
		}
		imports := tc.Imports()
		assert.Len(t, imports, 1)
		assert.Contains(t, imports, "time")
	})
}

func TestParseQualifiedType(t *testing.T) {
	t.Run("built-in type", func(t *testing.T) {
		pkg, alias, ok := parseQualifiedType("int64")
		assert.False(t, ok)
		assert.Empty(t, pkg)
		assert.Empty(t, alias)
	})

	t.Run("qualified type", func(t *testing.T) {
		pkg, alias, ok := parseQualifiedType("time.Time")
		assert.True(t, ok)
		assert.Equal(t, "time", pkg)
		assert.Equal(t, "time", alias)
	})

	t.Run("pointer qualified type", func(t *testing.T) {
		pkg, alias, ok := parseQualifiedType("*time.Time")
		assert.True(t, ok)
		assert.Equal(t, "time", pkg)
		assert.Equal(t, "time", alias)
	})

	t.Run("custom package", func(t *testing.T) {
		pkg, alias, ok := parseQualifiedType("uuid.UUID")
		assert.True(t, ok)
		assert.Equal(t, "uuid", pkg)
		assert.Equal(t, "uuid", alias)
	})
}
