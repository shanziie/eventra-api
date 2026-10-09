# Eventra API

Eventra API merupakan backend untuk sistem manajemen event dan tiket. API ini menyediakan fitur autentikasi pengguna, pengelolaan event, jenis tiket, registrasi peserta, pembayaran, check-in, serta pengaturan hak akses berdasarkan role (RBAC).

Proyek ini dikembangkan sebagai tugas Praktikum Pemrograman Backend Lanjut, D4 Teknik Informatika, Universitas Airlangga.

## 1. Teknologi yang Digunakan

* **Bahasa:** Go
* **Framework:** Fiber v2 (`github.com/gofiber/fiber/v2`)
* **Database:** PostgreSQL
* **Database Driver:** `github.com/jackc/pgx/v5` dan `pgxpool`
* **Environment Configuration:** `github.com/joho/godotenv`
* **Authentication:** `github.com/golang-jwt/v5` dengan JWT HS256
* **Password Hashing:** `golang.org/x/crypto/bcrypt`
* **Validation:** `github.com/go-playground/validator/v10`
* **Logging:** `log/slog` dan `gopkg.in/natefinch/lumberjack.v2`
* **Testing:** Go testing package

## 2. Persyaratan Sistem

Sebelum menjalankan aplikasi, pastikan sudah tersedia:

* Go versi 1.21 atau sesuai dengan kebutuhan di `go.mod`
* PostgreSQL
* Git

## 3. Instalasi

Clone repository dan masuk ke direktori proyek:

```bash
git clone https://github.com/shanziie/eventra-api.git
cd eventra-api
go mod download
```

## 4. Konfigurasi Environment

Salin file `.env.example` menjadi `.env`, kemudian sesuaikan konfigurasi database dan JWT dengan lingkungan lokal.

Untuk PowerShell Windows:

```powershell
Copy-Item .env.example .env
```

Pastikan `JWT_SECRET` memiliki panjang minimal 32 karakter. Gunakan secret sendiri untuk lingkungan lokal dan jangan mengunggah kredensial asli ke repository publik.

## 5. Menyiapkan Database

Buat database PostgreSQL sesuai konfigurasi pada file `.env`.

Jalankan file migrasi SQL di folder `migrations/` secara berurutan:

1. `001_rbac.sql`: tabel role, permission, dan relasinya.
2. `002_users_auth.sql`: tabel pengguna dan refresh token.
3. `003_categories.sql`: tabel kategori event.
4. `004_events.sql`: tabel event dan jenis tiket.
5. `005_registrations.sql`: tabel registrasi dan pembayaran.

Pastikan semua migrasi berhasil dijalankan sebelum memulai aplikasi.

## 6. Menjalankan Aplikasi

Jalankan perintah berikut dari direktori utama proyek:

```bash
go run .
```

Server akan berjalan menggunakan port yang ditentukan dalam konfigurasi aplikasi. Periksa konfigurasi lokal untuk mengetahui alamat dan port yang digunakan.

## 7. Struktur Folder

```text
eventra-api/
├── app/
│   ├── model/        # Model dan struktur request/response
│   ├── repository/   # Query dan akses database
│   └── service/      # Logika bisnis aplikasi
├── config/           # Konfigurasi aplikasi
├── database/         # Koneksi database
├── helper/           # Fungsi bantuan
├── middleware/       # Middleware autentikasi dan otorisasi
├── migrations/       # Migrasi database
├── postman/          # Koleksi request API
├── route/            # Definisi endpoint API
├── logs/             # File log lokal
├── main.go
├── go.mod
└── README.md
```

## 8. Authentication

Fitur autentikasi yang tersedia meliputi:

* Registrasi akun dengan password yang di-hash menggunakan bcrypt.
* Login menggunakan JWT access token.
* Refresh token untuk memperbarui sesi autentikasi.
* Logout dengan pencabutan refresh token.
* Pembatasan request login berdasarkan alamat IP.
* Penanganan login yang membantu mencegah *user enumeration*.

Saat registrasi, role pengguna ditentukan oleh server sesuai aturan aplikasi.

## 9. Authorization dan RBAC

Eventra API menerapkan *Role-Based Access Control* (RBAC) untuk mengatur akses pengguna terhadap endpoint dan data.

Penerapannya mencakup:

* Pemeriksaan autentikasi sebelum pemeriksaan permission.
* Pengaturan permission berdasarkan role.
* Pembatasan akses berdasarkan kepemilikan data.
* Proteksi operasi pengelolaan pengguna, event, dan registrasi sesuai hak akses.

## 10. Fitur dan Endpoint API

Endpoint menggunakan prefix `/api/v1`. Daftar route lengkap dapat dilihat di `route/route.go`.

| Modul          | Fitur                                                       |
| -------------- | ----------------------------------------------------------- |
| Health         | Pemeriksaan status aplikasi                                 |
| Authentication | Register, login, refresh token, logout, dan profil pengguna |
| Users          | Pengelolaan pengguna dan role                               |
| Categories     | Pengelolaan kategori event                                  |
| Events         | Pembuatan, pengambilan, perubahan, dan penghapusan event    |
| Ticket Types   | Pengelolaan jenis tiket, harga, dan kuota                   |
| Registrations  | Pendaftaran peserta dan pengelolaan registrasi              |
| Payments       | Pengajuan dan verifikasi pembayaran                         |
| Check-in       | Pencatatan kehadiran peserta                                |

Beberapa endpoint juga mendukung pagination dan pengambilan daftar registrasi dalam format CSV sesuai implementasinya.

## 11. Pengujian

Untuk menjalankan pengujian pada seluruh package, gunakan:

```bash
go test ./...
```

Koleksi request untuk pengujian API tersedia di:

`postman/eventra-api.postman_collection.json`

Import file tersebut ke Postman, lalu sesuaikan base URL, token autentikasi, dan data request dengan konfigurasi lokal.

## 12. Informasi Proyek

**Eventra API**
Praktikum Pemrograman Backend Lanjut
D4 Teknik Informatika, Universitas Airlangga
