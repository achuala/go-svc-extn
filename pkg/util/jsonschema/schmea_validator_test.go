package jsonschema_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/achuala/go-svc-extn/pkg/util/jsonschema"
)

func createTestSchemaFiles(dir string, t *testing.T) {
	schemas := map[string]string{
		"schema1.json": `{
			"id": "http://example.com/schema1",
			"type": "object",
			"properties": {
				"name": {"type": "string"},
				"age": {"type": "integer"}
			},
			"required": ["name"]
		}`,
		"schema2.json": `{
			"id": "http://example.com/schema2",
			"type": "object",
			"properties": {
				"title": {"type": "string"},
				"content": {"type": "string"}
			},
			"required": ["title"],
			"uniqueKeys": ["title"]
		}`,
	}

	for fname, content := range schemas {
		fpath := filepath.Join(dir, fname)
		err := os.WriteFile(fpath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to create schema file : %v", err)
		}
	}
}

func TestNewJsonSchemaValidator(t *testing.T) {
	tempDir := t.TempDir()
	createTestSchemaFiles(tempDir, t)

	_, err := jsonschema.NewJsonSchemaValidator(tempDir)
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}

}

func TestValidateJson(t *testing.T) {
	tempDir := t.TempDir()
	createTestSchemaFiles(tempDir, t)

	validator, err := jsonschema.NewJsonSchemaValidator(tempDir)
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}

	validJson := map[string]any{
		"name": "John Doe",
		"age":  30,
	}

	err = validator.ValidateJson("http://example.com/schema1", validJson)
	if err != nil {
		t.Errorf("expected valid JSON to pass validation, got error: %v", err)
	}

	invalidJson := map[string]any{
		"age": 30,
	}

	err = validator.ValidateJson("http://example.com/schema1", invalidJson)
	if err == nil {
		t.Errorf("expected invalid JSON to fail validation, got no error")
	}
}

func TestValidateMap(t *testing.T) {
	tempDir := t.TempDir()
	createTestSchemaFiles(tempDir, t)

	validator, err := jsonschema.NewJsonSchemaValidator(tempDir)
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}

	validMap := map[string]any{
		"name": "John Doe",
		"age":  "30",
	}

	err = validator.ValidateMap("http://example.com/schema1", validMap)
	if err != nil {
		t.Errorf("expected valid map to pass validation, got error: %v", err)
	}

	invalidMap := map[string]any{
		"age": "30",
	}

	err = validator.ValidateMap("http://example.com/schema1", invalidMap)
	if err == nil {
		t.Errorf("expected invalid map to fail validation, got no error")
	}
}

func TestGetUniqueKeys(t *testing.T) {
	tempDir := t.TempDir()
	createTestSchemaFiles(tempDir, t)

	validator, err := jsonschema.NewJsonSchemaValidator(tempDir)
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}

	uniqueKeys, err := validator.GetUniqueKeys("http://example.com/schema2")
	if err != nil {
		t.Errorf("expected to get unique keys, got error: %v", err)
	}

	expectedUniqueKeys := []string{"title"}
	if !equalStringSlices(uniqueKeys, expectedUniqueKeys) {
		t.Errorf("expected unique keys %v, got %v", expectedUniqueKeys, uniqueKeys)
	}

	_, err = validator.GetUniqueKeys("http://example.com/schema1")
	if err == nil {
		t.Errorf("expected error for schema without unique keys, got no error")
	}
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
