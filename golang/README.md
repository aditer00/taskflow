# TaskFlow (Go) — Studi Kasus 12-Factor App

TaskFlow versi **Golang** adalah aplikasi *task management* fullstack yang dikompilasi menjadi satu *static binary* (`taskflow`), dirancang sebagai **laboratorium praktik 12-Factor App** untuk DevOps Engineer menggunakan Docker & Docker Compose.

---

## Arsitektur Container

| Service | Role | Port |
| :--- | :--- | :--- |
| `web` | HTTP server (`taskflow web`, direct port binding) | `http://localhost:8080` |
| `worker` | Queue consumer (`taskflow worker`, Redis `BRPop`) | - |
| `scheduler` | Periodic ticker (`taskflow scheduler`, tiap 1 menit) | - |
| `postgres` | PostgreSQL `16.16-alpine` (Database utama) | `5432` (internal) |
| `redis` | Redis `7.4.12-alpine` (Session & Queue) | `6379` (internal) |
| `minio` | RustFS `1.0.6` (S3-compatible object storage untuk lampiran) | `9000` (API), `http://localhost:9001` (Console) |
| `mailpit` | Mailpit `v1.34.0` (SMTP lokal + Web UI untuk email async) | `1025` (SMTP), `http://localhost:8025` (UI) |

---

## Quickstart

```bash
# 1. Siapkan environment config (Faktor III)
cp .env.example .env

# 2. Build image multi-stage & jalankan seluruh stack
docker compose build
docker compose up -d

# 3. Jalankan migrasi & seeder sebagai one-off admin process (Faktor XII)
docker compose run --rm web migrate
docker compose run --rm web seed
```

Akses layanan:
- **Aplikasi TaskFlow:** http://localhost:8080 (akun seeder: `test@example.com` / `password`)
- **Healthcheck Endpoint:** http://localhost:8080/health
- **Mailpit UI (Email Async):** http://localhost:8025
- **S3 Console (RustFS):** http://localhost:9001 (`minioadmin` / `minioadmin`)

---

## Lab Verifikasi 12-Factor App

1. **I. Codebase** — Satu repo Git, banyak deploy:
   ```bash
   git status && git log --oneline -n 5
   ```
2. **II. Dependencies** — Terisolasi di `go.mod`/`go.sum` dan `package.json`, host tidak perlu Go atau Node saat runtime:
   ```bash
   docker compose exec web taskflow --help 2>&1 || true
   ```
3. **III. Config** — Konfigurasi disuntikkan via env vars, tidak ada `.env` di dalam image:
   ```bash
   docker compose run --rm --no-deps --entrypoint sh web -c "ls -la /app/.env 2>&1 || true"
   docker compose exec web env | grep -E 'PORT|DB_|REDIS_|AWS_|MAIL_'
   ```
4. **IV. Backing Services** — PostgreSQL, Redis, S3 (RustFS), dan Mailpit terhubung sebagai *attached resources*:
   ```bash
   curl -s http://localhost:8080/health
   ```
5. **V. Build, Release, Run** — Multi-stage `Dockerfile` (`assets` Node -> `builder` Go -> `runtime` Alpine 48.4 MB):
   ```bash
   docker images golang-web
   ```
6. **VI. Processes** — Proses `web` sepenuhnya *stateless* (session di Redis, lampiran di S3):
   ```bash
   docker compose exec redis redis-cli KEYS "session:*"
   ```
7. **VII. Port Binding** — Binary `taskflow web` langsung bind port `:8080` tanpa Nginx terpisah:
   ```bash
   docker compose port web 8080
   ```
8. **VIII. Concurrency** — Scale out tipe proses secara independen:
   ```bash
   docker compose up -d --scale worker=2
   docker compose ps
   ```
9. **IX. Disposability** — Startup hitungan milidetik & *graceful shutdown* pada `SIGTERM`:
   ```bash
   docker compose kill -s SIGTERM worker
   docker compose logs --tail=10 worker
   docker compose up -d worker
   ```
10. **X. Dev/Prod Parity** — Engine sama persis antara lokal dan production (`postgres:16.16-alpine`, `redis:7.4.12-alpine`, `rustfs:1.0.6`).
11. **XI. Logs** — Structured JSON log (`log/slog`) mengalir langsung ke `stdout`:
    ```bash
    docker compose logs -f web worker scheduler
    ```
12. **XII. Admin Processes** — Migrasi & seeder dijalankan sebagai *one-off process* dari binary yang sama:
    ```bash
    docker compose run --rm web migrate
    docker compose run --rm web seed
    ```

---

## Menjalankan Automated Tests

```bash
docker run --rm -u "$(id -u):$(id -g)" -e HOME=/tmp -v "$(pwd):/app" -w /app golang:1.24.3-alpine go test -v ./...
```
