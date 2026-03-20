package openapi

import "sync"

// Global pending queue for route documentation.
// Any code — plugins, user application code, or Aegis internals —
// calls Doc() to register a route. If the OpenAPI plugin hasn't
// loaded yet, the registration is buffered. When the plugin
// initializes, it drains the queue.
var (
	mu      sync.Mutex
	pending []Route
	active  *Plugin
)

// Doc registers a route for OpenAPI documentation.
//
// This is the single public function for documenting API routes.
// It is safe to call at any point — during init(), inside MountRoutes,
// or anywhere in application startup — regardless of whether the
// OpenAPI plugin has been registered yet.
//
// If the OpenAPI plugin is already loaded, the route is registered
// immediately. Otherwise it is buffered until the plugin initializes.
//
// Example:
//
//	openapi.Doc(openapi.Route{
//	    Method:  "POST",
//	    Path:    "/api/users/{id}/invite",
//	    Summary: "Invite a user",
//	    Tags:    []string{"Users"},
//	    Auth:    true,
//	    Params: []openapi.Param{
//	        {Name: "id", In: "path", Type: "string", Required: true},
//	    },
//	    Body:   openapi.BodyOf[InviteRequest](),
//	    Responses: openapi.Responses{
//	        200: openapi.ResponseOf[InviteResponse]("Invitation sent"),
//	        404: openapi.Text("User not found"),
//	    },
//	})
func Doc(r Route) {
	mu.Lock()
	defer mu.Unlock()
	if active != nil {
		active.register(r) // plugin already loaded, register immediately
	} else {
		pending = append(pending, r) // buffer until plugin initializes
	}
}

// drainPending is called inside Plugin.Init() to process all
// buffered route registrations.
func drainPending(p *Plugin) {
	mu.Lock()
	defer mu.Unlock()
	active = p
	for _, r := range pending {
		p.register(r)
	}
	pending = nil
}

// resetQueue resets the global queue state. This is only used for testing.
func resetQueue() {
	mu.Lock()
	defer mu.Unlock()
	active = nil
	pending = nil
}
