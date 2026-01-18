package openapi

import (
	"testing"

	"github.com/theinventorylib/aegis/core"
)

type TestStruct struct {
	Name string `json:"name" validate:"required"`
	Age  int    `json:"age"`
}

func TestRegisterSchemaFromType(t *testing.T) {
	p := New(nil)
	p.RegisterSchemaFromType("TestSchema", TestStruct{})

	spec := p.GetSpec()
	if spec.Components == nil {
		t.Fatal("Components should not be nil")
	}
	if spec.Components.Schemas == nil {
		t.Fatal("Schemas should not be nil")
	}

	schema, ok := spec.Components.Schemas["TestSchema"]
	if !ok {
		t.Fatal("TestSchema not found in components")
	}

	if schema.Type != "object" {
		t.Errorf("Expected type object, got %s", schema.Type)
	}

	if _, ok := schema.Properties["name"]; !ok {
		t.Error("Property 'name' not found in schema")
	}

	if _, ok := schema.Properties["age"]; !ok {
		t.Error("Property 'age' not found in schema")
	}

	foundRequired := false
	for _, r := range schema.Required {
		if r == "name" {
			foundRequired = true
			break
		}
	}
	if !foundRequired {
		t.Error("Property 'name' should be required")
	}
}

func TestUpdateSpecExcludesOpenAPIRoutes(t *testing.T) {
	p := New(nil)

	metadata := []core.RouteMetadata{
		{
			Method:  "GET",
			Path:    "/auth/openapi.json",
			Tags:    []string{"OpenAPI"},
			Summary: "Should be skipped",
		},
		{
			Method:  "POST",
			Path:    "/auth/login",
			Tags:    []string{"Authentication"},
			Summary: "Should be included",
		},
	}

	p.UpdateSpec(metadata)
	spec := p.GetSpec()

	if _, ok := spec.Paths["/auth/openapi.json"]; ok {
		t.Error("OpenAPI route should have been skipped")
	}

	if _, ok := spec.Paths["/auth/login"]; !ok {
		t.Error("Login route should have been included")
	}
}
