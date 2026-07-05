package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"project-prism/server/internal/auth"
	"project-prism/server/internal/models"
	"project-prism/server/internal/respond"
)

type authService interface {
	Login(ctx context.Context, email, password string) (*auth.Session, error)
	Refresh(ctx context.Context, rawRefreshToken string) (*auth.Session, error)
	Logout(ctx context.Context, rawRefreshToken string) error
	GetInvitation(ctx context.Context, rawToken string) (*auth.InvitationPreview, error)
	ActivateInvitation(ctx context.Context, rawToken, password string) (*auth.Session, error)
	RefreshTokenFromRequest(r *http.Request) string
	SetRefreshCookie(w http.ResponseWriter, token string, expiresAt time.Time)
	ClearRefreshCookie(w http.ResponseWriter)
}

type AuthHandler struct {
	auth authService
}

func NewAuthHandler(auth authService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type sessionResponse struct {
	User                 *models.User `json:"user"`
	AccessToken          string       `json:"access_token"`
	AccessTokenExpiresAt time.Time    `json:"access_token_expires_at"`
}

type activateInvitationRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.Email) == "" || req.Password == "" {
		respond.Error(w, http.StatusUnprocessableEntity, "email and password are required")
		return
	}

	session, err := h.auth.Login(r.Context(), req.Email, req.Password)
	switch {
	case errors.Is(err, auth.ErrInvalidCredentials):
		respond.Error(w, http.StatusUnauthorized, "invalid email or password")
		return
	case errors.Is(err, auth.ErrInvitationPending):
		respond.Error(w, http.StatusForbidden, "account invitation is pending activation")
		return
	case errors.Is(err, auth.ErrInactiveUser):
		respond.Error(w, http.StatusForbidden, "user is inactive")
		return
	case err != nil:
		respond.Error(w, http.StatusInternalServerError, "failed to sign in")
		return
	}

	h.auth.SetRefreshCookie(w, session.RefreshToken, session.RefreshTokenExpiresAt)
	respond.JSON(w, http.StatusOK, sessionResponse{
		User:                 session.User,
		AccessToken:          session.AccessToken,
		AccessTokenExpiresAt: session.AccessTokenExpiresAt,
	})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	session, err := h.auth.Refresh(r.Context(), h.auth.RefreshTokenFromRequest(r))
	switch {
	case errors.Is(err, auth.ErrInvalidRefreshToken):
		respond.Error(w, http.StatusUnauthorized, "invalid refresh token")
		return
	case errors.Is(err, auth.ErrInactiveUser):
		respond.Error(w, http.StatusForbidden, "user is inactive")
		return
	case errors.Is(err, auth.ErrForbiddenUser):
		respond.Error(w, http.StatusForbidden, "user is not authorized")
		return
	case err != nil:
		respond.Error(w, http.StatusInternalServerError, "failed to refresh session")
		return
	}

	h.auth.SetRefreshCookie(w, session.RefreshToken, session.RefreshTokenExpiresAt)
	respond.JSON(w, http.StatusOK, sessionResponse{
		User:                 session.User,
		AccessToken:          session.AccessToken,
		AccessTokenExpiresAt: session.AccessTokenExpiresAt,
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if err := h.auth.Logout(r.Context(), h.auth.RefreshTokenFromRequest(r)); err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to sign out")
		return
	}

	h.auth.ClearRefreshCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) GetInvitation(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(chi.URLParam(r, "token"))
	if token == "" {
		respond.Error(w, http.StatusUnprocessableEntity, "invitation token is required")
		return
	}

	invitation, err := h.auth.GetInvitation(r.Context(), token)
	switch {
	case errors.Is(err, auth.ErrInvalidInvitationToken):
		respond.Error(w, http.StatusNotFound, "invitation not found or expired")
		return
	case err != nil:
		respond.Error(w, http.StatusInternalServerError, "failed to load invitation")
		return
	}

	respond.JSON(w, http.StatusOK, invitation)
}

func (h *AuthHandler) ActivateInvitation(w http.ResponseWriter, r *http.Request) {
	var req activateInvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.Token) == "" || req.Password == "" {
		respond.Error(w, http.StatusUnprocessableEntity, "token and password are required")
		return
	}

	session, err := h.auth.ActivateInvitation(r.Context(), req.Token, req.Password)
	switch {
	case errors.Is(err, auth.ErrInvalidInvitationToken):
		respond.Error(w, http.StatusNotFound, "invitation not found or expired")
		return
	case errors.Is(err, auth.ErrWeakPassword):
		respond.Error(w, http.StatusUnprocessableEntity, "password must be at least 12 characters")
		return
	case err != nil:
		respond.Error(w, http.StatusInternalServerError, "failed to activate invitation")
		return
	}

	h.auth.SetRefreshCookie(w, session.RefreshToken, session.RefreshTokenExpiresAt)
	respond.JSON(w, http.StatusOK, sessionResponse{
		User:                 session.User,
		AccessToken:          session.AccessToken,
		AccessTokenExpiresAt: session.AccessTokenExpiresAt,
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		respond.Error(w, http.StatusUnauthorized, "missing current user")
		return
	}

	respond.JSON(w, http.StatusOK, user)
}
