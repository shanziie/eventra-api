# AGENTS.md — Eventra API

Project tugas kuliah Praktikum Pemrograman Backend Lanjut (D4 Teknik Informatika, UNAIR).
Pemilik project adalah mahasiswa yang HARUS bisa menjelaskan setiap baris ke dosen.
Tulis kode seperti mahasiswa yang rapi dan mengikuti modul dosen, bukan seperti template AI.

## 1. Sumber kebenaran (urutan prioritas)

1. Modul dosen di `docs/modul/` (Modul 1–7)
2. `docs/PRD.md` dan `docs/IDE.md`
3. File ini
4. Praktik umum Go

Bila bertentangan, yang di atas menang. Laporkan pertentangannya, jangan diam-diam memilih.

PENTING: Modul 7 Bagian B sengaja berisi kode yang salah. Jangan salin mentah.
Ikuti Bagian A (teori) dan Bagian C (spesifikasi penerimaan) Modul 7.

## 2. Stack (tidak boleh diganti)

Go stabil terbaru, Fiber v2 (BUKAN v3), PostgreSQL + pgx/v5 (pgxpool), godotenv,
golang-jwt/v5, x/crypto/bcrypt (cost 12), go-playground/validator/v10,
log/slog + lumberjack, testing bawaan Go.

DILARANG tanpa izin: GORM, sqlx, ent, testify, wire, Gin, Echo, Prisma, library generik baru.

## 3. Struktur folder (baku Modul 4)

```
eventra-api/
├── app/
│   ├── model/        entitas, request, response (tag json + validate)
│   ├── repository/   query SQL, interface repository, sentinel error
│   └── service/      method menerima *fiber.Ctx + business rules murni (*_rules.go)
├── config/           app.go, env.go, logger.go
├── database/         postgres.go (pool)
├── helper/           response, errors, validator, jwt, security, cursor, negotiate, authz
├── middleware/       middleware.go, auth.go, authz.go
├── route/            route.go
├── migrations/       001_xxx.sql dst
├── docs/             PRD, IDE, modul, PENJELASAN_KODE, AI_USAGE, ASUMSI
├── logs/             (tidak di-commit)
└── main.go
```

Tidak ada folder `controller/`, `domain/`, `usecase/`, `utils/`, `pkg/`, `internal/`.
Peran controller dipegang method service yang menerima `*fiber.Ctx`.

## 4. Dependency rule (package mana boleh impor package mana)

| Package | Boleh impor dari proyek sendiri |
|---|---|
| app/model | tidak ada |
| app/repository | app/model |
| app/service | app/model, app/repository, helper |
| helper | app/model |
| middleware | helper |
| route | app/service, middleware, helper |
| config | app/service, route, middleware, helper |
| database | config |
| main.go | config, database, app/repository, app/service |

Business rules ditulis sebagai fungsi biasa (struct masuk, hasil keluar) di `app/service/*_rules.go`
tanpa `fiber`, tanpa repository, tanpa SQL, supaya bisa di-unit-test.

## 5. Gaya kode (supaya natural dan bisa dijelaskan)

- Identifier bahasa Inggris, konsisten dengan modul (`UserService`, `FindByID`, `translateError`).
- Komentar Bahasa Indonesia, pendek, hanya menjelaskan MENGAPA (aturan bisnis, risiko keamanan,
  keputusan desain). Jangan mengulang apa yang sudah jelas dari kode.
- Doc comment hanya untuk fungsi exported yang tidak jelas dari namanya. Jangan semua fungsi.
- Tanpa pemisah dekoratif (`// -----`, `// =====`), tanpa emoji, tanpa blok komentar paket panjang.
- Fungsi pendek (sekitar 40 baris maksimal), satu tugas, early return, if/else biasa.
- Tanpa generics, tanpa reflection (selain validator), tanpa goroutine (selain graceful shutdown),
  tanpa functional options atau builder. Interface hanya untuk repository.
- Jangan membuat abstraksi sebelum ada tiga pemakaian. Tidak ada base repository generik.
  Tidak ada fungsi atau konstanta yang tidak dipakai.
- Error: bungkus dengan `%w` + konteks pendek. Jangan menelan error. Jangan `panic` di jalur request.
- Pesan response Bahasa Indonesia huruf kecil seperti modul ("user tidak ditemukan").
- Dilarang: TODO, FIXME, placeholder, fungsi kosong yang `return nil`, data dummy palsu.
  Fitur yang belum dikerjakan tidak boleh muncul di route.
- Jangan menambah fitur di luar PRD. Ingin menambah, tanya dulu.
- Jangan menulis komentar yang menyamarkan penulis kode. Penggunaan AI dicatat jujur di `docs/AI_USAGE.md`.

## 6. Perilaku API yang wajib (dari modul)

Response sukses: `{success, message, data, meta?}`.
Response gagal: `{success:false, code, message, fields?, request_id}`.
Handler gagal dengan MENGEMBALIKAN `*helper.AppError`. Hanya satu ErrorHandler di `config/app.go`
yang menulis response gagal. Tidak ada `helper.Fail`.

