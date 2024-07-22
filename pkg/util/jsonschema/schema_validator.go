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
	schemas          map[string]*jsonschema.Schema
	schemaUniqueKeys map[string][]string
}

func NewJsonSchemaValidator(schemaDirectory string) (*JsonSchemaValidator, error) {
	files, err := os.ReadDir(schemaDirectory)
	if err != nil {
		return nil, fmt.Errorf("error reading schema directory: %w", err)
	}
	c := jsonschema.NewCompiler()
	schemaUniqueKeys := make(map[string][]string, 0)
	var schemaIds []string
	for _, f := range files {
		fname := filepath.Join(schemaDirectory, f.Name())
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
		if uk, ok := jsonElems["uniqueKeys"].([]interface{}); ok {
			if uniqueKeys, err := convertInterfaceSliceToStringSlice(uk); err == nil {
				if len(uniqueKeys) > 0 {
					schemaUniqueKeys[schemaId] = uniqueKeys
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
	return &JsonSchemaValidator{schemas: compiledSchemas, schemaUniqueKeys: schemaUniqueKeys}, nil
}

func (v *JsonSchemaValidator) ValidateJson(schemaId string, jsonObject any) error {
	schema := v.schemas[schemaId]
	if schema == nil {
		return errors.New("invalid schema id " + schemaId)
	}
	if mapData, ok := jsonObject.(map[string]string); ok {
		v, err := convertMapToAny(mapData)
		if err != nil {
			return fmt.Errorf("unable to convert map to json: %w", err)
		}
		return schema.Validate(v)
	}
	return schema.Validate(jsonObject)
}

// New function for validating map with generic type parameter
func ValidateMap[T any](schema *jsonschema.Schema, data map[string]T) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error marshaling data to json: %w", err)
	}

	var jsonObject interface{}
	if err := json.Unmarshal(jsonData, &jsonObject); err != nil {
		return fmt.Errorf("error unmarshaling json data: %w", err)
	}

	return schema.Validate(jsonObject)
}

func (v *JsonSchemaValidator) ValidateMap(schemaId string, data map[string]any) error {
	schema := v.schemas[schemaId]
	if schema == nil {
		return errors.New("invalid schema id " + schemaId)
	}

	return ValidateMap(schema, data)
}

func (v *JsonSchemaValidator) GetUniqueKeys(schemaId string) ([]string, error) {
	schemaUniqueKeys := v.schemaUniqueKeys[schemaId]
	if schemaUniqueKeys == nil {
		return nil, errors.New("invalid schema id " + schemaId)
	}
	return schemaUniqueKeys, nil
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

// ConvertInterfaceSliceToStringSlice converts a slice of interface{} to a slice of string
func convertInterfaceSliceToStringSlice(input []interface{}) ([]string, error) {
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
