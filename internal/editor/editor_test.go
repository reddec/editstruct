package editor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFile(t *testing.T) {
	t.Run("valid file", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "types.go")
		err := os.WriteFile(filePath, []byte(`package test

type Example struct {
	ID int64
}
`), 0644)
		require.NoError(t, err)

		ed, err := ParseFile(filePath)
		require.NoError(t, err)
		require.NotNil(t, ed)
		assert.NotNil(t, ed.file)
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := ParseFile("/nonexistent/path.go")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "read file")
	})

	t.Run("invalid go syntax", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "types.go")
		err := os.WriteFile(filePath, []byte(`package test
type Example struct {`), 0644)
		require.NoError(t, err)

		_, err = ParseFile(filePath)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parse file")
	})
}

func TestEditor_StructNames(t *testing.T) {
	t.Run("single struct", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "types.go")
		err := os.WriteFile(filePath, []byte(`package test

type Example struct {
	ID int64
}
`), 0644)
		require.NoError(t, err)

		ed, err := ParseFile(filePath)
		require.NoError(t, err)

		names := ed.StructNames()
		assert.Equal(t, []string{"Example"}, names)
	})

	t.Run("multiple structs", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "types.go")
		err := os.WriteFile(filePath, []byte(`package test

type Example struct {
	ID int64
}

type Order struct {
	Count int
}
`), 0644)
		require.NoError(t, err)

		ed, err := ParseFile(filePath)
		require.NoError(t, err)

		names := ed.StructNames()
		assert.Equal(t, []string{"Example", "Order"}, names)
	})

	t.Run("no structs", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "types.go")
		err := os.WriteFile(filePath, []byte(`package test

const Foo = "bar"
`), 0644)
		require.NoError(t, err)

		ed, err := ParseFile(filePath)
		require.NoError(t, err)

		names := ed.StructNames()
		assert.Empty(t, names)
	})

	t.Run("ignores non-type declarations", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "types.go")
		err := os.WriteFile(filePath, []byte(`package test

type Example struct {
	ID int64
}

func DoSomething() {}

const MaxSize = 100
`), 0644)
		require.NoError(t, err)

		ed, err := ParseFile(filePath)
		require.NoError(t, err)

		names := ed.StructNames()
		assert.Equal(t, []string{"Example"}, names)
	})
}

