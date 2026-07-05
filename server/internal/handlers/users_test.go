package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"project-prism/server/internal/auth"
	"project-prism/server/internal/models"
	"project-prism/server/internal/store"
)

type fakeUserAdminService struct {
	listFn         func(context.Context) ([]*models.User, error)
	inviteFn       func(context.Context, string, string) (*models.User, error)
	updateStatusFn func(context.Context, string, models.UserStatus) (*models.User, error)
}

func (f *fakeUserAdminService) ListUsers(ctx context.Context) ([]*models.User, error) {
	return f.listFn(ctx)
}

func (f *fakeUserAdminService) InviteAdmin(ctx context.Context, invitedByEmail, email string) (*models.User, error) {
	return f.inviteFn(ctx, invitedByEmail, email)
}

func (f *fakeUserAdminService) UpdateUserStatus(ctx context.Context, userID string, status models.UserStatus) (*models.User, error) {
	return f.updateStatusFn(ctx, userID, status)
}

func TestUserHandlerList(t *testing.T) {
	handler := NewUserHandler(&fakeUserAdminService{
		listFn: func(context.Context) ([]*models.User, error) { return nil, nil },
		inviteFn: func(context.Context, string, string) (*models.User, error) {
			return nil, nil
		},
		updateStatusFn: func(context.Context, string, models.UserStatus) (*models.User, error) {
			return nil, nil
		},
	})

	rec := httptest.NewRecorder()
	handler.List(rec, httptest.NewRequest(http.MethodGet, "/api/v1/users", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("expected empty array response, got %q", rec.Body.String())
	}
}

func TestUserHandlerCreateRejectsInvalidEmail(t *testing.T) {
	handler := NewUserHandler(&fakeUserAdminService{
		listFn: func(context.Context) ([]*models.User, error) { return nil, nil },
		inviteFn: func(context.Context, string, string) (*models.User, error) {
			return nil, auth.ErrInvalidEmail
		},
		updateStatusFn: func(context.Context, string, models.UserStatus) (*models.User, error) {
			return nil, nil
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(`{"email":"bad-email"}`))
	req = req.WithContext(auth.WithUser(req.Context(), &models.User{Email: "owner@example.com"}))
	handler.Create(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d", http.StatusUnprocessableEntity, rec.Code)
	}
}

func TestUserHandlerCreateHandlesConflict(t *testing.T) {
	handler := NewUserHandler(&fakeUserAdminService{
		listFn: func(context.Context) ([]*models.User, error) { return nil, nil },
		inviteFn: func(context.Context, string, string) (*models.User, error) {
			return nil, store.ErrConflict
		},
		updateStatusFn: func(context.Context, string, models.UserStatus) (*models.User, error) {
			return nil, nil
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(`{"email":"admin@example.com"}`))
	req = req.WithContext(auth.WithUser(req.Context(), &models.User{Email: "owner@example.com"}))
	handler.Create(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, rec.Code)
	}
}

func TestUserHandlerCreateInvitesAdmin(t *testing.T) {
	var gotInvitedBy, gotEmail string
	handler := NewUserHandler(&fakeUserAdminService{
		listFn: func(context.Context) ([]*models.User, error) { return nil, nil },
		inviteFn: func(_ context.Context, invitedByEmail, email string) (*models.User, error) {
			gotInvitedBy = invitedByEmail
			gotEmail = email
			return &models.User{
				ID:     "user_123",
				Email:  email,
				Role:   models.UserRoleAdmin,
				Status: models.UserStatusInvited,
			}, nil
		},
		updateStatusFn: func(context.Context, string, models.UserStatus) (*models.User, error) {
			return nil, nil
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(`{"email":"admin@example.com"}`))
	req = req.WithContext(auth.WithUser(req.Context(), &models.User{Email: "owner@example.com"}))

	handler.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
	if gotInvitedBy != "owner@example.com" || gotEmail != "admin@example.com" {
		t.Fatalf("expected invite params to be passed through, got %q / %q", gotInvitedBy, gotEmail)
	}
}

func TestUserHandlerUpdateStatus(t *testing.T) {
	handler := NewUserHandler(&fakeUserAdminService{
		listFn: func(context.Context) ([]*models.User, error) { return nil, nil },
		inviteFn: func(context.Context, string, string) (*models.User, error) {
			return nil, nil
		},
		updateStatusFn: func(_ context.Context, userID string, status models.UserStatus) (*models.User, error) {
			return &models.User{
				ID:     userID,
				Email:  "admin@example.com",
				Role:   models.UserRoleAdmin,
				Status: status,
			}, nil
		},
	})

	rec := httptest.NewRecorder()
	req := newRequestWithURLParams(http.MethodPatch, "/", `{"status":"inactive"}`, map[string]string{"id": "user_123"})
	handler.UpdateStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestUserHandlerUpdateStatusPreventsLastAdminDeactivation(t *testing.T) {
	handler := NewUserHandler(&fakeUserAdminService{
		listFn: func(context.Context) ([]*models.User, error) { return nil, nil },
		inviteFn: func(context.Context, string, string) (*models.User, error) {
			return nil, nil
		},
		updateStatusFn: func(context.Context, string, models.UserStatus) (*models.User, error) {
			return nil, auth.ErrLastActiveAdmin
		},
	})

	rec := httptest.NewRecorder()
	req := newRequestWithURLParams(http.MethodPatch, "/", `{"status":"inactive"}`, map[string]string{"id": "user_123"})
	handler.UpdateStatus(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, rec.Code)
	}
}
