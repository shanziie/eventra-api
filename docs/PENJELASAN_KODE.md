# Penjelasan Kode — Eventra API

Dokumen ini mencatat ringkasan teknis dan persiapan tanya jawab dengan dosen penguji untuk setiap fase pengerjaan.

---

## Fase F0 — Fondasi (Modul 1, 4, 7)

### Ringkasan Pengerjaan
Pada fase ini dibuat kerangka dasar arsitektur backend Eventra:
1. `config/env.go`, `config/logger.go`, `database/postgres.go`: Konfigurasi terpusat via env, structured logging JSON dengan slog dan rotasi lumberjack, serta connection pool `pgxpool` dengan ping saat startup.
2. `app/model/response.go`: Standar respons sukses (`WebResponse`, `WebListResponse`, `WebCursorResponse`) dan respons error (`ErrorResponse`) sesuai PRD §8.1.
3. `helper/`: Manajemen error terpusat (`AppError` tanpa mengekspos cause), response helper tanpa `helper.Fail`, validator singleton dengan nama field JSON dan custom rules (`nospace`, `username`, `strongpassword`), serta pembaca query offset dan request context (timeout 5 detik).
4. `middleware/`: Registrasi global middleware (`requestid` → `requestLogger` → `recover` → `helmet` → `cors`) dan `RequireJSON`.
5. `config/app.go` & `route/route.go`: Instance Fiber dengan ErrorHandler terpusat, route `/api/v1/health` dengan ping timeout 2 detik (503 jika DB mati), dan catch-all 404.
6. `main.go`: Entry point bersih tanpa handler, validasi keamanan `JWT_SECRET >= 32`, dan graceful shutdown via `os.Signal`.

### Pertanyaan Dosen & Jawaban

1. **Kenapa tidak ada `helper.Fail`, melainkan handler mengembalikan `*helper.AppError` ke ErrorHandler terpusat?**
   *Jawaban:* Agar pembentukan JSON response gagal hanya dilakukan di satu tempat tunggal (`config/app.go`). Ini menjamin struktur `{success:false, code, message, fields, request_id}` selalu seragam di seluruh aplikasi. Selain itu, error teknis internal (`cause`) tidak bocor ke client namun tetap dicatat di server log bersama `request_id` untuk keperluan debugging.

2. **Kenapa urutan middleware diatur `requestid` → `requestLogger` → `recover` → `helmet` → `cors`?**
   *Jawaban:* `requestid` harus pertama agar header `X-Request-Id` sudah ada sebelum middleware lain bekerja. `requestLogger` ditaruh sebelum `recover` agar ketika handler atau middleware downstream mengalami `panic`, `recover` menangkapnya dan menghasilkan HTTP 500, lalu `requestLogger` tetap dapat mencatat status final 500 tersebut beserta durasi request-nya secara utuh.

3. **Kenapa validator dibuat singleton dan memakai `RegisterTagNameFunc`?**
   *Jawaban:* Validator reflection di Go memakan alokasi memori jika dibuat berulang kali per request; singleton memastikan struct metadata di-cache sekali saat program mulai. `RegisterTagNameFunc` membaca tag `json` pada struct agar nama field yang dikembalikan ke client adalah nama JSON API (misalnya `full_name`) bukan nama internal field struct Go (`FullName`).

---

## Fase F1 — Autentikasi (Modul 5)

### Ringkasan Pengerjaan
Pada fase ini diimplementasikan sistem autentikasi aman berbasis JWT & Refresh Token:
1. `migrations/`: `001_rbac.sql` (tabel roles, permissions, role_permissions + seed awal PRD 7.1) dan `002_users_auth.sql` (tabel users dan refresh_tokens beserta indeks unik case-insensitive `LOWER(username)` dan `LOWER(email)`).
2. `app/model/`: Model `User` (password `json:"-"`), `RegisterRequest` (tanpa field role sesuai BR-U1), `LoginRequest`, `RefreshRequest`, `TokenPair`, `RefreshToken`, `AuthUser`, dan `MeResponse`.
3. `app/repository/`: `UserRepository` dan `TokenRepository` dengan terjemahan error pgx terpusat (`TranslateError`), membedakan duplikat username vs email via `DuplicateError.Constraint`.
4. `helper/`: `security.go` (bcrypt cost 12, DummyHash pencegah timing attack, token acak 32 byte, SHA-256 hex), `jwt.go` (JWTManager dengan validasi metode HMAC eksplisit), `context.go` (helper pembaca ID dan role user), serta `authz.go` (pemuatan `PermissionSet` sekali saat startup, fail-closed).
5. `middleware/`: `RequireAuth` (memvalidasi access token dan menyematkan header WWW-Authenticate saat 401) serta `LoginRateLimiter` (maksimal 5 kali percobaan per menit per IP address dengan header Retry-After saat 429).
6. `app/service/auth_service.go` & `route/route.go`: Endpoint Register (201 + Location), Login (200), Refresh (rotasi token 200), Logout (200), dan Me (200 data user + daftar permission).

### Pertanyaan Dosen & Jawaban

1. **Kenapa saat login gagal karena username tidak ditemukan, kita tetap mengeksekusi `DummyHash` dan mengembalikan pesan yang sama persis dengan password salah?**
   *Jawaban:* Untuk mencegah serangan *user enumeration via timing attack*. Algoritma hashing bcrypt sengaja lambat (~100-300 ms pada cost 12). Jika saat username tidak ditemukan server langsung membalas dalam waktu <5 ms, penyerang dapat mengukur selisih waktu respons untuk membedakan akun mana yang sudah terdaftar di sistem. `DummyHash` menyamakan beban waktu komputasi sehingga waktu respons identik.

2. **Kenapa refresh token disimpan dalam bentuk hash SHA-256 di database, bukan plain text atau bcrypt?**
   *Jawaban:* Refresh token dihasilkan dari 32 byte cryptographically secure random bytes yang sudah memiliki entropi sangat tinggi (256-bit), sehingga tidak rentan terhadap serangan brute force kamus (dictionary attack) yang memerlukan perlambatan komputasi bcrypt. Namun nilai aslinya tetap tidak boleh disimpan sebagai plain text agar jika isi tabel database bocor, penyerang tidak dapat langsung menggunakan token tersebut untuk membajak sesi pengguna.

3. **Bagaimana mekanisme rotasi refresh token bekerja dan apa manfaat keamanannya?**
   *Jawaban:* Setiap kali endpoint `POST /auth/refresh` berhasil diproses, token penyegar yang digunakan langsung dicabut status aktifnya (`revoked_at = NOW()`) di database, lalu diterbitkan pasangan token baru. Jika token lama yang sudah dicabut dicoba dipakai lagi, server menolaknya (401). Mekanisme ini mencegah token yang pernah ditransmisikan disalahgunakan berulang kali apabila sempat tersadap.
