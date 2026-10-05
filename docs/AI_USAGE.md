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