func TestEditor_EditStruct(t *testing.T) {
	t.Run("change pointer type to value", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "types.go")
		original := `package test

type Example struct {
	ID    int64
	Total *int64
}
`
		err := os.WriteFile(filePath, []byte(original), 0644)
		require.NoError(t, err)

		ed, err := ParseFile(filePath)
		require.NoError(t, err)

		modified, err := ed.EditStruct("Example", map[string]string{"Total": "uint64"})
		require.NoError(t, err)
		assert.True(t, modified)

		ed.Apply()

		assert.Contains(t, string(ed.Source()), "Total uint64")
	})

	t.Run("change value type to pointer", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "types.go")
		original := `package test

type Example struct {
	Name string
}
`
		err := os.WriteFile(filePath, []byte(original), 0644)
		require.NoError(t, err)

		ed, err := ParseFile(filePath)
		require.NoError(t, err)

		modified, err := ed.EditStruct("Example", map[string]string{"Name": "*string"})
		require.NoError(t, err)
		assert.True(t, modified)

		ed.Apply()

		assert.Contains(t, string(ed.Source()), "Name *string")
	})

	t.Run("change to qualified type", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "types.go")
		original := `package test

type Example struct {
	CreatedAt string
}
`
		err := os.WriteFile(filePath, []byte(original), 0644)
		require.NoError(t, err)

		ed, err := ParseFile(filePath)
		require.NoError(t, err)

		modified, err := ed.EditStruct("Example", map[string]string{"CreatedAt": "time.Time"})
		require.NoError(t, err)
		assert.True(t, modified)

		ed.Apply()

		assert.Contains(t, string(ed.Source()), "CreatedAt time.Time")
	})

	t.Run("preserve struct tags", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "types.go")
		original := `package test

type Example struct {
	Total *int64 ` + "`" + `json:"total"` + "`" + `
}
`
		err := os.WriteFile(filePath, []byte(original), 0644)
		require.NoError(t, err)

		ed, err := ParseFile(filePath)
		require.NoError(t, err)

		modified, err := ed.EditStruct("Example", map[string]string{"Total": "uint64"})
		require.NoError(t, err)
		assert.True(t, modified)

		ed.Apply()

		src := string(ed.Source())
		assert.Contains(t, src, "Total uint64")
		assert.Contains(t, src, "`json:\"total\"`")
	})

	t.Run("preserve comments", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "types.go")
		original := `package test

type Example struct {
	// Total is the sum
	Total *int64
}
`
		err := os.WriteFile(filePath, []byte(original), 0644)
		require.NoError(t, err)

		ed, err := ParseFile(filePath)
		require.NoError(t, err)

		modified, err := ed.EditStruct("Example", map[string]string{"Total": "uint64"})
		require.NoError(t, err)
		assert.True(t, modified)

		ed.Apply()

		src := string(ed.Source())
		assert.Contains(t, src, "Total uint64")
		assert.Contains(t, src, "// Total is the sum")
	})

	t.Run("struct not found", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "types.go")
		original := `package test

type Example struct {
	ID int64
}
`
		err := os.WriteFile(filePath, []byte(original), 0644)
		require.NoError(t, err)

		ed, err := ParseFile(filePath)
		require.NoError(t, err)

		modified, err := ed.EditStruct("NonExistent", map[string]string{"ID": "string"})
		require.NoError(t, err)
		assert.False(t, modified)
	})

	t.Run("field not found", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "types.go")
		original := `package test

type Example struct {
	ID int64
}
`
		err := os.WriteFile(filePath, []byte(original), 0644)
		require.NoError(t, err)

		ed, err := ParseFile(filePath)
		require.NoError(t, err)

		modified, err := ed.EditStruct("Example", map[string]string{"NonExistent": "string"})
		require.NoError(t, err)
		assert.False(t, modified)
	})

	t.Run("same type no change", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "types.go")
		original := `package test

type Example struct {
	ID int64
}
`
		err := os.WriteFile(filePath, []byte(original), 0644)
		require.NoError(t, err)

		ed, err := ParseFile(filePath)
		require.NoError(t, err)

		modified, err := ed.EditStruct("Example", map[string]string{"ID": "int64"})
		require.NoError(t, err)
		assert.False(t, modified)
	})

	t.Run("multiple fields", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "types.go")
		original := `package test

type Example struct {
	ID    int64
	Name  string
	Count int
}
`
		err := os.WriteFile(filePath, []byte(original), 0644)
		require.NoError(t, err)

		ed, err := ParseFile(filePath)
		require.NoError(t, err)

		modified, err := ed.EditStruct("Example", map[string]string{
			"ID":    "string",
			"Count": "int64",
		})
		require.NoError(t, err)
		assert.True(t, modified)

		ed.Apply()

		src := string(ed.Source())
		assert.Contains(t, src, "ID")
		assert.Contains(t, src, "string")
		assert.Contains(t, src, "Count")
		assert.Contains(t, src, "int64")
		assert.Contains(t, src, "Name")
	})

	t.Run("skips embedded fields", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "types.go")
		original := `package test

type Base struct {
	ID int64
}

type Example struct {
	Base
	Name string
}
`
		err := os.WriteFile(filePath, []byte(original), 0644)
		require.NoError(t, err)

		ed, err := ParseFile(filePath)
		require.NoError(t, err)

		modified, err := ed.EditStruct("Example", map[string]string{"Name": "int"})
		require.NoError(t, err)
		assert.True(t, modified)

		ed.Apply()

		src := string(ed.Source())
		assert.Contains(t, src, "Name int")
		assert.Contains(t, src, "Base")
	})

	t.Run("non-struct type declaration", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "types.go")
		original := `package test

type ID int64
`
		err := os.WriteFile(filePath, []byte(original), 0644)
		require.NoError(t, err)

		ed, err := ParseFile(filePath)
		require.NoError(t, err)

		modified, err := ed.EditStruct("ID", map[string]string{"Foo": "string"})
		require.NoError(t, err)
		assert.False(t, modified)
	})
}

