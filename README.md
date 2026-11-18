# TaskFlow — Studi Kasus 12-Factor App (Multi-Language)

Repository ini berisi implementasi **TaskFlow** (*task management* fullstack) dalam beberapa bahasa/framework untuk mempelajari dan membandingkan penerapan **12-Factor App** dari perspektif DevOps.

## Struktur Direktori

| Path | Keterangan | Panduan / Dokumen |
| :--- | :--- | :--- |
| `laravel/` | Implementasi Laravel + Livewire + Nginx + PHP-FPM `8.4.27` (port `8080`) | [laravel/README.md](laravel/README.md) |
| `golang/` | Implementasi Go `1.24.3` (`net/http` + `html/template`, single binary, port `8080`) | [golang/README.md](golang/README.md) |
| `docs/` | Dokumen desain arsitektur (`specs/`) dan rencana implementasi (`plans/`) | [docs/specs/](docs/specs/) |
| `DESIGN.md` | Panduan desain UI bersama (*Notion-inspired*, light mode, aksen `#0075de`) | [DESIGN.md](DESIGN.md) |

> Catatan: Karena stack `laravel/` dan `golang/` sama-sama memakai port `8080`, jalankan satu stack pada satu waktu (`docker compose down` di folder sebelumnya sebelum `docker compose up -d` di folder lain).

## Menjalankan Salah Satu Versi

### Versi Laravel
```bash
cd laravel
cp .env.example .env
docker compose up -d
```

### Versi Golang
```bash
cd golang
cp .env.example .env
docker compose up -d
docker compose run --rm web migrate
docker compose run --rm web seed
```
