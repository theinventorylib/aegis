package aegis

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/theinventorylib/aegis/v2/config"
	"github.com/theinventorylib/aegis/v2/plugins"
	"github.com/theinventorylib/aegis/v2/plugins/openapi"
	"github.com/theinventorylib/aegis/v2/router"
	"github.com/theinventorylib/aegis/v2/router/routers"
)

// ── fake SQL driver ─────────────────────────────────────────────────────────
//
// Aegis only needs a non-nil *sql.DB at construction time; no query is run
// during New or MountRoutes. This driver lets the tests build a fully wired
// Aegis instance without an external database.

type mountTestDriver struct{}

func (mountTestDriver) Open(string) (driver.Conn, error) { return mountTestConn{}, nil }

type mountTestConn struct{}

func (mountTestConn) Prepare(string) (driver.Stmt, error) { return nil, driver.ErrSkip }
func (mountTestConn) Close() error                        { return nil }
func (mountTestConn) Begin() (driver.Tx, error)           { return nil, driver.ErrSkip }

func init() { sql.Register("aegis-mount-test", mountTestDriver{}) }

// ── stub router ─────────────────────────────────────────────────────────────

// mountRecordingRouter records every (method, path) registered on it,
// mirroring the path semantics of the real adapters via router.JoinPath.
type mountRecordingRouter struct {
	registered []string
}

func (r *mountRecordingRouter) record(method, path string) {
	r.registered = append(r.registered, method+" "+path)
}

func (r *mountRecordingRouter) GET(path string, _ http.HandlerFunc) {
	r.record(http.MethodGet, path)
}
func (r *mountRecordingRouter) POST(path string, _ http.HandlerFunc) {
	r.record(http.MethodPost, path)
}
func (r *mountRecordingRouter) PUT(path string, _ http.HandlerFunc) {
	r.record(http.MethodPut, path)
}
func (r *mountRecordingRouter) PATCH(path string, _ http.HandlerFunc) {
	r.record(http.MethodPatch, path)
}
func (r *mountRecordingRouter) DELETE(path string, _ http.HandlerFunc) {
	r.record(http.MethodDelete, path)
}
func (r *mountRecordingRouter) Use(func(http.Handler) http.Handler)          {}
func (r *mountRecordingRouter) ServeHTTP(http.ResponseWriter, *http.Request) {}
func (r *mountRecordingRouter) Group(path, _ string) router.GroupRouter {
	return &mountRecordingGroup{parent: r, prefix: path}
}

func (r *mountRecordingRouter) paths() []string {
	out := append([]string(nil), r.registered...)
	sort.Strings(out)
	return out
}

type mountRecordingGroup struct {
	parent *mountRecordingRouter
	prefix string
}

func (g *mountRecordingGroup) full(path string) string { return router.JoinPath(g.prefix, path) }

func (g *mountRecordingGroup) GET(path string, h http.HandlerFunc)   { g.parent.GET(g.full(path), h) }
func (g *mountRecordingGroup) POST(path string, h http.HandlerFunc)  { g.parent.POST(g.full(path), h) }
func (g *mountRecordingGroup) PUT(path string, h http.HandlerFunc)   { g.parent.PUT(g.full(path), h) }
func (g *mountRecordingGroup) PATCH(path string, h http.HandlerFunc) { g.parent.PATCH(g.full(path), h) }
func (g *mountRecordingGroup) DELETE(path string, h http.HandlerFunc) {
	g.parent.DELETE(g.full(path), h)
}
func (g *mountRecordingGroup) Use(func(http.Handler) http.Handler) {}
func (g *mountRecordingGroup) Group(path, _ string) router.GroupRouter {
	return &mountRecordingGroup{parent: g.parent, prefix: g.full(path)}
}

// ── fake plugin ─────────────────────────────────────────────────────────────

// mountTestPlugin mimics the real plugins: it mounts a grouped route and can
// document it through openapi.Doc.
type mountTestPlugin struct {
	name     string
	document bool
}