func TestEditor_AddImports(t *testing.T) {
	t.Run("add import to file with existing block", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "types.go")
		original := `package test

import (
	"fmt"
)

type Example struct {
	ID int64
}
`
		err := os.WriteFile(filePath, []byte(original), 0644)
		require.NoError(t, err)

		ed, err := ParseFile(filePath)
		require.NoError(t, err)

		err = ed.AddImports(map[string]string{"time": "time"})
		require.NoError(t, err)

		src := string(ed.Source())
		assert.Contains(t, src, `"fmt"`)
		assert.Contains(t, src, `"time"`)
	})

	t.Run("add import to file without imports", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "types.go")
		original := `package test

type Example struct {
	ID int64
}
`
		err := os.WriteFile(filePath, []byte(original), 0644)
		require.NoError(t, err)

		ed, err := ParseFile(filePath)
		require.NoError(t, err)

		err = ed.AddImports(map[string]string{"time": "time"})
		require.NoError(t, err)

		src := string(ed.Source())
		assert.Contains(t, src, "import (")
		assert.Contains(t, src, `"time"`)
		assert.Contains(t, src, "type Example struct")
	})

	t.Run("skip existing import", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "types.go")
		original := `package test

import (
	"time"
)

type Example struct {
	ID int64
}
`
		err := os.WriteFile(filePath, []byte(original), 0644)
		require.NoError(t, err)

		ed, err := ParseFile(filePath)
		require.NoError(t, err)

		err = ed.AddImports(map[string]string{"time": "time"})
		require.NoError(t, err)

		src := string(ed.Source())
		assert.Contains(t, src, `"time"`)
		assert.Equal(t, 1, countSubstring(src, `"time"`))
	})

	t.Run("add multiple imports", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "types.go")
		original := `package test

type Example struct {
	ID int64
}
`
		err := os.WriteFile(filePath, []byte(original), 0644)
		require.NoError(t, err)

		ed, err := ParseFile(filePath)
		require.NoError(t, err)

		err = ed.AddImports(map[string]string{
			"time": "time",
			"fmt":  "fmt",
		})
		require.NoError(t, err)

		src := string(ed.Source())
		assert.Contains(t, src, `"time"`)
		assert.Contains(t, src, `"fmt"`)
	})

	t.Run("empty required imports", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "types.go")
		original := `package test

type Example struct {
	ID int64
}
`
		err := os.WriteFile(filePath, []byte(original), 0644)
		require.NoError(t, err)

		ed, err := ParseFile(filePath)
		require.NoError(t, err)

		err = ed.AddImports(map[string]string{})
		require.NoError(t, err)

		src := string(ed.Source())
		assert.NotContains(t, src, "import")
	})
}

func TestEditor_WriteTo(t *testing.T) {
	t.Run("write modified file", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "types.go")
		original := `package test

type Example struct {
	Total *int64
}
`
		err := os.WriteFile(filePath, []byte(original), 0644)
		require.NoError(t, err)

		ed, err := ParseFile(filePath)
		require.NoError(t, err)

		modified, err := ed.EditStruct("Example", map[string]string{"Total": "uint64"})
		require.NoError(t, err)
		assert.True(t, modified)

		ed.Apply()

		err = ed.WriteTo(filePath)
		require.NoError(t, err)

		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "Total uint64")
	})
}

func TestEditor_Source(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "types.go")
	original := `package test

type Example struct {
	ID int64
}
`
	err := os.WriteFile(filePath, []byte(original), 0644)
	require.NoError(t, err)

	ed, err := ParseFile(filePath)
	require.NoError(t, err)

	src := ed.Source()
	assert.Equal(t, original, string(src))
}

func TestParseTypeString(t *testing.T) {
	t.Run("built-in type", func(t *testing.T) {
		pkgPath, typeName, isPointer := ParseTypeString("int64")
		assert.Empty(t, pkgPath)
		assert.Equal(t, "int64", typeName)
		assert.False(t, isPointer)
	})

	t.Run("pointer to built-in", func(t *testing.T) {
		pkgPath, typeName, isPointer := ParseTypeString("*int64")
		assert.Empty(t, pkgPath)
		assert.Equal(t, "int64", typeName)
		assert.True(t, isPointer)
	})

	t.Run("qualified type", func(t *testing.T) {
		pkgPath, typeName, isPointer := ParseTypeString("time.Time")
		assert.Equal(t, "time", pkgPath)
		assert.Equal(t, "Time", typeName)
		assert.False(t, isPointer)
	})

	t.Run("pointer to qualified type", func(t *testing.T) {
		pkgPath, typeName, isPointer := ParseTypeString("*time.Time")
		assert.Equal(t, "time", pkgPath)
		assert.Equal(t, "Time", typeName)
		assert.True(t, isPointer)
	})

	t.Run("whitespace handling", func(t *testing.T) {
		pkgPath, typeName, isPointer := ParseTypeString("  *string  ")
		assert.Empty(t, pkgPath)
		assert.Equal(t, "string", typeName)
		assert.True(t, isPointer)
	})
}

