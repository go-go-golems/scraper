package submission

import (
	"fmt"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

type fieldRef struct {
	section schema.Section
	def     *fields.Definition
}

func ValuesFromRequest(
	description *cmds.CommandDescription,
	flat map[string]interface{},
	sectionInput map[string]map[string]interface{},
) (*values.Values, error) {
	if description == nil {
		return nil, fmt.Errorf("command description is required")
	}

	parsedValues, err := description.Schema.InitializeFromDefaults()
	if err != nil {
		return nil, fmt.Errorf("initialize defaults: %w", err)
	}

	fieldIndex := map[string]fieldRef{}
	description.Schema.ForEach(func(_ string, section schema.Section) {
		section.GetDefinitions().ForEach(func(def *fields.Definition) {
			if _, exists := fieldIndex[def.Name]; exists {
				return
			}
			fieldIndex[def.Name] = fieldRef{section: section, def: def}
		})
	})

	for sectionSlug, valuesByField := range sectionInput {
		section, ok := description.Schema.Get(sectionSlug)
		if !ok {
			return nil, &ValidationError{Field: sectionSlug, Message: "unknown section"}
		}
		if err := applySectionValues(parsedValues, section, valuesByField); err != nil {
			return nil, err
		}
	}

	for fieldName, rawValue := range flat {
		ref, ok := fieldIndex[fieldName]
		if !ok {
			return nil, &ValidationError{Field: fieldName, Message: "unknown field"}
		}
		if err := parsedValues.GetOrCreate(ref.section).Fields.UpdateValue(
			fieldName,
			ref.def,
			rawValue,
			fields.WithSource("http-body"),
		); err != nil {
			return nil, &ValidationError{Field: fieldName, Message: err.Error()}
		}
	}

	return parsedValues, nil
}

func applySectionValues(parsedValues *values.Values, section schema.Section, input map[string]interface{}) error {
	for fieldName, rawValue := range input {
		def, ok := section.GetDefinitions().Get(fieldName)
		if !ok {
			return &ValidationError{Field: section.GetSlug() + "." + fieldName, Message: "unknown field"}
		}
		if err := parsedValues.GetOrCreate(section).Fields.UpdateValue(
			fieldName,
			def,
			rawValue,
			fields.WithSource("http-body"),
		); err != nil {
			return &ValidationError{Field: section.GetSlug() + "." + fieldName, Message: err.Error()}
		}
	}
	return nil
}
