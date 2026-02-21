package editor

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

type Editor struct {
	fset    *token.FileSet
	file    *ast.File
	src     []byte
	imports *importManager
}

type FieldEdit struct {
	OldType string
	NewType string
}

func ParseFile(path string) (*Editor, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("parse file: %w", err)
	}

	return &Editor{
		fset:    fset,
		file:    file,
		src:     src,
		imports: newImportManager(file, fset, src),
	}, nil
}

func (e *Editor) StructNames() []string {
	var names []string
	for _, decl := range e.file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			names = append(names, ts.Name.Name)
		}
	}
	return names
}

func (e *Editor) EditStruct(structName string, fieldEdits map[string]string) (bool, error) {
	var modified bool

	for _, decl := range e.file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}

		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || ts.Name.Name != structName {
				continue
			}

			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}

			changed := e.editFields(st, fieldEdits)
			if changed {
				modified = true
			}
		}
	}

	return modified, nil
}

func (e *Editor) editFields(st *ast.StructType, fieldEdits map[string]string) bool {
	var modified bool

	for _, field := range st.Fields.List {
		if len(field.Names) == 0 {
			continue
		}

		for _, name := range field.Names {
			newType, ok := fieldEdits[name.Name]
			if !ok {
				continue
			}

			oldType := e.typeString(field.Type)
			if oldType == newType {
				continue
			}

			e.replaceType(field.Type, newType)
			modified = true
		}
	}

	return modified
}

func (e *Editor) typeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return fmt.Sprintf("%s.%s", e.typeString(t.X), t.Sel.Name)
	case *ast.StarExpr:
		return "*" + e.typeString(t.X)
	case *ast.ArrayType:
		return "[]" + e.typeString(t.Elt)
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", e.typeString(t.Key), e.typeString(t.Value))
	default:
		return ""
	}
}

func (e *Editor) replaceType(expr ast.Expr, newType string) {
	start := e.fset.Position(expr.Pos()).Offset
	end := e.fset.Position(expr.End()).Offset

	e.src = append(e.src[:start], append([]byte(newType), e.src[end:]...)...)
}

func (e *Editor) AddImports(required map[string]string) error {
	return e.imports.add(required, &e.src)
}

func (e *Editor) Source() []byte {
	return e.src
}

func (e *Editor) WriteTo(path string) error {
	return os.WriteFile(path, e.src, 0644)
}

func ParseTypeString(typeStr string) (pkgPath string, typeName string, isPointer bool) {
	typeStr = strings.TrimSpace(typeStr)
	isPointer = strings.HasPrefix(typeStr, "*")
	if isPointer {
		typeStr = strings.TrimPrefix(typeStr, "*")
	}

	parts := strings.SplitN(typeStr, ".", 2)
	if len(parts) == 2 {
		return parts[0], parts[1], isPointer
	}
	return "", typeStr, isPointer
}

func (e *Editor) RequiredImports(fieldEdits map[string]string) map[string]string {
	imports := make(map[string]string)
	for _, typeStr := range fieldEdits {
		pkgPath, _, _ := ParseTypeString(typeStr)
		if pkgPath != "" {
			imports[pkgPath] = pkgPath
		}
	}
	return imports
}
