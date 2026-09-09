package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"awesomeProject/internal/repository"
)

type OrderJobRepository struct {
	db *sql.DB
}

func NewOrderJobRepository(db *sql.DB) *OrderJobRepository {
	return &OrderJobRepository{db: db}
}

func (r *OrderJobRepository) Enqueue(job repository.OrderJob) (repository.OrderJob, error) {
	job.ID = uuid.NewString()
	job.Status = repository.OrderJobPending
	job.CreatedAt = time.Now()
	const q = `
		INSERT INTO order_jobs (id, order_id, from_status, to_status, run_at, status, attempts, last_error, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, 0, '', $7)
	`
	if _, err := r.db.ExecContext(context.Background(), q,
		job.ID, job.OrderID, job.FromStatus, job.ToStatus, job.RunAt, job.Status, job.CreatedAt,
	); err != nil {
		return repository.OrderJob{}, fmt.Errorf("enqueue order job: %w", err)
	}
	return job, nil
}

func (r *OrderJobRepository) ListByOrder(orderID string) ([]repository.OrderJob, error) {
	const q = `
		SELECT id, order_id, from_status, to_status, run_at, status, attempts, last_error, created_at, processed_at
		FROM order_jobs
		WHERE order_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.db.QueryContext(context.Background(), q, orderID)
	if err != nil {
		return nil, fmt.Errorf("list order jobs: %w", err)
	}
	defer rows.Close()

	out := make([]repository.OrderJob, 0)
	for rows.Next() {
		job, err := scanOrderJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, job)
	}
	return out, rows.Err()
}

// ClaimDueJobs locks and returns pending jobs ready to run.
func (r *OrderJobRepository) ClaimDueJobs(limit int) ([]repository.OrderJob, error) {
	if limit <= 0 {
		limit = 10
	}
	ctx := context.Background()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	const q = `
		SELECT id, order_id, from_status, to_status, run_at, status, attempts, last_error, created_at, processed_at
		FROM order_jobs
		WHERE status = 'pending' AND run_at <= NOW()
		ORDER BY run_at ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`
	rows, err := tx.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, fmt.Errorf("claim order jobs: %w", err)
	}
	defer rows.Close()

	jobs := make([]repository.OrderJob, 0)
	for rows.Next() {
		job, err := scanOrderJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return jobs, nil
}

func (r *OrderJobRepository) MarkDone(jobID string) error {
	_, err := r.db.ExecContext(context.Background(), `
		UPDATE order_jobs
		SET status = 'done', processed_at = NOW(), last_error = ''
		WHERE id = $1
	`, jobID)
	return err
}

func (r *OrderJobRepository) MarkCancelled(jobID, reason string) error {
	_, err := r.db.ExecContext(context.Background(), `
		UPDATE order_jobs
		SET status = 'cancelled', processed_at = NOW(), last_error = $2
		WHERE id = $1
	`, jobID, reason)
	return err
}

func (r *OrderJobRepository) MarkFailed(jobID string, attempts int32, reason string) error {
	_, err := r.db.ExecContext(context.Background(), `
		UPDATE order_jobs
		SET status = 'failed', attempts = $2, processed_at = NOW(), last_error = $3
		WHERE id = $1
	`, jobID, attempts, reason)
	return err
}

func (r *OrderJobRepository) BumpAttempts(jobID string) error {
	_, err := r.db.ExecContext(context.Background(), `
		UPDATE order_jobs SET attempts = attempts + 1 WHERE id = $1
	`, jobID)
	return err
}

func scanOrderJob(row rowScanner) (repository.OrderJob, error) {
	var job repository.OrderJob
	var processed sql.NullTime
	err := row.Scan(
		&job.ID, &job.OrderID, &job.FromStatus, &job.ToStatus, &job.RunAt,
		&job.Status, &job.Attempts, &job.LastError, &job.CreatedAt, &processed,
	)
	if err != nil {
		return repository.OrderJob{}, err
	}
	if processed.Valid {
		t := processed.Time
		job.ProcessedAt = &t
	}
	return job, nil
}
