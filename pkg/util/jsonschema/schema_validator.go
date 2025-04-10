package jsonschema

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

type JsonSchemaValidator struct {
	schemas            map[string]*jsonschema.Schema
	schemaUniqueKeys   map[string][]string
	schemaReadOnlyKeys map[string][]string
}

func NewJsonSchemaValidator(schemaDirectory string) (*JsonSchemaValidator, error) {
	// Validate and clean the base path
	basePath := filepath.Clean(schemaDirectory)
	basePath, err := filepath.Abs(basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for schema base path: %w", err)
	}
	if !filepath.IsAbs(basePath) {
		return nil, errors.New("schema base path must be absolute")
	}

	files, err := os.ReadDir(basePath)
	if err != nil {
		return nil, fmt.Errorf("error reading schema directory: %w", err)
	}
	c := jsonschema.NewCompiler()
	schemaUniqueKeys := make(map[string][]string, 0)
	schemaReadOnlyKeys := make(map[string][]string, 0)
	var schemaIds []string
	for _, f := range files {
		// Skip non-regular files and hidden files
		if !f.Type().IsRegular() || strings.HasPrefix(f.Name(), ".") {
			continue
		}

		// Validate filename
		if strings.Contains(f.Name(), "..") {
			continue
		}

		fname := filepath.Join(basePath, f.Name())
		cleanPath := filepath.Clean(fname)
		if !strings.HasPrefix(cleanPath, basePath) {
			continue
		}
		jsonData, err := os.ReadFile(fname)
		if err != nil {
			return nil, fmt.Errorf("error reading schema file: %w", err)
		}
		jsonElems := make(map[string]any)
		err = json.Unmarshal(jsonData, &jsonElems)
		if err != nil {
			return nil, errors.Join(err)
		}
		schemaId := jsonElems["id"].(string)
		if schemaId == "" {
			return nil, errors.New("missing id in the json schema - " + f.Name())
		}
		// If there are any unique keys defined we will collect and store as well.
		if uk, ok := jsonElems["uniqueKeys"].([]any); ok {
			if uniqueKeys, err := convertInterfaceSliceToStringSlice(uk); err == nil {
				if len(uniqueKeys) > 0 {
					schemaUniqueKeys[schemaId] = uniqueKeys
				}
			}
		} else if nuk, ok := jsonElems["readOnlyKeys"].([]any); ok {
			if readOnlyKeys, err := convertInterfaceSliceToStringSlice(nuk); err == nil {
				if len(readOnlyKeys) > 0 {
					schemaReadOnlyKeys[schemaId] = readOnlyKeys
				}
			}
		}
		if err := c.AddResource(schemaId, strings.NewReader(string(jsonData))); err != nil {
			return nil, fmt.Errorf("unable to add schema: %w", err)
		}
		schemaIds = append(schemaIds, schemaId)
	}
	compiledSchemas := make(map[string]*jsonschema.Schema, 0)
	for _, sid := range schemaIds {
		sch, err := c.Compile(sid)
		if err != nil {
			return nil, fmt.Errorf("error compiling schema :%w", err)
		}
		compiledSchemas[sid] = sch
	}
	return &JsonSchemaValidator{schemas: compiledSchemas, schemaUniqueKeys: schemaUniqueKeys, schemaReadOnlyKeys: schemaReadOnlyKeys}, nil
}

func (v *JsonSchemaValidator) ValidateJson(schemaId string, jsonObject any) ([]*SchemaFieldViolation, error) {
	schema := v.schemas[schemaId]
	if schema == nil {
		return nil, errors.New("invalid schema id " + schemaId)
	}
	if mapData, ok := jsonObject.(map[string]string); ok {
		v, err := convertMapToAny(mapData)
		if err != nil {
			return nil, fmt.Errorf("unable to convert map to json: %w", err)
		}
		return validateWithSchema(schema, v)
	}
	// Convert protobuf/other objects to JSON format
	jsonBytes, err := json.Marshal(jsonObject)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal object to json: %w", err)
	}
	var jsonData any
	err = json.Unmarshal(jsonBytes, &jsonData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal json to interface: %w", err)
	}
	return validateWithSchema(schema, jsonData)

}

func validateWithSchema(schema *jsonschema.Schema, jsonObject any) ([]*SchemaFieldViolation, error) {
	if err := schema.Validate(jsonObject); err != nil {
		if validationErr, ok := err.(*jsonschema.ValidationError); ok {
			validationErrors := mapSchemaValidationErrors(validationErr)
			return validationErrors, errors.New("validation errors")
		}
		return nil, err
	}
	return nil, nil
}

