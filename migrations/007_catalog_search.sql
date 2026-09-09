-- Categories and server-side catalog filters.

CREATE TABLE IF NOT EXISTS categories (
    id    UUID PRIMARY KEY,
    slug  TEXT NOT NULL UNIQUE,
    name  TEXT NOT NULL
);

INSERT INTO categories (id, slug, name) VALUES
    ('770e8400-e29b-41d4-a716-446655440001', 'phones', 'Смартфоны'),
    ('770e8400-e29b-41d4-a716-446655440002', 'tablets', 'Планшеты'),
    ('770e8400-e29b-41d4-a716-446655440003', 'laptops', 'Ноутбуки'),
    ('770e8400-e29b-41d4-a716-446655440004', 'wearables', 'Гаджеты'),
    ('770e8400-e29b-41d4-a716-446655440005', 'gpu', 'Видеокарты'),
    ('770e8400-e29b-41d4-a716-446655440006', 'storage', 'Накопители и прочее')
ON CONFLICT (id) DO NOTHING;

ALTER TABLE products
    ADD COLUMN IF NOT EXISTS category_id UUID REFERENCES categories (id);

-- phones: iPhone 15/Pro/Max, Galaxy S24/+/Ultra, Z Fold/Flip
UPDATE products SET category_id = '770e8400-e29b-41d4-a716-446655440001'
WHERE id IN (
    '550e8400-e29b-41d4-a716-446655440001',
    '550e8400-e29b-41d4-a716-446655440002',
    '550e8400-e29b-41d4-a716-446655440003',
    '550e8400-e29b-41d4-a716-446655440016',
    '550e8400-e29b-41d4-a716-446655440017',
    '550e8400-e29b-41d4-a716-446655440018',
    '550e8400-e29b-41d4-a716-446655440019',
    '550e8400-e29b-41d4-a716-446655440020'
);

-- tablets
UPDATE products SET category_id = '770e8400-e29b-41d4-a716-446655440002'
WHERE id IN (
    '550e8400-e29b-41d4-a716-446655440007',
    '550e8400-e29b-41d4-a716-446655440008',
    '550e8400-e29b-41d4-a716-446655440009',
    '550e8400-e29b-41d4-a716-446655440021',
    '550e8400-e29b-41d4-a716-446655440022'
);

-- laptops
UPDATE products SET category_id = '770e8400-e29b-41d4-a716-446655440003'
WHERE id IN (
    '550e8400-e29b-41d4-a716-446655440004',
    '550e8400-e29b-41d4-a716-446655440005',
    '550e8400-e29b-41d4-a716-446655440006',
    '550e8400-e29b-41d4-a716-446655440026'
);

-- wearables: watches, earbuds, AirPods, Buds
UPDATE products SET category_id = '770e8400-e29b-41d4-a716-446655440004'
WHERE id IN (
    '550e8400-e29b-41d4-a716-446655440010',
    '550e8400-e29b-41d4-a716-446655440011',
    '550e8400-e29b-41d4-a716-446655440012',
    '550e8400-e29b-41d4-a716-446655440013',
    '550e8400-e29b-41d4-a716-446655440023',
    '550e8400-e29b-41d4-a716-446655440024',
    '550e8400-e29b-41d4-a716-446655440025'
);

-- gpu: NVIDIA GPUs + AMD Radeon
UPDATE products SET category_id = '770e8400-e29b-41d4-a716-446655440005'
WHERE id IN (
    '550e8400-e29b-41d4-a716-446655440031',
    '550e8400-e29b-41d4-a716-446655440032',
    '550e8400-e29b-41d4-a716-446655440033',
    '550e8400-e29b-41d4-a716-446655440034',
    '550e8400-e29b-41d4-a716-446655440035',
    '550e8400-e29b-41d4-a716-446655440036',
    '550e8400-e29b-41d4-a716-446655440045',
    '550e8400-e29b-41d4-a716-446655440046',
    '550e8400-e29b-41d4-a716-446655440047'
);

-- storage / other: rest
UPDATE products SET category_id = '770e8400-e29b-41d4-a716-446655440006'
WHERE category_id IS NULL;

-- One out-of-stock SKU for in_stock filter tests
UPDATE products SET stock_quantity = 0
WHERE id = '550e8400-e29b-41d4-a716-446655440015';

CREATE INDEX IF NOT EXISTS idx_products_brand ON products (brand);
CREATE INDEX IF NOT EXISTS idx_products_price ON products (price_cents);
CREATE INDEX IF NOT EXISTS idx_products_category ON products (category_id);
CREATE INDEX IF NOT EXISTS idx_products_stock ON products (stock_quantity);
