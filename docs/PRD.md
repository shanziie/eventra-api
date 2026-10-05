PRD — Eventra API
Event Registration & Ticketing Backend
Praktikum Pemrograman Backend Lanjut — D4 Teknik Informatika, Universitas Airlangga
Versi 1.0 — Stack: Go + Fiber v2 + PostgreSQL (pgx)
---
1. Ringkasan
Eventra adalah REST API untuk mengelola event dari pembuatan sampai check-in peserta:
penyelenggara membuat event dan jenis tiket, peserta mendaftar, membayar, lalu check-in di hari acara.
API ini sengaja memuat proses bisnis yang punya state, kuota, waktu, kepemilikan data, dan hak akses berlapis,
supaya seluruh materi Modul 1–7 dipakai pada kebutuhan nyata, bukan latihan terpisah.
1.1 Masalah yang diselesaikan
Masalah	Akibat tanpa sistem
Pendaftaran event lewat chat/form manual	Kuota jebol, peserta ganda, data tersebar
Verifikasi pembayaran manual tanpa jejak	Sengketa "sudah bayar" tidak bisa dibuktikan
Siapa saja boleh mengubah data event	Event orang lain bisa diedit atau dihapus
Daftar hadir dibuat ulang tiap acara	Tidak ada rekap, tidak bisa diekspor
1.2 Tujuan
ID	Tujuan
G1	Seluruh requirement Modul 1–7 terimplementasi nyata di kode, bukan hanya di laporan
G2	Proses bisnis end-to-end berjalan: buat event, jual tiket, daftar, bayar, verifikasi, check-in
G3	Kuota tidak pernah oversold walau dua request datang bersamaan
G4	Hak akses benar: role (RBAC) dan kepemilikan data (ownership) sama-sama ditegakkan
G5	Mahasiswa dapat menjelaskan setiap file ke dosen tanpa membuka AI
1.3 Bukan tujuan (non-goals)
Payment gateway sungguhan, email/notifikasi, upload file (bukti transfer berupa nomor referensi saja),
background job/scheduler (Modul 11), MongoDB (Modul 10), Swagger UI (Modul 13), Docker, caching/Redis,
OAuth/MFA, frontend. Event yang lewat tanggal tidak berubah status otomatis; penyelenggara menandainya `finished`.
1.4 Metrik keberhasilan
Metrik	Target
Endpoint dalam matriks hak akses yang terbukti diuji (curl/Postman)	100%
Pemeriksaan kebocoran layer (README bagian Arsitektur)	0 pelanggaran
Unit test business rules (table-driven, tanpa server/DB)	minimal 25 kasus, semua hijau
Daftar periksa C.8 Modul 7	14 dari 14 benar
`go build ./... && go vet ./... && go test ./...`	hijau
---
2. Aktor dan Role
Role	Siapa	Cara mendapat role
`admin`	Pengelola platform	Diangkat lewat SQL (akun pertama) lalu lewat `PATCH /users/:id/role`
`organizer`	Penyelenggara event	Diangkat admin lewat `PATCH /users/:id/role`
`participant`	Peserta event	Default saat register (ditentukan server, tidak dari body)
Persona singkat:
Admin Dewi: memantau seluruh user dan event, menangani sengketa.
Organizer Raka: membuat event "Workshop Go Backend", mengatur tiket, memverifikasi pembayaran, check-in di lokasi.
Participant Sari: mencari event, mendaftar, mengirim nomor referensi pembayaran, memegang kode tiket.
---
3. Ruang Lingkup
Prioritas	Isi
P0 (wajib)	Seluruh endpoint di bagian 8 kecuali yang bertanda P1
P1 (bila waktu ada)	Pembatalan event massal (batalkan semua registrasi aktif + lepas kuota dalam satu transaksi), filter tambahan `starts_after`, endpoint statistik event
Di luar scope	Lihat 1.3
---
4. Proses Bisnis Utama
```
Organizer: buat event (draft) -> tambah jenis tiket -> publish
Participant: lihat event -> daftar (pilih tiket)
     |
     +-- tiket gratis  -> registrasi langsung CONFIRMED + ticket_code
     +-- tiket berbayar-> registrasi PENDING + payment UNPAID (jumlah = harga saat ini)
                          Participant kirim nomor referensi  -> payment SUBMITTED
                          Organizer verifikasi:
                             verified -> payment VERIFIED, registrasi CONFIRMED + ticket_code
                             rejected -> payment REJECTED (+catatan), peserta boleh kirim ulang
Hari acara : Organizer check-in memakai ticket_code -> registrasi CHECKED_IN
Pembatalan : Participant (maks H-24 jam) / Organizer / Admin -> CANCELLED, kuota kembali
Selesai    : Organizer tandai event FINISHED setelah ends_at
```
4.1 State machine
Registrasi:
Dari	Ke	Oleh	Syarat
(baru)	pending	participant	tiket berbayar
(baru)	confirmed	sistem	tiket gratis
pending	confirmed	organizer pemilik event / admin	payment `submitted`, lewat verifikasi pembayaran
pending	cancelled	pemilik registrasi / organizer pemilik event / admin	bebas
confirmed	cancelled	pemilik registrasi	sekarang <= `starts_at` - 24 jam
confirmed	cancelled	organizer pemilik event / admin	belum check-in
confirmed	checked_in	organizer pemilik event / admin	dalam jendela check-in
checked_in, cancelled	(akhir)	-	tidak bisa diubah
Payment: `unpaid` -> `submitted` (PUT peserta) -> `verified` atau `rejected` (PATCH organizer). `rejected` -> `submitted` (PUT ulang). `verified` akhir.
Event: `draft` -> `published`; `draft` atau `published` -> `cancelled`; `published` -> `finished` (hanya bila sekarang >= `ends_at`). `cancelled` dan `finished` akhir.
---
5. Aturan Bisnis
ID	Aturan	Gagal dengan
BR-U1	Role saat register selalu `participant`; body tidak boleh memuat role	(field diabaikan oleh struct)
BR-U2	`is_active` hanya boleh diubah oleh pemegang `user:update:any`	403
BR-U3	User nonaktif tidak bisa login dan seluruh refresh token-nya dicabut	403 saat login
BR-U4	Tidak boleh mengubah role atau menghapus akun sendiri	422 (role), 403 (hapus)
BR-U5	User yang masih jadi organizer event atau punya registrasi tidak bisa dihapus (nonaktifkan saja)	409
BR-C1	Nama kategori unik tanpa membedakan huruf besar/kecil	409
BR-C2	Kategori yang masih dipakai event tidak bisa dihapus	409
BR-E1	`organizer_id` selalu dari token, tidak dari body	-
BR-E2	`ends_at` > `starts_at`; `registration_deadline` <= `starts_at`; `starts_at` > sekarang saat dibuat	422
BR-E3	Jumlah kuota seluruh jenis tiket <= `capacity` event	422
BR-E4	Publish butuh minimal 1 jenis tiket	422
BR-E5	Event yang punya registrasi aktif: `starts_at`, `ends_at`, `capacity` tidak boleh diubah	409
BR-E6	Hanya event `draft` yang boleh dihapus	409
BR-E7	Event dengan registrasi aktif tidak boleh dibatalkan (MVP; P1 = batalkan massal)	409
BR-E8	`finished` hanya bila sekarang >= `ends_at`	409
BR-E9	Visibilitas: participant hanya melihat `published` dan `finished`; organizer ditambah event miliknya; admin semua. Event yang tidak terlihat dijawab 404	404
BR-T1	Maksimal 10 jenis tiket per event, nama unik per event	409
BR-T2	Harga dalam Rupiah bulat (BIGINT), bukan float	422
BR-T3	`quota` tidak boleh di bawah `sold`	422
BR-T4	Jenis tiket yang sudah punya registrasi tidak bisa dihapus	409
BR-T5	Jenis tiket hanya bisa ditambah/diubah selagi event `draft` atau `published`	409
BR-R1	Hanya event `published` yang menerima pendaftaran, dan sekarang <= `registration_deadline`	409 `REGISTRATION_CLOSED`
BR-R2	Jenis tiket harus milik event yang didaftar	422
BR-R3	Satu user hanya boleh punya satu registrasi aktif (non-cancelled) per event	409 `ALREADY_REGISTERED`
BR-R4	Kuota habis ditolak, dijaga constraint `CHECK (sold <= quota)` di database	409 `QUOTA_FULL`
BR-R5	Harga disalin ke registrasi (`price_at_purchase`); perubahan harga tidak memengaruhi registrasi lama	-
BR-R6	Tiket gratis langsung `confirmed` dan mendapat `ticket_code`	-
BR-R7	Pembatalan mengembalikan kuota; operasi pendaftaran dan pembatalan atomik (satu transaksi)	-
BR-P1	Pengiriman bukti hanya untuk registrasi `pending` dan payment `unpaid`/`rejected`	409
BR-P2	Verifikasi hanya bila payment `submitted`	409
BR-P3	Verifikasi `verified` sekaligus mengonfirmasi registrasi dan membuat `ticket_code` (satu transaksi)	-
BR-K1	Check-in hanya untuk registrasi `confirmed`, mulai `starts_at` - 2 jam sampai `ends_at`	409 `CHECKIN_NOT_OPEN` / `NOT_CONFIRMED`
BR-K2	Satu tiket hanya bisa check-in sekali	409 `ALREADY_CHECKED_IN`
Kode status: 422 untuk pelanggaran aturan pada ISI request (tidak masuk akal), 409 untuk benturan dengan KEADAAN data
(duplikat, kuota, state salah, relasi masih dipakai), 403 untuk tidak berhak.
---
6. Model Data
Aturan umum: tipe waktu `TIMESTAMPTZ`; uang `BIGINT` (Rupiah); status berupa `VARCHAR` + `CHECK ... IN (...)`;
semua FK eksplisit dengan `ON DELETE` yang disengaja; keunikan dijaga database, bukan loop di Go.
6.1 Daftar tabel dan relasi
Relasi	Jenis	Alasan bisnis
roles - permissions (lewat role_permissions)	Many-to-Many	Satu role punya banyak hak, satu hak dipakai banyak role (Modul 6)
roles - users	One-to-Many	Setiap user punya tepat satu role, dikunci FK
users - refresh_tokens	One-to-Many	Satu user bisa login dari banyak perangkat
users - events (organizer)	One-to-Many	Penyelenggara punya banyak event
categories - events	One-to-Many	Klasifikasi event
events - ticket_types	One-to-Many	Satu event punya beberapa jenis tiket
users - registrations	One-to-Many	Peserta punya banyak registrasi
ticket_types - registrations	One-to-Many	Kuota dihitung per jenis tiket
registrations - payments	One-to-One (opsional)	Pembayaran punya siklus status sendiri; tiket gratis tidak punya payment
6.2 Spesifikasi tabel
`roles`: `name` VARCHAR(20) PK; `description` VARCHAR(150) NOT NULL; `created_at` TIMESTAMPTZ NOT NULL DEFAULT NOW().
`permissions`: `name` VARCHAR(50) PK; `description` VARCHAR(150) NOT NULL.
`role_permissions`: `role_name` FK roles ON DELETE CASCADE; `permission_name` FK permissions ON DELETE CASCADE; PK gabungan keduanya.
`users`: `id` SERIAL PK; `username` VARCHAR(50) NOT NULL; `email` VARCHAR(255) NOT NULL; `full_name` VARCHAR(100) NOT NULL;
`password` VARCHAR(255) NOT NULL (hash bcrypt); `role` VARCHAR(20) NOT NULL DEFAULT 'participant' FK roles(name) ON UPDATE CASCADE;
`is_active` BOOLEAN NOT NULL DEFAULT TRUE; `created_at` TIMESTAMPTZ NOT NULL DEFAULT NOW().
UNIQUE INDEX `LOWER(username)`, UNIQUE INDEX `LOWER(email)`, INDEX `role`, INDEX `(created_at DESC, id DESC)`.
`refresh_tokens`: `id` BIGSERIAL PK; `user_id` INT NOT NULL FK users ON DELETE CASCADE; `token_hash` TEXT NOT NULL UNIQUE;
`expires_at` TIMESTAMPTZ NOT NULL; `revoked_at` TIMESTAMPTZ NULL; `created_at` TIMESTAMPTZ NOT NULL DEFAULT NOW(). INDEX `user_id`.
`categories`: `id` SERIAL PK; `name` VARCHAR(80) NOT NULL; `description` VARCHAR(255) NOT NULL DEFAULT ''; `created_at`.
UNIQUE INDEX `LOWER(name)`. Seed: Seminar, Workshop, Kompetisi, Konser, Volunteer.
`events`: `id` SERIAL PK; `organizer_id` INT NOT NULL FK users ON DELETE RESTRICT; `category_id` INT NOT NULL FK categories ON DELETE RESTRICT;
`title` VARCHAR(150) NOT NULL; `description` TEXT NOT NULL DEFAULT ''; `location` VARCHAR(200) NOT NULL;
`capacity` INT NOT NULL CHECK (capacity > 0); `starts_at`, `ends_at`, `registration_deadline` TIMESTAMPTZ NOT NULL;
`status` VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK IN ('draft','published','cancelled','finished');
`created_at`, `updated_at` TIMESTAMPTZ NOT NULL DEFAULT NOW().
CHECK `(ends_at > starts_at)`, CHECK `(registration_deadline <= starts_at)`.
INDEX `(created_at DESC, id DESC)` (cursor), INDEX `organizer_id`, INDEX `category_id`, INDEX `status`.
`ticket_types`: `id` SERIAL PK; `event_id` INT NOT NULL FK events ON DELETE CASCADE; `name` VARCHAR(80) NOT NULL;
`price` BIGINT NOT NULL CHECK (price >= 0); `quota` INT NOT NULL CHECK (quota > 0); `sold` INT NOT NULL DEFAULT 0;
CHECK `(sold >= 0 AND sold <= quota)`; `created_at`. UNIQUE `(event_id, LOWER(name))` lewat unique index; UNIQUE `(id, event_id)` untuk FK gabungan.
`registrations`: `id` SERIAL PK; `user_id` INT NOT NULL FK users ON DELETE RESTRICT; `event_id` INT NOT NULL; `ticket_type_id` INT NOT NULL;
FOREIGN KEY `(ticket_type_id, event_id)` REFERENCES `ticket_types(id, event_id)` ON DELETE RESTRICT (menjamin tiket milik event yang sama);
FK `event_id` ke events ON DELETE RESTRICT; `status` VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK IN ('pending','confirmed','checked_in','cancelled');
`price_at_purchase` BIGINT NOT NULL CHECK (>= 0); `ticket_code` VARCHAR(32) NULL UNIQUE; `confirmed_at`, `checked_in_at`, `cancelled_at` TIMESTAMPTZ NULL;
`created_at`, `updated_at`.
UNIQUE INDEX `(user_id, event_id) WHERE status <> 'cancelled'` (satu registrasi aktif per event).
INDEX `(event_id, created_at DESC, id DESC)` (cursor daftar peserta), INDEX `(user_id, created_at DESC)`.
`payments`: `id` SERIAL PK; `registration_id` INT NOT NULL UNIQUE FK registrations ON DELETE CASCADE (inilah One-to-One);
`amount` BIGINT NOT NULL CHECK (>= 0); `method` VARCHAR(30) NULL; `reference_number` VARCHAR(60) NULL;
`status` VARCHAR(20) NOT NULL DEFAULT 'unpaid' CHECK IN ('unpaid','submitted','verified','rejected'); `note` VARCHAR(200) NOT NULL DEFAULT '';
`submitted_at`, `verified_at` TIMESTAMPTZ NULL; `verified_by` INT NULL FK users; `created_at`.
6.3 Migrasi
File	Isi
001_rbac.sql	roles, permissions, role_permissions + seed role dan permission
002_users_auth.sql	users, refresh_tokens
003_categories.sql	categories + seed
004_events.sql	events, ticket_types, index cursor
005_registrations.sql	registrations, payments, index
Migrasi dijalankan dengan `psql -f` seperti modul. Akun admin pertama: register lewat API, lalu `UPDATE users SET role='admin' ...` (didokumentasikan di README, mengikuti Modul 6 Langkah 9).
---
7. RBAC dan Ownership
7.1 Permission (pola `domain:action:scope`)
Permission	admin	organizer	participant
user:list	v		
user:read:any	v		
user:update:any	v		
user:delete	v		
role:assign	v		
category:create	v		
category:update	v		
category:delete	v		
event:create	v	v	
event:read:any	v		
event:update:any	v		
event:delete:any	v		
registration:create			v
registration:read:any	v		
registration:manage:any	v		
Hak atas data sendiri tidak butuh permission, mengalir dari kepemilikan (prinsip Modul 6 A.4).
7.2 Ownership
Resource	Pemilik	Hak pemilik	Hak lewat permission `:any`
user	user itu sendiri	baca, PUT, PATCH profil	admin: baca/ubah semua, hapus
event	`events.organizer_id`	ubah, hapus (draft), ubah status, kelola tiket, lihat peserta, verifikasi pembayaran, check-in	admin lewat `event:update:any`, `event:delete:any`, `registration:*:any`
registration	`registrations.user_id`	baca, kirim pembayaran, batalkan (H-24)	admin
registration (sisi organizer)	organizer event induknya	baca, verifikasi, batalkan, check-in	admin
Fungsi murni di `app/service/*_rules.go`: `CanAccessUser`, `CanManageEvent`, `CanReadRegistration`, `CanManageRegistration`.
Keputusan yang tidak butuh data (mis. `GET /users`) -> middleware `RequirePermission`. Keputusan yang butuh data -> service.
---
8. Spesifikasi API
Base path `/api/v1`. Semua endpoint selain yang bertanda Publik wajib `Authorization: Bearer <access_token>`.
`Content-Type: application/json` wajib untuk POST/PUT/PATCH (selain itu 415).
8.1 Konvensi response
Sukses:
```json
{ "success": true, "message": "event ditemukan", "data": { } }
```
Sukses daftar offset: tambahan `"meta": {"page":1,"limit":10,"total":37,"total_pages":4}`.
Sukses daftar cursor: `"meta": {"limit":10,"next_cursor":"MTc4...","has_more":true}` (halaman terakhir tanpa `next_cursor`).
Gagal:
```json
{ "success": false, "code": "VALIDATION_ERROR", "message": "validasi gagal",
  "fields": { "email": "format email tidak valid" }, "request_id": "30155df4-..." }
```
Kode error: `VALIDATION_ERROR`, `BAD_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `CONFLICT`, `UNSUPPORTED_MEDIA_TYPE`,
`NOT_ACCEPTABLE`, `TOO_MANY_REQUESTS`, `PAYLOAD_TOO_LARGE`, `INTERNAL_ERROR`, `SERVICE_UNAVAILABLE`.
Perluasan domain (status 409): `QUOTA_FULL`, `ALREADY_REGISTERED`, `REGISTRATION_CLOSED`, `INVALID_TRANSITION`,
`CHECKIN_NOT_OPEN`, `NOT_CONFIRMED`, `ALREADY_CHECKED_IN`, `HAS_DEPENDENCIES`, `HAS_ACTIVE_REGISTRATIONS`.
8.2 Daftar endpoint
Kolom Akses: P = publik, L = login saja, perm = permission middleware, own = ownership di service.
Sistem dan Auth (Modul 5)
#	Endpoint	Akses	Sukses	Error utama
1	GET /health	P	200	503 DB mati
2	POST /auth/register	P	201 + Location	422, 409
3	POST /auth/login	P, rate limit 5/menit/IP	200 (access + refresh token)	401 (pesan sama), 403 nonaktif, 422, 429 + Retry-After
4	POST /auth/refresh	P	200 (rotasi)	400, 401
5	POST /auth/logout	P	200	400
6	GET /auth/me	L	200 (user + daftar permission)	401
Users (Modul 2, 3, 6)
#	Endpoint	Akses	Sukses	Error utama
7	GET /users	perm `user:list`	200 offset: `page,limit,search,sort,order,is_active,role`	401, 403
8	GET /users/:id	own atau `user:read:any`	200	403, 404
9	PUT /users/:id	own atau `user:update:any`	200	403, 404, 409, 422
10	PATCH /users/:id	own atau `user:update:any`	200	400 (kosong), 403, 404, 409, 422
11	DELETE /users/:id	perm `user:delete`, bukan diri sendiri	204	403, 404, 409 `HAS_DEPENDENCIES`
12	PATCH /users/:id/role	perm `role:assign`, bukan diri sendiri	200	404, 422
Categories
#	Endpoint	Akses	Sukses	Error utama
13	GET /categories	L	200 offset	-
14	POST /categories	perm `category:create`	201 + Location	403, 409, 422
15	PUT /categories/:id	perm `category:update`	200	403, 404, 409, 422
16	DELETE /categories/:id	perm `category:delete`	204	403, 404, 409 `HAS_DEPENDENCIES`
Events (Modul 2, 3, 6, 7)
#	Endpoint	Akses	Sukses	Error utama
17	GET /events	L, visibilitas BR-E9	200 CURSOR: `limit,cursor,search,category_id,status`	400 (cursor rusak), 401
18	GET /events/:id	L, visibilitas BR-E9	200	404
19	POST /events	perm `event:create`	201 + Location	403, 422
20	PUT /events/:id	own atau `event:update:any`	200	403, 404, 409 BR-E5, 422
21	PATCH /events/:id	own atau `event:update:any`	200	400, 403, 404, 409, 422
22	DELETE /events/:id	own atau `event:delete:any`, hanya draft	204	403, 404, 409 BR-E6
23	PATCH /events/:id/status	own atau `event:update:any`	200	403, 404, 409 `INVALID_TRANSITION` / `HAS_ACTIVE_REGISTRATIONS`, 422 BR-E4
Ticket types
#	Endpoint	Akses	Sukses	Error utama
24	GET /events/:id/ticket-types	L, event terlihat	200 (tanpa paginasi: dibatasi maks 10 per event, BR-T1)	404
25	POST /events/:id/ticket-types	own atau `event:update:any`	201 + Location	403, 404, 409, 422 BR-E3
26	PUT /ticket-types/:id	own (event induk) atau `event:update:any`	200	403, 404, 409, 422 BR-T3
27	DELETE /ticket-types/:id	own atau `event:update:any`	204	403, 404, 409 BR-T4
Registrations, payment, check-in (Modul 6, 7)
#	Endpoint	Akses	Sukses	Error utama
28	POST /events/:id/registrations	perm `registration:create`	201 + Location	404, 409 `QUOTA_FULL`/`ALREADY_REGISTERED`/`REGISTRATION_CLOSED`, 422 BR-R2
29	GET /events/:id/registrations	own atau `registration:read:any`	200 CURSOR; `Accept: text/csv` untuk ekspor; filter `status`	403, 404, 406
30	GET /registrations	L; participant hanya miliknya, `registration:read:any` melihat semua	200 offset: `page,limit,sort,order,status,event_id`	400
31	GET /registrations/:id	pemilik, organizer event induk, atau `registration:read:any`	200	403, 404
32	PUT /registrations/:id/payment	pemilik registrasi	200 (payment `submitted`)	403, 404, 409 BR-P1, 422
33	PATCH /registrations/:id/payment	own event atau `registration:manage:any`	200 (`verified`/`rejected`)	403, 404, 409 BR-P2, 422
34	PATCH /registrations/:id/status	pemilik (H-24), own event, atau `registration:manage:any`	200 (`cancelled`)	403, 404, 409 `INVALID_TRANSITION`
35	POST /events/:id/check-ins	own atau `registration:manage:any`	201 (registrasi `checked_in`)	403, 404, 409 `CHECKIN_NOT_OPEN`/`NOT_CONFIRMED`/`ALREADY_CHECKED_IN`, 422
Total 35 endpoint (termasuk /health). Nomor ini dipakai di matriks pengujian dan laporan.
8.3 Request body dan validasi (tag struct, Modul 7)
Struct	Aturan
RegisterRequest	`username` required,min=3,max=30,username; `email` required,email,max=120; `full_name` required,min=2,max=100; `password` required,max=72,strongpassword
LoginRequest	`username` required; `password` required (tanpa aturan kekuatan)
PutUserRequest	`username`, `email`, `full_name` seperti Register; `is_active` bool
PatchUserRequest	semua field POINTER, tag `omitnil`; body kosong -> 400
AssignRoleRequest	`role` required
CategoryRequest	`name` required,min=3,max=80; `description` max=255
CreateEventRequest / PutEventRequest	`title` required,min=5,max=150; `description` max=2000; `location` required,max=200; `capacity` required,min=1,max=100000; `starts_at`, `ends_at`, `registration_deadline` required (RFC3339); `category_id` required,min=1
PatchEventRequest	semua field pointer + `omitnil`
EventStatusRequest	`status` required,oneof=published cancelled finished
TicketTypeRequest	`name` required,min=2,max=80; `price` POINTER required,min=0,max=100000000; `quota` required,min=1,max=100000
RegisterEventRequest	`ticket_type_id` required,min=1
SubmitPaymentRequest	`method` required,oneof=bank_transfer ewallet qris; `reference_number` required,min=6,max=60,nospace
VerifyPaymentRequest	`status` required,oneof=verified rejected; `note` max=200 (wajib bila rejected, aturan antar-field di service)
CancelRegistrationRequest	`status` required,oneof=cancelled
CheckInRequest	`ticket_code` required,len=16
Catatan jebakan: `price` bernilai 0 itu sah (tiket gratis), tetapi `required` pada tipe `int64` menolak 0.
Pakai pointer + `required` agar "tidak dikirim" (nil) dibedakan dari "dikirim 0". Ini pemakaian pointer Modul 1 dan 2 pada kasus nyata.
Aturan antar-field yang bukan tag (fungsi murni di `event_rules.go` dll): urutan waktu BR-E2, jumlah kuota BR-E3, patch kosong,
catatan wajib saat rejected, ketergantungan role pada `is_active`.
8.4 Contoh request dan response
Register tiket berbayar:
```
POST /api/v1/events/12/registrations
Authorization: Bearer <token participant>
{ "ticket_type_id": 31 }

