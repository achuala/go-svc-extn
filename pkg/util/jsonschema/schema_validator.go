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

// SchemaFieldViolation represents a validation error for a specific field in the JSON schema.
// It contains the field path and a list of error messages associated with that field.
type SchemaFieldViolation struct {
	Field    string   // The path to the field that failed validation
	Messages []string // List of validation error messages for this field
}

// validateWithSchema validates a JSON object against a JSON schema.
// It returns a list of field violations if validation fails, or nil if validation succeeds.
// The error returned contains the original validation error for debugging purposes.
//
// Parameters:
//   - schema: The JSON schema to validate against
//   - jsonObject: The JSON object to validate
//
// Returns:
//   - []*SchemaFieldViolation: List of field violations if validation fails
//   - error: The original validation error if validation fails, nil if validation succeeds
func validateWithSchema(schema *jsonschema.Schema, jsonObject any) ([]*SchemaFieldViolation, error) {
	if err := schema.Validate(jsonObject); err != nil {
		if validationErr, ok := err.(*jsonschema.ValidationError); ok {
			validationErrors := mapSchemaValidationErrors(validationErr)
			if len(validationErrors) > 0 {
				return validationErrors, fmt.Errorf("validation failed: %w", err)
			}
		}
		return nil, err
	}
	return nil, nil
}

// mapSchemaValidationErrors processes a ValidationError and converts it into a list of field violations.
// It handles both the main validation error and its nested causes, ensuring all validation errors are captured.
//
// Parameters:
//   - validationErr: The validation error to process
//
// Returns:
//   - []*SchemaFieldViolation: List of field violations extracted from the validation error
func mapSchemaValidationErrors(validationErr *jsonschema.ValidationError) []*SchemaFieldViolation {
	if validationErr == nil {
		return nil
	}

	fieldErrorMap := make(map[string][]string)

	// Process all causes to collect detailed errors
	for _, cause := range validationErr.Causes {
		processValidationCause(cause, fieldErrorMap)
	}

	// Process the main error if it has a message
	if validationErr.Message != "" {
		field := normalizeFieldPath(validationErr.InstanceLocation)
		if field != "" {
			fieldErrorMap[field] = append(fieldErrorMap[field], validationErr.Message)
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

// processValidationCause processes a single validation error cause and its nested causes.
// It uses a queue-based approach to handle nested causes iteratively, preventing stack overflow
// with deeply nested validation errors.
//
// Parameters:
//   - cause: The validation error cause to process
//   - fieldErrorMap: Map to store field violations
func processValidationCause(cause *jsonschema.ValidationError, fieldErrorMap map[string][]string) {
	if cause == nil {
		return
	}

	// Process current cause
	if cause.Message != "" {
		field := normalizeFieldPath(cause.InstanceLocation)
		if field != "" {
			fieldErrorMap[field] = append(fieldErrorMap[field], cause.Message)
		}
	}

	// Process nested causes using a queue-based approach
	causesQueue := cause.Causes
	for len(causesQueue) > 0 {
		current := causesQueue[0]
		causesQueue = causesQueue[1:]

		if current.Message != "" {
			field := normalizeFieldPath(current.InstanceLocation)
			if field != "" {
				fieldErrorMap[field] = append(fieldErrorMap[field], current.Message)
			}
		}

		// Add any nested causes to the queue
		if len(current.Causes) > 0 {
			causesQueue = append(causesQueue, current.Causes...)
		}
	}
}

// normalizeFieldPath normalizes a JSON Schema field path by removing common prefixes and formatting.
// It makes field paths more readable and consistent across different types of validation errors.
//
// Parameters:
//   - path: The original field path from the validation error
//
// Returns:
//   - string: The normalized field path
func normalizeFieldPath(path string) string {
	if path == "" {
		return ""
	}

	// Remove common JSON Schema path prefixes
	path = strings.NewReplacer(
		"properties/", "",
		"meta/$ref/", "",
		"items/", "",
		"additionalProperties/", "",
	).Replace(path)

	// Remove leading slash if present
	if strings.HasPrefix(path, "/") {
		path = path[1:]
	}

	return path
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
