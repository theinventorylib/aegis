package routers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-chi/chi/v5"
	"github.com/labstack/echo/v4"

	"github.com/theinventorylib/aegis/core"
)

// helpers for reading response bodies

type response struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
	ID      string `json:"id,omitempty"`
	TeamID  string `json:"teamId,omitempty"`
	UserID  string `json:"userId,omitempty"`
	Page    int    `json:"page,omitempty"`
	Limit   int    `json:"limit,omitempty"`
}

func decodeResponse(body io.Reader) response {
	var r response
	json.NewDecoder(body).Decode(&r)
	return r
}

// =============================================================================
// Chi Router Tests
// =============================================================================

// TC-CHI-001: Single path parameter extraction via chi v5
func TestChiRouter_SinglePathParam(t *testing.T) {
	// Given
	mux := chi.NewRouter()
	router := NewChiRouter(mux)

	handler := func(w http.ResponseWriter, r *http.Request) {
		id := core.GetSanitizedPathParam(r, "id")
		core.WriteJSON(w, http.StatusOK, &core.Response{
			Success: true,
			Message: id,
		})
	}

	router.GET("/organizations/:id", handler)

	// When
	req := httptest.NewRequest(http.MethodGet, "/organizations/org_abc123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Then
	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}
	resp := decodeResponse(w.Body)
	if resp.Message != "org_abc123" {
		t.Errorf("id = %q, want %q", resp.Message, "org_abc123")
	}
}

// TC-CHI-002: Multiple path parameters extraction via chi v5
func TestChiRouter_MultiplePathParams(t *testing.T) {
	// Given
	mux := chi.NewRouter()
	router := NewChiRouter(mux)

	handler := func(w http.ResponseWriter, r *http.Request) {
		id := core.GetSanitizedPathParam(r, "id")
		teamID := core.GetSanitizedPathParam(r, "teamId")
		core.WriteJSON(w, http.StatusOK, response{
			ID:     id,
			TeamID: teamID,
		})
	}

	router.GET("/organizations/:id/teams/:teamId", handler)

	// When
	req := httptest.NewRequest(http.MethodGet, "/organizations/org_abc/teams/team_xyz", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Then
	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}
	resp := decodeResponse(w.Body)
	if resp.ID != "org_abc" {
		t.Errorf("id = %q, want %q", resp.ID, "org_abc")
	}
	if resp.TeamID != "team_xyz" {
		t.Errorf("teamId = %q, want %q", resp.TeamID, "team_xyz")
	}
}

