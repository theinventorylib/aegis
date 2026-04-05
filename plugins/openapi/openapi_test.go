package openapi

import (
	"testing"
)

type TestStruct struct {
	Name string `json:"name" validate:"required"`
	Age  int    `json:"age"`
}

type InviteRequest struct {
	Email   string `json:"email" validation:"Required"`
	Message string `json:"message,omitempty"`
}

type InviteResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func TestRegisterSchemaFromType(t *testing.T) {
	resetQueue()
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

func TestDocPendingQueue(t *testing.T) {
	resetQueue()

	// Doc calls before plugin initialization should be buffered
	Doc(Route{
		Method:  "POST",
		Path:    "/auth/login",
		Summary: "Login",
		Tags:    []string{"Auth"},
	})

	Doc(Route{
		Method:  "GET",
		Path:    "/api/users/{id}",
		Summary: "Get user",
		Tags:    []string{"Users"},
		Auth:    true,
		Params: []Param{
			{Name: "id", In: "path", Type: "string"},
		},
		Responses: Responses{
			200: TextResponse("User found"),
			404: TextResponse("User not found"),
		},
	})

	// Create and initialize plugin — this drains the queue
	p := New(nil)
	drainPending(p)

	spec := p.GetSpec()

	// Verify login route was registered
	if _, ok := spec.Paths["/auth/login"]; !ok {
		t.Error("Login route should have been registered from queue")
	}

	// Verify user route was registered
	userPath, ok := spec.Paths["/api/users/{id}"]
	if !ok {
		t.Fatal("User route should have been registered from queue")
	}

	if userPath.Get == nil {
		t.Fatal("User GET operation should exist")
	}

	// Verify security was set for Auth: true
	if len(userPath.Get.Security) == 0 {
		t.Error("User GET should have security requirements")
	}
}

func TestDocAfterInit(t *testing.T) {
	resetQueue()

	// Initialize plugin first
	p := New(nil)
	drainPending(p)

	// Doc calls after initialization should register immediately
	Doc(Route{
		Method:  "DELETE",
		Path:    "/api/items/{id}",
		Summary: "Delete item",
		Tags:    []string{"Items"},
		Auth:    true,
		Params: []Param{
			{Name: "id", In: "path", Type: "string", Required: true},
		},
		Responses: Responses{
			204: TextResponse("Item deleted"),
		},
	})

	spec := p.GetSpec()

	if _, ok := spec.Paths["/api/items/{id}"]; !ok {
		t.Error("Item route should have been registered immediately")
	}
}

func TestDeriveOperationID(t *testing.T) {
	tests := []struct {
		method   string
		path     string
		expected string
	}{
		{"POST", "/auth/email/signup", "postAuthEmailSignup"},
		{"GET", "/api/users/{id}", "getApiUsersById"},
		{"DELETE", "/api/users/{id}", "deleteApiUsersById"},
		{"GET", "/api/dashboard", "getApiDashboard"},
		{"GET", "/users/:id", "getUsersById"},
		{"POST", "/", "post"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			result := deriveOperationID(tt.method, tt.path)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestPathParamForcedRequired(t *testing.T) {
	resetQueue()

	p := New(nil)
	drainPending(p)

	Doc(Route{
		Method:  "GET",
		Path:    "/users/{id}",
		Summary: "Get user",
		Params: []Param{
			{Name: "id", In: "path", Type: "string", Required: false}, // Caller sets false
		},
		Responses: Responses{
			200: TextResponse("OK"),
		},
	})

	spec := p.GetSpec()
	pathItem, ok := spec.Paths["/users/{id}"]
	if !ok {
		t.Fatal("Path should exist")
	}

	if pathItem.Get == nil {
		t.Fatal("GET operation should exist")
	}

	if len(pathItem.Get.Parameters) == 0 {
		t.Fatal("Should have parameters")
	}

	param := pathItem.Get.Parameters[0]
	if !param.Required {
		t.Error("Path parameter 'id' should be forced to required: true")
	}
}

func TestBodyOfAndResponseOf(t *testing.T) {
	resetQueue()

	p := New(nil)
	drainPending(p)

	Doc(Route{
		Method:  "POST",
		Path:    "/invite",
		Summary: "Invite user",
		Body:    BodyOf[InviteRequest](),
		Responses: Responses{
			200: ResponseOf[InviteResponse]("Invitation sent"),
			404: TextResponse("User not found"),
		},
	})

	spec := p.GetSpec()

	// Verify request body schema was registered
	if _, ok := spec.Components.Schemas["InviteRequest"]; !ok {
		t.Error("InviteRequest schema should be registered in components")
	}

	// Verify response schema was registered
	if _, ok := spec.Components.Schemas["InviteResponse"]; !ok {
		t.Error("InviteResponse schema should be registered in components")
	}

	// Verify the path exists
	pathItem, ok := spec.Paths["/invite"]
	if !ok {
		t.Fatal("Path should exist")
	}

	if pathItem.Post == nil {
		t.Fatal("POST operation should exist")
	}

	if pathItem.Post.RequestBody == nil {
		t.Fatal("Request body should exist")
	}

	// Verify 404 response has no content (Text response)
	resp404 := pathItem.Post.Responses["404"]
	if resp404 == nil {
		t.Fatal("404 response should exist")
		return
	}
	if resp404.Content != nil {
		t.Error("Text response should have no content")
	}
	if resp404.Description != "User not found" {
		t.Errorf("Expected description 'User not found', got %q", resp404.Description)
	}
}

func TestNormalizePathParams(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/users/:id", "/users/{id}"},
		{"/orgs/:orgId/members/:userId", "/orgs/{orgId}/members/{userId}"},
		{"/static/file.css", "/static/file.css"},
		{"/users/{id}", "/users/{id}"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizePathParams(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestAutoTagRegistration(t *testing.T) {
	resetQueue()

	p := New(nil)
	drainPending(p)

	Doc(Route{
		Method:  "GET",
		Path:    "/items",
		Summary: "List items",
		Tags:    []string{"CustomTag"},
	})

	spec := p.GetSpec()

	found := false
	for _, tag := range spec.Tags {
		if tag.Name == "CustomTag" {
			found = true
			break
		}
	}
	if !found {
		t.Error("CustomTag should have been auto-registered")
	}
}

func TestOperationIDExplicit(t *testing.T) {
	resetQueue()

	p := New(nil)
	drainPending(p)

	Doc(Route{
		Method:      "POST",
		Path:        "/auth/email/signup",
		OperationID: "signUpViaEmail",
		Summary:     "Sign up",
	})

	spec := p.GetSpec()
	pathItem := spec.Paths["/auth/email/signup"]
	if pathItem == nil || pathItem.Post == nil {
		t.Fatal("Path should exist")
	}

	if pathItem.Post.OperationID != "signUpViaEmail" {
		t.Errorf("Expected explicit operationId 'signUpViaEmail', got %q", pathItem.Post.OperationID)
	}
}
