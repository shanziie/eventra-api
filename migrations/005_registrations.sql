-- 005_registrations.sql
-- Tabel registrations dan payments beserta indeks dan constraint bisnis.

CREATE TABLE IF NOT EXISTS registrations (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    event_id INT NOT NULL REFERENCES events(id) ON DELETE RESTRICT,
    ticket_type_id INT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','confirmed','checked_in','cancelled')),
    price_at_purchase BIGINT NOT NULL CHECK (price_at_purchase >= 0),
    ticket_code VARCHAR(32) NULL UNIQUE,
    confirmed_at TIMESTAMPTZ NULL,
    checked_in_at TIMESTAMPTZ NULL,
    cancelled_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (ticket_type_id, event_id) REFERENCES ticket_types(id, event_id) ON DELETE RESTRICT
);

-- BR-R3: Satu user hanya boleh punya satu registrasi aktif (non-cancelled) per event
CREATE UNIQUE INDEX IF NOT EXISTS idx_registrations_user_event_active
ON registrations (user_id, event_id) WHERE status <> 'cancelled';

CREATE INDEX IF NOT EXISTS idx_registrations_event_cursor ON registrations (event_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_registrations_user ON registrations (user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS payments (
    id SERIAL PRIMARY KEY,
    registration_id INT NOT NULL UNIQUE REFERENCES registrations(id) ON DELETE CASCADE,
    amount BIGINT NOT NULL CHECK (amount >= 0),
    method VARCHAR(30) NULL,
    reference_number VARCHAR(60) NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'unpaid' CHECK (status IN ('unpaid','submitted','verified','rejected')),
    note VARCHAR(200) NOT NULL DEFAULT '',
    submitted_at TIMESTAMPTZ NULL,
    verified_at TIMESTAMPTZ NULL,
    verified_by INT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payments_registration ON payments (registration_id);
