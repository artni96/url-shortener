package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type FieldInfo struct {
	Name         string
	DefaultValue string
	IsSlice      bool
}

type StructInfo struct {
	PackageName string
	StructName  string
	Fields      []FieldInfo
}

type TemplateData struct {
	PackageName string
	Structs     []StructInfo
}

const resetTemplate = `//
package {{.PackageName}}

{{range .Structs}}
func (s *{{.StructName}}) Reset() {
	{{- range .Fields}}
	s.{{.Name}} = {{.DefaultValue}}
	{{- end}}
}
{{end}}
`

func main() {
	absDirPath, err := filepath.Abs("./internal/model")
	if err != nil {
		log.Fatal(fmt.Errorf("Error getting absolute path: %v\n", err))
	}

	if _, err = os.Stat(absDirPath); os.IsNotExist(err) {
		log.Fatal(fmt.Errorf("Directory does not exist: %s\n", absDirPath))
	}

	err = processDirectory(absDirPath)
	if err != nil {
		log.Fatal(fmt.Errorf("Error: %v\n", err))
	}
}

func processDirectory(dir string) error {
	var allStructs []StructInfo
	var pkgName string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "vendor" || name == "testdata" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, content, parser.ParseComments)
		if err != nil {
			return nil
		}

		if file.Name != nil {
			if pkgName == "" {
				pkgName = file.Name.Name
			} else if pkgName != file.Name.Name {
				return nil
			}
		}
		structsInFile := findStructsWithResetComment(file)
		if len(structsInFile) > 0 {
			for _, s := range structsInFile {
				s.PackageName = file.Name.Name
				allStructs = append(allStructs, s)
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	if len(allStructs) == 0 {
		return nil
	}

	return generateResetFile(dir, pkgName, allStructs)
}

func findStructsWithResetComment(file *ast.File) []StructInfo {
	var structs []StructInfo

	ast.Inspect(file, func(n ast.Node) bool {
		switch obj := n.(type) {
		case *ast.TypeSpec:
			_, ok := obj.Type.(*ast.StructType)
			if !ok {
				return true
			}

			structType := obj.Type.(*ast.StructType)

			var fields []FieldInfo
			if structType.Fields != nil && structType.Fields.List != nil {
				for _, field := range structType.Fields.List {
					if len(field.Names) == 0 {
						fieldTypeName := field.Type.(*ast.Ident).Name
						defaultValue := getDefaultValue(fieldTypeName)
						fields = append(fields, FieldInfo{
							Name:         fieldTypeName,
							DefaultValue: defaultValue,
						})
					} else {
						for _, name := range field.Names {
							fieldTypeName := field.Type.(*ast.Ident).Name
							defaultValue := getDefaultValue(fieldTypeName)
							fields = append(fields, FieldInfo{
								Name:         name.Name,
								DefaultValue: defaultValue,
							})
						}
					}
				}
			}
			structs = append(structs, StructInfo{
				StructName: obj.Name.Name,
				Fields:     fields,
			})
		}
		return true
	})

	return structs
}

func getDefaultValue(typeName string) string {
	switch typeName {
	case "bool":
		return "false"
	case "string":
		return `""`
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr", "byte":
		return "0"
	case "float32", "float64":
		return "0.0"

	case "interface{}", "error":
		return "nil"
	default:
		if strings.HasPrefix(typeName, "*") {
			return "nil"
		}
		if strings.HasPrefix(typeName, "[]") {
			return "nil"
		}
		if strings.HasPrefix(typeName, "map[") {
			return "nil"
		}
		if strings.HasPrefix(typeName, "func") {
			return "nil"
		}
		return typeName
	}
}

func generateResetFile(dir, pkgName string, structs []StructInfo) error {
	data := TemplateData{
		PackageName: pkgName,
		Structs:     structs,
	}

	tmpl, err := template.New("reset").Parse(resetTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return fmt.Errorf("failed to execute template: %v", err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		fmt.Printf("Warning: could not format generated code: %v\n", err)
		formatted = buf.Bytes()
	}

	outputPath := filepath.Join(dir, "reset.gen.go")
	err = os.WriteFile(outputPath, formatted, 0644)
	if err != nil {
		return fmt.Errorf("failed to write %s: %v", outputPath, err)
	}
	return nil
}