func TestEditor_RequiredImports(t *testing.T) {
	t.Run("no qualified types", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "types.go")
		err := os.WriteFile(filePath, []byte(`package test
type Example struct{ ID int64 }
`), 0644)
		require.NoError(t, err)

		ed, err := ParseFile(filePath)
		require.NoError(t, err)

		imports := ed.RequiredImports(map[string]string{"ID": "string"})
		assert.Empty(t, imports)
	})

	t.Run("with qualified types", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "types.go")
		err := os.WriteFile(filePath, []byte(`package test
type Example struct{ ID int64 }
`), 0644)
		require.NoError(t, err)

		ed, err := ParseFile(filePath)
		require.NoError(t, err)

		imports := ed.RequiredImports(map[string]string{
			"ID":        "uuid.UUID",
			"CreatedAt": "time.Time",
		})
		assert.Len(t, imports, 2)
		assert.Contains(t, imports, "uuid")
		assert.Contains(t, imports, "time")
	})
}

func TestEditor_TypeString(t *testing.T) {
	t.Run("simple types", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "types.go")
		err := os.WriteFile(filePath, []byte(`package test
type Example struct {
	Int    int64
	Str    string
	Ptr    *int64
	Slice  []string
	MapVal map[string]int
}
`), 0644)
		require.NoError(t, err)

		ed, err := ParseFile(filePath)
		require.NoError(t, err)

		src := ed.Source()
		assert.Contains(t, string(src), "Int    int64")
		assert.Contains(t, string(src), "Str    string")
		assert.Contains(t, string(src), "Ptr    *int64")
		assert.Contains(t, string(src), "Slice  []string")
		assert.Contains(t, string(src), "MapVal map[string]int")
	})
}

func TestEditor_EditComplexTypes(t *testing.T) {
	t.Run("slice type", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "types.go")
		original := `package test

type Example struct {
	Items []string
}
`
		err := os.WriteFile(filePath, []byte(original), 0644)
		require.NoError(t, err)

		ed, err := ParseFile(filePath)
		require.NoError(t, err)

		modified, err := ed.EditStruct("Example", map[string]string{"Items": "[]int64"})
		require.NoError(t, err)
		assert.True(t, modified)

		ed.Apply()

		assert.Contains(t, string(ed.Source()), "Items []int64")
	})

	t.Run("map type", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "types.go")
		original := `package test

type Example struct {
	Data map[string]int
}
`
		err := os.WriteFile(filePath, []byte(original), 0644)
		require.NoError(t, err)

		ed, err := ParseFile(filePath)
		require.NoError(t, err)

		modified, err := ed.EditStruct("Example", map[string]string{"Data": "map[int]string"})
		require.NoError(t, err)
		assert.True(t, modified)

		ed.Apply()

		assert.Contains(t, string(ed.Source()), "Data map[int]string")
	})

	t.Run("pointer to slice", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "types.go")
		original := `package test

type Example struct {
	Items *[]string
}
`
		err := os.WriteFile(filePath, []byte(original), 0644)
		require.NoError(t, err)

		ed, err := ParseFile(filePath)
		require.NoError(t, err)

		modified, err := ed.EditStruct("Example", map[string]string{"Items": "[]int"})
		require.NoError(t, err)
		assert.True(t, modified)

		ed.Apply()

		assert.Contains(t, string(ed.Source()), "Items []int")
	})

	t.Run("qualified type with pointer", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "types.go")
		original := `package test

type Example struct {
	Timestamp *string
}
`
		err := os.WriteFile(filePath, []byte(original), 0644)
		require.NoError(t, err)

		ed, err := ParseFile(filePath)
		require.NoError(t, err)

		modified, err := ed.EditStruct("Example", map[string]string{"Timestamp": "time.Time"})
		require.NoError(t, err)
		assert.True(t, modified)

		ed.Apply()

		assert.Contains(t, string(ed.Source()), "Timestamp time.Time")
	})
}

func TestEditor_AddImports_SingleToBlock(t *testing.T) {
	t.Run("convert single import to block", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "types.go")
		original := `package test

import "fmt"

type Example struct {
	ID int64
}
`
		err := os.WriteFile(filePath, []byte(original), 0644)
		require.NoError(t, err)

		ed, err := ParseFile(filePath)
		require.NoError(t, err)

		err = ed.AddImports(map[string]string{"time": "time"})
		require.NoError(t, err)

		src := string(ed.Source())
		assert.Contains(t, src, "import (")
		assert.Contains(t, src, `"fmt"`)
		assert.Contains(t, src, `"time"`)
	})
}

