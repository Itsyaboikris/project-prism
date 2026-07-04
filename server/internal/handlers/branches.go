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
	ListByExperimentID(ctx context.Context, experimentID string) ([]*models.Branch, error)
	SaveAll(ctx context.Context, experimentID string, branches []store.SaveBranchParams) ([]*models.Branch, error)
	GetByID(ctx context.Context, experimentID, id string) (*models.Branch, error)
	Update(ctx context.Context, experimentID, id string, p store.UpdateBranchParams) (*models.Branch, error)
	Delete(ctx context.Context, experimentID, id string) error
}

type branchExperimentStore interface {
	GetByID(ctx context.Context, applicationID, id string) (*models.Experiment, error)
}

type BranchHandler struct {
	store    branchStore
	expStore branchExperimentStore
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

type saveBranchRequest struct {
	ID           string          `json:"id"`
	Key          string          `json:"key"`
	Name         string          `json:"name"`
	Weight       float64         `json:"weight"`
	MetadataJSON json.RawMessage `json:"metadata_json"`
}

type saveBranchesRequest struct {
	Branches []saveBranchRequest `json:"branches"`
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

	if err := validateBranchKey(req.Key); err != nil {
		respond.Error(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if err := validateBranchName(req.Name); err != nil {
		respond.Error(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if err := validateBranchMetadataJSON(req.MetadataJSON); err != nil {
		respond.Error(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if err := validateBranchWeightValue(req.Weight); err != nil {
		respond.Error(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	existingBranches, err := h.store.ListByExperimentID(r.Context(), experimentID)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to validate branch weights")
		return
	}

	candidateBranches := append(append([]*models.Branch{}, existingBranches...), &models.Branch{
		Weight: req.Weight,
	})
	if err := validateBranchWeights(candidateBranches); err != nil {
		respond.Error(w, http.StatusUnprocessableEntity, err.Error())
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

func (h *BranchHandler) SaveAll(w http.ResponseWriter, r *http.Request) {
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

	var req saveBranchesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	candidateBranches := make([]*models.Branch, 0, len(req.Branches))
	params := make([]store.SaveBranchParams, 0, len(req.Branches))
	seenIDs := make(map[string]struct{}, len(req.Branches))
	seenKeys := make(map[string]struct{}, len(req.Branches))

	for _, branch := range req.Branches {
		branch.ID = strings.TrimSpace(branch.ID)
		branch.Key = strings.TrimSpace(branch.Key)
		branch.Name = strings.TrimSpace(branch.Name)

		if err := validateBranchKey(branch.Key); err != nil {
			respond.Error(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		if err := validateBranchName(branch.Name); err != nil {
			respond.Error(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		if err := validateBranchMetadataJSON(branch.MetadataJSON); err != nil {
			respond.Error(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		if err := validateBranchWeightValue(branch.Weight); err != nil {
			respond.Error(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		if branch.ID != "" {
			if _, ok := seenIDs[branch.ID]; ok {
				respond.Error(w, http.StatusUnprocessableEntity, "duplicate branch id in request")
				return
			}
			seenIDs[branch.ID] = struct{}{}
		}
		if _, ok := seenKeys[branch.Key]; ok {
			respond.Error(w, http.StatusConflict, "a branch with that key already exists for this experiment")
			return
		}
		seenKeys[branch.Key] = struct{}{}

		candidateBranches = append(candidateBranches, &models.Branch{Weight: branch.Weight})
		params = append(params, store.SaveBranchParams{
			ID:           branch.ID,
			Key:          branch.Key,
			Name:         branch.Name,
			Weight:       branch.Weight,
			MetadataJSON: branch.MetadataJSON,
		})
	}

	if err := validateBranchWeights(candidateBranches); err != nil {
		respond.Error(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	branches, err := h.store.SaveAll(r.Context(), experimentID, params)
	if errors.Is(err, store.ErrNotFound) {
		respond.Error(w, http.StatusNotFound, "branch not found")
		return
	}
	if errors.Is(err, store.ErrConflict) {
		respond.Error(w, http.StatusConflict, "a branch with that key already exists for this experiment")
		return
	}
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to save branches")
		return
	}

	respond.JSON(w, http.StatusOK, branches)
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
	if err := validateBranchName(req.Name); err != nil {
		respond.Error(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if err := validateBranchMetadataJSON(req.MetadataJSON); err != nil {
		respond.Error(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if err := validateBranchWeightValue(req.Weight); err != nil {
		respond.Error(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	existingBranches, err := h.store.ListByExperimentID(r.Context(), experimentID)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to validate branch weights")
		return
	}

	found := false
	candidateBranches := make([]*models.Branch, 0, len(existingBranches))
	for _, branch := range existingBranches {
		if branch.ID == id {
			found = true
			candidateBranches = append(candidateBranches, &models.Branch{
				ID:     branch.ID,
				Weight: req.Weight,
			})
			continue
		}
		candidateBranches = append(candidateBranches, branch)
	}
	if !found {
		respond.Error(w, http.StatusNotFound, "branch not found")
		return
	}
	if err := validateBranchWeights(candidateBranches); err != nil {
		respond.Error(w, http.StatusUnprocessableEntity, err.Error())
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

	existingBranches, err := h.store.ListByExperimentID(r.Context(), experimentID)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to validate branch weights")
		return
	}

	found := false
	remainingBranches := make([]*models.Branch, 0, len(existingBranches))
	for _, branch := range existingBranches {
		if branch.ID == id {
			found = true
			continue
		}
		remainingBranches = append(remainingBranches, branch)
	}
	if !found {
		respond.Error(w, http.StatusNotFound, "branch not found")
		return
	}
	if err := validateBranchWeights(remainingBranches); err != nil {
		respond.Error(w, http.StatusUnprocessableEntity, err.Error())
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
