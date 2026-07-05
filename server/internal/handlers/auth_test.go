package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"project-prism/server/internal/auth"
	"project-prism/server/internal/models"
)

type fakeAuthService struct {
	loginFn              func(context.Context, string, string) (*auth.Session, error)
	refreshFn            func(context.Context, string) (*auth.Session, error)
	logoutFn             func(context.Context, string) error
	getInvitationFn      func(context.Context, string) (*auth.InvitationPreview, error)
	activateInvitationFn func(context.Context, string, string) (*auth.Session, error)
	refreshTokenFn       func(*http.Request) string
	setCookieFn          func(http.ResponseWriter, string, time.Time)
	clearCookieFn        func(http.ResponseWriter)
}

func (f *fakeAuthService) Login(ctx context.Context, email, password string) (*auth.Session, error) {
	return f.loginFn(ctx, email, password)
}

func (f *fakeAuthService) Refresh(ctx context.Context, rawRefreshToken string) (*auth.Session, error) {
	return f.refreshFn(ctx, rawRefreshToken)
}

func (f *fakeAuthService) Logout(ctx context.Context, rawRefreshToken string) error {
	return f.logoutFn(ctx, rawRefreshToken)
}

func (f *fakeAuthService) GetInvitation(ctx context.Context, rawToken string) (*auth.InvitationPreview, error) {
	return f.getInvitationFn(ctx, rawToken)
}

func (f *fakeAuthService) ActivateInvitation(ctx context.Context, rawToken, password string) (*auth.Session, error) {
	return f.activateInvitationFn(ctx, rawToken, password)
}

func (f *fakeAuthService) RefreshTokenFromRequest(r *http.Request) string {
	return f.refreshTokenFn(r)
}

func (f *fakeAuthService) SetRefreshCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	f.setCookieFn(w, token, expiresAt)
}

func (f *fakeAuthService) ClearRefreshCookie(w http.ResponseWriter) {
	f.clearCookieFn(w)
}

