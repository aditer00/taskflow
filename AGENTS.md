# TaskFlow Repository Guidelines

This is a multi-language **12-Factor App** case study repository. Each language implementation lives in its own directory and runs via Docker Compose.

## Repository Layout

- `laravel/` — Laravel + Livewire implementation (Nginx + PHP-FPM `8.4.27`)
- `golang/` — Go implementation (`net/http` + `html/template`, single static binary)
- `docs/` — Architecture specs (`docs/specs/`) and implementation plans (`docs/plans/`)
- `DESIGN.md` — Shared UI design direction (light mode, Notion-inspired, `#0075de` primary accent)

## Environment & Tooling

- Host runtimes (`php`, `composer`, `go`) are not required; run builds, migrations, and tests inside each folder's Docker containers.
- Both stacks bind port `8080`. Stop one stack (`docker compose down`) before starting another.
- Keep `.env` files untracked and never bake secrets into Docker images.

## Verification Commands

- **Laravel (`laravel/`):** `docker compose exec -T app php artisan test`
- **Golang (`golang/`):** `docker run --rm -u "$(id -u):$(id -g)" -e HOME=/tmp -v "$(pwd):/app" -w /app golang:1.24.3-alpine go test ./...`

## 12-Factor & UI Rules

- Preserve strict 12-Factor compliance: config via env vars, backing services (`postgres`, `redis`, `minio`, `mailpit`), stateless app processes, logs to `stdout`/`stderr`, and one-off admin processes (`migrate`, `seed`).
- For UI work, read `DESIGN.md` for visual direction and apply `antislop` quality rules.
