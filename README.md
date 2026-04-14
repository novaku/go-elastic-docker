# go-elastic-search

REST API siap produksi yang dibangun dengan **Go + Gin + Elasticsearch 8**, dikontainerisasi menggunakan **Docker Compose**.  
Mendukung pencarian teks penuh, fuzzy matching, filter, paginasi, pengurutan, bulk indexing, dan graceful shutdown.

---

## Teknologi yang Digunakan

| Lapisan | Teknologi |
| ------- | --------- |
| Bahasa | Go 1.26 |
| Framework HTTP | Gin |
| Search engine | Elasticsearch 8 |
| ES client | `go-elasticsearch/v8` |
| Auth | JWT (HS256) via `golang-jwt/jwt/v5` |
| Logger | `go.uber.org/zap` |
| Container | Docker (distroless runtime image) |
| Orkestrasi | Docker Compose (dev + prod) |

---

## Mulai Cepat (lokal)

```bash
# 1. Clone dan konfigurasi
cp .env.example .env

# 2. Jalankan stack (ES + API)
make dev

# 3. Cek kesehatan layanan
curl http://localhost:8080/health

# 4. Isi data awal
make seed

# 5. Buka QA UI (end-to-end test harness)
open http://localhost:8080/qa/
```

## QA End-to-End UI

API sudah dilengkapi frontend QA berbasis browser di path berikut:

```text
GET /qa/
```

Halaman ini mengimplementasikan semua endpoint backend:

- `POST /auth/login`
- `GET /health`
- `GET /ready`
- `GET /v1/products`
- `POST /v1/products`
- `POST /v1/products/bulk`
- `GET /v1/products/:id`
- `PUT /v1/products/:id`
- `DELETE /v1/products/:id`

Fitur untuk QA:

- Login form — halaman terkunci di balik overlay login; token JWT disimpan di `localStorage`
- Input `API Base URL` (default ke origin saat ini, bisa diarahkan ke environment lain)
- Header `Authorization: Bearer <token>` disisipkan otomatis ke setiap request
- Logout otomatis saat token kedaluwarsa (401)
- Form terpisah untuk search, create, get by id, update, delete, dan bulk index
- Panel hasil response JSON per endpoint
- Request log ringkas untuk pelacakan langkah uji

---

## Referensi API

### Health

```http
GET /health     → { "status": "ok" }
GET /ready      → { "status": "ready" }
```

### Autentikasi

Semua endpoint `/v1/*` dilindungi JWT. Dapatkan token terlebih dahulu:

```http
POST /auth/login
Content-Type: application/json

{ "username": "admin", "password": "admin123" }
```

Respons:

```json
{ "token": "<jwt>", "expires_in": 86400 }
```

Gunakan token pada setiap request berikutnya:

```http
Authorization: Bearer <jwt>
```

### Produk

| Method | Path | Deskripsi |
| ------ | ---- | --------- |
| `GET` | `/v1/products` | Cari / tampilkan daftar produk |
| `POST` | `/v1/products` | Buat produk baru |
| `POST` | `/v1/products/bulk` | Bulk-index produk |
| `GET` | `/v1/products/:id` | Ambil produk berdasarkan ID |
| `PUT` | `/v1/products/:id` | Perbarui produk |
| `DELETE` | `/v1/products/:id` | Hapus produk |

### Parameter query pencarian

| Parameter | Tipe | Deskripsi |
| --------- | ---- | --------- |
| `q` | string | Query pencarian teks penuh (fuzzy, multi-field) |
| `category` | string | Pencocokan tepat berdasarkan kategori |
| `brand` | string | Pencocokan tepat berdasarkan merek |
| `min_price` | float | Batas harga minimum |
| `max_price` | float | Batas harga maksimum |
| `is_active` | bool | Filter berdasarkan status aktif |
| `tags` | string (multi) | Filter berdasarkan tag |
| `sort_by` | string | Field untuk pengurutan (contoh: `price`, `created_at`) |
| `sort_dir` | string | `asc` atau `desc` |
| `page` | int | Nomor halaman (default: 1) |
| `page_size` | int | Jumlah hasil per halaman (default: 20, maks: 100) |

### Contoh permintaan

