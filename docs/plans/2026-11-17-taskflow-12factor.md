# TaskFlow 12-Factor App Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build **TaskFlow**, a fullstack Laravel + Livewire task management MVP packaged as a production-style multi-container Docker Compose stack that explicitly implements and demonstrates all 12-Factor App principles.

**Architecture:** A multi-stage `Dockerfile` builds Composer dependencies and Vite frontend assets into an immutable PHP-FPM runtime image. `docker-compose.yml` runs separate process types (`web` Nginx, `app` PHP-FPM, `worker` Redis queue worker, `scheduler` artisan schedule runner, and one-off `migrate` admin jobs) connected via environment variables to backing services (`postgres`, `redis`, `minio`, `mailpit`), with all logs streaming to `stdout`/`stderr`.

**Tech Stack:** Docker & Docker Compose, PHP 8.3 (FPM Alpine), Nginx Alpine, Laravel 11/12, Livewire 3, Tailwind CSS (Vite), PostgreSQL 16, Redis 7, MinIO (S3), Mailpit (SMTP), PHPUnit.

**Spec:** [2026-11-17-taskflow-12factor-design.md](file:///home/aditya/taskflow/docs/specs/2026-11-17-taskflow-12factor-design.md)

## Global Constraints

- Host has `docker`, `node`, and `npm`, and **no `php` or `composer`** on the host; all PHP/Composer/Artisan commands must execute inside Docker containers.
- Zero local state in container filesystem: sessions, cache, and queue must use `redis`; file attachments must use `s3` (MinIO); logs must use `stderr` (`LOG_CHANNEL=stderr`).
- `.env` and secrets must **never** be baked into the Docker image (`.dockerignore` must exclude `.env*`).
- Database migrations must **never** run automatically inside `web`, `app`, `worker`, or `scheduler` startup; migrations run strictly as a one-off admin process (`docker compose run --rm app php artisan migrate --force`).
- All processes (`web`, `app`, `worker`, `scheduler`) must run in separate containers and handle `SIGTERM` gracefully.

## Review Focus

- Unauthenticated user visiting `/projects` or `/projects/{project}` redirects to `/login` (pinned in Task 3).
- User attempting to view, edit, or delete another user's project or task receives HTTP `403` (pinned in Task 4).
- Task attachment upload exceeding size limit or using disallowed type fails validation without writing to local disk (pinned in Task 4).
- `GET /health` returns HTTP `503` JSON when PostgreSQL or Redis is unreachable instead of an unhandled 500 exception (pinned in Task 2).
- `SendTaskNotification` job retries cleanly on transient failure and logs structured context to `stderr` (pinned in Task 5).

---

### Task 1: Project Scaffolding, Multi-Stage Dockerfile & 12-Factor Docker Compose Stack

**Files:**
- Create: `.gitignore`, `.dockerignore`, `.env.example`
- Create: `Dockerfile`
- Create: `docker/nginx/default.conf`
- Create: `docker-compose.yml`
- Create: Laravel skeleton via `composer create-project` in Docker + `livewire/livewire`, `league/flysystem-aws-s3-v3`, `predis/predis`

**Interfaces:**
- Produces: Running Docker Compose stack (`web` on `:8080`, `app` PHP-FPM `:9000`, `worker`, `scheduler`, `postgres:5432`, `redis:6379`, `minio:9000/:9001`, `mailpit:1025/:8025`) and `docker compose run --rm app php artisan test` test runner.

- [ ] **Step 1: Initialize Git repository and scaffold Laravel project using Dockerized Composer**

Run:
```bash
git init
docker run --rm -u "$(id -u):$(id -g)" -v "$(pwd):/app" -w /app composer:2 create-project laravel/laravel . --no-interaction --prefer-dist
docker run --rm -u "$(id -u):$(id -g)" -v "$(pwd):/app" -w /app composer:2 require livewire/livewire league/flysystem-aws-s3-v3 predis/predis --no-interaction
```
Expected: Laravel skeleton created in `/home/aditya/taskflow/laravel` with `livewire/livewire`, `league/flysystem-aws-s3-v3`, and `predis/predis` in `composer.json`.

- [ ] **Step 2: Configure `.dockerignore` and `.env.example` for 12-Factor defaults**

Update `.dockerignore` to exclude `.env`, `.env.*`, `.git`, `vendor`, `node_modules`, `storage/logs/*`, `tests`.
Update `.env.example` with 12-Factor defaults:
- `APP_NAME=TaskFlow`, `APP_ENV=local`, `APP_DEBUG=false`, `APP_URL=http://localhost:8080`
- `LOG_CHANNEL=stderr`, `LOG_LEVEL=info`
- `DB_CONNECTION=pgsql`, `DB_HOST=postgres`, `DB_PORT=5432`, `DB_DATABASE=taskflow`, `DB_USERNAME=taskflow`, `DB_PASSWORD=secret`
- `SESSION_DRIVER=redis`, `CACHE_STORE=redis`, `QUEUE_CONNECTION=redis`, `REDIS_CLIENT=predis`, `REDIS_HOST=redis`, `REDIS_PORT=6379`
- `FILESYSTEM_DISK=s3`, `AWS_ACCESS_KEY_ID=minioadmin`, `AWS_SECRET_ACCESS_KEY=minioadmin`, `AWS_DEFAULT_REGION=us-east-1`, `AWS_BUCKET=taskflow`, `AWS_ENDPOINT=http://minio:9000`, `AWS_USE_PATH_STYLE_ENDPOINT=true`
- `MAIL_MAILER=smtp`, `MAIL_HOST=mailpit`, `MAIL_PORT=1025`, `MAIL_FROM_ADDRESS=noreply@taskflow.local`

- [ ] **Step 3: Create multi-stage `Dockerfile` and `docker/nginx/default.conf`**

`Dockerfile` must have:
1. `vendor` stage (`composer:2`): `composer install --no-dev --no-interaction --prefer-dist --optimize-autoloader` (plus a `dev` target or build-arg for dev/test dependencies when running tests locally).
2. `assets` stage (`node:22-alpine`): `npm ci && npm run build`.
3. `runtime` stage (`php:8.3-fpm-alpine`): install `pdo_pgsql`, `pcntl`, `zip`, `redis` (via pecl or predis); copy app source, `vendor/`, and `public/build/` from build stages; configure PHP-FPM to log to `/proc/self/fd/2` (`stderr`) and run as non-root `www-data`.

`docker/nginx/default.conf`:
- Listen on port `80`, root `/var/www/html/public`, `access_log /dev/stdout`, `error_log /dev/stderr warn`, pass `*.php` to `app:9000`.

- [ ] **Step 4: Create `docker-compose.yml` with all process types & backing services**

Services:
- `web` (`nginx:alpine`, ports `8080:80`, mounts `docker/nginx/default.conf` and built public assets volume or image)
- `app` (`build: .`, target `dev` or `runtime`, `env_file: .env`, depends on `postgres` & `redis` healthy)
- `worker` (same image as `app`, command: `php artisan queue:work redis --tries=3 --timeout=60 --sleep=3`, `stop_grace_period: 30s`)
- `scheduler` (same image as `app`, command: `php artisan schedule:work`, `stop_grace_period: 15s`)
- `postgres` (`postgres:16-alpine` with healthcheck `pg_isready`)
- `redis` (`redis:7-alpine` with healthcheck `redis-cli ping`)
- `minio` (`minio/minio`, ports `9000:9000`, `9001:9001`, healthcheck) + `minio-init` (`minio/mc` one-off to create `taskflow` bucket)
- `mailpit` (`axllent/mailpit`, ports `8025:8025`, `1025:1025`)

- [ ] **Step 5: Build images, generate `APP_KEY`, boot stack, and verify PHPUnit runs**

Run:
```bash
cp .env.example .env
docker compose build
docker compose up -d postgres redis minio mailpit
docker compose run --rm app php artisan key:generate
docker compose up -d
docker compose run --rm app php artisan test
```
Expected: All containers healthy/running; `php artisan test` exits 0 on default Laravel tests.

- [ ] **Step 6: Commit**

```bash
git add .
git commit -m "chore: scaffold Laravel + Livewire with 12-factor multi-stage Docker stack"
```

---

### Task 2: Healthcheck Endpoint (`/health`) & Streamed Logging

**Files:**
- Create: `app/Http/Controllers/HealthController.php`
- Modify: `routes/web.php`
- Test: `tests/Feature/HealthCheckTest.php`

**Interfaces:**
- Produces: `GET /health` returning `200` JSON `{"status":"ok","checks":{"database":"ok","redis":"ok"}}` when healthy, or `503` JSON `{"status":"degraded","checks":{...}}` when either check fails.

- [ ] **Step 1: Write failing tests in `tests/Feature/HealthCheckTest.php`**

- `test_health_endpoint_returns_200_when_services_are_healthy`: mocks/runs DB + Redis ping, asserts `200` and JSON `status: ok`.
- `test_health_endpoint_returns_503_when_redis_is_unreachable`: forces `Redis::shouldReceive('connection->ping')->andThrow(new \RuntimeException('Redis down'))`, asserts `503` and JSON `status: degraded`, `checks.redis: error`.

- [ ] **Step 2: Run test to verify it fails**

Run: `docker compose run --rm app php artisan test tests/Feature/HealthCheckTest.php`
Expected: FAIL (route `/health` not found / 404).

- [ ] **Step 3: Implement `HealthController::__invoke()` and register `GET /health` in `routes/web.php`**

Check `DB::connection()->getPdo()` and `Redis::connection()->ping()` inside `try/catch` blocks; return `response()->json(..., $allHealthy ? 200 : 503)`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `docker compose run --rm app php artisan test tests/Feature/HealthCheckTest.php`
Expected: PASS (2 tests passed).

- [ ] **Step 5: Commit**

```bash
git add app/Http/Controllers/HealthController.php routes/web.php tests/Feature/HealthCheckTest.php
git commit -m "feat: add /health endpoint checking PostgreSQL and Redis"
```

---

### Task 3: Database Schema, Models & Redis-Backed Authentication

**Files:**
- Create: `database/migrations/2026_11_17_000001_create_projects_table.php`
- Create: `database/migrations/2026_11_17_000002_create_tasks_table.php`
- Create: `app/Models/Project.php`, `app/Models/Task.php`
- Create: `app/Livewire/Auth/Login.php`, `app/Livewire/Auth/Register.php`
- Create: `resources/views/components/layouts/app.blade.php`, `resources/views/livewire/auth/login.blade.php`, `resources/views/livewire/auth/register.blade.php`
- Modify: `routes/web.php`, `app/Models/User.php`
- Test: `tests/Feature/AuthTest.php`

**Interfaces:**
- Produces:
  - `Project` model (`id`, `user_id`, `name`, `description`, `tasks()` HasMany, `user()` BelongsTo).
  - `Task` model (`id`, `project_id`, `user_id`, `title`, `description`, `status` in `['todo','in_progress','done']`, `priority` in `['low','medium','high']`, `due_date`, `attachment_path`, `project()` BelongsTo, `user()` BelongsTo).
  - Routes: `GET /login`, `GET /register`, `POST /logout`, and `auth` middleware protecting `/projects`.

- [ ] **Step 1: Write failing tests in `tests/Feature/AuthTest.php`**

- `test_unauthenticated_user_is_redirected_to_login`: `GET /projects` redirects to `/login`.
- `test_user_can_register_and_is_redirected_to_projects`: Livewire `Register` component creates user, authenticates, redirects to `/projects`.
- `test_user_can_login_and_logout`: Livewire `Login` component authenticates valid user, `POST /logout` logs out and redirects to `/login`.

- [ ] **Step 2: Run test to verify it fails**

Run: `docker compose run --rm app php artisan test tests/Feature/AuthTest.php`
Expected: FAIL (routes/components missing).

- [ ] **Step 3: Create migrations, `Project` & `Task` models, layout, and Livewire Auth components**

- Run one-off migration on dev DB: `docker compose run --rm app php artisan migrate --force`.
- Implement `layouts/app.blade.php` with Tailwind + `@livewireStyles` / `@livewireScripts`.
- Implement `Login` & `Register` Livewire components and `/logout` POST route.

- [ ] **Step 4: Run tests to verify they pass**

Run: `docker compose run --rm app php artisan test tests/Feature/AuthTest.php`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add database/ app/Models/ app/Livewire/Auth/ resources/views/ routes/web.php tests/Feature/AuthTest.php
git commit -m "feat: add projects/tasks schema, models, and Redis-backed Livewire auth"
```

---

### Task 4: Livewire Project & Task Management + S3/MinIO File Attachments

**Files:**
- Create: `app/Livewire/Projects/ProjectIndex.php`, `resources/views/livewire/projects/project-index.blade.php`
- Create: `app/Livewire/Tasks/TaskBoard.php`, `resources/views/livewire/tasks/task-board.blade.php`
- Modify: `routes/web.php`
- Test: `tests/Feature/ProjectAndTaskTest.php`

**Interfaces:**
- Consumes: `Project`, `Task` models and `auth` middleware from Task 3.
- Produces:
  - `GET /projects` (`ProjectIndex` Livewire component)
  - `GET /projects/{project}` (`TaskBoard` Livewire component)
  - `GET /tasks/{task}/attachment` (authorized download/stream from `s3` disk)

- [ ] **Step 1: Write failing tests in `tests/Feature/ProjectAndTaskTest.php`**

- `test_user_can_create_and_delete_own_project`
- `test_user_cannot_view_another_users_project`: asserts `403` when user B visits `/projects/{userAProject}`.
- `test_user_can_create_task_with_s3_attachment_and_update_status`: uses `Storage::fake('s3')` and `UploadedFile::fake()->create('spec.pdf', 100)`, asserts file stored on `s3` disk and task status transitions `todo` -> `done`.
- `test_task_attachment_rejects_oversized_or_disallowed_file`: asserts validation error for `.exe` or >5MB file and nothing written to `s3`.

- [ ] **Step 2: Run test to verify it fails**

Run: `docker compose run --rm app php artisan test tests/Feature/ProjectAndTaskTest.php`
Expected: FAIL.

- [ ] **Step 3: Implement `ProjectIndex`, `TaskBoard`, and attachment download route**

- Enforce `$project->user_id === Auth::id()` (`abort(403)` otherwise) on mount and all actions.
- Validate attachments (`nullable|file|max:5120|mimes:pdf,png,jpg,jpeg,txt,md`) and store using `Storage::disk('s3')->putFile('attachments', $this->attachment)`.
- Delete old S3 object when task is deleted.

- [ ] **Step 4: Run tests to verify they pass**

Run: `docker compose run --rm app php artisan test tests/Feature/ProjectAndTaskTest.php`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add app/Livewire/Projects/ app/Livewire/Tasks/ resources/views/livewire/ routes/web.php tests/Feature/ProjectAndTaskTest.php
git commit -m "feat: add Livewire project and task management with MinIO S3 attachments"
```

---

### Task 5: Async Queue Job, Mailpit Notifications & Scheduled Heartbeat

**Files:**
- Create: `app/Mail/TaskNotificationMail.php`, `resources/views/emails/task-notification.blade.php`
- Create: `app/Jobs/SendTaskNotification.php`
- Create: `app/Console/Commands/TaskHeartbeatCommand.php`
- Modify: `app/Livewire/Tasks/TaskBoard.php`, `routes/console.php`
- Test: `tests/Feature/AsyncAndScheduleTest.php`

**Interfaces:**
- Consumes: `Task` model from Task 3, `TaskBoard` from Task 4.
- Produces:
  - `SendTaskNotification(Task $task, string $event)` queued job on Redis.
  - `tasks:heartbeat` artisan command scheduled `everyMinute()` in `routes/console.php`.

- [ ] **Step 1: Write failing tests in `tests/Feature/AsyncAndScheduleTest.php`**

- `test_creating_or_completing_task_dispatches_notification_job`: uses `Queue::fake()`, asserts `SendTaskNotification::class` pushed when task created and when marked `done`.
- `test_send_task_notification_job_sends_email`: uses `Mail::fake()`, runs `SendTaskNotification::dispatchSync($task, 'created')`, asserts `TaskNotificationMail` sent to task owner.
- `test_heartbeat_command_runs_successfully`: runs `$this->artisan('tasks:heartbeat')->assertExitCode(0)`.

- [ ] **Step 2: Run test to verify it fails**

Run: `docker compose run --rm app php artisan test tests/Feature/AsyncAndScheduleTest.php`
Expected: FAIL.

- [ ] **Step 3: Implement `TaskNotificationMail`, `SendTaskNotification`, `TaskHeartbeatCommand`, and hook into `TaskBoard` & `routes/console.php`**

- Log structured JSON/context message to `Log::info(...)` (goes to `stderr`) inside `SendTaskNotification::handle()` and `TaskHeartbeatCommand::handle()`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `docker compose run --rm app php artisan test tests/Feature/AsyncAndScheduleTest.php`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add app/Mail/ app/Jobs/ app/Console/ app/Livewire/Tasks/TaskBoard.php resources/views/emails/ routes/console.php tests/Feature/AsyncAndScheduleTest.php
git commit -m "feat: add Redis-queued task notifications to Mailpit and scheduled heartbeat"
```

---

### Task 6: 12-Factor Verification Guide (`README.md`) & Full End-to-End Verification

**Files:**
- Create/Update: `README.md`

**Interfaces:**
- Consumes: Complete Docker Compose stack and test suite from Tasks 1–5.

- [ ] **Step 1: Write `README.md` tailored for DevOps learners**

Include:
- Quickstart (`cp .env.example .env`, `docker compose up -d`, `docker compose run --rm app php artisan key:generate`, `docker compose run --rm app php artisan migrate --force`).
- URLs: App (`http://localhost:8080`), Healthcheck (`http://localhost:8080/health`), MinIO Console (`http://localhost:9001`), Mailpit UI (`http://localhost:8025`).
- Hands-on verification commands for **each of the 12 factors** (e.g., `docker compose logs -f app worker scheduler`, `docker compose up -d --scale worker=2`, `docker compose kill -s SIGTERM worker`, `docker compose run --rm app php artisan tinker`).

- [ ] **Step 2: Run full automated test suite and live container smoke check**

Run:
```bash
docker compose run --rm app php artisan test
curl -sf http://localhost:8080/health
```
Expected: All PHPUnit feature/unit tests PASS; `/health` returns `200` with `{"status":"ok","checks":{"database":"ok","redis":"ok"}}`.

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "docs: add 12-factor verification lab guide in README"
```
