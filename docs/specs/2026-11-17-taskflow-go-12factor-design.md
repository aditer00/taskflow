# Desain Arsitektur: TaskFlow (Go) — Studi Kasus 12-Factor App

**Tanggal:** 2026-11-17  
**Target Pengguna:** DevOps Engineer yang mempelajari dan membandingkan implementasi *12-Factor App* pada aplikasi Go (*single static binary*) vs PHP/Laravel.

---

## 1. Tujuan & Latar Belakang

Membangun **TaskFlow (Go)** sebagai implementasi pembanding dari studi kasus TaskFlow Laravel menggunakan **Go (`net/http` + `html/template`)** dengan tampilan dan *backing services* yang identik.

Fokus utama versi Go ini adalah menunjukkan bagaimana bahasa kompilasi statis menyederhanakan lapisan runtime:
- **Tanpa web server terpisah (Nginx/PHP-FPM):** Binary `taskflow` langsung melakukan *port binding* HTTP di `:8080`.
- **Image runtime minimal (~48.4 MB):** Multi-stage build mengompilasi Tailwind CSS + HTML templates ke dalam satu binary statis di atas `alpine:3.21.3`.
- **Satu binary, banyak role proses:** Subcommand `web`, `worker`, `scheduler`, `migrate`, `seed`, dan `init-bucket`.

---

## 2. Arsitektur & Komponen

### Struktur Kode
- `cmd/taskflow/main.go`: Entrypoint CLI, inisialisasi `slog` JSON logger ke `stdout`, sinyal `SIGINT`/`SIGTERM`, dan routing subcommand.
- `internal/config/config.go`: Pembacaan seluruh konfigurasi dari environment variables.
- `internal/store/store.go` & `internal/store/migrations/001_init.sql`: Repository PostgreSQL (`pgx/v5`) + SQL migration ter-embed (`embed.FS`).
- `internal/storage/s3.go`: Client S3 (`aws-sdk-go-v2`) untuk upload/download/delete lampiran di RustFS/MinIO.
- `internal/queue/redis_queue.go`: Antrean asinkron berbasis Redis List (`LPush` / `BRPop`).
- `internal/mailer/smtp.go`: Pengirim email SMTP (`net/smtp`) ke Mailpit.
- `internal/web/server.go` & `internal/web/templates/*.html`: HTTP handler, session cookie Redis, validasi upload, serta UI mengikuti `DESIGN.md`.

### Fitur MVP
- **Autentikasi:** Register, Login, Logout dengan password `bcrypt` dan session tersimpan di Redis (`session:<id>`).
- **Projects & Tasks:** CRUD Project, CRUD Task, pembaruan status (`todo`, `in_progress`, `done`), serta upload/download lampiran ke S3 (maksimal 5 MB, ekstensi `.pdf`, `.png`, `.jpg`, `.jpeg`, `.txt`, `.md`).
- **Async Notification:** Membuat atau menyelesaikan task mendorong pesan ke Redis queue, diproses oleh `worker` untuk mengirim email ke `mailpit`.
- **Scheduler Heartbeat:** Setiap 1 menit, `scheduler` mencatat statistik task (`total_tasks`, `open_tasks`, `overdue_tasks`) ke `stdout`.
- **Healthcheck (`GET /health`):** Memeriksa koneksi PostgreSQL & Redis (`200 ok` / `503 degraded`).

---

## 3. Pemetaan 12-Factor App

| Faktor | Prinsip | Implementasi di TaskFlow (Go) |
| :--- | :--- | :--- |
| **I. Codebase** | Satu repo Git, banyak deploy | Subfolder `golang/` di repo `/home/aditya/taskflow` dengan satu codebase Go. |
| **II. Dependencies** | Deklarasi & isolasi eksplisit | `go.mod`/`go.sum` dan `package.json` di-resolve di dalam container build. |
| **III. Config** | Simpan config di environment | Semua kredensial & endpoint dibaca oleh `config.Load()` dari env vars; `.env` dikecualikan di `.dockerignore`. |
| **IV. Backing Services** | Perlakukan sebagai attached resources | PostgreSQL (`16.16-alpine`), Redis (`7.4.12-alpine`), RustFS (`1.0.6`), Mailpit (`v1.34.0`) terhubung via env vars. |
| **V. Build, Release, Run** | Pisahkan tahap secara ketat | Stage `assets` (Tailwind) -> `builder` (Go binary) -> `runtime` (`alpine:3.21.3` read-only binary). |
| **VI. Processes** | Proses stateless | Session di Redis, lampiran di S3; tidak ada state lokal di container `web`. |
| **VII. Port Binding** | Ekspor service via port binding | `taskflow web` langsung listen di `:$PORT` (`8080`) tanpa Nginx. |
| **VIII. Concurrency** | Scale out melalui model proses | Container terpisah untuk `web`, `worker`, dan `scheduler` dari satu image yang sama. |
| **IX. Disposability** | Startup cepat & graceful shutdown | `signal.NotifyContext` menangkap `SIGTERM` untuk menghentikan HTTP server (`Shutdown`) dan worker dengan rapi. |
| **X. Dev/Prod Parity** | Dev dan prod semirip mungkin | Menggunakan engine yang sama dan versi image yang di-pin eksplisit. |
| **XI. Logs** | Log sebagai event stream | `log/slog` menulis structured JSON langsung ke `os.Stdout`. |
| **XII. Admin Processes** | Tugas admin sebagai one-off | `docker compose run --rm web migrate` dan `seed` memakai binary & image yang sama. |

---

## 4. Perbandingan Singkat: Laravel vs Go

| Aspek | TaskFlow (Laravel) | TaskFlow (Go) |
| :--- | :--- | :--- |
| **Runtime Image** | Nginx + PHP-FPM 8.4 + `vendor/` | `alpine:3.21.3` + 1 static binary (**48.4 MB**) |
| **Port Binding** | Nginx (`80`) -> FastCGI (`9000`) | Go `net/http` langsung (`8080`) |
| **Frontend & Interaksi** | Vite + Livewire (AJAX snapshot) | Tailwind CLI + `html/template` (SSR + form POST) |
| **Startup & Shutdown** | PHP-FPM + Artisan queue worker | Binary startup <50ms + `http.Server.Shutdown` |
