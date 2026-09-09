-- Background jobs for automatic order status progression.

CREATE TABLE IF NOT EXISTS order_jobs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id      UUID NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    from_status   INTEGER NOT NULL,
    to_status     INTEGER NOT NULL,
    run_at        TIMESTAMPTZ NOT NULL,
    status        TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'done', 'failed', 'cancelled')),
    attempts      INTEGER NOT NULL DEFAULT 0,
    last_error    TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_order_jobs_pending_run
    ON order_jobs (run_at)
    WHERE status = 'pending';

CREATE INDEX IF NOT EXISTS idx_order_jobs_order_id ON order_jobs (order_id);
