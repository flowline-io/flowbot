// Command gendocs generates FieldDocs from pkg/config struct field comments.
package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "gendocs: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	dir := "."
	if _, err := os.Stat(filepath.Join(dir, "config.go")); err != nil {
		dir = ".."
	}
	docs, err := collectFieldDocs(dir)
	if err != nil {
		return err
	}
	src, err := render(docs)
	if err != nil {
		return fmt.Errorf("render: %w", err)
	}
	out := filepath.Join(dir, "field_docs_gen.go")
	if err := os.WriteFile(out, src, 0o644); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	_, _ = fmt.Printf("wrote %s (%d entries)\n", out, len(docs))
	if err := writeSettingsDescLocale(docs); err != nil {
		return fmt.Errorf("settings i18n: %w", err)
	}
	return nil
}

func collectFieldDocs(dir string) (map[string]string, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, gendocsFileFilter, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	pkg := pkgs["config"]
	if pkg == nil {
		return nil, fmt.Errorf("package config not found in %s", dir)
	}

	structs := collectStructs(pkg)
	out := map[string]string{}
	walkStructDocs(structs, "Type", "", map[string]bool{}, out)
	return out, nil
}

func gendocsFileFilter(fi os.FileInfo) bool {
	name := fi.Name()
	if !strings.HasSuffix(name, ".go") {
		return false
	}
	if strings.HasSuffix(name, "_test.go") || strings.HasSuffix(name, "_gen.go") {
		return false
	}
	return name != "generate.go"
}

func collectStructs(pkg *ast.Package) map[string]*ast.StructType {
	structs := map[string]*ast.StructType{}
	for _, f := range pkg.Files {
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					continue
				}
				structs[ts.Name.Name] = st
			}
		}
	}
	return structs
}

func walkStructDocs(
	structs map[string]*ast.StructType,
	typeName, prefix string,
	seen map[string]bool,
	out map[string]string,
) {
	if seen[typeName] {
		return
	}
	seen[typeName] = true
	st := structs[typeName]
	if st == nil {
		return
	}
	for _, field := range st.Fields.List {
		recordFieldDocs(structs, field, prefix, cloneSeen(seen), out)
	}
}

func recordFieldDocs(
	structs map[string]*ast.StructType,
	field *ast.Field,
	prefix string,
	seen map[string]bool,
	out map[string]string,
) {
	if len(field.Names) == 0 {
		return
	}
	for _, name := range field.Names {
		if !name.IsExported() {
			continue
		}
		key := pathKey(field, name.Name)
		if key == "-" {
			continue
		}
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}
		if doc := fieldDoc(field); doc != "" {
			out[path] = doc
		}
		for _, nested := range nestedTypeNames(field.Type) {
			walkStructDocs(structs, nested, path, seen, out)
		}
	}
}

func nestedTypeNames(expr ast.Expr) []string {
	switch t := expr.(type) {
	case *ast.Ident:
		return []string{t.Name}
	case *ast.StarExpr:
		return nestedTypeNames(t.X)
	case *ast.ArrayType:
		return nestedTypeNames(t.Elt)
	default:
		return nil
	}
}

func cloneSeen(in map[string]bool) map[string]bool {
	out := make(map[string]bool, len(in))
	maps.Copy(out, in)
	return out
}

func pathKey(field *ast.Field, fieldName string) string {
	if field.Tag == nil {
		return toSnakeAST(fieldName)
	}
	tag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
	for _, key := range []string{"yaml", "mapstructure", "json"} {
		v := tag.Get(key)
		if v == "" {
			continue
		}
		name := strings.Split(v, ",")[0]
		if name == "-" {
			continue
		}
		if name != "" {
			return name
		}
	}
	return toSnakeAST(fieldName)
}

func fieldDoc(field *ast.Field) string {
	var parts []string
	if field.Doc != nil {
		for _, c := range field.Doc.List {
			parts = append(parts, strings.TrimSpace(strings.TrimPrefix(c.Text, "//")))
		}
	}
	if field.Comment != nil {
		for _, c := range field.Comment.List {
			parts = append(parts, strings.TrimSpace(strings.TrimPrefix(c.Text, "//")))
		}
	}
	doc := strings.Join(parts, " ")
	doc = collapseSpace.ReplaceAllString(doc, " ")
	return strings.TrimSpace(doc)
}

var collapseSpace = regexp.MustCompile(`\s+`)

func toSnakeAST(name string) string {
	if name == "" {
		return name
	}
	var b strings.Builder
	for i, r := range name {
		if i > 0 && r >= 'A' && r <= 'Z' {
			_, _ = b.WriteRune('_')
		}
		if r >= 'A' && r <= 'Z' {
			_, _ = b.WriteRune(r - 'A' + 'a')
		} else {
			_, _ = b.WriteRune(r)
		}
	}
	return b.String()
}

func render(docs map[string]string) ([]byte, error) {
	keys := make([]string, 0, len(docs))
	for k := range docs {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	var b bytes.Buffer
	_, _ = b.WriteString("// Code generated by gendocs; DO NOT EDIT.\n\n")
	_, _ = b.WriteString("package config\n\n")
	_, _ = b.WriteString("// FieldDocs maps normalized yaml-style config paths to field godoc text.\n")
	_, _ = b.WriteString("var FieldDocs = map[string]string{\n")
	for _, k := range keys {
		_, _ = fmt.Fprintf(&b, "\t%q: %q,\n", k, docs[k])
	}
	_, _ = b.WriteString("}\n")
	formatted, err := format.Source(b.Bytes())
	if err != nil {
		return b.Bytes(), err
	}
	if !utf8.Valid(formatted) {
		return nil, fmt.Errorf("generated file is not valid UTF-8")
	}
	return formatted, nil
}

func writeSettingsDescLocale(docs map[string]string) error {
	localePath := filepath.Join("..", "..", "i18n", "locales", "settings_desc.zh.toml")
	existing, err := loadSettingsDescTranslations(localePath)
	if err != nil {
		return err
	}
	keys := make([]string, 0, len(docs))
	for k := range docs {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	var b bytes.Buffer
	_, _ = b.WriteString("# Code generated by gendocs; DO NOT EDIT.\n\n")
	for _, k := range keys {
		text := existing[k]
		if text == "" {
			text = docs[k]
		}
		_, _ = fmt.Fprintf(&b, "[\"settings.desc.%s\"]\nother = %q\n\n", k, text)
	}
	return os.WriteFile(localePath, b.Bytes(), 0o644)
}

func loadSettingsDescTranslations(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	out := make(map[string]string)
	lines := strings.Split(string(data), "\n")
	var currentKey string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[\"settings.desc.") && strings.HasSuffix(line, "\"]") {
			currentKey = strings.TrimPrefix(line, "[\"settings.desc.")
			currentKey = strings.TrimSuffix(currentKey, "\"]")
			continue
		}
		if currentKey != "" && strings.HasPrefix(line, "other = ") {
			val, parseErr := strconv.Unquote(strings.TrimPrefix(line, "other = "))
			if parseErr == nil {
				out[currentKey] = val
			}
			currentKey = ""
		}
	}
	return out, nil
}
