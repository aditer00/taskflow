# TaskFlow — Studi Kasus 12-Factor App (Laravel + Livewire)

TaskFlow adalah aplikasi *task management* fullstack berbasis **Laravel + Livewire** yang dirancang khusus sebagai **laboratorium praktik 12-Factor App** untuk DevOps Engineer menggunakan Docker & Docker Compose.

---

## Arsitektur Container

| Service | Role | Port |
| :--- | :--- | :--- |
| `web` | Nginx reverse proxy & static assets | `http://localhost:8080` |
| `app` | PHP-FPM 8.4 stateless application server | `9000` (internal) |
| `worker` | Queue worker (`php artisan queue:work redis`) | - |
| `scheduler` | Task scheduler (`php artisan schedule:work`) | - |
| `postgres` | PostgreSQL 16 (Database utama) | `5432` (internal) |
| `redis` | Redis 7 (Session, Cache, Queue) | `6379` (internal) |
| `minio` | RustFS (S3-compatible object storage untuk lampiran) | `9000` (API), `http://localhost:9001` (Console) |
| `mailpit` | SMTP server lokal + Web UI untuk email async | `1025` (SMTP), `http://localhost:8025` (UI) |

---

## Quickstart

```bash
# 1. Siapkan environment config (Faktor III)
cp .env.example .env

# 2. Build image multi-stage & jalankan backing services
docker compose build
docker compose up -d

# 3. Generate APP_KEY (tulis ke .env host)
APP_KEY=$(docker compose run --rm --no-deps app php artisan key:generate --show | tr -d '\r\n')
sed -i "s|^APP_KEY=.*|APP_KEY=${APP_KEY}|" .env
docker compose up -d

# 4. Jalankan migrasi database sebagai one-off admin process (Faktor XII)
docker compose run --rm app php artisan migrate --force
```

Akses aplikasi:
- **Aplikasi TaskFlow:** http://localhost:8080
- **Healthcheck Endpoint:** http://localhost:8080/health
- **Mailpit UI (Email Async):** http://localhost:8025
- **S3 Console (RustFS):** http://localhost:9001 (`minioadmin` / `minioadmin`)

---

## Lab Verifikasi 12-Factor App

Gunakan perintah berikut untuk menguji setiap faktor secara langsung:

1. **I. Codebase** — Satu repo Git, banyak deploy:
   ```bash
   git status && git log --oneline -n 5
   ```
2. **II. Dependencies** — Terisolasi di dalam container (`composer.lock` & `package-lock.json`), host tidak perlu PHP/Composer:
   ```bash
   docker compose exec app php -v
   ```
3. **III. Config** — Konfigurasi disuntikkan lewat env vars, tidak ada `.env` di dalam image:
   ```bash
   docker compose run --rm --no-deps app ls -la /var/www/html/.env 2>&1 || true
   docker compose exec app env | grep -E 'DB_|REDIS_|LOG_CHANNEL|SESSION_DRIVER'
   ```
4. **IV. Backing Services** — Database, Redis, S3, dan SMTP terhubung sebagai *attached resources*:
   ```bash
   curl -s http://localhost:8080/health
   ```
5. **V. Build, Release, Run** — `Dockerfile` memisahkan tahap `vendor` + `assets` (build) dan `runtime` (run):
   ```bash
   docker build --target runtime -t taskflow:prod .
   ```
6. **VI. Processes** — Proses web sepenuhnya *stateless* (session/cache di Redis, file di S3):
   ```bash
   docker compose exec redis redis-cli KEYS "*"
   ```
7. **VII. Port Binding** — `web` mengekspor `:8080` langsung tanpa bergantung web server host:
   ```bash
   docker compose port web 80
   ```
8. **VIII. Concurrency** — Scale out tipe proses secara independen:
   ```bash
   docker compose up -d --scale worker=2
   docker compose ps
   ```
9. **IX. Disposability** — Startup cepat & *graceful shutdown* dengan `SIGTERM`:
   ```bash
   docker compose kill -s SIGTERM worker
   docker compose logs --tail=15 worker
   docker compose up -d worker
   ```
10. **X. Dev/Prod Parity** — Engine sama persis antara lokal dan production (PostgreSQL, Redis, S3, Nginx + PHP-FPM).
11. **XI. Logs** — Semua log mengalir ke `stdout`/`stderr` sebagai *event stream*:
    ```bash
    docker compose logs -f app worker scheduler web
    ```
12. **XII. Admin Processes** — Migrasi & command admin dijalankan sebagai *one-off process* dari image yang sama:
    ```bash
    docker compose run --rm app php artisan migrate:status
    docker compose run --rm app php artisan tasks:heartbeat
    ```

---

## Menjalankan Automated Tests

```bash
docker compose exec -T app php artisan test
```
