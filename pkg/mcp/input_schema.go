package mcp

import (
	"bytes"
	"errors"
	"fmt"
	"slices"

	"github.com/bytedance/sonic"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

const inputSchemaLoc = "https://flowbot.local/mcp/input.schema.json"

var inputSchemaTypes = map[string]struct{}{
	"string": {}, "number": {}, "integer": {}, "boolean": {}, "object": {}, "array": {},
}

// forbiddenInputSchemaKeys are keywords model APIs commonly reject.
var forbiddenInputSchemaKeys = []string{
	"$ref", "$schema", "oneOf", "anyOf", "allOf", "not",
	"patternProperties", "unevaluatedProperties",
	"if", "then", "else", "dependentSchemas", "prefixItems",
}

type inputNodeRole int

const (
	inputNodeRoot inputNodeRole = iota
	inputNodeProperty
	inputNodeItems
	inputNodeNested
)

// ValidateInputSchema checks an advertised MCP inputSchema.
// The compiler is pinned to JSON Schema draft 2020-12. The schema object itself
// stays without a $schema keyword. Property descriptions are required at every
// depth. items only needs a type. Bare object and array schemas are allowed.
func ValidateInputSchema(schema map[string]any) error {
	if schema == nil {
		return errors.New("input schema is nil")
	}
	if err := checkInputNode(schema, inputNodeRoot, ""); err != nil {
		return err
	}
	return compileInputSchema(schema)
}

func compileInputSchema(schema map[string]any) error {
	raw, err := sonic.Marshal(schema)
	if err != nil {
		return err
	}
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		return err
	}
	c := jsonschema.NewCompiler()
	c.DefaultDraft(jsonschema.Draft2020)
	if err := c.AddResource(inputSchemaLoc, doc); err != nil {
		return err
	}
	if _, err := c.Compile(inputSchemaLoc); err != nil {
		return fmt.Errorf("compile input schema: %w", err)
	}
	return nil
}

func checkInputNode(node map[string]any, role inputNodeRole, path string) error {
	if err := checkForbiddenInputKeys(node, path); err != nil {
		return err
	}
	if err := checkInputType(node, role, path); err != nil {
		return err
	}
	if err := checkPropertyDescription(node, role, path); err != nil {
		return err
	}
	if err := checkInputEnum(node, path); err != nil {
		return err
	}
	if err := checkInputRequired(node, path); err != nil {
		return err
	}
	if err := checkInputProperties(node, path); err != nil {
		return err
	}
	if err := checkInputItems(node, path); err != nil {
		return err
	}
	return checkNestedInputSchemas(node, path)
}

func checkForbiddenInputKeys(node map[string]any, path string) error {
	for _, key := range forbiddenInputSchemaKeys {
		if _, ok := node[key]; ok {
			return fmt.Errorf("%s: forbidden keyword %s", displayInputPath(path), key)
		}
	}
	return nil
}

func checkPropertyDescription(node map[string]any, role inputNodeRole, path string) error {
	if role != inputNodeProperty {
		return nil
	}
	desc, ok := node["description"].(string)
	if !ok || desc == "" {
		return fmt.Errorf("%s: description is required", displayInputPath(path))
	}
	return nil
}

func checkInputProperties(node map[string]any, path string) error {
	raw, ok := node["properties"]
	if !ok {
		return nil
	}
	props, ok := raw.(map[string]any)
	if !ok {
		return fmt.Errorf("%s: properties must be an object", displayInputPath(path))
	}
	names := make([]string, 0, len(props))
	for name := range props {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		prop, ok := props[name].(map[string]any)
		if !ok {
			return fmt.Errorf("%s: property schema must be an object", displayInputPath(joinInputPath(path, name)))
		}
		if err := checkInputNode(prop, inputNodeProperty, joinInputPath(path, name)); err != nil {
			return err
		}
	}
	return nil
}

func checkInputItems(node map[string]any, path string) error {
	raw, ok := node["items"]
	if !ok {
		return nil
	}
	items, ok := raw.(map[string]any)
	if !ok {
		return fmt.Errorf("%s: items must be an object schema", displayInputPath(joinInputPath(path, "items")))
	}
	return checkInputNode(items, inputNodeItems, joinInputPath(path, "items"))
}

func checkNestedInputSchemas(node map[string]any, path string) error {
	for key, raw := range node {
		if key == "properties" || key == "items" {
			continue
		}
		child, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if err := checkInputNode(child, inputNodeNested, joinInputPath(path, key)); err != nil {
			return err
		}
	}
	return nil
}

func checkInputType(node map[string]any, role inputNodeRole, path string) error {
	raw, ok := node["type"]
	switch role {
	case inputNodeRoot:
		s, isString := raw.(string)
		if !ok || !isString || s != "object" {
			return fmt.Errorf("%s: top-level type must be object", displayInputPath(path))
		}
		return nil
	case inputNodeProperty, inputNodeItems:
		if !ok {
			return fmt.Errorf("%s: type is required", displayInputPath(path))
		}
		return checkSingleInputType(raw, path)
	default:
		if !ok {
			return nil
		}
		return checkSingleInputType(raw, path)
	}
}

func checkSingleInputType(raw any, path string) error {
	s, ok := raw.(string)
	if !ok {
		return fmt.Errorf("%s: type must be a single string", displayInputPath(path))
	}
	if _, ok := inputSchemaTypes[s]; !ok {
		return fmt.Errorf("%s: type %q is not allowed", displayInputPath(path), s)
	}
	return nil
}

func checkInputEnum(node map[string]any, path string) error {
	raw, ok := node["enum"]
	if !ok {
		return nil
	}
	list, err := inputAnyList(raw)
	if err != nil {
		return fmt.Errorf("%s: enum must be an array", displayInputPath(path))
	}
	if len(list) == 0 {
		return fmt.Errorf("%s: enum must not be empty", displayInputPath(path))
	}
	return nil
}

func checkInputRequired(node map[string]any, path string) error {
	raw, ok := node["required"]
	if !ok {
		return nil
	}
	list, err := inputAnyList(raw)
	if err != nil {
		return fmt.Errorf("%s: required must be an array of strings", displayInputPath(path))
	}
	names := make([]string, 0, len(list))
	for _, item := range list {
		name, ok := item.(string)
		if !ok {
			return fmt.Errorf("%s: required must be an array of strings", displayInputPath(path))
		}
		names = append(names, name)
	}
	props, ok := node["properties"].(map[string]any)
	if !ok {
		props = map[string]any{}
	}
	for _, name := range names {
		if _, ok := props[name]; !ok {
			return fmt.Errorf("%s: required field %s is not in properties", displayInputPath(path), name)
		}
	}
	return nil
}

func inputAnyList(raw any) ([]any, error) {
	switch list := raw.(type) {
	case []string:
		out := make([]any, len(list))
		for i, s := range list {
			out[i] = s
		}
		return out, nil
	case []any:
		return list, nil
	default:
		return nil, errors.New("not a list")
	}
}

func joinInputPath(path, elem string) string {
	if path == "" {
		return elem
	}
	return path + "." + elem
}

func displayInputPath(path string) string {
	if path == "" {
		return "schema"
	}
	return path
}