// TC-CHI-003: Query parameter extraction
func TestChiRouter_QueryParams(t *testing.T) {
	// Given
	mux := chi.NewRouter()
	router := NewChiRouter(mux)

	handler := func(w http.ResponseWriter, r *http.Request) {
		pagination := core.ParsePagination(r)
		core.WriteJSON(w, http.StatusOK, response{
			Page:  pagination.Page,
			Limit: pagination.Limit,
		})
	}

	router.GET("/organizations", handler)

	// When
	req := httptest.NewRequest(http.MethodGet, "/organizations?page=3&limit=50", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Then
	resp := decodeResponse(w.Body)
	if resp.Page != 3 {
		t.Errorf("page = %d, want %d", resp.Page, 3)
	}
	if resp.Limit != 50 {
		t.Errorf("limit = %d, want %d", resp.Limit, 50)
	}
}

// TC-CHI-004: Path param + query param combined
func TestChiRouter_PathAndQueryParams(t *testing.T) {
	// Given
	mux := chi.NewRouter()
	router := NewChiRouter(mux)

	handler := func(w http.ResponseWriter, r *http.Request) {
		id := core.GetSanitizedPathParam(r, "id")
		pagination := core.ParsePagination(r)
		core.WriteJSON(w, http.StatusOK, response{
			ID:    id,
			Page:  pagination.Page,
			Limit: pagination.Limit,
		})
	}

	router.GET("/organizations/:id/members", handler)

	// When
	req := httptest.NewRequest(http.MethodGet, "/organizations/org_abc/members?page=2&limit=30", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Then
	resp := decodeResponse(w.Body)
	if resp.ID != "org_abc" {
		t.Errorf("id = %q, want %q", resp.ID, "org_abc")
	}
	if resp.Page != 2 {
		t.Errorf("page = %d, want %d", resp.Page, 2)
	}
	if resp.Limit != 30 {
		t.Errorf("limit = %d, want %d", resp.Limit, 30)
	}
}

// TC-CHI-005: Route group with path params
func TestChiRouter_GroupWithPathParams(t *testing.T) {
	// Given
	mux := chi.NewRouter()
	router := NewChiRouter(mux)

	orgGroup := router.Group("/organizations", "Organizations")
	membersGroup := orgGroup.Group("/:id/members", "Members")

	handler := func(w http.ResponseWriter, r *http.Request) {
		id := core.GetSanitizedPathParam(r, "id")
		userID := core.GetSanitizedPathParam(r, "userId")
		core.WriteJSON(w, http.StatusOK, response{
			ID:     id,
			UserID: userID,
		})
	}

	membersGroup.PATCH("/:userId", handler)

	// When
	req := httptest.NewRequest(http.MethodPatch, "/organizations/org_abc/members/usr_def", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Then
	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}
	resp := decodeResponse(w.Body)
	if resp.ID != "org_abc" {
		t.Errorf("id = %q, want %q", resp.ID, "org_abc")
	}
	if resp.UserID != "usr_def" {
		t.Errorf("userId = %q, want %q", resp.UserID, "usr_def")
	}
}

// TC-CHI-006: Multiple HTTP methods with path params
func TestChiRouter_MultipleMethodsWithPathParams(t *testing.T) {
	// Given
	mux := chi.NewRouter()
	router := NewChiRouter(mux)

	getHandler := func(w http.ResponseWriter, r *http.Request) {
		id := core.GetSanitizedPathParam(r, "id")
		core.WriteJSON(w, http.StatusOK, response{ID: id, Message: "GET"})
	}
	putHandler := func(w http.ResponseWriter, r *http.Request) {
		id := core.GetSanitizedPathParam(r, "id")
		core.WriteJSON(w, http.StatusOK, response{ID: id, Message: "PUT"})
	}
	deleteHandler := func(w http.ResponseWriter, r *http.Request) {
		id := core.GetSanitizedPathParam(r, "id")
		core.WriteJSON(w, http.StatusOK, response{ID: id, Message: "DELETE"})
	}

	router.GET("/organizations/:id", getHandler)
	router.PUT("/organizations/:id", putHandler)
	router.DELETE("/organizations/:id", deleteHandler)

	tests := []struct {
		method  string
		message string
	}{
		{method: http.MethodGet, message: "GET"},
		{method: http.MethodPut, message: "PUT"},
		{method: http.MethodDelete, message: "DELETE"},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/organizations/org_test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
			}
			resp := decodeResponse(w.Body)
			if resp.ID != "org_test" {
				t.Errorf("id = %q, want %q", resp.ID, "org_test")
			}
			if resp.Message != tt.message {
				t.Errorf("message = %q, want %q", resp.Message, tt.message)
			}
		})
	}
}

// =============================================================================
// Gin Router Tests
// =============================================================================

// TC-GIN-001: Path parameter extraction via Gin
func TestGinRouter_SinglePathParam(t *testing.T) {
	// Given
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	router := NewGinRouter(engine)

	handler := func(w http.ResponseWriter, r *http.Request) {
		id := core.GetSanitizedPathParam(r, "id")
		core.WriteJSON(w, http.StatusOK, &core.Response{
			Success: true,
			Message: id,
		})
	}

	router.GET("/organizations/:id", handler)

	// When
	req := httptest.NewRequest(http.MethodGet, "/organizations/org_abc123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Then
	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}
	resp := decodeResponse(w.Body)
	if resp.Message != "org_abc123" {
		t.Errorf("id = %q, want %q", resp.Message, "org_abc123")
	}
}

