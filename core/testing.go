package core

import (
	"context"
	"fmt"
	"time"

	"github.com/theinventorylib/aegis/db"
	"github.com/theinventorylib/aegis/models"
)

// MockDB implements DBProvider for testing
type MockDB struct {
	users    map[string]*models.User
	sessions map[string]*models.Session
}

func NewMockDB() *MockDB {
	return &MockDB{
		users:    make(map[string]*models.User),
		sessions: make(map[string]*models.Session),
	}
}

func (m *MockDB) CreateUser(ctx context.Context) (*models.User, error) {
	user := &models.User{
		ID:        fmt.Sprintf("test-user-%d", len(m.users)),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.users[user.ID] = user
	return user, nil
}

func (m *MockDB) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	if u, ok := m.users[id]; ok {
		return u, nil
	}
	return nil, fmt.Errorf("user not found")
}

func (m *MockDB) UpdateUser(ctx context.Context, user *models.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *MockDB) DeleteUser(ctx context.Context, id string) error {
	delete(m.users, id)
	return nil
}

func (m *MockDB) ListUsers(ctx context.Context, offset, limit int) ([]*models.User, error) {
	users := make([]*models.User, 0, len(m.users))
	for _, u := range m.users {
		users = append(users, u)
	}
	return users, nil
}

func (m *MockDB) CountUsers(ctx context.Context) (int, error) {
	return len(m.users), nil
}

func (m *MockDB) CreateSession(ctx context.Context, session *models.Session) error {
	m.sessions[session.Token] = session
	return nil
}

func (m *MockDB) GetSession(ctx context.Context, token string) (*models.Session, error) {
	if s, ok := m.sessions[token]; ok {
		return s, nil
	}
	return nil, fmt.Errorf("session not found")
}

func (m *MockDB) GetSessionByRefreshToken(ctx context.Context, refreshToken string) (*models.Session, error) {
	for _, s := range m.sessions {
		if s.RefreshToken == refreshToken {
			return s, nil
		}
	}
	return nil, fmt.Errorf("session not found")
}

func (m *MockDB) UpdateSession(ctx context.Context, session *models.Session) error {
	m.sessions[session.Token] = session
	return nil
}

func (m *MockDB) DeleteSession(ctx context.Context, token string) error {
	delete(m.sessions, token)
	return nil
}

func (m *MockDB) GetUserSessions(ctx context.Context, userID string) ([]*models.Session, error) {
	return nil, nil
}

func (m *MockDB) DeleteUserSessions(ctx context.Context, userID string) error {
	return nil
}

// Query implements db.DBProvider
func (m *MockDB) Query(ctx context.Context, query string, args ...interface{}) (db.Rows, error) {
	return nil, fmt.Errorf("not implemented in mock")
}

// QueryRow implements db.DBProvider
func (m *MockDB) QueryRow(ctx context.Context, query string, args ...interface{}) db.Row {
	return &mockRow{err: fmt.Errorf("not implemented in mock")}
}

// Exec implements db.DBProvider
func (m *MockDB) Exec(ctx context.Context, query string, args ...interface{}) (db.Result, error) {
	return nil, fmt.Errorf("not implemented in mock")
}

// Begin implements db.DBProvider
func (m *MockDB) Begin(ctx context.Context) (db.Tx, error) {
	return nil, fmt.Errorf("not implemented in mock")
}

// Close implements db.DBProvider
func (m *MockDB) Close() error {
	return nil
}

type mockRow struct {
	err error
}

func (r *mockRow) Scan(dest ...interface{}) error {
	return r.err
}