201 Created
Location: /api/v1/registrations/204
{ "success": true, "message": "pendaftaran berhasil, silakan kirim bukti pembayaran",
  "data": { "id": 204, "event_id": 12, "ticket_type_id": 31, "status": "pending",
            "price_at_purchase": 75000, "ticket_code": null,
            "payment": { "status": "unpaid", "amount": 75000 } } }
```
Kuota habis:
```
409 Conflict
{ "success": false, "code": "QUOTA_FULL", "message": "kuota tiket sudah habis", "request_id": "..." }
```
Daftar peserta CSV (`Accept: text/csv`):
```
200 OK
Content-Type: text/csv; charset=utf-8
Content-Disposition: attachment; filename="registrations-event-12.csv"
id,full_name,email,ticket,status,ticket_code,created_at
204,Sari Wulandari,sari@mail.com,Reguler,confirmed,K7Q2M9XD4PA8WZ3B,2026-10-03T10:12:00Z
```
Pagination cursor:
```
GET /api/v1/events?limit=5
200 { ..., "meta": { "limit": 5, "next_cursor": "MTc4OTMwODUyMDM2MDg3NDAwMHw4", "has_more": true } }
```
---
9. Persyaratan Non-Fungsional
9.1 Keamanan (Modul 5 dan 6)
Ancaman	Penutup
Password bocor	bcrypt cost 12, `json:"-"`, tidak pernah di-log
Token palsu / algorithm confusion	cek method HMAC eksplisit, `iss` dan `exp` wajib, secret >= 32 karakter dari env, tolak start bila kurang
Brute force	rate limiter login 5/menit/IP, 429 + Retry-After
User enumeration	pesan login identik + dummy hash
Mass assignment	struct request terpisah, tidak ada `role`, `organizer_id`, `status`, `owner` pada body yang tidak semestinya
IDOR	ownership di service, hak diperiksa sebelum query data
SQL injection	parameter `$n`, whitelist ORDER BY
Refresh token dicuri	disimpan sebagai SHA-256, rotasi, logout dan nonaktif mencabut
Payload besar	`BodyLimit` 1 MB
CORS longgar	allowlist dari `ALLOWED_ORIGINS`
Kebocoran detail	500 generik, detail hanya di log dengan `request_id`
Keterbatasan yang diterima dan wajib ditulis di laporan: role di dalam JWT baru berlaku setelah access token (15 menit) kedaluwarsa (Modul 6 A.9);
HTTPS wajib di produksi; belum ada MFA, reset password, dan penguncian akun.
9.2 Performa
Pool koneksi `DB_MAX_CONNS`, timeout 5 detik tiap operasi, `/health` timeout 2 detik, limit maksimal 100, cursor pada daftar yang tumbuh
(events, peserta), index sesuai `ORDER BY`. Bukti `EXPLAIN ANALYZE` untuk satu query cursor dilampirkan di docs.
9.3 Logging dan observability
JSON slog ke stdout dan `logs/app.log` (lumberjack: 10 MB, 5 backup, 14 hari). Satu request, satu baris access log. 4xx -> WARN, 5xx -> ERROR.
Header `X-Request-Id` di setiap response.
9.4 Konfigurasi (`.env.example`, nilai dikosongkan)
`APP_NAME, APP_PORT, LOG_LEVEL, DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME, DB_SSLMODE, DB_MAX_CONNS, JWT_SECRET, JWT_ISSUER, JWT_ACCESS_TTL_MINUTES, JWT_REFRESH_TTL_DAYS, ALLOWED_ORIGINS`.
(Modul memakai variabel DB terpisah, bukan satu `DATABASE_URL`. Ikuti modul.)
---
10. Arsitektur (Modul 4)
Struktur folder dan tabel dependency rule ada di `AGENTS.md`. Ringkasnya: `route -> service -> repository -> model`,
dengan `middleware`, `helper`, `config`, `database` di lapisan luar.
Keputusan arsitektur yang harus bisa dijelaskan:
Keputusan	Alasan	Harga yang dibayar
Controller digabung ke service (struktur kuliah)	Sesuai Modul 4, jumlah package terkendali	Rules terikat Fiber, dimitigasi dengan fungsi murni `*_rules.go`
Infrastruktur Modul 7 (AppError, ErrorHandler, validator) dibangun di fase 0	Menghindari menulis ulang 35 handler nanti	Urutan belajar tidak sama persis dengan urutan modul
Transaksi pgx di dalam repository untuk daftar, batal, verifikasi	Kuota dan status harus berubah bersamaan	Satu hal di luar modul 1–7, dijelaskan di README. Alternatif satu statement CTE bila dosen keberatan
`CHECK (sold <= quota)` di database	Prinsip Modul 3: constraint lebih kuat dari pemeriksaan di aplikasi	Error 23514 harus diterjemahkan jadi 409
Cursor memakai `created_at, id` (bukan `starts_at`)	Kolom tidak berubah, sehingga cursor tidak bergeser	Urutan tidak bisa diganti client
Event tidak terlihat dijawab 404, bukan 403	Menyembunyikan keberadaan draft. Keputusannya butuh data, jadi prinsip "periksa hak sebelum ambil data" Modul 6 tidak berlaku di sini	Berbeda dari endpoint users yang memakai 403
Permission dimuat sekali saat start	Sesuai Modul 6	Perubahan hak butuh restart
---
11. Rencana Pengujian
11.1 Unit test (tanpa server dan tanpa DB, `go test ./...`)
Paket	Yang diuji	Kasus minimal
helper	strength password, JWT (valid, kedaluwarsa, `alg:none`, secret beda), cursor encode/decode (valid, rusak), negotiate (kosong, `*/*`, csv, xml), `PermissionSet.Can` (role/permission tak dikenal, nil)	10
service rules	`CanAccessUser`, `CanManageEvent`, `CanReadRegistration`, transisi registrasi, transisi event, batas H-24, jendela check-in, `ValidateEventTimes`, `ValidateQuotaSum`, `ApplyPatch`	15
11.2 Pengujian endpoint (Postman collection `postman/eventra.postman_collection.json` + environment)
Setiap request punya test script yang memeriksa status DAN bentuk body. Folder per area. Minimal mencakup:
Kebutuhan tugas	Skenario
register berhasil, data invalid	#2 201; #2 422 dengan `fields`; #2 409 username/email ganda
login berhasil, password salah	#3 200; #3 401 (pesan sama untuk username tak ada); enam kali gagal -> 429
tanpa token, token valid, token rusak	#7 401 + `WWW-Authenticate`; #6 200; token diubah satu karakter 401
role salah	participant ke #7 403; organizer ke #11 403
CRUD berhasil	kategori, event (PUT dan PATCH), ticket type
data tidak ditemukan	event/registrasi id 999999 -> 404
ownership	organizer B ke PUT event milik A -> 403; participant B ke GET registrasi milik A -> 403; mass assignment `"role":"admin"` saat register -> tetap participant
business process	daftar sampai check-in; kuota habis; pendaftaran ganda; batal lewat H-24; check-in di luar jendela
pagination dan negotiation	cursor tidak duplikat setelah sisip baris; `Accept: text/csv` berisi data; `application/xml` 406; `*/*` 200 JSON
kegagalan server	matikan PostgreSQL -> 500/503 tanpa detail teknis, ada `request_id`
11.3 Matriks hak akses
`docs/MATRIX_HAK_AKSES.md`: 35 endpoint x (admin, organizer pemilik, organizer bukan pemilik, participant pemilik, participant bukan pemilik, tanpa token)
berisi status HTTP yang benar-benar diperoleh. Sertakan minimal dua pengujian negatif (Modul 6 C.3).
---
12. Fase Pengerjaan
Fase	Fokus	Modul	Keluaran	Gerbang selesai
F0	Fondasi	1, 4, 7	Struktur folder, config, logger, pool, middleware global, helper (response, AppError, ErrorHandler terpusat, validator, query), `/health`, graceful shutdown	build+vet hijau, `/health` 200 dan 503, 404 endpoint tak dikenal berbentuk standar
F1	Autentikasi	5	Migrasi 001 (RBAC, dibutuhkan FK role) dan 002, register, login, refresh, logout, me, rate limiter	Delapan skenario uji keamanan Modul 5 lulus
F2	RBAC, users, categories	6, 2, 3	Migrasi 003, permission set, RequirePermission, ownership, endpoint #7–#16	Matriks hak akses users dan categories terisi
F3	Events dan tiket	2, 3, 7	Migrasi 004–005 (skema registrasi disiapkan lebih dulu karena BR-E5 dan BR-E7 membacanya), endpoint #17–#27, cursor, state machine event	Uji cursor tanpa duplikat, PUT vs PATCH terbukti
F4	Registrasi, payment, check-in	6, 7	Endpoint #28–#35, transaksi, CSV	Kuota tidak oversold pada dua request bersamaan, daftar sampai check-in jalan
F5	Hardening dan review	4, 5, 6, 7	Semua pemeriksaan kebocoran, checklist C.8, perbaikan	Nol pelanggaran
F6	Pengujian	4	Unit test, Postman, matriks hak akses	25+ kasus hijau, matriks lengkap
F7	Dokumentasi	semua	README, API.md, DATABASE.md, MODUL_MAPPING.md, PENJELASAN_KODE.md, AI_USAGE.md, bahan laporan	Semua berkas ada dan konsisten dengan kode
---
13. Deliverables
Berkas	Isi
README.md	14 bagian sesuai tugas: nama, deskripsi, teknologi, requirement, instalasi, env, menjalankan, struktur, database, authentication, authorization, daftar endpoint, testing, mapping Modul 4–7
docs/API.md	Setiap endpoint: method, path, akses, parameter, body, response, status, contoh
docs/DATABASE.md	Tabel, kolom, tipe, PK, FK, nullable, unique, relasi, ERD
docs/MODUL_MAPPING.md	Tabel Modul, Materi, Implementasi, File untuk Modul 1–7
docs/MATRIX_HAK_AKSES.md	Hasil uji matriks
docs/PENJELASAN_KODE.md	Per file: fungsi, alasan, pertanyaan dosen dan jawaban
docs/AI_USAGE.md	Bagian yang dibantu AI, apa yang disesuaikan, alasan memilih pendekatan
docs/ASUMSI.md	Asumsi dan keputusan
postman/	Collection dan environment
Laporan (BAB I–VI)	Dibuat dari dokumen di atas setelah kode selesai
---
14. Risiko dan Asumsi
Risiko / asumsi	Mitigasi
Agent menyalin kode salah dari Modul 7 Bagian B	`AGENTS.md` melarang, acuan Bagian C, review di F5
Kuota oversold saat request bersamaan	CHECK constraint + update atomik dalam transaksi, uji dua request paralel
Transaksi belum diajarkan dosen	Dijelaskan di README, alternatif CTE disiapkan
Scope terlalu besar	P1 bisa dipotong, urutan fase memungkinkan berhenti di F4 dengan produk utuh
Kode tidak bisa dijelaskan	PENJELASAN_KODE.md per fase, latihan menjelaskan satu alur utuh sebelum presentasi
Role di JWT basi sampai 15 menit	Diterima dan ditulis di laporan
Waktu server dan zona waktu	Semua waktu disimpan UTC (`TIMESTAMPTZ`), dikirim RFC3339
Asumsi: satu registrasi = satu tiket (tanpa kuantitas); verifikasi pembayaran manual oleh organizer; tidak ada refund;
event hanya satu lokasi teks; organizer tidak mendaftar ke event (pakai akun participant terpisah).
---
15. Definisi Selesai
[ ] `go build ./... && go vet ./... && go test ./...` hijau
[ ] Lima migrasi berjalan dari database kosong, README menjelaskan langkahnya
[ ] 35 endpoint terdaftar di `route/route.go` dan bisa dibaca sebagai peta hak akses
[ ] Pemeriksaan kebocoran layer (AGENTS.md bagian 8) kosong semua
[ ] Daftar periksa C.8 Modul 7: 14 dari 14
[ ] Matriks hak akses terisi dari hasil uji nyata, dua pengujian negatif ada
[ ] Tidak ada `.env`, `logs/`, secret, atau binary di Git; `.env.example` ada
[ ] Commit bertahap, tidak ada satu commit besar
[ ] `docs/AI_USAGE.md` terisi jujur, `docs/PENJELASAN_KODE.md` lengkap per fase
[ ] Pemilik project sudah berlatih menjelaskan alur: register -> login -> daftar event -> bayar -> verifikasi -> check-in, dari route sampai SQL
