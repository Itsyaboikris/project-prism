package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"project-prism/server/internal/apiauth"
	"project-prism/server/internal/models"
	"project-prism/server/internal/respond"
	"project-prism/server/internal/store"
)

type assignmentStore interface {
	Assign(ctx context.Context, p store.AssignParams) (*models.Branch, error)
	ListByExperiment(ctx context.Context, applicationID, experimentID string) (*models.ExperimentAssignmentsView, error)
	GetExperimentDashboard(ctx context.Context, applicationID, experimentID string) (*models.ExperimentDashboard, error)
}

type AssignmentHandler struct {
	store assignmentStore
}

func NewAssignmentHandler(s assignmentStore) *AssignmentHandler {
	return &AssignmentHandler{store: s}
}

type assignRequest struct {
	UserID        string `json:"user_id"`
	ExperimentKey string `json:"experiment_key"`
}

func (h *AssignmentHandler) Create(w http.ResponseWriter, r *http.Request) {
	app, ok := apiauth.ApplicationFromContext(r.Context())
	if !ok {
		respond.Error(w, http.StatusInternalServerError, "application context missing")
		return
	}

	var req assignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.UserID = strings.TrimSpace(req.UserID)
	req.ExperimentKey = strings.TrimSpace(req.ExperimentKey)

	if req.UserID == "" {
		respond.Error(w, http.StatusUnprocessableEntity, "user_id is required")
		return
	}
	if req.ExperimentKey == "" {
		respond.Error(w, http.StatusUnprocessableEntity, "experiment_key is required")
		return
	}

	branch, err := h.store.Assign(r.Context(), store.AssignParams{
		ApplicationID: app.ID,
		ExperimentKey: req.ExperimentKey,
		UserID:        req.UserID,
	})
	if errors.Is(err, store.ErrNotFound) {
		respond.Error(w, http.StatusNotFound, "experiment not found")
		return
	}
	if errors.Is(err, store.ErrNotEligible) {
		respond.Error(w, http.StatusConflict, "experiment is not eligible for assignment")
		return
	}
	if errors.Is(err, store.ErrMisconfigured) {
		respond.Error(w, http.StatusConflict, "experiment branches are misconfigured")
		return
	}
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to assign branch")
		return
	}

	respond.JSON(w, http.StatusOK, branch)
}

func (h *AssignmentHandler) ListByExperiment(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "appID")
	experimentID := chi.URLParam(r, "id")

	assignments, err := h.store.ListByExperiment(r.Context(), appID, experimentID)
	if errors.Is(err, store.ErrNotFound) {
		respond.Error(w, http.StatusNotFound, "experiment not found")
		return
	}
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to list assignments")
		return
	}

	respond.JSON(w, http.StatusOK, assignments)
}

func (h *AssignmentHandler) GetExperimentDashboard(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "appID")
	experimentID := chi.URLParam(r, "id")

	dashboard, err := h.store.GetExperimentDashboard(r.Context(), appID, experimentID)
	if errors.Is(err, store.ErrNotFound) {
		respond.Error(w, http.StatusNotFound, "experiment not found")
		return
	}
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to load experiment dashboard")
		return
	}

	respond.JSON(w, http.StatusOK, dashboard)
}