```bash
# 1. Login dan simpan token
TOKEN=$(curl -s -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | grep -o '"token":"[^"]*' | cut -d'"' -f4)

# Pencarian teks penuh
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/v1/products?q=laptop&category=Electronics"

# Rentang harga + hanya yang aktif
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/v1/products?min_price=100&max_price=500&is_active=true"

# Urutkan berdasarkan harga (ascending)
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/v1/products?sort_by=price&sort_dir=asc&page=1&page_size=5"

# Buat produk baru
curl -X POST http://localhost:8080/v1/products \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "Gaming Mouse",
    "description": "High DPI gaming mouse with RGB",
    "category": "Accessories",
    "brand": "GameGear",
    "price": 59.99,
    "stock": 200,
    "tags": ["gaming", "mouse"],
    "is_active": true
  }'

# Bulk index
curl -X POST http://localhost:8080/v1/products/bulk \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d @scripts/seed.json
```

---

## Pengembangan

```bash
make dev          # jalankan full stack Docker
make logs         # pantau log API
make restart-api  # rebuild + restart hanya API
make down         # matikan semua layanan
make run          # jalankan API secara lokal (ES harus sudah berjalan)
make test         # jalankan pengujian
make lint         # jalankan golangci-lint
```

## Konfigurasi

Seluruh konfigurasi API dipusatkan di file JSON pada folder `config/`:

- `config/local.json` untuk mode lokal
- `config/production.json` untuk mode produksi

Saat runtime, aplikasi hanya membaca `APP_ENV` untuk menentukan file mana yang dipakai:

```bash
APP_ENV=local       # load config/local.json
APP_ENV=production  # load config/production.json
```

### Konfigurasi JWT

Blok `jwt` pada setiap file config:

```json
"jwt": {
  "secret": "ganti-dengan-secret-min-32-karakter",
  "expiry_duration": "24h",
  "admin_username": "admin",
  "admin_password": "admin123"
}
```

> **Penting untuk produksi:** Gunakan secret acak minimal 32 karakter dan password yang kuat. Jangan commit nilai sensitif ke repository.

| Field | Default (local) | Keterangan |
| ----- | --------------- | ---------- |
| `secret` | `local-dev-secret-...` | Kunci penandatanganan HMAC-SHA256 |
| `expiry_duration` | `24h` | Masa berlaku token (format Go duration) |
| `admin_username` | `admin` | Username login |
| `admin_password` | `admin123` | Password login |

---

## Deploy ke Produksi

### Single-host (Docker Compose)

```bash
# APP_ENV memilih file config API
export APP_ENV=production

# Variable tambahan ini dipakai oleh service Elasticsearch dan image tag di compose
export ES_PASSWORD=your-strong-password
export API_IMAGE=yourorg/go-elastic-search:1.0.0

make prod-up
```

### CI/CD

```bash
# Build dan push image
make image VERSION=1.2.3
make push  VERSION=1.2.3

# Deploy (di server)
docker compose -f docker-compose.yml -f docker-compose.prod.yml pull
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

### Kubernetes / ECS / Cloud Run

Container API ini bersifat stateless. Skala secara horizontal dengan cara:

1. Deploy image `go-elastic-search` sebagai Deployment/Task
2. Kelola konfigurasi API via `config/production.json` (port, Elasticsearch, CORS, timeout, JWT)
3. Simpan `jwt.secret`, `jwt.admin_password`, dan kredensial Elasticsearch di secret manager, lalu inject saat build/release config

---

## Struktur Proyek

```text
.
├── cmd/api/          # Titik masuk aplikasi (main.go)
├── config/           # Konfigurasi JSON per environment (local/production)
├── internal/
│   ├── elasticsearch/ # Wrapper client ES
│   ├── handler/       # HTTP handler (Gin) — auth.go, product.go
│   ├── middleware/    # Logger, CORS, Recovery, RequestID, JWTAuth
│   └── service/       # Logika bisnis + pembangun query ES
├── scripts/           # Data awal (seed)
├── Dockerfile         # Multi-stage build (distroless runtime)
├── docker-compose.yml       # Lingkungan pengembangan lokal
└── docker-compose.prod.yml  # Overlay produksi
```

---

## Variabel Lingkungan

| Variabel | Default | Deskripsi |
| -------- | ------- | --------- |
| `APP_ENV` | `local` | Selector file config API: `local` atau `production` |
| `ES_PASSWORD` | `changeme` | Dipakai Docker Compose production untuk bootstrap Elasticsearch |
| `API_IMAGE` | `yourorg/go-elastic-search:latest` | Image API yang dipakai di compose production |
| `ES_CLUSTER_NAME` | `prod-cluster` | Nama cluster Elasticsearch untuk compose production |