func (p *mountTestPlugin) Name() string                              { return p.name }
func (p *mountTestPlugin) Version() string                           { return "0.0.0" }
func (p *mountTestPlugin) Description() string                       { return "mount test plugin" }
func (p *mountTestPlugin) Init(context.Context, plugins.Aegis) error { return nil }
func (p *mountTestPlugin) GetMigrations() []plugins.Migration        { return nil }
func (p *mountTestPlugin) Dependencies() []plugins.Dependency        { return nil }
func (p *mountTestPlugin) RequiresTables() []string                  { return nil }
func (p *mountTestPlugin) ProvidesAuthMethods() []string             { return nil }

func (p *mountTestPlugin) MountRoutes(r router.Router, prefix string) {
	g := r.Group(prefix, p.name)
	g.GET("/items", func(http.ResponseWriter, *http.Request) {})
	g.POST("/items", func(http.ResponseWriter, *http.Request) {})
	if p.document {
		openapi.Doc(openapi.Route{
			Method:  http.MethodGet,
			Path:    prefix + "/items",
			Summary: "List items",
		})
	}
}

// ── helpers ─────────────────────────────────────────────────────────────────

func newMountTestAegis(t *testing.T, r router.Router) *Aegis {
	t.Helper()

	db, err := sql.Open("aegis-mount-test", "")
	if err != nil {
		t.Fatalf("open fake db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	cfg := config.Default().
		WithDB(db).
		WithRouter(r).
		WithDialect(config.DialectPostgres).
		WithSecret([]byte("mount-test-secret-key-32-bytes-long!")).
		WithAPIOnlyMode(true)

	a, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return a
}

func containsPath(paths []string, want string) bool {
	for _, p := range paths {
		if p == want {
			return true
		}
	}
	return false
}

// TestMountRoutesModulePlacement verifies the per-call placement options and
// that the route registry records the same (non-doubled) paths the real
// router receives.
func TestMountRoutesModulePlacement(t *testing.T) {
	cases := []struct {
		name   string
		opts   []MountOption
		want   []string
		absent []string
	}{
		{
			name: "defaults unchanged",
			want: []string{
				"GET /auth/default/session",
				"POST /auth/default/login",
				"GET /auth/admin/items",
				"POST /auth/admin/items",
			},
			absent: []string{"POST /auth/login", "GET /auth/items"},
		},
		{
			name:   "core at prefix root",
			opts:   []MountOption{WithCorePath("")},
			want:   []string{"GET /auth/session", "POST /auth/login"},
			absent: []string{"GET /auth/default/session", "POST /auth/default/login"},
		},
		{
			name:   "plugin moved to absolute path",
			opts:   []MountOption{WithPluginPrefix("admin", "/admin")},
			want:   []string{"GET /admin/items", "GET /auth/default/session"},
			absent: []string{"GET /auth/admin/items"},
		},
		{
			name:   "plugin moved to relative path",
			opts:   []MountOption{WithPluginPrefix("admin", "backoffice")},
			want:   []string{"GET /auth/backoffice/items"},
			absent: []string{"GET /auth/admin/items"},
		},
		{
			name:   "plugin at server root",
			opts:   []MountOption{WithPluginPrefix("admin", "/")},
			want:   []string{"GET /items"},
			absent: []string{"GET /auth/admin/items"},
		},
		{
			name:   "unknown plugin override is ignored",
			opts:   []MountOption{WithPluginPrefix("does-not-exist", "/nope")},
			want:   []string{"GET /auth/admin/items"},
			absent: []string{"GET /nope/items"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := &mountRecordingRouter{}
			a := newMountTestAegis(t, stub)

			if err := a.Use(context.Background(), &mountTestPlugin{name: "admin"}); err != nil {
				t.Fatalf("Use: %v", err)
			}

			a.MountRoutes("/auth", tc.opts...)

			got := stub.paths()
			for _, want := range tc.want {
				if !containsPath(got, want) {
					t.Errorf("router is missing %q; registered: %v", want, got)
				}
			}
			for _, absent := range tc.absent {
				if containsPath(got, absent) {
					t.Errorf("router unexpectedly has %q; registered: %v", absent, got)
				}
			}

			// The introspection registry must mirror the real routing table.
			registered := make([]string, 0, len(a.Routes()))
			for _, e := range a.Routes() {
				registered = append(registered, e.Method+" "+e.Path)
			}
			sort.Strings(registered)
			if len(registered) != len(got) {
				t.Fatalf("Routes() count = %d, router count = %d\nregistry: %v\nrouter:   %v",
					len(registered), len(got), registered, got)
			}
			for i := range got {
				if registered[i] != got[i] {
					t.Fatalf("Routes()[%d] = %q, router[%d] = %q", i, registered[i], i, got[i])
				}
			}
		})
	}
}

// TestMountRoutesOpenAPIIntegration drives a real chi router end to end:
// authentication stays under /auth, an application plugin moves to /admin,
// and the OpenAPI UI/spec move to the server root, with the spec documenting
// the final paths.
func TestMountRoutesOpenAPIIntegration(t *testing.T) {
	mux := chi.NewRouter()
	r := routers.NewChiRouter(mux)
	a := newMountTestAegis(t, r)

	if err := a.Use(context.Background(), &mountTestPlugin{name: "admin", document: true}); err != nil {
		t.Fatalf("Use admin: %v", err)
	}
	oa := openapi.New(&openapi.Config{
		Title:      "Mount Test API",
		Version:    "1.0.0",
		EnableDocs: true,
		DocsPath:   "/docs",
		SpecPath:   "/openapi.json",
	})
	if err := a.Use(context.Background(), oa); err != nil {
		t.Fatalf("Use openapi: %v", err)
	}

	a.MountRoutes("/auth",
		WithPluginPrefix("admin", "/admin"),
		WithPluginPrefix("openapi", "/"),
	)

	statuses := []struct {
		method, path string
		want         int
	}{
		{http.MethodGet, "/auth/default/session", http.StatusUnauthorized}, // auth route exists
		{http.MethodGet, "/admin/items", http.StatusOK},                    // moved plugin route
		{http.MethodGet, "/auth/admin/items", http.StatusNotFound},         // no longer under /auth
		{http.MethodGet, "/docs", http.StatusOK},                           // docs at root
		{http.MethodGet, "/openapi.json", http.StatusOK},                   // spec at root
		{http.MethodGet, "/auth/openapi/docs", http.StatusNotFound},        // no longer under /auth
	}
	for _, tc := range statuses {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != tc.want {
			t.Errorf("%s %s = %d, want %d", tc.method, tc.path, rec.Code, tc.want)
		}
	}

	// The generated spec must document the final mounted paths.
	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var spec struct {
		Paths map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &spec); err != nil {
		t.Fatalf("decode spec: %v", err)
	}
	for _, path := range []string{"/auth/default/session", "/admin/items"} {
		if _, ok := spec.Paths[path]; !ok {
			t.Errorf("spec is missing documented path %q", path)
		}
	}
}

// TestMountPathResolution covers the option resolution rules directly.
func TestMountPathResolution(t *testing.T) {
	cases := []struct {
		name   string
		prefix string
		path   string
		want   string
	}{
		{"relative nests", "/auth", "signin", "/auth/signin"},
		{"empty stays at prefix root", "/auth", "", "/auth"},
		{"absolute escapes prefix", "/auth", "/admin", "/admin"},
		{"absolute root collapses", "/auth", "/", ""},
		{"trailing slashes are trimmed", "/auth/", "signin/", "/auth/signin"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveMountPath(tc.prefix, tc.path); got != tc.want {
				t.Errorf("resolveMountPath(%q, %q) = %q, want %q", tc.prefix, tc.path, got, tc.want)
			}
		})
	}
}
