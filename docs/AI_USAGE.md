# Catatan Penggunaan AI — Eventra API

Sesuai aturan di `AGENTS.md` bagian 5 dan 7, penggunaan AI dicatat secara jujur untuk setiap fase implementasi.

| Fase | Berkas / Bagian | Bantuan AI | Yang Disesuaikan / Dipelajari Mahasiswa | Alasan Pemilihan Pendekatan |
|---|---|---|---|---|
| F0 | `config/env.go`, `config/logger.go`, `database/postgres.go` | Pembuatan struktur konfigurasi dan logger dengan rotasi file | Memahami integrasi slog JSON multi-writer dan lifecycle connection pool pgx | Standar logging terstruktur memudahkan observabilitas tanpa library berlebihan |
| F0 | `app/model/response.go` | Pembuatan struct standar response API | Menyesuaikan format JSON tag dan omitempty pada Meta | Format konsisten `{success, message, data, meta}` memudahkan integrasi client |
| F0 | `helper/errors.go`, `helper/response.go` | Desain AppError dan helper response | Memastikan field cause tidak diekspor agar tidak bocor ke client | Memisahkan detail internal dari pesan yang aman untuk publik |
| F0 | `helper/validator.go`, `helper/validator_test.go` | Singleton validator, custom rules, dan unit test | Memahami `RegisterTagNameFunc` dan validasi regex/rune | Mengumpulkan seluruh error validasi sekaligus daripada gagal di field pertama |
| F0 | `middleware/middleware.go` | Middleware requestid, requestLogger, recover, helmet, cors | Mengatur urutan agar status code 5xx/4xx terhitung akurat pada access log | Urutan middleware menentukan apakah log mencatat status sebelum atau sesudah recover |
| F0 | `config/app.go`, `route/route.go`, `main.go` | ErrorHandler terpusat, route health, graceful shutdown | Mempelajari graceful shutdown via `os.Signal` channel di Go | Menjamin request in-flight selesai diproses saat server dimatikan |
| F1 | `migrations/001_rbac.sql`, `002_users_auth.sql` | Penulisan skema DDL SQL roles, permissions, users, dan refresh token | Mempelajari indeks unik case-insensitive `LOWER(...)` dan relasi cascade | Integritas data dijaga langsung di level basis data menggunakan constraint |
| F1 | `app/repository/user_repository.go`, `token_repository.go`, `errors.go` | Isolasi query SQL dan sentralisasi penanganan kode error PostgreSQL | Memahami pemetaan kode error `23505` menjadi constraint-aware DuplicateError | Mencegah error teknis driver basis data bocor ke layer atas |
| F1 | `helper/security.go`, `helper/jwt.go`, `helper/context.go`, `helper/authz.go` | Implementasi bcrypt cost 12, DummyHash, verifikasi algoritma JWT eksplisit, dan in-memory PermissionSet | Mempelajari pencegahan timing attack dan pencegahan kerentanan alg "none" pada JWT | Menjamin keamanan standar industri untuk autentikasi dan penanganan token |
| F1 | `middleware/auth.go`, `app/service/auth_service.go`, `route/route.go` | Implementasi alur register, login, refresh token rotation, logout, me, dan in-memory rate limiter | Memahami mekanisme rotasi refresh token dan header HTTP standar (WWW-Authenticate, Retry-After) | Menjamin kepatuhan standar REST API dan perlindungan terhadap serangan brute force |
| F1 | `helper/jwt_test.go`, `helper/security_test.go` | Penulisan unit test untuk skenario valid, expired, alg none, dan pengujian cost bcrypt | Memverifikasi ketepatan klaim token JWT dan keandalan fungsi keamanan | Memastikan proteksi keamanan teruji secara otomatis tanpa bergantung pada DB |
