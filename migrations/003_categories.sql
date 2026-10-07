-- 003_categories.sql
-- Tabel categories dan seed data awal.

CREATE TABLE IF NOT EXISTS categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(80) NOT NULL,
    description VARCHAR(255) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- BR-C1: Nama kategori unik tanpa membedakan huruf besar/kecil
CREATE UNIQUE INDEX IF NOT EXISTS idx_categories_lower_name ON categories (LOWER(name));

-- Seed Categories
INSERT INTO categories (name, description) VALUES
    ('Seminar', 'Seminar dan talkshow edukatif'),
    ('Workshop', 'Pelatihan praktis dan hands-on'),
    ('Kompetisi', 'Lomba dan kompetisi berbagai bidang'),
    ('Konser', 'Pertunjukan musik dan hiburan'),
    ('Volunteer', 'Kegiatan sosial dan relawan')
ON CONFLICT (LOWER(name)) DO NOTHING;
