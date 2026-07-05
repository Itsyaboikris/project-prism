package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"project-prism/server/internal/auth"
	"project-prism/server/internal/models"
	"project-prism/server/internal/respond"
	"project-prism/server/internal/store"
)

type userAdminService interface {
	ListUsers(ctx context.Context) ([]*models.User, error)
	InviteAdmin(ctx context.Context, invitedByEmail, email string) (*models.User, error)
	UpdateUserStatus(ctx context.Context, userID string, status models.UserStatus) (*models.User, error)
}

type UserHandler struct {
	auth userAdminService
}

func NewUserHandler(auth userAdminService) *UserHandler {
	return &UserHandler{auth: auth}
}

type createUserRequest struct {
	Email string `json:"email"`
}

type updateUserStatusRequest struct {
	Status models.UserStatus `json:"status"`
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.auth.ListUsers(r.Context())
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to list users")
		return
	}
	if users == nil {
		users = []*models.User{}
	}

	respond.JSON(w, http.StatusOK, users)
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.Email) == "" {
		respond.Error(w, http.StatusUnprocessableEntity, "email is required")
		return
	}

	currentUser, ok := auth.UserFromContext(r.Context())
	if !ok {
		respond.Error(w, http.StatusUnauthorized, "missing current user")
		return
	}

	user, err := h.auth.InviteAdmin(r.Context(), currentUser.Email, req.Email)
	switch {
	case errors.Is(err, auth.ErrInvalidEmail):
		respond.Error(w, http.StatusUnprocessableEntity, "email must be a valid email address")
		return
	case errors.Is(err, auth.ErrInviteAlreadyPending):
		respond.Error(w, http.StatusConflict, "an active invite already exists for this email")
		return
	case errors.Is(err, store.ErrConflict):
		respond.Error(w, http.StatusConflict, "user already exists")
		return
	case errors.Is(err, auth.ErrMailerUnavailable):
		respond.Error(w, http.StatusServiceUnavailable, "invite email is not configured")
		return
	case err != nil:
		respond.Error(w, http.StatusInternalServerError, "failed to send invite")
		return
	}

	respond.JSON(w, http.StatusCreated, user)
}

func (h *UserHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")

	var req updateUserStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if !req.Status.Valid() {
		respond.Error(w, http.StatusUnprocessableEntity, "status must be one of: active, inactive")
		return
	}

	user, err := h.auth.UpdateUserStatus(r.Context(), userID, req.Status)
	switch {
	case errors.Is(err, store.ErrNotFound):
		respond.Error(w, http.StatusNotFound, "user not found")
		return
	case errors.Is(err, auth.ErrLastActiveAdmin):
		respond.Error(w, http.StatusConflict, "cannot deactivate the last active admin")
		return
	case err != nil:
		respond.Error(w, http.StatusInternalServerError, "failed to update user")
		return
	}

	respond.JSON(w, http.StatusOK, user)
}
