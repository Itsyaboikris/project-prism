package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"project-prism/server/internal/models"
	"project-prism/server/internal/respond"
	"project-prism/server/internal/store"
)

type branchStore interface {
	Create(ctx context.Context, p store.CreateBranchParams) (*models.Branch, error)
	GetByID(ctx context.Context, experimentID, id string) (*models.Branch, error)
	Update(ctx context.Context, experimentID, id string, p store.UpdateBranchParams) (*models.Branch, error)
	Delete(ctx context.Context, experimentID, id string) error
}

type branchExperimentStore interface {
	GetByID(ctx context.Context, applicationID, id string) (*models.Experiment, error)
}

type BranchHandler struct {
	store     branchStore
	expStore  branchExperimentStore
}

func NewBranchHandler(s branchStore, expStore branchExperimentStore) *BranchHandler {
	return &BranchHandler{store: s, expStore: expStore}
}

type createBranchRequest struct {
	Key          string          `json:"key"`
	Name         string          `json:"name"`
	Weight       float64         `json:"weight"`
	MetadataJSON json.RawMessage `json:"metadata_json"`
}

type updateBranchRequest struct {
	Name         string          `json:"name"`
	Weight       float64         `json:"weight"`
	MetadataJSON json.RawMessage `json:"metadata_json"`
}

func (h *BranchHandler) Create(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "appID")
	experimentID := chi.URLParam(r, "experimentID")

	if _, err := h.expStore.GetByID(r.Context(), appID, experimentID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respond.Error(w, http.StatusNotFound, "experiment not found")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "failed to verify experiment")
		return
	}

	var req createBranchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Key = strings.TrimSpace(req.Key)
	req.Name = strings.TrimSpace(req.Name)

	if req.Key == "" {
		respond.Error(w, http.StatusUnprocessableEntity, "key is required")
		return
	}
	if req.Name == "" {
		respond.Error(w, http.StatusUnprocessableEntity, "name is required")
		return
	}
	if req.Weight < 0 || req.Weight > 1 {
		respond.Error(w, http.StatusUnprocessableEntity, "weight must be between 0 and 1")
		return
	}

	branch, err := h.store.Create(r.Context(), store.CreateBranchParams{
		ExperimentID: experimentID,
		Key:          req.Key,
		Name:         req.Name,
		Weight:       req.Weight,
		MetadataJSON: req.MetadataJSON,
	})
	if errors.Is(err, store.ErrConflict) {
		respond.Error(w, http.StatusConflict, "a branch with that key already exists for this experiment")
		return
	}
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to create branch")
		return
	}

	respond.JSON(w, http.StatusCreated, branch)
}

func (h *BranchHandler) Update(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "appID")
	experimentID := chi.URLParam(r, "experimentID")
	id := chi.URLParam(r, "id")

	if _, err := h.expStore.GetByID(r.Context(), appID, experimentID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respond.Error(w, http.StatusNotFound, "experiment not found")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "failed to verify experiment")
		return
	}

	var req updateBranchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		respond.Error(w, http.StatusUnprocessableEntity, "name is required")
		return
	}
	if req.Weight < 0 || req.Weight > 1 {
		respond.Error(w, http.StatusUnprocessableEntity, "weight must be between 0 and 1")
		return
	}

	branch, err := h.store.Update(r.Context(), experimentID, id, store.UpdateBranchParams{
		Name:         req.Name,
		Weight:       req.Weight,
		MetadataJSON: req.MetadataJSON,
	})
	if errors.Is(err, store.ErrNotFound) {
		respond.Error(w, http.StatusNotFound, "branch not found")
		return
	}
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to update branch")
		return
	}

	respond.JSON(w, http.StatusOK, branch)
}

func (h *BranchHandler) Delete(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "appID")
	experimentID := chi.URLParam(r, "experimentID")
	id := chi.URLParam(r, "id")

	if _, err := h.expStore.GetByID(r.Context(), appID, experimentID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respond.Error(w, http.StatusNotFound, "experiment not found")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "failed to verify experiment")
		return
	}

	if err := h.store.Delete(r.Context(), experimentID, id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respond.Error(w, http.StatusNotFound, "branch not found")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "failed to delete branch")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