func TestEditor_AddImports_WithAlias(t *testing.T) {
	t.Run("import with alias already exists", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "types.go")
		original := `package test

import (
	mytime "time"
)

type Example struct {
	ID int64
}
`
		err := os.WriteFile(filePath, []byte(original), 0644)
		require.NoError(t, err)

		ed, err := ParseFile(filePath)
		require.NoError(t, err)

		err = ed.AddImports(map[string]string{"mytime": "time"})
		require.NoError(t, err)

		src := string(ed.Source())
		assert.Contains(t, src, `mytime "time"`)
		assert.Equal(t, 1, countSubstring(src, `"time"`))
	})
}

func TestEditor_InsertImportBeforeType(t *testing.T) {
	t.Run("import inserted before first type declaration", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "types.go")
		original := `package test

const X = 1

type Example struct {
	ID int64
}
`
		err := os.WriteFile(filePath, []byte(original), 0644)
		require.NoError(t, err)

		ed, err := ParseFile(filePath)
		require.NoError(t, err)

		err = ed.AddImports(map[string]string{"time": "time"})
		require.NoError(t, err)

		src := string(ed.Source())
		assert.Contains(t, src, "import (")
		assert.Contains(t, src, `"time"`)
		assert.Contains(t, src, "type Example struct")
	})
}

func TestEditor_InsertImportBeforeFunc(t *testing.T) {
	t.Run("import inserted before function", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "types.go")
		original := `package test

func DoSomething() {}
`
		err := os.WriteFile(filePath, []byte(original), 0644)
		require.NoError(t, err)

		ed, err := ParseFile(filePath)
		require.NoError(t, err)

		err = ed.AddImports(map[string]string{"fmt": "fmt"})
		require.NoError(t, err)

		src := string(ed.Source())
		assert.Contains(t, src, "import (")
		assert.Contains(t, src, `"fmt"`)
		assert.Contains(t, src, "func DoSomething()")
	})
}

func TestEditor_EditStruct_MultipleTypes(t *testing.T) {
	t.Run("multiple structs same file", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "types.go")
		original := `package test

type First struct {
	Value int
}

type Second struct {
	Data string
}
`
		err := os.WriteFile(filePath, []byte(original), 0644)
		require.NoError(t, err)

		ed, err := ParseFile(filePath)
		require.NoError(t, err)

		modified1, err := ed.EditStruct("First", map[string]string{"Value": "int64"})
		require.NoError(t, err)
		assert.True(t, modified1)

		modified2, err := ed.EditStruct("Second", map[string]string{"Data": "[]byte"})
		require.NoError(t, err)
		assert.True(t, modified2)

		ed.Apply()

		src := string(ed.Source())
		assert.Contains(t, src, "Value int64")
		assert.Contains(t, src, "Data []byte")
	})
}

func TestEditor_TypeString_QualifiedPointer(t *testing.T) {
	t.Run("pointer to qualified type", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "types.go")
		original := `package test

type Example struct {
	Time *time.Time
}
`
		err := os.WriteFile(filePath, []byte(original), 0644)
		require.NoError(t, err)

		ed, err := ParseFile(filePath)
		require.NoError(t, err)

		modified, err := ed.EditStruct("Example", map[string]string{"Time": "uuid.UUID"})
		require.NoError(t, err)
		assert.True(t, modified)

		ed.Apply()

		assert.Contains(t, string(ed.Source()), "Time uuid.UUID")
	})
}

func TestEditor_InsertImportOnlyImports(t *testing.T) {
	t.Run("insert after import block with only imports", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "types.go")
		original := `package test

import "fmt"
`
		err := os.WriteFile(filePath, []byte(original), 0644)
		require.NoError(t, err)

		ed, err := ParseFile(filePath)
		require.NoError(t, err)

		err = ed.AddImports(map[string]string{"time": "time"})
		require.NoError(t, err)

		src := string(ed.Source())
		assert.Contains(t, src, "import (")
		assert.Contains(t, src, `"fmt"`)
		assert.Contains(t, src, `"time"`)
	})
}

func countSubstring(s, substr string) int {
	count := 0
	for {
		idx := len(s) - len(substr)
		found := false
		for i := 0; i <= idx; i++ {
			if s[i:i+len(substr)] == substr {
				count++
				s = s[i+len(substr):]
				found = true
				break
			}
		}
		if !found {
			break
		}
	}
	return count
}
