-- 004_events.sql
-- Tabel events dan ticket_types beserta indeks dan constraint bisnis.

CREATE TABLE IF NOT EXISTS events (
    id SERIAL PRIMARY KEY,
    organizer_id INT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    category_id INT NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
    title VARCHAR(150) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    location VARCHAR(200) NOT NULL,
    capacity INT NOT NULL CHECK (capacity > 0),
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    registration_deadline TIMESTAMPTZ NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','published','cancelled','finished')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_event_times CHECK (ends_at > starts_at),
    CONSTRAINT chk_registration_deadline CHECK (registration_deadline <= starts_at)
);

CREATE INDEX IF NOT EXISTS idx_events_cursor ON events (created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_events_organizer ON events (organizer_id);
CREATE INDEX IF NOT EXISTS idx_events_category ON events (category_id);
CREATE INDEX IF NOT EXISTS idx_events_status ON events (status);

CREATE TABLE IF NOT EXISTS ticket_types (
    id SERIAL PRIMARY KEY,
    event_id INT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    name VARCHAR(80) NOT NULL,
    price BIGINT NOT NULL CHECK (price >= 0),
    quota INT NOT NULL CHECK (quota > 0),
    sold INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_ticket_sold_quota CHECK (sold >= 0 AND sold <= quota),
    CONSTRAINT uq_ticket_id_event UNIQUE (id, event_id)
);

-- BR-T1: Nama jenis tiket unik per event
CREATE UNIQUE INDEX IF NOT EXISTS idx_ticket_types_event_lower_name ON ticket_types (event_id, LOWER(name));
CREATE INDEX IF NOT EXISTS idx_ticket_types_event ON ticket_types (event_id);
