-- 001_rbac.sql
-- Tabel roles, permissions, role_permissions dan data awal (seed).

CREATE TABLE IF NOT EXISTS roles (
    name VARCHAR(20) PRIMARY KEY,
    description VARCHAR(150) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS permissions (
    name VARCHAR(50) PRIMARY KEY,
    description VARCHAR(150) NOT NULL
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_name VARCHAR(20) NOT NULL REFERENCES roles(name) ON DELETE CASCADE,
    permission_name VARCHAR(50) NOT NULL REFERENCES permissions(name) ON DELETE CASCADE,
    PRIMARY KEY (role_name, permission_name)
);

-- Seed Roles
INSERT INTO roles (name, description) VALUES
    ('admin', 'Pengelola platform'),
    ('organizer', 'Penyelenggara event'),
    ('participant', 'Peserta event')
ON CONFLICT (name) DO NOTHING;

-- Seed Permissions (sesuai PRD 7.1)
INSERT INTO permissions (name, description) VALUES
    ('user:list', 'Melihat daftar seluruh pengguna'),
    ('user:read:any', 'Melihat detail profil pengguna mana pun'),
    ('user:update:any', 'Mengubah profil pengguna mana pun'),
    ('user:delete', 'Menghapus pengguna'),
    ('role:assign', 'Mengubah role pengguna'),
    ('category:create', 'Membuat kategori event baru'),
    ('category:update', 'Mengubah data kategori event'),
    ('category:delete', 'Menghapus kategori event'),
    ('event:create', 'Membuat event baru'),
    ('event:read:any', 'Melihat event mana pun termasuk draft'),
    ('event:update:any', 'Mengubah event mana pun'),
    ('event:delete:any', 'Menghapus event mana pun'),
    ('registration:create', 'Mendaftar pada suatu event'),
    ('registration:read:any', 'Melihat seluruh data pendaftaran event'),
    ('registration:manage:any', 'Mengelola pendaftaran, verifikasi bayar, dan check-in mana pun')
ON CONFLICT (name) DO NOTHING;

-- Seed Role Permissions (admin mendapatkan semua permission manajemen, organizer event:create, participant registration:create)
INSERT INTO role_permissions (role_name, permission_name) VALUES
    ('admin', 'user:list'),
    ('admin', 'user:read:any'),
    ('admin', 'user:update:any'),
    ('admin', 'user:delete'),
    ('admin', 'role:assign'),
    ('admin', 'category:create'),
    ('admin', 'category:update'),
    ('admin', 'category:delete'),
    ('admin', 'event:create'),
    ('admin', 'event:read:any'),
    ('admin', 'event:update:any'),
    ('admin', 'event:delete:any'),
    ('admin', 'registration:read:any'),
    ('admin', 'registration:manage:any'),
    ('organizer', 'event:create'),
    ('participant', 'registration:create')
ON CONFLICT (role_name, permission_name) DO NOTHING;