func TestAuthHandlerLogin(t *testing.T) {
	expiresAt := time.Date(2026, time.July, 4, 12, 0, 0, 0, time.UTC).Add(15 * time.Minute)
	var gotEmail, gotPassword string
	var cookieToken string

	handler := NewAuthHandler(&fakeAuthService{
		loginFn: func(_ context.Context, email, password string) (*auth.Session, error) {
			gotEmail, gotPassword = email, password
			return &auth.Session{
				User: &models.User{
					ID:     "user_123",
					Email:  "admin@example.com",
					Role:   models.UserRoleAdmin,
					Status: models.UserStatusActive,
				},
				AccessToken:           "access_token",
				AccessTokenExpiresAt:  expiresAt,
				RefreshToken:          "refresh_token",
				RefreshTokenExpiresAt: expiresAt.Add(24 * time.Hour),
			}, nil
		},
		refreshFn:            func(context.Context, string) (*auth.Session, error) { return nil, nil },
		logoutFn:             func(context.Context, string) error { return nil },
		getInvitationFn:      func(context.Context, string) (*auth.InvitationPreview, error) { return nil, nil },
		activateInvitationFn: func(context.Context, string, string) (*auth.Session, error) { return nil, nil },
		refreshTokenFn:       func(*http.Request) string { return "" },
		setCookieFn: func(w http.ResponseWriter, token string, _ time.Time) {
			cookieToken = token
			http.SetCookie(w, &http.Cookie{Name: "prism_refresh", Value: token})
		},
		clearCookieFn: func(http.ResponseWriter) {},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"admin@example.com","password":"correct horse battery staple"}`))

	handler.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if gotEmail != "admin@example.com" || gotPassword != "correct horse battery staple" {
		t.Fatalf("expected credentials to be passed through, got %q / %q", gotEmail, gotPassword)
	}
	if cookieToken != "refresh_token" {
		t.Fatalf("expected refresh cookie token %q, got %q", "refresh_token", cookieToken)
	}
}

func TestAuthHandlerLoginRejectsInvalidCredentials(t *testing.T) {
	handler := NewAuthHandler(&fakeAuthService{
		loginFn: func(context.Context, string, string) (*auth.Session, error) {
			return nil, auth.ErrInvalidCredentials
		},
		refreshFn:            func(context.Context, string) (*auth.Session, error) { return nil, nil },
		logoutFn:             func(context.Context, string) error { return nil },
		getInvitationFn:      func(context.Context, string) (*auth.InvitationPreview, error) { return nil, nil },
		activateInvitationFn: func(context.Context, string, string) (*auth.Session, error) { return nil, nil },
		refreshTokenFn:       func(*http.Request) string { return "" },
		setCookieFn:          func(http.ResponseWriter, string, time.Time) {},
		clearCookieFn:        func(http.ResponseWriter) {},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"admin@example.com","password":"bad"}`))
	handler.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestAuthHandlerLoginRejectsPendingInvitation(t *testing.T) {
	handler := NewAuthHandler(&fakeAuthService{
		loginFn: func(context.Context, string, string) (*auth.Session, error) {
			return nil, auth.ErrInvitationPending
		},
		refreshFn:            func(context.Context, string) (*auth.Session, error) { return nil, nil },
		logoutFn:             func(context.Context, string) error { return nil },
		getInvitationFn:      func(context.Context, string) (*auth.InvitationPreview, error) { return nil, nil },
		activateInvitationFn: func(context.Context, string, string) (*auth.Session, error) { return nil, nil },
		refreshTokenFn:       func(*http.Request) string { return "" },
		setCookieFn:          func(http.ResponseWriter, string, time.Time) {},
		clearCookieFn:        func(http.ResponseWriter) {},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"admin@example.com","password":"bad"}`))
	handler.Login(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestAuthHandlerLogoutClearsCookie(t *testing.T) {
	var cleared bool
	handler := NewAuthHandler(&fakeAuthService{
		loginFn:   func(context.Context, string, string) (*auth.Session, error) { return nil, nil },
		refreshFn: func(context.Context, string) (*auth.Session, error) { return nil, nil },
		logoutFn: func(_ context.Context, token string) error {
			if token != "refresh_token" {
				t.Fatalf("expected refresh token %q, got %q", "refresh_token", token)
			}
			return nil
		},
		getInvitationFn:      func(context.Context, string) (*auth.InvitationPreview, error) { return nil, nil },
		activateInvitationFn: func(context.Context, string, string) (*auth.Session, error) { return nil, nil },
		refreshTokenFn:       func(*http.Request) string { return "refresh_token" },
		setCookieFn:          func(http.ResponseWriter, string, time.Time) {},
		clearCookieFn: func(http.ResponseWriter) {
			cleared = true
		},
	})

	rec := httptest.NewRecorder()
	handler.Logout(rec, httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil))

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}
	if !cleared {
		t.Fatal("expected refresh cookie to be cleared")
	}
}

func TestAuthHandlerMe(t *testing.T) {
	handler := NewAuthHandler(&fakeAuthService{
		loginFn:        func(context.Context, string, string) (*auth.Session, error) { return nil, nil },
		refreshFn:            func(context.Context, string) (*auth.Session, error) { return nil, nil },
		logoutFn:             func(context.Context, string) error { return nil },
		getInvitationFn:      func(context.Context, string) (*auth.InvitationPreview, error) { return nil, nil },
		activateInvitationFn: func(context.Context, string, string) (*auth.Session, error) { return nil, nil },
		refreshTokenFn:       func(*http.Request) string { return "" },
		setCookieFn:          func(http.ResponseWriter, string, time.Time) {},
		clearCookieFn:        func(http.ResponseWriter) {},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req = req.WithContext(auth.WithUser(req.Context(), &models.User{
		ID:     "user_123",
		Email:  "admin@example.com",
		Role:   models.UserRoleAdmin,
		Status: models.UserStatusActive,
	}))
	rec := httptest.NewRecorder()

	handler.Me(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestAuthHandlerRefreshHandlesFailure(t *testing.T) {
	handler := NewAuthHandler(&fakeAuthService{
		loginFn: func(context.Context, string, string) (*auth.Session, error) { return nil, nil },
		refreshFn: func(context.Context, string) (*auth.Session, error) {
			return nil, errors.New("boom")
		},
		logoutFn:             func(context.Context, string) error { return nil },
		getInvitationFn:      func(context.Context, string) (*auth.InvitationPreview, error) { return nil, nil },
		activateInvitationFn: func(context.Context, string, string) (*auth.Session, error) { return nil, nil },
		refreshTokenFn:       func(*http.Request) string { return "refresh_token" },
		setCookieFn:          func(http.ResponseWriter, string, time.Time) {},
		clearCookieFn:        func(http.ResponseWriter) {},
	})

	rec := httptest.NewRecorder()
	handler.Refresh(rec, httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestAuthHandlerRefreshRejectsInvalidRefreshToken(t *testing.T) {
	handler := NewAuthHandler(&fakeAuthService{
		loginFn: func(context.Context, string, string) (*auth.Session, error) { return nil, nil },
		refreshFn: func(context.Context, string) (*auth.Session, error) {
			return nil, auth.ErrInvalidRefreshToken
		},
		logoutFn:             func(context.Context, string) error { return nil },
		getInvitationFn:      func(context.Context, string) (*auth.InvitationPreview, error) { return nil, nil },
		activateInvitationFn: func(context.Context, string, string) (*auth.Session, error) { return nil, nil },
		refreshTokenFn:       func(*http.Request) string { return "refresh_token" },
		setCookieFn:          func(http.ResponseWriter, string, time.Time) {},
		clearCookieFn:        func(http.ResponseWriter) {},
	})

	rec := httptest.NewRecorder()
	handler.Refresh(rec, httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestAuthHandlerGetInvitation(t *testing.T) {
	handler := NewAuthHandler(&fakeAuthService{
		loginFn:        func(context.Context, string, string) (*auth.Session, error) { return nil, nil },
		refreshFn:      func(context.Context, string) (*auth.Session, error) { return nil, nil },
		logoutFn:       func(context.Context, string) error { return nil },
		getInvitationFn: func(_ context.Context, token string) (*auth.InvitationPreview, error) {
			if token != "token_123" {
				t.Fatalf("expected token %q, got %q", "token_123", token)
			}
			return &auth.InvitationPreview{Email: "invitee@example.com", ExpiresAt: time.Now().UTC()}, nil
		},
		activateInvitationFn: func(context.Context, string, string) (*auth.Session, error) { return nil, nil },
		refreshTokenFn:       func(*http.Request) string { return "" },
		setCookieFn:          func(http.ResponseWriter, string, time.Time) {},
		clearCookieFn:        func(http.ResponseWriter) {},
	})

	rec := httptest.NewRecorder()
	req := newRequestWithURLParams(http.MethodGet, "/", "", map[string]string{"token": "token_123"})
	handler.GetInvitation(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestAuthHandlerActivateInvitation(t *testing.T) {
	expiresAt := time.Now().UTC().Add(24 * time.Hour)
	handler := NewAuthHandler(&fakeAuthService{
		loginFn:   func(context.Context, string, string) (*auth.Session, error) { return nil, nil },
		refreshFn: func(context.Context, string) (*auth.Session, error) { return nil, nil },
		logoutFn:  func(context.Context, string) error { return nil },
		getInvitationFn: func(context.Context, string) (*auth.InvitationPreview, error) {
			return nil, nil
		},
		activateInvitationFn: func(_ context.Context, token, password string) (*auth.Session, error) {
			if token != "token_123" || password != "correct horse battery staple" {
				t.Fatalf("unexpected activation payload %q / %q", token, password)
			}
			return &auth.Session{
				User: &models.User{
					ID:     "user_123",
					Email:  "invitee@example.com",
					Role:   models.UserRoleAdmin,
					Status: models.UserStatusActive,
				},
				AccessToken:           "access_token",
				AccessTokenExpiresAt:  time.Now().UTC().Add(15 * time.Minute),
				RefreshToken:          "refresh_token",
				RefreshTokenExpiresAt: expiresAt,
			}, nil
		},
		refreshTokenFn: func(*http.Request) string { return "" },
		setCookieFn:    func(http.ResponseWriter, string, time.Time) {},
		clearCookieFn:  func(http.ResponseWriter) {},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/invitations/activate", strings.NewReader(`{"token":"token_123","password":"correct horse battery staple"}`))
	handler.ActivateInvitation(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}
