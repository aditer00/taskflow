# Desain Arsitektur: TaskFlow — Studi Kasus 12-Factor App dengan Laravel & Livewire

**Tanggal:** 2026-11-17  
**Target Pengguna:** DevOps Engineer yang ingin mempelajari dan mempraktikkan ke-12 prinsip *12-Factor App* pada aplikasi web fullstack nyata.

---

## 1. Tujuan & Latar Belakang (Intent & Why)

Membangun **TaskFlow**, sebuah aplikasi *task management* fullstack berbasis **Laravel + Livewire**, yang dirancang khusus sebagai studi kasus nyata penerapan seluruh **12-Factor App**.

Karena pengguna berasal dari latar belakang DevOps dan tidak berfokus pada kompleksitas kode bisnis aplikasi, desain ini menekankan:
- **Fitur aplikasi yang ringkas dan jelas**, cukup untuk memicu kebutuhan infrastruktur nyata (database relasional, session/cache terpusat, antrean/queue asinkron, object storage, email SMTP, dan scheduled job).
- **Pemisahan proses, konfigurasi, build/release/run, logging, dan backing services yang eksplisit**, sehingga setiap faktor dapat diamati dan diuji langsung menggunakan Docker & Docker Compose.

---

## 2. Goals, Non-Goals, dan Anti-Goals

### Goals
- Menerapkan seluruh **12 Faktor** secara nyata, bukan sekadar teori.
- Menjalankan seluruh stack (build hingga runtime) melalui Docker & Docker Compose tanpa bergantung pada PHP/Composer di mesin host.
- Menyediakan endpoint `/health` untuk memeriksa konektivitas *backing services* (PostgreSQL & Redis).
- Menyediakan dokumentasi verifikasi per faktor agar pengguna dapat mengetes langsung (mis. *horizontal scaling*, *log streaming*, *graceful shutdown*, dan *one-off admin task*).

### Non-Goals
- Fitur manajemen proyek kompleks (Gantt chart, time tracking, multi-tenant RBAC, OAuth sosial).
- Manifest Kubernetes/Helm atau pipeline CI/CD eksternal (fokus pada Docker Compose yang *container-ready* dan *Kubernetes-friendly*).

### Anti-Goals
- Menyimpan session, cache, queue, file upload, atau log ke filesystem lokal container (`storage/`).
- Menanamkan file `.env` atau *secrets* ke dalam Docker image.
- Menggabungkan proses web, queue worker, dan scheduler dalam satu container/entrypoint menggunakan supervisor monolitik.

---

## 3. Batasan Lingkungan (Constraints)

- Mesin host hanya memiliki `docker`, `node`, dan `npm`; **tidak ada `php` atau `composer` di host**.
- Seluruh proses *scaffolding*, instalasi dependency (`composer`), kompilasi asset (`npm run build`), dan eksekusi aplikasi harus berjalan di dalam container Docker.

---

## 4. Pendekatan yang Dipertimbangkan (Approaches)

### Pendekatan 1: Multi-Stage Dockerfile + Nginx + PHP-FPM + Process Terpisah *(Dipilih)*
- **Gambaran:** Menggunakan multi-stage Dockerfile (`composer` + `node` di tahap build, lalu image runtime PHP-FPM yang ramping). Di Docker Compose, setiap tipe proses dipisah menjadi service tersendiri (`web`/Nginx, `app`/PHP-FPM, `worker`, `scheduler`) bersama backing services (`postgres`, `redis`, `minio`, `mailpit`).
- **Trade-off:**
  - *Plus:* Pola standar industri di production/Kubernetes; bebas risiko kebocoran state memori antar-request di Livewire; sangat jelas untuk mempraktikkan *Port Binding*, *Concurrency* (`--scale`), dan *Disposability*.
  - *Minus:* Memerlukan konfigurasi terpisah untuk Nginx dan PHP-FPM FastCGI.

### Pendekatan 2: Laravel Octane (FrankenPHP) + Process Terpisah *(Ditolak)*
- **Gambaran:** Menggabungkan web server dan PHP runtime ke satu container FrankenPHP yang langsung melakukan port binding.
- **Trade-off:** Startup cepat dan container lebih sedikit, namun model *long-lived memory* berisiko memunculkan bug state antar-request pada Livewire dan kurang merepresentasikan mayoritas deployment Laravel standar.

### Pendekatan 3: Laravel Sail *(Ditolak)*
- **Gambaran:** Wrapper Docker dev bawaan Laravel.
- **Trade-off:** Menggabungkan tool dev dan runtime dalam satu container besar sehingga batas *Build, Release, Run* (Faktor V) dan *Dev/Prod Parity* (Faktor X) menjadi kabur.

**Alasan Pemilihan:** Pendekatan 1 dipilih karena paling aman dari sisi kode aplikasi, paling realistis untuk skenario production DevOps, dan membuat batas setiap faktor terlihat jelas.

---

## 5. Arsitektur Sistem & Fitur MVP

### Topologi Container (`docker-compose.yml`)
1. **Application Processes (dari image yang sama, beda command/role):**
   - `web` (`nginx:alpine`): Melakukan *port binding* HTTP (`8080:80`), melayani static assets hasil build, dan meneruskan request PHP ke `app:9000`.
   - `app` (`taskflow-app`, PHP-FPM): Menjalankan aplikasi Laravel + Livewire secara *stateless*.
   - `worker` (`taskflow-app`, `php artisan queue:work redis --tries=3 --timeout=60`): Memproses job asinkron dari Redis.
   - `scheduler` (`taskflow-app`, `php artisan schedule:work`): Menjalankan tugas terjadwal tiap menit.
   - `migrate` (profil one-off / `docker compose run --rm app php artisan migrate --force`): Menjalankan migrasi database sebagai *admin process*.
