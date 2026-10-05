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
