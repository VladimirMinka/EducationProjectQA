package worker

import (
	"context"
	"log/slog"
	"os"
	"time"

	"awesomeProject/internal/logging"
	"awesomeProject/internal/repository"
	"awesomeProject/internal/service"
)

type OrderStore interface {
	GetOrder(orderID string) (repository.Order, error)
	UpdateOrderStatus(orderID string, fromStatus, toStatus int32) (repository.Order, error)
}

type JobStore interface {
	ClaimDueJobs(limit int) ([]repository.OrderJob, error)
	Enqueue(job repository.OrderJob) (repository.OrderJob, error)
	MarkDone(jobID string) error
	MarkCancelled(jobID, reason string) error
	MarkFailed(jobID string, attempts int32, reason string) error
	BumpAttempts(jobID string) error
}

type OrderStatusWorker struct {
	orders OrderStore
	jobs   JobStore
	delay  time.Duration
	poll   time.Duration
	log    *slog.Logger
}

func NewOrderStatusWorker(orders OrderStore, jobs JobStore, delay time.Duration) *OrderStatusWorker {
	poll := 2 * time.Second
	if raw := os.Getenv("ORDER_JOB_POLL_INTERVAL"); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil && d > 0 {
			poll = d
		}
	}
	return &OrderStatusWorker{
		orders: orders,
		jobs:   jobs,
		delay:  delay,
		poll:   poll,
		log:    logging.Logger,
	}
}

func (w *OrderStatusWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.poll)
	defer ticker.Stop()
	w.log.Info("order status worker started", "poll", w.poll.String(), "delay", w.delay.String())
	for {
		select {
		case <-ctx.Done():
			w.log.Info("order status worker stopped")
			return
		case <-ticker.C:
			w.tick()
		}
	}
}

func (w *OrderStatusWorker) tick() {
	jobs, err := w.jobs.ClaimDueJobs(20)
	if err != nil {
		w.log.Error("claim order jobs", "error", err)
		return
	}
	for _, job := range jobs {
		w.process(job)
	}
}

func (w *OrderStatusWorker) process(job repository.OrderJob) {
	_ = w.jobs.BumpAttempts(job.ID)

	order, err := w.orders.GetOrder(job.OrderID)
	if err != nil {
		_ = w.jobs.MarkCancelled(job.ID, "order not found")
		w.log.Info("order job cancelled", "job_id", job.ID, "order_id", job.OrderID, "reason", "order not found")
		return
	}
	if order.Status != job.FromStatus {
		_ = w.jobs.MarkCancelled(job.ID, "status mismatch")
		w.log.Info("order job cancelled",
			"job_id", job.ID,
			"order_id", job.OrderID,
			"from", job.FromStatus,
			"to", job.ToStatus,
			"actual_status", order.Status,
			"reason", "status mismatch",
		)
		return
	}

	updated, err := w.orders.UpdateOrderStatus(job.OrderID, job.FromStatus, job.ToStatus)
	if err != nil {
		_ = w.jobs.MarkFailed(job.ID, job.Attempts+1, err.Error())
		w.log.Error("order job failed", "job_id", job.ID, "order_id", job.OrderID, "error", err)
		return
	}

	if err := w.jobs.MarkDone(job.ID); err != nil {
		w.log.Error("mark job done", "job_id", job.ID, "error", err)
	}

	w.log.Info("order status transition",
		"job_id", job.ID,
		"order_id", job.OrderID,
		"from", job.FromStatus,
		"to", job.ToStatus,
	)

	if updated.Status == service.OrderStatusShipped {
		_, err := w.jobs.Enqueue(repository.OrderJob{
			OrderID:    job.OrderID,
			FromStatus: service.OrderStatusShipped,
			ToStatus:   service.OrderStatusCompleted,
			RunAt:      time.Now().Add(w.delay),
		})
		if err != nil {
			w.log.Error("enqueue completed job", "order_id", job.OrderID, "error", err)
		}
	}
}
