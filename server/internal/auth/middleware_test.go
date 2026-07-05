package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"project-prism/server/internal/models"
)

type fakeAccessAuthenticator struct {
	user  *models.User
	err   error
	token string
}

func (f *fakeAccessAuthenticator) AuthenticateAccessToken(_ context.Context, token string) (*models.User, error) {
	f.token = token
	return f.user, f.err
}

func TestRequireAdminRejectsMissingToken(t *testing.T) {
	mw := NewMiddleware(&fakeAccessAuthenticator{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/applications", nil)
	rec := httptest.NewRecorder()

	mw.RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestRequireAdminRejectsInvalidToken(t *testing.T) {
	authenticator := &fakeAccessAuthenticator{err: ErrInvalidAccessToken}
	mw := NewMiddleware(authenticator)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/applications", nil)
	req.Header.Set("Authorization", "Bearer invalid")
	rec := httptest.NewRecorder()

	mw.RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
	if authenticator.token != "invalid" {
		t.Fatalf("expected token %q, got %q", "invalid", authenticator.token)
	}
}

func TestRequireAdminRejectsInactiveUser(t *testing.T) {
	mw := NewMiddleware(&fakeAccessAuthenticator{err: ErrInactiveUser})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/applications", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()

	mw.RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestRequireAdminInjectsUserIntoContext(t *testing.T) {
	wantUser := &models.User{
		ID:     "user_123",
		Email:  "admin@example.com",
		Role:   models.UserRoleAdmin,
		Status: models.UserStatusActive,
	}
	mw := NewMiddleware(&fakeAccessAuthenticator{user: wantUser})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/applications", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()

	mw.RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser, ok := UserFromContext(r.Context())
		if !ok {
			t.Fatal("expected user in context")
		}
		if gotUser.ID != wantUser.ID {
			t.Fatalf("expected user %q, got %q", wantUser.ID, gotUser.ID)
		}
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}
}
