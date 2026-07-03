package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"project-prism/server/internal/apikey"
	"project-prism/server/internal/models"
	"project-prism/server/internal/respond"
	"project-prism/server/internal/store"
)

type applicationStore interface {
	Create(ctx context.Context, name, apiKey string) (*models.Application, error)
	List(ctx context.Context) ([]*models.Application, error)
	GetByID(ctx context.Context, id string) (*models.Application, error)
	Update(ctx context.Context, id string, p store.UpdateApplicationParams) (*models.Application, error)
	Delete(ctx context.Context, id string) error
}

type ApplicationHandler struct {
	store applicationStore
}

func NewApplicationHandler(s applicationStore) *ApplicationHandler {
	return &ApplicationHandler{store: s}
}

type createApplicationRequest struct {
	Name string `json:"name"`
}

type updateApplicationRequest struct {
	Name   string                    `json:"name"`
	Status *models.ApplicationStatus `json:"status"`
}

func (h *ApplicationHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createApplicationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		respond.Error(w, http.StatusUnprocessableEntity, "name is required")
		return
	}

	key, err := apikey.Generate()
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to generate API key")
		return
	}

	app, err := h.store.Create(r.Context(), req.Name, key)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to create application")
		return
	}

	respond.JSON(w, http.StatusCreated, app)
}

func (h *ApplicationHandler) List(w http.ResponseWriter, r *http.Request) {
	apps, err := h.store.List(r.Context())
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to list applications")
		return
	}

	// return an empty array rather than null when there are no applications
	if apps == nil {
		apps = []*models.Application{}
	}

	respond.JSON(w, http.StatusOK, apps)
}

func (h *ApplicationHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	app, err := h.store.GetByID(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		respond.Error(w, http.StatusNotFound, "application not found")
		return
	}
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to get application")
		return
	}

	respond.JSON(w, http.StatusOK, app)
}

func (h *ApplicationHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateApplicationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		respond.Error(w, http.StatusUnprocessableEntity, "name is required")
		return
	}
	if req.Status != nil && !req.Status.Valid() {
		respond.Error(w, http.StatusUnprocessableEntity, "status must be one of: active, inactive")
		return
	}

	app, err := h.store.Update(r.Context(), id, store.UpdateApplicationParams{
		Name:   req.Name,
		Status: req.Status,
	})
	if errors.Is(err, store.ErrNotFound) {
		respond.Error(w, http.StatusNotFound, "application not found")
		return
	}
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to update application")
		return
	}

	respond.JSON(w, http.StatusOK, app)
}

func (h *ApplicationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.store.Delete(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respond.Error(w, http.StatusNotFound, "application not found")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "failed to delete application")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
