-- Delivery: user addresses, pickup points, order delivery snapshot.

CREATE TABLE IF NOT EXISTS user_addresses (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    title           TEXT NOT NULL DEFAULT '',
    city            TEXT NOT NULL,
    street          TEXT NOT NULL,
    building        TEXT NOT NULL,
    apartment       TEXT NOT NULL DEFAULT '',
    postal_code     TEXT NOT NULL DEFAULT '',
    recipient_name  TEXT NOT NULL,
    phone           TEXT NOT NULL,
    is_default      BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_addresses_user_id ON user_addresses (user_id);

CREATE TABLE IF NOT EXISTS pickup_points (
    id          UUID PRIMARY KEY,
    code        TEXT NOT NULL UNIQUE,
    city        TEXT NOT NULL,
    address     TEXT NOT NULL,
    work_hours  TEXT NOT NULL DEFAULT '',
    active      BOOLEAN NOT NULL DEFAULT TRUE
);

INSERT INTO pickup_points (id, code, city, address, work_hours, active) VALUES
    ('660e8400-e29b-41d4-a716-446655440001', 'MSK-001', 'Москва', 'ул. Тверская, 12', '10:00–22:00', TRUE),
    ('660e8400-e29b-41d4-a716-446655440002', 'MSK-002', 'Москва', 'Кутузовский пр-т, 48', '09:00–21:00', TRUE),
    ('660e8400-e29b-41d4-a716-446655440003', 'MSK-003', 'Москва', 'ул. Арбат, 25', '10:00–22:00', TRUE),
    ('660e8400-e29b-41d4-a716-446655440004', 'SPB-001', 'Санкт-Петербург', 'Невский пр-т, 28', '10:00–22:00', TRUE),
    ('660e8400-e29b-41d4-a716-446655440005', 'SPB-002', 'Санкт-Петербург', 'ул. Рубинштейна, 15', '09:00–21:00', TRUE),
    ('660e8400-e29b-41d4-a716-446655440006', 'MSK-X01', 'Москва', 'закрытый ПВЗ (тест)', '—', FALSE)
ON CONFLICT (id) DO NOTHING;

ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS delivery_method INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS delivery_fee_cents BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS delivery_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb;
