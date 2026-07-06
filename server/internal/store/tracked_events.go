package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"project-prism/server/internal/models"
)

type CreateTrackedEventParams struct {
	ExperimentID string
	Key          string
	Name         string
	Description  *string
}

type UpdateTrackedEventParams struct {
	Name        string
	Description *string
}

type TrackedEventStore struct {
	pool DB
}

func NewTrackedEventStore(pool DB) *TrackedEventStore {
	return &TrackedEventStore{pool: pool}
}

func (s *TrackedEventStore) ListByExperimentID(ctx context.Context, experimentID string) ([]*models.TrackedEvent, error) {
	const q = `
		SELECT
			te.id,
			te.experiment_id,
			te.key,
			te.name,
			te.description,
			te.created_at,
			te.updated_at,
			COUNT(e.id)::bigint AS occurrence_count,
			MAX(e.occurred_at) AS last_occurred_at
		FROM tracked_events te
		LEFT JOIN events e
			ON e.experiment_id = te.experiment_id
			AND e.event_name = te.key
		WHERE te.experiment_id = $1
		  AND te.deleted_at IS NULL
		GROUP BY te.id
		ORDER BY te.name`

	rows, err := s.pool.Query(ctx, q, experimentID)
	if err != nil {
		return nil, fmt.Errorf("list tracked events: %w", err)
	}
	defer rows.Close()

	var trackedEvents []*models.TrackedEvent
	for rows.Next() {
		item, err := scanTrackedEvent(rows)
		if err != nil {
			return nil, err
		}
		trackedEvents = append(trackedEvents, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list tracked events rows: %w", err)
	}

	return trackedEvents, nil
}

func (s *TrackedEventStore) GetByID(ctx context.Context, experimentID, id string) (*models.TrackedEvent, error) {
	const q = `
		SELECT
			te.id,
			te.experiment_id,
			te.key,
			te.name,
			te.description,
			te.created_at,
			te.updated_at,
			COUNT(e.id)::bigint AS occurrence_count,
			MAX(e.occurred_at) AS last_occurred_at
		FROM tracked_events te
		LEFT JOIN events e
			ON e.experiment_id = te.experiment_id
			AND e.event_name = te.key
		WHERE te.id = $1
		  AND te.experiment_id = $2
		  AND te.deleted_at IS NULL
		GROUP BY te.id`

	item, err := scanTrackedEvent(s.pool.QueryRow(ctx, q, id, experimentID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get tracked event: %w", err)
	}

	return item, nil
}

func (s *TrackedEventStore) Create(ctx context.Context, p CreateTrackedEventParams) (*models.TrackedEvent, error) {
	const q = `
		INSERT INTO tracked_events (experiment_id, key, name, description)
		VALUES ($1, $2, $3, $4)
		RETURNING id, experiment_id, key, name, description, created_at, updated_at`

	item := &models.TrackedEvent{}
	err := s.pool.QueryRow(ctx, q, p.ExperimentID, p.Key, p.Name, p.Description).Scan(
		&item.ID,
		&item.ExperimentID,
		&item.Key,
		&item.Name,
		&item.Description,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return nil, classifyTrackedEventErr("create tracked event", err)
	}

	return item, nil
}

func (s *TrackedEventStore) Update(ctx context.Context, experimentID, id string, p UpdateTrackedEventParams) (*models.TrackedEvent, error) {
	const q = `
		UPDATE tracked_events
		SET name = $1, description = $2, updated_at = NOW()
		WHERE id = $3 AND experiment_id = $4 AND deleted_at IS NULL
		RETURNING id, experiment_id, key, name, description, created_at, updated_at`

	item := &models.TrackedEvent{}
	err := s.pool.QueryRow(ctx, q, p.Name, p.Description, id, experimentID).Scan(
		&item.ID,
		&item.ExperimentID,
		&item.Key,
		&item.Name,
		&item.Description,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, classifyTrackedEventErr("update tracked event", err)
	}

	return item, nil
}

func (s *TrackedEventStore) Delete(ctx context.Context, experimentID, id string) error {
	const q = `
		UPDATE tracked_events
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND experiment_id = $2 AND deleted_at IS NULL`

	tag, err := s.pool.Exec(ctx, q, id, experimentID)
	if err != nil {
		return fmt.Errorf("delete tracked event: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *TrackedEventStore) IsRegistered(ctx context.Context, experimentID, key string) (bool, error) {
	const q = `
		SELECT EXISTS (
			SELECT 1
			FROM tracked_events
			WHERE experiment_id = $1
			  AND key = $2
			  AND deleted_at IS NULL
		)`

	var exists bool
	if err := s.pool.QueryRow(ctx, q, experimentID, key).Scan(&exists); err != nil {
		return false, fmt.Errorf("check tracked event registration: %w", err)
	}

	return exists, nil
}

type trackedEventScanner interface {
	Scan(dest ...any) error
}

func scanTrackedEvent(row trackedEventScanner) (*models.TrackedEvent, error) {
	item := &models.TrackedEvent{}
	var lastOccurredAt sql.NullTime
	if err := row.Scan(
		&item.ID,
		&item.ExperimentID,
		&item.Key,
		&item.Name,
		&item.Description,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.OccurrenceCount,
		&lastOccurredAt,
	); err != nil {
		return nil, fmt.Errorf("scan tracked event: %w", err)
	}
	if lastOccurredAt.Valid {
		item.LastOccurredAt = &lastOccurredAt.Time
	}

	return item, nil
}

func classifyTrackedEventErr(op string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return ErrConflict
		case "23503":
			return ErrNotFound
		}
	}
	return fmt.Errorf("%s: %w", op, err)
}