2. **Backing Services:**
   - `postgres` (`postgres:16-alpine`): Database utama.
   - `redis` (`redis:7-alpine`): Penyimpanan Session, Cache, dan Queue.
   - `minio` (`minio/minio`): Object storage S3-compatible untuk lampiran file task.
   - `mailpit` (`axllent/mailpit`): SMTP server lokal + Web UI untuk menangkap email notifikasi async.

### Fitur Aplikasi (MVP)
- **Autentikasi:** Register, Login, Logout menggunakan session Laravel (disimpan di Redis).
- **Projects:** Membuat, melihat daftar, dan menghapus project.
- **Tasks (Livewire):** CRUD task di dalam project (judul, deskripsi, status `todo`/`in_progress`/`done`, prioritas `low`/`medium`/`high`, dan upload lampiran file ke MinIO/S3).
- **Async Notification:** Membuat atau menyelesaikan task men-dispatch job ke Redis (`SendTaskNotification`), yang diproses oleh `worker` untuk mengirim email ke `mailpit` dan menulis log ke `stderr`.
- **Scheduled Heartbeat / Overdue Check:** Command artisan terjadwal yang berjalan tiap menit via `scheduler` dan mencatat ringkasan ke `stderr`.
- **Healthcheck Endpoint (`GET /health`):** Memeriksa koneksi ke PostgreSQL dan Redis, mengembalikan HTTP `200` jika sehat atau `503` jika ada backing service yang bermasalah.

---

## 6. Pemetaan 12-Factor App

| Faktor | Prinsip | Implementasi di TaskFlow |
| :--- | :--- | :--- |
| **I. Codebase** | Satu codebase dilacak di version control, banyak deploy | Satu repo Git `/home/aditya/taskflow` (subfolder `laravel/`) untuk seluruh environment (local/staging/prod). |
| **II. Dependencies** | Deklarasi & isolasi dependency secara eksplisit | `composer.json`/`composer.lock` & `package.json`/`package-lock.json` di-install terisolasi di dalam stage build Docker. |
| **III. Config** | Simpan konfigurasi di environment | Semua kredensial & endpoint (`DB_*`, `REDIS_*`, `AWS_*`, `MAIL_*`, `APP_KEY`) disuntikkan via env vars (`docker-compose.yml` / `.env` runtime), tidak ada yang di-hardcode atau masuk ke image. |
| **IV. Backing Services** | Perlakukan backing services sebagai attached resources | PostgreSQL, Redis, MinIO (S3), dan Mailpit terhubung lewat URL/host env var dan dapat ditukar ke managed service cloud tanpa ubah kode. |
| **V. Build, Release, Run** | Pisahkan tahap build, release, dan run secara ketat | Dockerfile multi-stage mengkompilasi vendor & asset Vite di tahap **build**; **release** menggabungkan image + env config; **run** menjalankan proses read-only tanpa build ulang. |
| **VI. Processes** | Eksekusi aplikasi sebagai proses stateless | Tidak ada state di disk lokal container; session & cache di Redis, file upload di MinIO. Container `app` dapat di-scale (`--scale app=2`) atau dihapus kapan saja. |
| **VII. Port Binding** | Ekspor service via port binding | `web` (Nginx) mengekspor port HTTP (`8080:80`), `app` (PHP-FPM) bind di `:9000`, tanpa bergantung pada web server eksternal di host. |
| **VIII. Concurrency** | Scale out melalui model proses | Pemisahan eksplisit tipe proses: `app` (web), `worker` (queue), `scheduler` (cron). Masing-masing dapat di-scale independen. |
| **IX. Disposability** | Startup cepat dan graceful shutdown | `worker` merespons `SIGTERM` untuk menyelesaikan job yang sedang berjalan sebelum berhenti; `stop_grace_period` dikonfigurasi di Compose. |
| **X. Dev/Prod Parity** | Jaga dev, staging, dan prod semirip mungkin | Menggunakan engine yang sama (PostgreSQL, Redis, S3-API via MinIO, Nginx + PHP-FPM) di lokal maupun production. |
| **XI. Logs** | Perlakukan log sebagai event stream | `LOG_CHANNEL=stderr`; Nginx & PHP-FPM juga diarahkan ke `stdout`/`stderr` agar bisa dibaca via `docker compose logs -f`. |
| **XII. Admin Processes** | Jalankan tugas admin sebagai one-off processes | Migrasi DB (`php artisan migrate --force`) dan `tinker` dijalankan sebagai one-off container (`docker compose run --rm app ...`), bukan di-hardcode di startup web. |

---

## 7. Ruang Lingkup yang Diserahkan ke Builder (What's Left to the Builder)

- Pemilihan versi spesifik PHP (8.3/8.4), Laravel (11/12), dan Livewire (3) yang stabil beserta ekstensi PHP yang dibutuhkan (`pdo_pgsql`, `redis`, `pcntl`, `mbstring`, `xml`, `curl`, `zip`).
- Detail konfigurasi `nginx.conf`, `php-fpm`, serta *entrypoint script* yang bersih (tanpa menjalankan migrasi otomatis di container web/worker).
- Desain UI Tailwind CSS yang rapi dan sederhana untuk halaman auth, daftar project, dan papan task.
- Penulisan *automated tests* (Feature tests untuk autentikasi, CRUD Project/Task, dispatch queue job, dan endpoint `/health`) beserta panduan verifikasi 12-Factor di `README.md`.
