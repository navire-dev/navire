package template

import (
	"fmt"
	"sort"
	textparse "text/template/parse"
)

func extractFields(name, source string) ([]string, error) {
	tmpl, err := compile(name, source)
	if err != nil {
		return nil, err
	}

	fields := make(map[string]struct{})
	if err := inspectNode(tmpl.Root, fields); err != nil {
		return nil, fmt.Errorf("inspect %s: %w", name, err)
	}

	result := make([]string, 0, len(fields))
	for field := range fields {
		result = append(result, field)
	}
	sort.Strings(result)
	return result, nil
}

func inspectNode(node textparse.Node, fields map[string]struct{}) error {
	switch node := node.(type) {
	case nil, *textparse.TextNode, *textparse.CommentNode:
		return nil
	case *textparse.ListNode:
		for _, child := range node.Nodes {
			if err := inspectNode(child, fields); err != nil {
				return err
			}
		}
		return nil
	case *textparse.ActionNode:
		return inspectPipe(node.Pipe, fields)
	case *textparse.IfNode:
		if err := inspectPipe(node.Pipe, fields); err != nil {
			return err
		}
		if err := inspectNode(node.List, fields); err != nil {
			return err
		}
		return inspectNode(node.ElseList, fields)
	case *textparse.RangeNode, *textparse.WithNode:
		return fmt.Errorf("only top-level data fields are supported")
	case *textparse.TemplateNode:
		return fmt.Errorf("template calls are not supported")
	default:
		return fmt.Errorf("unsupported template expression %T", node)
	}
}

func inspectPipe(pipe *textparse.PipeNode, fields map[string]struct{}) error {
	if pipe == nil {
		return nil
	}
	for _, command := range pipe.Cmds {
		for _, argument := range command.Args {
			switch argument := argument.(type) {
			case *textparse.FieldNode:
				if len(argument.Ident) != 1 {
					return fmt.Errorf("nested data fields are not supported")
				}
				fields[argument.Ident[0]] = struct{}{}
			case *textparse.ChainNode, *textparse.VariableNode, *textparse.DotNode:
				return fmt.Errorf("only top-level data fields are supported")
			case *textparse.PipeNode:
				if err := inspectPipe(argument, fields); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (d *Definition) ValidateData(variantName string, data map[string]any) error {
	variant, ok := d.Variants[variantName]
	if !ok {
		return fmt.Errorf("variant %q not found", variantName)
	}
	fields := variant.RequiredFields
	if fields == nil {
		var err error
		fields, err = fieldsForVariant(variantName, variant)
		if err != nil {
			return err
		}
	}

	missing := make([]string, 0)
	for _, field := range fields {
		if _, ok := data[field]; !ok {
			missing = append(missing, field)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("variant %q requires data fields: %v", variantName, missing)
	}
	return nil
}

func fieldsForVariant(name string, variant Variant) ([]string, error) {
	titleFields, err := extractFields(name+".title", variant.Title)
	if err != nil {
		return nil, err
	}
	bodyFields, err := extractFields(name+".body", variant.Body)
	if err != nil {
		return nil, err
	}
	return mergeFields(titleFields, bodyFields), nil
}

func mergeFields(groups ...[]string) []string {
	fields := make(map[string]struct{})
	for _, group := range groups {
		for _, field := range group {
			fields[field] = struct{}{}
		}
	}
	result := make([]string, 0, len(fields))
	for field := range fields {
		result = append(result, field)
	}
	sort.Strings(result)
	return result
}
