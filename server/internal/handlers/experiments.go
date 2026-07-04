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
	Delete(ctx context.Context, applicationID, id string) error
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
	Key         string                            `json:"key"`
	Name        string                            `json:"name"`
	Description *string                           `json:"description"`
	StartDate   *time.Time                        `json:"start_date"`
	EndDate     *time.Time                        `json:"end_date"`
	Branches    []createBranchInExperimentRequest `json:"branches"`
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

	if err := validateExperimentKey(req.Key); err != nil {
		respond.Error(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if err := validateExperimentName(req.Name); err != nil {
		respond.Error(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if err := validateExperimentDescription(req.Description); err != nil {
		respond.Error(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if err := validateExperimentDates(req.StartDate, req.EndDate); err != nil {
		respond.Error(w, http.StatusUnprocessableEntity, err.Error())
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
	branchWeights := make([]*models.Branch, 0, len(req.Branches))

	for i, b := range req.Branches {
		b.Key = strings.TrimSpace(b.Key)
		b.Name = strings.TrimSpace(b.Name)
		if err := validateBranchKey(b.Key); err != nil {
			respond.Error(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		if err := validateBranchName(b.Name); err != nil {
			respond.Error(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		if err := validateBranchMetadataJSON(b.MetadataJSON); err != nil {
			respond.Error(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		if err := validateBranchWeightValue(b.Weight); err != nil {
			respond.Error(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		req.Branches[i] = b
		branchWeights = append(branchWeights, &models.Branch{Weight: b.Weight})
		params.Branches = append(params.Branches, store.CreateBranchParams{
			Key:          b.Key,
			Name:         b.Name,
			Weight:       b.Weight,
			MetadataJSON: b.MetadataJSON,
		})
	}
	if err := validateBranchWeights(branchWeights); err != nil {
		respond.Error(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	exp, err := h.store.Create(r.Context(), params)
	if errors.Is(err, store.ErrNotFound) {
		respond.Error(w, http.StatusNotFound, "application not found")
		return
	}
	if errors.Is(err, store.ErrInactive) {
		respond.Error(w, http.StatusConflict, "application is inactive")
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
	if err := validateExperimentName(req.Name); err != nil {
		respond.Error(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if err := validateExperimentDescription(req.Description); err != nil {
		respond.Error(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if req.Status == "" {
		req.Status = models.ExperimentStatusDraft
	}
	if !req.Status.Valid() {
		respond.Error(w, http.StatusUnprocessableEntity, "status must be one of: draft, active, paused, completed")
		return
	}
	if err := validateExperimentDates(req.StartDate, req.EndDate); err != nil {
		respond.Error(w, http.StatusUnprocessableEntity, err.Error())
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

func (h *ExperimentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "appID")
	id := chi.URLParam(r, "id")

	if err := h.store.Delete(r.Context(), appID, id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respond.Error(w, http.StatusNotFound, "experiment not found")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "failed to delete experiment")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
