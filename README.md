# Eventra API — Backend Sistem Manajemen Event & Tiket

Eventra API adalah sistem backend untuk manajemen event, penjualan tiket, registrasi, pembayaran, check-in, dan kontrol akses berbasis peran (RBAC). Proyek ini dikembangkan sebagai tugas Praktikum Pemrograman Backend Lanjut (D4 Teknik Informatika, UNAIR).

## 1. Teknologi & Stack
- **Bahasa**: Go (versi stabil terbaru)
- **Framework Web**: Fiber v2 (`github.com/gofiber/fiber/v2`)
- **Database**: PostgreSQL
- **Driver / Pool**: `github.com/jackc/pgx/v5` dan `pgxpool`
- **Konfigurasi**: `github.com/joho/godotenv`
- **Autentikasi**: `github.com/golang-jwt/v5` (JWT HS256) & `golang.org/x/crypto/bcrypt` (cost 12)
- **Validasi**: `github.com/go-playground/validator/v10`
- **Logging**: `log/slog` dengan rotasi file `gopkg.in/natefinch/lumberjack.v2`
- **Pengujian**: Testing bawaan Go (`testing`)

## 2. System Requirements
- Go 1.21+ terinstal
- PostgreSQL aktif (lokal atau remote)

## 3. Instalasi & Setup Repository
```bash
git clone <repository-url>
cd eventra-api
go mod download
```

## 4. Konfigurasi Environment (`.env`)
Salin `.env.example` menjadi `.env` dan sesuaikan nilai koneksi database serta rahasia JWT:
```bash
cp .env.example .env
```
*(Catatan: `JWT_SECRET` wajib memiliki panjang minimal 32 karakter, jika kurang aplikasi akan menghentikan eksekusi saat startup).*

## 5. Cara Menjalankan Aplikasi
```bash
go run main.go
```
Server akan berjalan di `http://localhost:8080` (sesuaikan port di `.env`).

## 6. Struktur Folder Proyek (Baku Modul 4)
```
eventra-api/
├── app/
│   ├── model/        # Entitas, request, response (tag json + validate)
│   ├── repository/   # Query SQL, interface repository, sentinel error
│   └── service/      # Method Fiber (*fiber.Ctx) + business rules murni (*_rules.go)
├── config/           # app.go, env.go, logger.go
├── database/         # postgres.go (pool)
├── helper/           # response, errors, validator, jwt, security, cursor, negotiate, authz
├── middleware/       # middleware.go, auth.go, authz.go
├── route/            # route.go (35+ endpoint)
├── migrations/       # 001_rbac.sql s.d 005_registrations.sql
├── docs/             # Dokumentasi lengkap (API, DATABASE, PRD, MATRIX, dll.)
├── postman/          # Postman collection untuk 35 endpoint
├── logs/             # (tidak di-commit)
└── main.go
```

## 7. Skema Database & Migrasi
Migrasi SQL di folder `migrations/` dijalankan secara berurutan:
1. `001_rbac.sql`: Tabel `roles`, `permissions`, `role_permissions` beserta seed data.
2. `002_users_auth.sql`: Tabel `users` (dengan unique index lowercase `username` dan `email`) dan `refresh_tokens`.
3. `003_categories.sql`: Tabel `categories` untuk kategori event.
4. `004_events.sql`: Tabel `events` dan `ticket_types` dengan constraint kapasitas, kuota, `chk_event_times`, dan `chk_ticket_sold_quota`.
5. `005_registrations.sql`: Tabel `registrations` (dengan unique index aktif `idx_registrations_user_event_active`) dan `payments`.

## 8. Authentication
- Pendaftaran akun dengan password terenkripsi bcrypt (cost 12, max 72 karakter). Role otomatis diset `participant` dari server.
- Proteksi User Enumeration: Jika username tidak ditemukan saat login, sistem tetap menjalankan *dummy hash* dan memberikan pesan error yang sama persis.
- JWT HS256 dengan access token berlaku 15 menit, refresh token acak 32 byte disimpan sebagai hash SHA-256 dan dapat di-revoke saat logout.
- Rate limiter login (5 request per menit per IP).

## 9. Authorization & RBAC
- Role-Based Access Control (RBAC) dengan tabel dinamis di database. Permission dimuat sekali saat start (`main.go`).
- Middleware `RequireAuth` selalu dijalankan sebelum `RequirePermission`.
- Keputusan berdasarkan data/kepemilikan (ownership) diimplementasikan melalui fungsi murni di layer service (`CanAccessUser`, `CanManageEvent`, `CanAccessRegistration`).

## 10. Daftar Endpoint API (35 Endpoint)
Terdaftar di `route/route.go` meliputi:
- **Health**: `GET /api/v1/health`
- **Auth**: Register, Login, Refresh, Logout, Me (#2–#6)
- **Users**: List, Get by ID, Put, Patch, Delete, Assign Role, Get Registrations (#7–#12, #35)
- **Categories**: List, Create, Update, Delete (#13–#16)
- **Events & Ticket Types**: List, Get by ID, Create, Put, Patch, Delete, Status, Get Ticket Types, Create Ticket Type, Update Ticket Type, Delete Ticket Type (#17–#27)
- **Registrations & Payments**: Create Registration, Get Registrations by Event (JSON/CSV), Get Registration by ID, Submit Payment, Verify Payment, Cancel Registration, Check-in (#28–#34)

## 11. Pengujian & Testing
- Unit tests untuk business rules (`app/service/*_test.go`) dan helper (`helper/*_test.go`) menggunakan framework `testing` bawaan Go.
- Postman Collection tersedia di `postman/eventra-api.postman_collection.json`.
- Matriks Hak Akses tersedia di `docs/MATRIX_HAK_AKSES.md`.

## 12. Pemetaan Modul 1–7
Detail pemetaan materi modul praktikum tercantum di `docs/MODUL_MAPPING.md`.

## 13. Asumsi & Keputusan Desain
Tercantum lengkap di `docs/ASUMSI.md`.

## 14. Kontak & Pemilik Proyek
Mahasiswa D4 Teknik Informatika, Universitas Airlangga (Praktikum Pemrograman Backend Lanjut).