// New function for validating map with generic type parameter
func ValidateMap[T any](schema *jsonschema.Schema, data map[string]T) ([]*SchemaFieldViolation, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("error marshaling data to json: %w", err)
	}

	var jsonObject any
	if err := json.Unmarshal(jsonData, &jsonObject); err != nil {
		return nil, fmt.Errorf("error unmarshaling json data: %w", err)
	}

	return validateWithSchema(schema, jsonObject)
}

func (v *JsonSchemaValidator) ValidateMap(schemaId string, data map[string]any) ([]*SchemaFieldViolation, error) {
	schema := v.schemas[schemaId]
	if schema == nil {
		return nil, errors.New("invalid schema id " + schemaId)
	}

	return ValidateMap(schema, data)
}

func (v *JsonSchemaValidator) GetUniqueKeys(schemaId string) ([]string, error) {
	schemaUniqueKeys, ok := v.schemaUniqueKeys[schemaId]
	if !ok {
		return nil, errors.New("invalid schema id " + schemaId)
	}
	return schemaUniqueKeys, nil
}

func (v *JsonSchemaValidator) GetReadOnlyKeys(schemaId string) ([]string, error) {
	schemaReadOnlyKeys, ok := v.schemaReadOnlyKeys[schemaId]
	if !ok {
		return nil, errors.New("invalid schema id " + schemaId)
	}
	return schemaReadOnlyKeys, nil
}

func (v *JsonSchemaValidator) GetSchema(schemaId string) (*jsonschema.Schema, error) {
	schema, ok := v.schemas[schemaId]
	if !ok {
		return nil, errors.New("invalid schema id " + schemaId)
	}
	return schema, nil
}

func convertMapToAny(mapData map[string]string) (any, error) {
	jb, err := json.Marshal(mapData)
	if err != nil {
		return nil, err
	}
	var v any
	err = json.Unmarshal(jb, &v)
	return v, err
}

// ConvertInterfaceSliceToStringSlice converts a slice of any to a slice of string
func convertInterfaceSliceToStringSlice(input []any) ([]string, error) {
	output := make([]string, len(input))
	for i, v := range input {
		str, ok := v.(string)
		if !ok {
			return nil, errors.New("element is not a string")
		}
		output[i] = str
	}
	return output, nil
}

type SchemaFieldViolation struct {
	Field    string
	Messages []string
}

func mapSchemaValidationErrors(validationErr *jsonschema.ValidationError) []*SchemaFieldViolation {
	if validationErr == nil {
		return nil
	}

	fieldErrorMap := make(map[string][]string)

	// Process all causes to collect detailed errors
	for _, cause := range validationErr.Causes {
		field := cause.InstanceLocation
		msg := cause.Message
		if msg != "" {
			fieldErrorMap[field] = append(fieldErrorMap[field], msg)
		}
		for _, internalCause := range cause.Causes {
			field = internalCause.InstanceLocation
			msg = internalCause.Message
			fieldErrorMap[field] = append(fieldErrorMap[field], msg)
		}
	}

	// If no causes but has error, use the main error
	if len(validationErr.Causes) == 0 {
		if field, msg := extractFieldAndError(validationErr.Error()); field != "" {
			fieldErrorMap[field] = append(fieldErrorMap[field], msg)
		}
	}

	fieldViolations := make([]*SchemaFieldViolation, 0, len(fieldErrorMap))
	for field, messages := range fieldErrorMap {
		fieldViolations = append(fieldViolations, &SchemaFieldViolation{
			Field:    field,
			Messages: messages,
		})
	}

	return fieldViolations
}

func extractFieldAndError(errorMessage string) (field, message string) {
	if !strings.Contains(errorMessage, "#/") {
		return "", errorMessage
	}

	parts := strings.Split(errorMessage, "#/")
	if len(parts) <= 1 {
		return "", errorMessage
	}

	fieldPath := strings.NewReplacer(
		"properties/", "",
		"meta/$ref/", "",
		"items/", "",
		"additionalProperties/", "",
	).Replace(parts[1])

	// Split into field and message if possible
	if idx := strings.Index(fieldPath, ":"); idx != -1 {
		field = fieldPath[:idx]
		message = strings.TrimSpace(fieldPath[idx+1:])
		return field, message
	}

	return fieldPath, parts[0]
}
