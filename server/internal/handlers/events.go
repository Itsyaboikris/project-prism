package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"project-prism/server/internal/apiauth"
	"project-prism/server/internal/models"
	"project-prism/server/internal/respond"
	"project-prism/server/internal/store"
)

type eventStore interface {
	Create(ctx context.Context, p store.CreateEventParams) (*models.Event, error)
	ListByExperiment(ctx context.Context, p store.ListEventsParams) (*models.ExperimentEventsView, error)
}

type EventHandler struct {
	store eventStore
}

func NewEventHandler(s eventStore) *EventHandler {
	return &EventHandler{store: s}
}

type createEventRequest struct {
	UserID        string          `json:"user_id"`
	EventName     string          `json:"event_name"`
	ExperimentKey string          `json:"experiment_key"`
	Properties    json.RawMessage `json:"properties"`
}

func (h *EventHandler) Create(w http.ResponseWriter, r *http.Request) {
	app, ok := apiauth.ApplicationFromContext(r.Context())
	if !ok {
		respond.Error(w, http.StatusInternalServerError, "application context missing")
		return
	}

	var req createEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.UserID = strings.TrimSpace(req.UserID)
	req.EventName = strings.TrimSpace(req.EventName)
	req.ExperimentKey = strings.TrimSpace(req.ExperimentKey)

	if req.UserID == "" {
		respond.Error(w, http.StatusUnprocessableEntity, "user_id is required")
		return
	}
	if err := validateEventName(req.EventName); err != nil {
		respond.Error(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if err := validateEventPropertiesJSON(req.Properties); err != nil {
		respond.Error(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	event, err := h.store.Create(r.Context(), store.CreateEventParams{
		ApplicationID:  app.ID,
		UserID:         req.UserID,
		EventName:      req.EventName,
		ExperimentKey:  req.ExperimentKey,
		PropertiesJSON: req.Properties,
	})
	if errors.Is(err, store.ErrNotFound) {
		respond.Error(w, http.StatusNotFound, "experiment not found")
		return
	}
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to create event")
		return
	}

	respond.JSON(w, http.StatusCreated, event)
}

func (h *EventHandler) ListByExperiment(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "appID")
	experimentID := chi.URLParam(r, "id")

	limit := 100
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			respond.Error(w, http.StatusUnprocessableEntity, "limit must be a positive integer")
			return
		}
		if parsed > 500 {
			parsed = 500
		}
		limit = parsed
	}

	offset := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("offset")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			respond.Error(w, http.StatusUnprocessableEntity, "offset must be a non-negative integer")
			return
		}
		offset = parsed
	}

	events, err := h.store.ListByExperiment(r.Context(), store.ListEventsParams{
		ApplicationID: appID,
		ExperimentID:  experimentID,
		EventName:     strings.TrimSpace(r.URL.Query().Get("event_name")),
		Limit:         limit,
		Offset:        offset,
	})
	if errors.Is(err, store.ErrNotFound) {
		respond.Error(w, http.StatusNotFound, "experiment not found")
		return
	}
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to list events")
		return
	}

	respond.JSON(w, http.StatusOK, events)
}
