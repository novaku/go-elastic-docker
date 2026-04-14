# go-elastic-search

REST API siap produksi yang dibangun dengan **Go + Gin + Elasticsearch 8**, dikontainerisasi menggunakan **Docker Compose**.  
Mendukung pencarian teks penuh, fuzzy matching, filter, paginasi, pengurutan, bulk indexing, dan graceful shutdown.

---

## Teknologi yang Digunakan

| Lapisan | Teknologi |
| ------- | --------- |
| Bahasa | Go 1.22 |
| Framework HTTP | Gin |
| Search engine | Elasticsearch 8 |
| ES client | `go-elasticsearch/v8` |
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
```

---

## Referensi API

### Health

```http
GET /health     → { "status": "ok" }
GET /ready      → { "status": "ready" }
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
# Pencarian teks penuh
curl "http://localhost:8080/v1/products?q=laptop&category=Electronics"

# Rentang harga + hanya yang aktif
curl "http://localhost:8080/v1/products?min_price=100&max_price=500&is_active=true"

# Urutkan berdasarkan harga (ascending)
curl "http://localhost:8080/v1/products?sort_by=price&sort_dir=asc&page=1&page_size=5"

# Buat produk baru
curl -X POST http://localhost:8080/v1/products \
  -H "Content-Type: application/json" \
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
2. Kelola konfigurasi API via `config/production.json` (port, Elasticsearch, CORS, timeout)
3. Simpan kredensial sensitif dengan secret manager, lalu inject saat build/release config

---

## Struktur Proyek

```text
.
├── cmd/api/          # Titik masuk aplikasi (main.go)
├── config/           # Konfigurasi JSON per environment (local/production)
├── internal/
│   ├── elasticsearch/ # Wrapper client ES
│   ├── handler/       # HTTP handler (Gin)
│   ├── middleware/    # Logger, CORS, Recovery, RequestID
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
