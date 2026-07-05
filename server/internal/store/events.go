package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"project-prism/server/internal/models"
)

type CreateEventParams struct {
	ApplicationID  string
	UserID         string
	EventName      string
	ExperimentKey  string
	PropertiesJSON json.RawMessage
}

type ListEventsParams struct {
	ApplicationID string
	ExperimentID  string
	EventName     string
	Limit         int
	Offset        int
}

type EventStore struct {
	pool DB
}

type queryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func NewEventStore(pool DB) *EventStore {
	return &EventStore{pool: pool}
}

func (s *EventStore) Create(ctx context.Context, p CreateEventParams) (*models.Event, error) {
	var experimentID *string
	var branchID *string

	if p.ExperimentKey != "" {
		resolvedExperimentID, err := getExperimentIDByKey(ctx, s.pool, p.ApplicationID, p.ExperimentKey)
		if err != nil {
			return nil, err
		}
		experimentID = &resolvedExperimentID

		assignedBranchID, err := getAssignedBranchID(ctx, s.pool, resolvedExperimentID, p.UserID)
		if err != nil {
			return nil, err
		}
		if assignedBranchID != "" {
			branchID = &assignedBranchID
		}
	}

	const q = `
		INSERT INTO events (application_id, experiment_id, branch_id, user_id, event_name, properties_json)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, application_id, experiment_id, branch_id, user_id, event_name, properties_json, occurred_at, created_at`

	event := &models.Event{}
	var properties []byte
	var experimentIDScan, branchIDScan sql.NullString
	err := s.pool.QueryRow(ctx, q,
		p.ApplicationID,
		nullableString(experimentID),
		nullableString(branchID),
		p.UserID,
		p.EventName,
		nullRawJSON(p.PropertiesJSON),
	).Scan(
		&event.ID,
		&event.ApplicationID,
		&experimentIDScan,
		&branchIDScan,
		&event.UserID,
		&event.EventName,
		&properties,
		&event.OccurredAt,
		&event.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create event: %w", err)
	}
	if experimentIDScan.Valid {
		event.ExperimentID = &experimentIDScan.String
	}
	if branchIDScan.Valid {
		event.BranchID = &branchIDScan.String
	}
	if len(properties) > 0 {
		event.Properties = json.RawMessage(properties)
	}

	return event, nil
}

func (s *EventStore) ListByExperiment(ctx context.Context, p ListEventsParams) (*models.ExperimentEventsView, error) {
	experiment, err := s.getExperimentView(ctx, p.ApplicationID, p.ExperimentID)
	if err != nil {
		return nil, err
	}

	const q = `
		SELECT e.id, e.application_id, e.experiment_id, e.branch_id, e.user_id, e.event_name,
		       e.properties_json, e.occurred_at, e.created_at, b.key, b.name
		FROM events e
		LEFT JOIN branches b ON b.id = e.branch_id
		WHERE e.application_id = $1
		  AND e.experiment_id = $2
		  AND ($3 = '' OR e.event_name = $3)
		ORDER BY e.occurred_at DESC, e.id DESC
		LIMIT $4 OFFSET $5`

	rows, err := s.pool.Query(ctx, q, p.ApplicationID, p.ExperimentID, p.EventName, p.Limit, p.Offset)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()

	view := &models.ExperimentEventsView{
		ExperimentID:     experiment.ID,
		ExperimentKey:    experiment.Key,
		ExperimentName:   experiment.Name,
		ExperimentStatus: experiment.Status,
		Events:           []*models.ExperimentEventListItem{},
	}

	for rows.Next() {
		item := &models.ExperimentEventListItem{}
		var properties []byte
		var experimentIDScan, branchIDScan, branchKeyScan, branchNameScan sql.NullString
		if err := rows.Scan(
			&item.ID,
			&item.ApplicationID,
			&experimentIDScan,
			&branchIDScan,
			&item.UserID,
			&item.EventName,
			&properties,
			&item.OccurredAt,
			&item.CreatedAt,
			&branchKeyScan,
			&branchNameScan,
		); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		if experimentIDScan.Valid {
			item.ExperimentID = &experimentIDScan.String
		}
		if branchIDScan.Valid {
			item.BranchID = &branchIDScan.String
		}
		if branchKeyScan.Valid {
			item.BranchKey = &branchKeyScan.String
		}
		if branchNameScan.Valid {
			item.BranchName = &branchNameScan.String
		}
		if len(properties) > 0 {
			item.Properties = json.RawMessage(properties)
		}
		view.Events = append(view.Events, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list events rows: %w", err)
	}

	return view, nil
}

func (s *EventStore) GetEventMetricsByExperiment(ctx context.Context, experimentID, eventName string) (map[string]models.EventBranchMetrics, error) {
	const q = `
		SELECT b.id, COUNT(e.id)::bigint, COUNT(DISTINCT e.user_id)::bigint
		FROM branches b
		LEFT JOIN events e
		  ON e.branch_id = b.id
		 AND e.experiment_id = $1
		 AND e.event_name = $2
		WHERE b.experiment_id = $1
		  AND b.deleted_at IS NULL
		GROUP BY b.id`

	rows, err := s.pool.Query(ctx, q, experimentID, eventName)
	if err != nil {
		return nil, fmt.Errorf("get event metrics: %w", err)
	}
	defer rows.Close()

	metrics := make(map[string]models.EventBranchMetrics)
	for rows.Next() {
		var branchID string
		var eventCount int64
		var uniqueUsers int64
		if err := rows.Scan(&branchID, &eventCount, &uniqueUsers); err != nil {
			return nil, fmt.Errorf("scan event metrics: %w", err)
		}
		metrics[branchID] = models.EventBranchMetrics{
			BranchID:         branchID,
			EventCount:       int(eventCount),
			UniqueEventUsers: int(uniqueUsers),
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("event metrics rows: %w", err)
	}

	return metrics, nil
}

type eventExperimentView struct {
	ID     string
	Key    string
	Name   string
	Status models.ExperimentStatus
}

func (s *EventStore) getExperimentView(ctx context.Context, applicationID, experimentID string) (*eventExperimentView, error) {
	const q = `
		SELECT id, key, name, status
		FROM experiments
		WHERE application_id = $1
		  AND id = $2
		  AND deleted_at IS NULL`

	experiment := &eventExperimentView{}
	err := s.pool.QueryRow(ctx, q, applicationID, experimentID).Scan(
		&experiment.ID,
		&experiment.Key,
		&experiment.Name,
		&experiment.Status,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get experiment view: %w", err)
	}

	return experiment, nil
}

func getExperimentIDByKey(ctx context.Context, q queryer, applicationID, experimentKey string) (string, error) {
	const query = `
		SELECT id
		FROM experiments
		WHERE application_id = $1 AND key = $2 AND deleted_at IS NULL`

	var experimentID string
	err := q.QueryRow(ctx, query, applicationID, experimentKey).Scan(&experimentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("get experiment id by key: %w", err)
	}

	return experimentID, nil
}

func getAssignedBranchID(ctx context.Context, q queryer, experimentID, userID string) (string, error) {
	const query = `
		SELECT a.branch_id
		FROM assignments a
		JOIN branches b ON b.id = a.branch_id
		WHERE a.experiment_id = $1
		  AND a.user_id = $2
		  AND b.deleted_at IS NULL`

	var branchID string
	err := q.QueryRow(ctx, query, experimentID, userID).Scan(&branchID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get assigned branch id: %w", err)
	}

	return branchID, nil
}

func nullRawJSON(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	return []byte(raw)
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}
