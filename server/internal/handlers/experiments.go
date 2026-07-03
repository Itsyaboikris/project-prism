package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"project-prism/server/internal/models"
	"project-prism/server/internal/respond"
	"project-prism/server/internal/store"
)

type experimentStore interface {
	Create(ctx context.Context, p store.CreateExperimentParams) (*models.Experiment, error)
	List(ctx context.Context, applicationID string) ([]*models.Experiment, error)
	GetByID(ctx context.Context, applicationID, id string) (*models.Experiment, error)
	Update(ctx context.Context, applicationID, id string, p store.UpdateExperimentParams) (*models.Experiment, error)
}

type experimentBranchStore interface {
	ListByExperimentIDs(ctx context.Context, experimentIDs []string) (map[string][]*models.Branch, error)
}

type ExperimentHandler struct {
	store       experimentStore
	branchStore experimentBranchStore
}

func NewExperimentHandler(s experimentStore, bs experimentBranchStore) *ExperimentHandler {
	return &ExperimentHandler{store: s, branchStore: bs}
}

type createBranchInExperimentRequest struct {
	Key          string          `json:"key"`
	Name         string          `json:"name"`
	Weight       float64         `json:"weight"`
	MetadataJSON json.RawMessage `json:"metadata_json"`
}

type createExperimentRequest struct {
	Key         string                             `json:"key"`
	Name        string                             `json:"name"`
	Description *string                            `json:"description"`
	StartDate   *time.Time                         `json:"start_date"`
	EndDate     *time.Time                         `json:"end_date"`
	Branches    []createBranchInExperimentRequest  `json:"branches"`
}

type updateExperimentRequest struct {
	Name        string                  `json:"name"`
	Description *string                 `json:"description"`
	Status      models.ExperimentStatus `json:"status"`
	StartDate   *time.Time              `json:"start_date"`
	EndDate     *time.Time              `json:"end_date"`
}

func (h *ExperimentHandler) Create(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "appID")

	var req createExperimentRequest
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

	params := store.CreateExperimentParams{
		ApplicationID: appID,
		Key:           req.Key,
		Name:          req.Name,
		Description:   req.Description,
		StartDate:     req.StartDate,
		EndDate:       req.EndDate,
	}

	for i, b := range req.Branches {
		b.Key = strings.TrimSpace(b.Key)
		b.Name = strings.TrimSpace(b.Name)
		if b.Key == "" {
			respond.Error(w, http.StatusUnprocessableEntity, "branch key is required")
			return
		}
		if b.Name == "" {
			respond.Error(w, http.StatusUnprocessableEntity, "branch name is required")
			return
		}
		if b.Weight < 0 || b.Weight > 1 {
			respond.Error(w, http.StatusUnprocessableEntity, "branch weight must be between 0 and 1")
			return
		}
		req.Branches[i] = b
		params.Branches = append(params.Branches, store.CreateBranchParams{
			Key:          b.Key,
			Name:         b.Name,
			Weight:       b.Weight,
			MetadataJSON: b.MetadataJSON,
		})
	}

	exp, err := h.store.Create(r.Context(), params)
	if errors.Is(err, store.ErrNotFound) {
		respond.Error(w, http.StatusNotFound, "application not found")
		return
	}
	if errors.Is(err, store.ErrConflict) {
		respond.Error(w, http.StatusConflict, "an experiment with that key already exists for this application")
		return
	}
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to create experiment")
		return
	}

	respond.JSON(w, http.StatusCreated, exp)
}

func (h *ExperimentHandler) List(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "appID")

	exps, err := h.store.List(r.Context(), appID)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to list experiments")
		return
	}

	if len(exps) == 0 {
		respond.JSON(w, http.StatusOK, []*models.Experiment{})
		return
	}

	ids := make([]string, len(exps))
	for i, e := range exps {
		ids[i] = e.ID
	}

	branchMap, err := h.branchStore.ListByExperimentIDs(r.Context(), ids)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to load branches")
		return
	}

	for _, exp := range exps {
		if branches, ok := branchMap[exp.ID]; ok {
			exp.Branches = branches
		}
	}

	respond.JSON(w, http.StatusOK, exps)
}

func (h *ExperimentHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "appID")
	id := chi.URLParam(r, "id")

	exp, err := h.store.GetByID(r.Context(), appID, id)
	if errors.Is(err, store.ErrNotFound) {
		respond.Error(w, http.StatusNotFound, "experiment not found")
		return
	}
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to get experiment")
		return
	}

	branchMap, err := h.branchStore.ListByExperimentIDs(r.Context(), []string{exp.ID})
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to load branches")
		return
	}
	if branches, ok := branchMap[exp.ID]; ok {
		exp.Branches = branches
	}

	respond.JSON(w, http.StatusOK, exp)
}

func (h *ExperimentHandler) Update(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "appID")
	id := chi.URLParam(r, "id")

	var req updateExperimentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		respond.Error(w, http.StatusUnprocessableEntity, "name is required")
		return
	}
	if req.Status == "" {
		req.Status = models.ExperimentStatusDraft
	}
	if !req.Status.Valid() {
		respond.Error(w, http.StatusUnprocessableEntity, "status must be one of: draft, active, paused, completed")
		return
	}

	exp, err := h.store.Update(r.Context(), appID, id, store.UpdateExperimentParams{
		Name:        req.Name,
		Description: req.Description,
		Status:      req.Status,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
	})
	if errors.Is(err, store.ErrNotFound) {
		respond.Error(w, http.StatusNotFound, "experiment not found")
		return
	}
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to update experiment")
		return
	}

	branchMap, err := h.branchStore.ListByExperimentIDs(r.Context(), []string{exp.ID})
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to load branches")
		return
	}
	if branches, ok := branchMap[exp.ID]; ok {
		exp.Branches = branches
	}

	respond.JSON(w, http.StatusOK, exp)
}
