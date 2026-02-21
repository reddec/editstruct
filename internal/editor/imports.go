package editor

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

type importManager struct {
	file     *ast.File
	fset     *token.FileSet
	existing map[string]string
}

func newImportManager(file *ast.File, fset *token.FileSet, src []byte) *importManager {
	existing := make(map[string]string)
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		var name string
		if imp.Name != nil {
			name = imp.Name.Name
		} else {
			parts := strings.Split(path, "/")
			name = parts[len(parts)-1]
		}
		existing[name] = path
	}
	return &importManager{
		file:     file,
		fset:     fset,
		existing: existing,
	}
}

func (im *importManager) add(required map[string]string, src *[]byte) error {
	var toAdd []importSpec

	for alias, pkgPath := range required {
		if _, exists := im.existing[alias]; !exists {
			toAdd = append(toAdd, importSpec{alias: alias, path: pkgPath})
			im.existing[alias] = pkgPath
		}
	}

	if len(toAdd) == 0 {
		return nil
	}

	if len(im.file.Imports) == 0 {
		return im.insertNewImportBlock(toAdd, src)
	}

	importDecl := im.findImportDecl()
	if importDecl != nil && importDecl.Lparen.IsValid() {
		return im.addToBlock(importDecl, toAdd, src)
	}

	return im.convertToBlock(toAdd, src)
}

func (im *importManager) insertNewImportBlock(toAdd []importSpec, src *[]byte) error {
	insertPos := im.findInsertPosition()
	start := im.fset.Position(insertPos).Offset

	var lines []string
	lines = append(lines, "import (")
	for _, spec := range toAdd {
		lines = append(lines, fmt.Sprintf("\t\"%s\"", spec.path))
	}
	lines = append(lines, ")\n\n")

	newBlock := strings.Join(lines, "\n")
	*src = append((*src)[:start], append([]byte(newBlock), (*src)[start:]...)...)
	return nil
}

func (im *importManager) findInsertPosition() token.Pos {
	for _, decl := range im.file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			if d.Tok == token.IMPORT {
				continue
			}
			return d.Pos()
		case *ast.FuncDecl:
			return d.Pos()
		}
	}
	return im.file.FileEnd
}

func (im *importManager) addToBlock(importDecl *ast.GenDecl, toAdd []importSpec, src *[]byte) error {
	start := im.fset.Position(importDecl.Lparen).Offset
	end := im.fset.Position(importDecl.Rparen).Offset + 1

	var existingImports []string
	for _, imp := range importDecl.Specs {
		existingImports = append(existingImports, fmt.Sprintf("\t%s", im.specString(imp)))
	}
	for _, spec := range toAdd {
		existingImports = append(existingImports, fmt.Sprintf("\t\"%s\"", spec.path))
	}

	newBlock := fmt.Sprintf("(\n%s\n)", strings.Join(existingImports, "\n"))
	*src = append((*src)[:start], append([]byte(newBlock), (*src)[end:]...)...)
	return nil
}

func (im *importManager) convertToBlock(toAdd []importSpec, src *[]byte) error {
	for _, decl := range im.file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.IMPORT {
			continue
		}

		start := im.fset.Position(gd.Pos()).Offset
		end := im.fset.Position(gd.End()).Offset

		var imports []string
		for _, spec := range gd.Specs {
			imports = append(imports, fmt.Sprintf("\t%s", im.specString(spec)))
		}
		for _, spec := range toAdd {
			imports = append(imports, fmt.Sprintf("\t\"%s\"", spec.path))
		}

		newBlock := fmt.Sprintf("import (\n%s\n)", strings.Join(imports, "\n"))
		*src = append((*src)[:start], append([]byte(newBlock), (*src)[end:]...)...)
		return nil
	}
	return fmt.Errorf("no import declaration found")
}

func (im *importManager) findImportDecl() *ast.GenDecl {
	for _, decl := range im.file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.IMPORT {
			continue
		}
		return gd
	}
	return nil
}

func (im *importManager) specString(spec ast.Spec) string {
	is, ok := spec.(*ast.ImportSpec)
	if !ok {
		return ""
	}
	if is.Name != nil {
		return fmt.Sprintf("%s %s", is.Name.Name, is.Path.Value)
	}
	return is.Path.Value
}

type importSpec struct {
	alias string
	path  string
}