Status: 200, 201 (+Location), 204, 400 (JSON rusak, id/cursor/query tidak sah), 401 (+WWW-Authenticate),
403, 404, 406, 409, 415, 422 (validasi tag dan pelanggaran aturan bisnis pada isi request),
429 (+Retry-After), 500 (pesan generik, detail hanya di log), 503 (/health saat DB mati).

Log: 4xx = WARN `request_rejected`, 5xx = ERROR `request_failed`. Access log memakai status final
yang benar (dihitung dari error yang dikembalikan) dan memuat request_id, method, path, status,
duration, ip, user_id, role. Error yang tidak dikenali `translateError` menjadi Internal (500), tidak pernah nil.

Validasi: tag `validate` pada struct request. Validator dibuat SEKALI. `RegisterTagNameFunc` agar nama
field = nama JSON. Custom: `nospace`, `username`, `strongpassword` (return true bila password KUAT).
PATCH: semua field pointer + `omitnil`. Aturan antar-field ditulis sebagai fungsi murni di service.
Semua pelanggaran dilaporkan sekaligus, pesan tidak boleh string kosong.

SQL: semua nilai klien lewat `$n`. ORDER BY lewat whitelist map. Pencarian `ILIKE`. `COUNT(*)` dengan
filter yang sama. `helper.RequestContext` (timeout) di setiap operasi DB. `defer rows.Close()`.
Terjemahkan error pgx di repository: no rows -> ErrNotFound, 23505 -> ErrDuplicate,
23503 dan 23514 -> ErrConflict. Error pgx tidak boleh naik ke layer atas.

Offset pagination: `page`, `limit` (default 10, maks 100), `search`, `sort` (whitelist), `order`, filter;
meta `{page, limit, total, total_pages}`.
Cursor pagination: `ORDER BY created_at DESC, id DESC`, `WHERE (created_at, id) < ($1, $2)`, ambil `limit+1`,
index gabungan DESC yang cocok, cursor rusak -> 400, meta `{limit, next_cursor, has_more}`,
halaman terakhir tanpa `next_cursor`.

Content negotiation: Accept kosong atau `*/*` -> JSON. `text/csv` pakai `encoding/csv`.
Format tak didukung -> 406, diputuskan SEBELUM query.

Auth: bcrypt cost 12, password maks 72 karakter. Username tidak ada -> jalankan dummy hash dan balas
pesan yang sama persis. JWT HS256 dengan pemeriksaan method eksplisit, claims sub/role/iss/iat/exp,
access TTL 15 menit. Refresh token acak 32 byte, disimpan sebagai SHA-256, rotasi tiap dipakai, logout
mencabut. `JWT_SECRET` kurang dari 32 karakter -> aplikasi berhenti. Rate limiter login 5 per menit per IP.
`BodyLimit` 1 MB. CORS allowlist dari env. Role saat register SELALU `participant` dari server.
Field password memakai `json:"-"`.

RBAC: tabel roles, permissions, role_permissions di DB. `PermissionSet` dimuat SEKALI saat start
(gagal -> exit). `Can()` fail closed. `RequireAuth` selalu sebelum `RequirePermission`.
Keputusan tanpa data -> middleware. Keputusan yang butuh data (ownership) -> fungsi murni di service.
Periksa hak SEBELUM mengambil data. Tidak boleh mengubah role atau menghapus akun sendiri.

## 7. Cara kerja

1. Kerjakan SATU fase PRD pada satu waktu, urut. Mulai dengan rencana singkat: file yang dibuat/diubah
   dan asumsi. Jangan lompat fase.
2. Fase dianggap selesai hanya bila `go build ./...`, `go vet ./...`, `go test ./...` hijau, migrasi
   sudah dijalankan ke PostgreSQL lokal, dan endpoint fase itu benar-benar dicoba dengan curl.
   Laporkan status HTTP apa adanya. Jangan klaim lulus tanpa menjalankan.
3. Commit kecil per langkah logis (Conventional Commits, ringkas). Jangan satu commit besar.
4. Setelah tiap fase: ringkasan maksimal 15 baris (apa dibuat, alasannya, 3 pertanyaan dosen yang mungkin
   beserta jawaban singkat). Tambahkan ke `docs/PENJELASAN_KODE.md`. Catat bagian yang dibantu AI ke
   `docs/AI_USAGE.md`.
5. Tidak yakin -> tulis asumsi di `docs/ASUMSI.md`. Jangan mengarang.
6. Jangan mengubah file migrasi yang sudah dijalankan. Buat migrasi baru bernomor.
7. Jangan commit `.env`, `logs/`, atau binary. Jangan menulis secret asli di mana pun.

## 8. Pemeriksaan kebocoran layer (jalankan di akhir tiap fase)

- `go list -deps ./app/repository | grep gofiber` harus kosong
- `grep -rniE "SELECT|INSERT|UPDATE|DELETE" app/service` harus kosong
- `app/model` tidak mengimpor package proyek sendiri
- `route/*.go` tidak berisi `if` validasi atau business rules
- `main.go` tidak berisi handler
- `grep -rn "helper.Fail" .` harus kosong
