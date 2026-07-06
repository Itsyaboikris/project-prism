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

type trackedEventStore interface {
	ListByExperimentID(ctx context.Context, experimentID string) ([]*models.TrackedEvent, error)
	GetByID(ctx context.Context, experimentID, id string) (*models.TrackedEvent, error)
	Create(ctx context.Context, p store.CreateTrackedEventParams) (*models.TrackedEvent, error)
	Update(ctx context.Context, experimentID, id string, p store.UpdateTrackedEventParams) (*models.TrackedEvent, error)
	Delete(ctx context.Context, experimentID, id string) error
}

type trackedEventExperimentStore interface {
	GetByID(ctx context.Context, applicationID, id string) (*models.Experiment, error)
}

type TrackedEventHandler struct {
	store    trackedEventStore
	expStore trackedEventExperimentStore
}

func NewTrackedEventHandler(s trackedEventStore, expStore trackedEventExperimentStore) *TrackedEventHandler {
	return &TrackedEventHandler{store: s, expStore: expStore}
}

type createTrackedEventRequest struct {
	Key         string  `json:"key"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

type updateTrackedEventRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

func (h *TrackedEventHandler) List(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "appID")
	experimentID := chi.URLParam(r, "id")

	if _, err := h.expStore.GetByID(r.Context(), appID, experimentID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respond.Error(w, http.StatusNotFound, "experiment not found")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "failed to verify experiment")
		return
	}

	trackedEvents, err := h.store.ListByExperimentID(r.Context(), experimentID)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to list tracked events")
		return
	}

	if trackedEvents == nil {
		trackedEvents = []*models.TrackedEvent{}
	}

	respond.JSON(w, http.StatusOK, trackedEvents)
}

func (h *TrackedEventHandler) Create(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "appID")
	experimentID := chi.URLParam(r, "id")

	if _, err := h.expStore.GetByID(r.Context(), appID, experimentID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respond.Error(w, http.StatusNotFound, "experiment not found")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "failed to verify experiment")
		return
	}

	var req createTrackedEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Key = strings.TrimSpace(req.Key)
	req.Name = strings.TrimSpace(req.Name)
	if req.Description != nil {
		trimmed := strings.TrimSpace(*req.Description)
		if trimmed == "" {
			req.Description = nil
		} else {
			req.Description = &trimmed
		}
	}

	if err := validateTrackedEventKey(req.Key); err != nil {
		respond.Error(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if err := validateTrackedEventName(req.Name); err != nil {
		respond.Error(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if err := validateTrackedEventDescription(req.Description); err != nil {
		respond.Error(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	trackedEvent, err := h.store.Create(r.Context(), store.CreateTrackedEventParams{
		ExperimentID: experimentID,
		Key:          req.Key,
		Name:         req.Name,
		Description:  req.Description,
	})
	if errors.Is(err, store.ErrConflict) {
		respond.Error(w, http.StatusConflict, "a tracked event with that key already exists for this experiment")
		return
	}
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to create tracked event")
		return
	}

	respond.JSON(w, http.StatusCreated, trackedEvent)
}

func (h *TrackedEventHandler) Update(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "appID")
	experimentID := chi.URLParam(r, "id")
	trackedEventID := chi.URLParam(r, "trackedEventID")

	if _, err := h.expStore.GetByID(r.Context(), appID, experimentID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respond.Error(w, http.StatusNotFound, "experiment not found")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "failed to verify experiment")
		return
	}

	var req updateTrackedEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Description != nil {
		trimmed := strings.TrimSpace(*req.Description)
		if trimmed == "" {
			req.Description = nil
		} else {
			req.Description = &trimmed
		}
	}

	if err := validateTrackedEventName(req.Name); err != nil {
		respond.Error(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if err := validateTrackedEventDescription(req.Description); err != nil {
		respond.Error(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	trackedEvent, err := h.store.Update(r.Context(), experimentID, trackedEventID, store.UpdateTrackedEventParams{
		Name:        req.Name,
		Description: req.Description,
	})
	if errors.Is(err, store.ErrNotFound) {
		respond.Error(w, http.StatusNotFound, "tracked event not found")
		return
	}
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to update tracked event")
		return
	}

	respond.JSON(w, http.StatusOK, trackedEvent)
}

func (h *TrackedEventHandler) Delete(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "appID")
	experimentID := chi.URLParam(r, "id")
	trackedEventID := chi.URLParam(r, "trackedEventID")

	if _, err := h.expStore.GetByID(r.Context(), appID, experimentID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respond.Error(w, http.StatusNotFound, "experiment not found")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "failed to verify experiment")
		return
	}

	if err := h.store.Delete(r.Context(), experimentID, trackedEventID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respond.Error(w, http.StatusNotFound, "tracked event not found")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "failed to delete tracked event")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