// TC-GIN-002: Multiple path parameters via Gin
func TestGinRouter_MultiplePathParams(t *testing.T) {
	// Given
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	router := NewGinRouter(engine)

	handler := func(w http.ResponseWriter, r *http.Request) {
		id := core.GetSanitizedPathParam(r, "id")
		teamID := core.GetSanitizedPathParam(r, "teamId")
		core.WriteJSON(w, http.StatusOK, response{
			ID:     id,
			TeamID: teamID,
		})
	}

	router.GET("/organizations/:id/teams/:teamId", handler)

	// When
	req := httptest.NewRequest(http.MethodGet, "/organizations/org_abc/teams/team_xyz", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Then
	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}
	resp := decodeResponse(w.Body)
	if resp.ID != "org_abc" {
		t.Errorf("id = %q, want %q", resp.ID, "org_abc")
	}
	if resp.TeamID != "team_xyz" {
		t.Errorf("teamId = %q, want %q", resp.TeamID, "team_xyz")
	}
}

// TC-GIN-003: Query parameter extraction via Gin
func TestGinRouter_QueryParams(t *testing.T) {
	// Given
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	router := NewGinRouter(engine)

	handler := func(w http.ResponseWriter, r *http.Request) {
		pagination := core.ParsePagination(r)
		core.WriteJSON(w, http.StatusOK, response{
			Page:  pagination.Page,
			Limit: pagination.Limit,
		})
	}

	router.GET("/organizations", handler)

	// When
	req := httptest.NewRequest(http.MethodGet, "/organizations?page=3&limit=50", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Then
	resp := decodeResponse(w.Body)
	if resp.Page != 3 {
		t.Errorf("page = %d, want %d", resp.Page, 3)
	}
	if resp.Limit != 50 {
		t.Errorf("limit = %d, want %d", resp.Limit, 50)
	}
}

// TC-GIN-004: Path param + query param combined via Gin
func TestGinRouter_PathAndQueryParams(t *testing.T) {
	// Given
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	router := NewGinRouter(engine)

	handler := func(w http.ResponseWriter, r *http.Request) {
		id := core.GetSanitizedPathParam(r, "id")
		pagination := core.ParsePagination(r)
		core.WriteJSON(w, http.StatusOK, response{
			ID:    id,
			Page:  pagination.Page,
			Limit: pagination.Limit,
		})
	}

	router.GET("/organizations/:id/members", handler)

	// When
	req := httptest.NewRequest(http.MethodGet, "/organizations/org_abc/members?page=2&limit=30", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Then
	resp := decodeResponse(w.Body)
	if resp.ID != "org_abc" {
		t.Errorf("id = %q, want %q", resp.ID, "org_abc")
	}
	if resp.Page != 2 {
		t.Errorf("page = %d, want %d", resp.Page, 2)
	}
	if resp.Limit != 30 {
		t.Errorf("limit = %d, want %d", resp.Limit, 30)
	}
}

// TC-GIN-005: Route group with path params via Gin
func TestGinRouter_GroupWithPathParams(t *testing.T) {
	// Given
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	router := NewGinRouter(engine)

	orgGroup := router.Group("/organizations", "Organizations")
	membersGroup := orgGroup.Group("/:id/members", "Members")

	handler := func(w http.ResponseWriter, r *http.Request) {
		id := core.GetSanitizedPathParam(r, "id")
		userID := core.GetSanitizedPathParam(r, "userId")
		core.WriteJSON(w, http.StatusOK, response{
			ID:     id,
			UserID: userID,
		})
	}

	membersGroup.PATCH("/:userId", handler)

	// When
	req := httptest.NewRequest(http.MethodPatch, "/organizations/org_abc/members/usr_def", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Then
	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}
	resp := decodeResponse(w.Body)
	if resp.ID != "org_abc" {
		t.Errorf("id = %q, want %q", resp.ID, "org_abc")
	}
	if resp.UserID != "usr_def" {
		t.Errorf("userId = %q, want %q", resp.UserID, "usr_def")
	}
}

// =============================================================================
// Echo Router Tests
// =============================================================================

// TC-ECHO-001: Path parameter extraction via Echo
func TestEchoRouter_SinglePathParam(t *testing.T) {
	// Given
	e := echo.New()
	router := NewEchoRouter(e)

	handler := func(w http.ResponseWriter, r *http.Request) {
		id := core.GetSanitizedPathParam(r, "id")
		core.WriteJSON(w, http.StatusOK, &core.Response{
			Success: true,
			Message: id,
		})
	}

	router.GET("/organizations/:id", handler)

	// When
	req := httptest.NewRequest(http.MethodGet, "/organizations/org_abc123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Then
	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}
	resp := decodeResponse(w.Body)
	if resp.Message != "org_abc123" {
		t.Errorf("id = %q, want %q", resp.Message, "org_abc123")
	}
}

// TC-ECHO-002: Multiple path parameters via Echo
func TestEchoRouter_MultiplePathParams(t *testing.T) {
	// Given
	e := echo.New()
	router := NewEchoRouter(e)

	handler := func(w http.ResponseWriter, r *http.Request) {
		id := core.GetSanitizedPathParam(r, "id")
		teamID := core.GetSanitizedPathParam(r, "teamId")
		core.WriteJSON(w, http.StatusOK, response{
			ID:     id,
			TeamID: teamID,
		})
	}

	router.GET("/organizations/:id/teams/:teamId", handler)

	// When
	req := httptest.NewRequest(http.MethodGet, "/organizations/org_abc/teams/team_xyz", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Then
	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}
	resp := decodeResponse(w.Body)
	if resp.ID != "org_abc" {
		t.Errorf("id = %q, want %q", resp.ID, "org_abc")
	}
	if resp.TeamID != "team_xyz" {
		t.Errorf("teamId = %q, want %q", resp.TeamID, "team_xyz")
	}
}

// TC-ECHO-003: Query parameter extraction via Echo
func TestEchoRouter_QueryParams(t *testing.T) {
	// Given
	e := echo.New()
	router := NewEchoRouter(e)

	handler := func(w http.ResponseWriter, r *http.Request) {
		pagination := core.ParsePagination(r)
		core.WriteJSON(w, http.StatusOK, response{
			Page:  pagination.Page,
			Limit: pagination.Limit,
		})
	}

	router.GET("/organizations", handler)

	// When
	req := httptest.NewRequest(http.MethodGet, "/organizations?page=3&limit=50", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Then
	resp := decodeResponse(w.Body)
	if resp.Page != 3 {
		t.Errorf("page = %d, want %d", resp.Page, 3)
	}
	if resp.Limit != 50 {
		t.Errorf("limit = %d, want %d", resp.Limit, 50)
	}
}

// TC-ECHO-004: Path param + query param combined via Echo
func TestEchoRouter_PathAndQueryParams(t *testing.T) {
	// Given
	e := echo.New()
	router := NewEchoRouter(e)

	handler := func(w http.ResponseWriter, r *http.Request) {
		id := core.GetSanitizedPathParam(r, "id")
		pagination := core.ParsePagination(r)
		core.WriteJSON(w, http.StatusOK, response{
			ID:    id,
			Page:  pagination.Page,
			Limit: pagination.Limit,
		})
	}

	router.GET("/organizations/:id/members", handler)

	// When
	req := httptest.NewRequest(http.MethodGet, "/organizations/org_abc/members?page=2&limit=30", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Then
	resp := decodeResponse(w.Body)
	if resp.ID != "org_abc" {
		t.Errorf("id = %q, want %q", resp.ID, "org_abc")
	}
	if resp.Page != 2 {
		t.Errorf("page = %d, want %d", resp.Page, 2)
	}
	if resp.Limit != 30 {
		t.Errorf("limit = %d, want %d", resp.Limit, 30)
	}
}

// TC-ECHO-005: Route group with path params via Echo
func TestEchoRouter_GroupWithPathParams(t *testing.T) {
	// Given
	e := echo.New()
	router := NewEchoRouter(e)

	orgGroup := router.Group("/organizations", "Organizations")
	membersGroup := orgGroup.Group("/:id/members", "Members")

	handler := func(w http.ResponseWriter, r *http.Request) {
		id := core.GetSanitizedPathParam(r, "id")
		userID := core.GetSanitizedPathParam(r, "userId")
		core.WriteJSON(w, http.StatusOK, response{
			ID:     id,
			UserID: userID,
		})
	}

	membersGroup.PATCH("/:userId", handler)

	// When
	req := httptest.NewRequest(http.MethodPatch, "/organizations/org_abc/members/usr_def", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Then
	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}
	resp := decodeResponse(w.Body)
	if resp.ID != "org_abc" {
		t.Errorf("id = %q, want %q", resp.ID, "org_abc")
	}
	if resp.UserID != "usr_def" {
		t.Errorf("userId = %q, want %q", resp.UserID, "usr_def")
	}
}

// =============================================================================
// NormalizePathToOpenAPI Integration Tests (Chi route registration)
// =============================================================================

// TC-REG-001: Chi adapter converts :param to {param} for route registration
func TestChiRouter_ParamSyntaxConversion(t *testing.T) {
	tests := []struct {
		name          string
		registerPath  string
		requestPath   string
		method        string
		paramName     string
		expectedValue string
	}{
		{
			name:          "single param :id",
			registerPath:  "/organizations/:id",
			requestPath:   "/organizations/org_123",
			method:        http.MethodGet,
			paramName:     "id",
			expectedValue: "org_123",
		},
		{
			name:          "nested params",
			registerPath:  "/organizations/:id/teams/:teamId",
			requestPath:   "/organizations/org_abc/teams/team_xyz",
			method:        http.MethodGet,
			paramName:     "teamId",
			expectedValue: "team_xyz",
		},
		{
			name:          "member userId param",
			registerPath:  "/organizations/:id/members/:userId",
			requestPath:   "/organizations/org_1/members/usr_2",
			method:        http.MethodPatch,
			paramName:     "userId",
			expectedValue: "usr_2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mux := chi.NewRouter()
			router := NewChiRouter(mux)

			handler := func(w http.ResponseWriter, r *http.Request) {
				val := core.GetSanitizedPathParam(r, tt.paramName)
				core.WriteJSON(w, http.StatusOK, response{Message: val})
			}

			switch tt.method {
			case http.MethodGet:
				router.GET(tt.registerPath, handler)
			case http.MethodPost:
				router.POST(tt.registerPath, handler)
			case http.MethodPut:
				router.PUT(tt.registerPath, handler)
			case http.MethodPatch:
				router.PATCH(tt.registerPath, handler)
			case http.MethodDelete:
				router.DELETE(tt.registerPath, handler)
			}

			// When
			switch tt.method {
			case http.MethodPatch:
				req := httptest.NewRequest(tt.method, tt.requestPath, nil)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				// Also need to set content-type for some frameworks
				_ = w
			default:
				req := httptest.NewRequest(tt.method, tt.requestPath, nil)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				_ = w
			}

			req := httptest.NewRequest(tt.method, tt.requestPath, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Then
			if w.Code != http.StatusOK {
				t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
			}
			resp := decodeResponse(w.Body)
			if resp.Message != tt.expectedValue {
				t.Errorf("param %q = %q, want %q", tt.paramName, resp.Message, tt.expectedValue)
			}
		})
	}
}

// TC-REG-002: Chi group route with :param syntax converts correctly
func TestChiRouter_GroupParamConversion(t *testing.T) {
	// Given
	mux := chi.NewRouter()
	router := NewChiRouter(mux)

	orgGroup := router.Group("/organizations", "Organizations")
	teamGroup := orgGroup.Group("/teams", "Teams")

	handler := func(w http.ResponseWriter, r *http.Request) {
		teamID := core.GetSanitizedPathParam(r, "teamId")
		core.WriteJSON(w, http.StatusOK, response{TeamID: teamID})
	}

	teamGroup.GET("/:teamId", handler)

	// When
	req := httptest.NewRequest(http.MethodGet, "/organizations/teams/team_test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Then
	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}
	resp := decodeResponse(w.Body)
	if resp.TeamID != "team_test" {
		t.Errorf("teamId = %q, want %q", resp.TeamID, "team_test")
	}
}

// =============================================================================
// Full organizations-like route simulation via Chi
// =============================================================================

// TC-ORG-001: Simulated organizations plugin routes via Chi
func TestChiRouter_OrganizationsRouteSimulation(t *testing.T) {
	// Given
	mux := chi.NewRouter()
	router := NewChiRouter(mux)
	prefix := "/auth/organizations"

	// Simulate the organizations plugin route structure
	orgGroup := router.Group(prefix, "Organizations")

	orgGroup.GET("/:id", func(w http.ResponseWriter, r *http.Request) {
		id := core.GetSanitizedPathParam(r, "id")
		core.WriteJSON(w, http.StatusOK, response{ID: id, Message: "get_org"})
	})
	orgGroup.PUT("/:id", func(w http.ResponseWriter, r *http.Request) {
		id := core.GetSanitizedPathParam(r, "id")
		core.WriteJSON(w, http.StatusOK, response{ID: id, Message: "update_org"})
	})
	orgGroup.DELETE("/:id", func(w http.ResponseWriter, r *http.Request) {
		id := core.GetSanitizedPathParam(r, "id")
		core.WriteJSON(w, http.StatusOK, response{ID: id, Message: "delete_org"})
	})

	membersGroup := orgGroup.Group("/:id/members", "Members")
	membersGroup.GET("/", func(w http.ResponseWriter, r *http.Request) {
		id := core.GetSanitizedPathParam(r, "id")
		pagination := core.ParsePagination(r)
		core.WriteJSON(w, http.StatusOK, response{ID: id, Page: pagination.Page, Limit: pagination.Limit, Message: "list_members"})
	})
	membersGroup.PATCH("/:userId", func(w http.ResponseWriter, r *http.Request) {
		id := core.GetSanitizedPathParam(r, "id")
		userID := core.GetSanitizedPathParam(r, "userId")
		core.WriteJSON(w, http.StatusOK, response{ID: id, UserID: userID, Message: "update_member"})
	})

	teamGroup := orgGroup.Group("/teams", "Teams")
	teamGroup.GET("/:teamId", func(w http.ResponseWriter, r *http.Request) {
		teamID := core.GetSanitizedPathParam(r, "teamId")
		core.WriteJSON(w, http.StatusOK, response{TeamID: teamID, Message: "get_team"})
	})

	tests := []struct {
		name           string
		method         string
		path           string
		expectedID     string
		expectedTeamID string
		expectedUserID string
		expectedPage   int
		expectedLimit  int
		expectedMsg    string
	}{
		{
			name:   "GET org by id",
			method: http.MethodGet, path: "/auth/organizations/org_1",
			expectedID: "org_1", expectedMsg: "get_org",
		},
		{
			name:   "PUT org",
			method: http.MethodPut, path: "/auth/organizations/org_2",
			expectedID: "org_2", expectedMsg: "update_org",
		},
		{
			name:   "DELETE org",
			method: http.MethodDelete, path: "/auth/organizations/org_3",
			expectedID: "org_3", expectedMsg: "delete_org",
		},
		{
			name:   "GET members with pagination",
			method: http.MethodGet, path: "/auth/organizations/org_4/members?page=5&limit=10",
			expectedID: "org_4", expectedPage: 5, expectedLimit: 10, expectedMsg: "list_members",
		},
		{
			name:   "PATCH member role",
			method: http.MethodPatch, path: "/auth/organizations/org_5/members/usr_6",
			expectedID: "org_5", expectedUserID: "usr_6", expectedMsg: "update_member",
		},
		{
			name:   "GET team by id",
			method: http.MethodGet, path: "/auth/organizations/teams/team_7",
			expectedTeamID: "team_7", expectedMsg: "get_team",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("Status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
			}

			resp := decodeResponse(w.Body)

			if tt.expectedID != "" && resp.ID != tt.expectedID {
				t.Errorf("id = %q, want %q", resp.ID, tt.expectedID)
			}
			if tt.expectedTeamID != "" && resp.TeamID != tt.expectedTeamID {
				t.Errorf("teamId = %q, want %q", resp.TeamID, tt.expectedTeamID)
			}
			if tt.expectedUserID != "" && resp.UserID != tt.expectedUserID {
				t.Errorf("userId = %q, want %q", resp.UserID, tt.expectedUserID)
			}
			if tt.expectedPage != 0 && resp.Page != tt.expectedPage {
				t.Errorf("page = %d, want %d", resp.Page, tt.expectedPage)
			}
			if tt.expectedLimit != 0 && resp.Limit != tt.expectedLimit {
				t.Errorf("limit = %d, want %d", resp.Limit, tt.expectedLimit)
			}
			if resp.Message != tt.expectedMsg {
				t.Errorf("message = %q, want %q", resp.Message, fmt.Sprintf(tt.expectedMsg, ""))
			}
		})
	}
}
