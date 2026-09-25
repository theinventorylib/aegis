package core

import (
	"context"
	"testing"

	"github.com/theinventorylib/aegis/v2/auth"
)

type recordingSink struct {
	events []*AuditEvent
}

func (r *recordingSink) OnAuthEvent(_ context.Context, event *AuditEvent) {
	r.events = append(r.events, event)
}

func (r *recordingSink) find(eventType AuditEventType) *AuditEvent {
	for _, event := range r.events {
		if event.EventType == eventType {
			return event
		}
	}
	return nil
}

func TestAuditSinkReceivesEventsAlongsidePrimary(t *testing.T) {
	ctx := context.Background()
	as, _, _, primary := newSecurityTestAuth()
	sink := &recordingSink{}
	as.AddAuditSink(sink)

	if _, err := as.User.CreateUser(ctx, auth.User{Name: "Alice", Email: "alice@example.com"}, "Str0ngPassword1!"); err != nil {
		t.Fatalf("create user: %v", err)
	}

	if !primary.has(AuditEventUserCreated) {
		t.Error("primary audit logger missed user_created")
	}
	if sink.find(AuditEventUserCreated) == nil {
		t.Error("sink missed user_created")
	}
}

func TestAuditSinkCarriesRequestMeta(t *testing.T) {
	as, _, _, _ := newSecurityTestAuth()
	sink := &recordingSink{}
	as.AddAuditSink(sink)
	ctx := WithRequestMeta(context.Background(), &RequestMeta{IPAddress: "203.0.113.7", UserAgent: "aegis-test"})

	if _, err := as.User.CreateUser(ctx, auth.User{Name: "Alice", Email: "alice@example.com"}, "Str0ngPassword1!"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	event := sink.find(AuditEventUserCreated)
	if event == nil {
		t.Fatal("sink missed user_created")
	}
	if event.IPAddress != "203.0.113.7" || event.UserAgent != "aegis-test" {
		t.Errorf("event meta = %q/%q, want 203.0.113.7/aegis-test", event.IPAddress, event.UserAgent)
	}
}

func TestUserLifecycleEmitsAuditEvents(t *testing.T) {
	ctx := context.Background()
	as, _, _, primary := newSecurityTestAuth()
	u, err := as.User.CreateUser(ctx, auth.User{Name: "Alice", Email: "alice@example.com"}, "Str0ngPassword1!")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	u.Name = "Alice Cooper"
	if err := as.User.UpdateUser(ctx, u); err != nil {
		t.Fatalf("update user: %v", err)
	}
	if err := as.User.UpdateUserEmail(ctx, u.GetID(), "new@example.com"); err != nil {
		t.Fatalf("update email: %v", err)
	}
	if err := as.User.DeleteUser(ctx, u.GetID()); err != nil {
		t.Fatalf("delete user: %v", err)
	}
	for _, want := range []AuditEventType{
		AuditEventUserCreated,
		AuditEventUserUpdated,
		AuditEventEmailChanged,
		AuditEventUserDeleted,
	} {
		if !primary.has(want) {
			t.Errorf("missing audit event %s", want)
		}
	}
}
