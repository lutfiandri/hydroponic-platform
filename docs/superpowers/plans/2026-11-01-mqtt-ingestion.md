# MQTT Ingestion Implementation Plan

**Goal:** Backend Go menerima JSON MQTT dan menyimpannya ke PocketBase melalui REST API.

**Architecture:** Satu proses, satu file handler per collection. Tidak menambah service/repository layer atau mengubah schema database.

**Tech stack:** Go 1.25.1, Eclipse Paho MQTT Go, standard-library `net/http`, `encoding/json`, dan `log/slog`.

## Aturan

- Semua logging wajib menggunakan `log/slog`.
- Error per pesan cukup di-log lalu pesan dilewati; tidak ada retry penyimpanan, queue, atau dead-letter topic.
- Config/koneksi awal yang tidak valid di-log lalu proses berhenti dengan exit code nonzero.
- Jangan log password, token, atau payload mentah.
- Device harus sudah terdaftar; tidak membuat device otomatis.
- Tidak mengubah status/last_seen_at device dalam scope ini.

## Struktur

```text
backend/
├── AGENTS.md
├── go.mod
├── go.sum
├── main.go
├── pocketbase.go
└── handler/
    ├── device_reading.go
    ├── device_reading_test.go
    ├── device_event.go
    └── device_event_test.go
```

- `backend/main.go`: config env, wiring dependency, MQTT subscription, dan graceful shutdown.
- `backend/pocketbase.go`: autentikasi superuser, lookup device_code, dan penyimpanan record melalui HTTP dengan timeout serta pemeriksaan status response.
- `backend/handler/device_reading.go`: decode, validasi, mapping, dan simpan ke `device_reading`.
- `backend/handler/device_event.go`: decode, validasi, mapping, dan simpan ke `device_event`.
- Main memberikan fungsi lookup dan save ke handler; package handler tidak mengimpor package main.

## Config

Tambahkan ke `.env.example` root:

```dotenv
MQTT_BROKER=tcp://localhost:1883
MQTT_CLIENT_ID=hydroponic-backend
MQTT_USERNAME=
MQTT_PASSWORD=
POCKETBASE_URL=http://localhost:8090
```

Gunakan `POCKETBASE_ADMIN_EMAIL` dan `POCKETBASE_ADMIN_PASSWORD` yang sudah ada. Go membaca environment, bukan file `.env` otomatis. Dokumentasikan cara mengekspor env saat menjalankan backend lokal; konfigurasi container baru di luar scope. Dalam container, gunakan hostname service PocketBase, bukan localhost.

Subscribe QoS 1 ke dua topic berikut, aktifkan reconnect dan subscribe ulang setelah reconnect:

```text
devices/+/reading
devices/+/event
```

Publisher memakai retain false. Abaikan pesan retained yang diterima agar histori lama tidak tersimpan ulang. Identitas berasal dari segmen `device_code` pada topic; lookup menjadi record ID PocketBase dengan parameter filter yang di-escape dengan benar. Deployment di luar jaringan tepercaya memerlukan TLS dan broker ACL per device.

## Kontrak JSON

Topic `devices/{device_code}/reading`:

```json
{
  "recorded_at": "2026-11-01T10:00:00Z",
  "water_level": 75,
  "ppm": 850,
  "water_temperature": 26.5,
  "battery_percentage": 90
}
```

- Timestamp wajib RFC3339; normalisasi ke UTC saat disimpan.
- Sensor opsional: omit atau null berarti tidak ada pengukuran; minimal satu sensor harus terisi.
- Gunakan pointer angka: nilai nol tetap dianggap pengukuran.
- Backend mengisi setiap flag `has_*`; sensor kosong disimpan sebagai angka 0 dengan flag false.
- Validasi water_level dan battery_percentage 0–100, ppm >= 0; water_temperature dalam Celsius.

Topic `devices/{device_code}/event`:

```json
{
  "occurred_at": "2026-11-01T10:01:00Z",
  "event_type": "pump_on",
  "source": "automatic",
  "status": "success",
  "message": "Pompa dinyalakan"
}
```

- occurred_at wajib RFC3339; normalisasi ke UTC.
- event_type wajib teks nonkosong; tidak dibatasi ke daftar event tertentu.
- source wajib automatic/manual/system; status wajib pending/success/failed.
- message opsional.
- Handler membentuk field yang disimpan secara eksplisit; payload tidak boleh mengganti device_id dari hasil lookup.

## Langkah implementasi

- [x] Tulis test handler dengan lookup/save palsu: reading valid, sensor nol, sensor kosong/null, tidak ada sensor, rentang invalid, event valid, enum invalid, timestamp invalid, JSON rusak, serta device tidak dikenal. Jalankan dari backend: `go test ./handler`, pastikan gagal sebelum implementasi.
- [x] Implementasikan kedua handler sesuai kontrak; test juga memastikan kegagalan lookup/save dikembalikan tanpa retry. Jalankan `go test ./handler` sampai lulus.
- [x] Tambahkan test `httptest` untuk login, lookup device_code yang mengandung karakter filter, penyimpanan record, response non-2xx, dan response rusak. Implementasikan fungsi PocketBase di `backend/pocketbase.go`; kredensial dan token hanya di memori.
- [x] Implementasikan `backend/main.go`, tambahkan dependency Paho, validasi env, gunakan slog untuk error dengan atribut topic/device_code/error. Daftarkan subscription dan tangani signal shutdown tanpa retry penyimpanan.
- [x] Perbarui `.env.example` dan `README.md` dengan config serta contoh publish kedua JSON. Tidak menambah broker MQTT atau deployment baru.
- [x] Jalankan dari backend: `go test ./...`, `go vet ./...`, dan `go build ./...`. Smoke test dengan broker dan PocketBase: publish kedua payload, verifikasi record serta flag sensor, lalu publish payload invalid dan pastikan hanya menghasilkan log.

## Hasil verifikasi

- `go test -race -cover ./...`, `go vet ./...`, dan build lulus. Coverage handler 100%.
- Smoke test memakai Mosquitto dan PocketBase lokal: reading/event tersimpan, flag nol/kosong benar, timestamp UTC, lookup device_code dengan quote/backslash benar, dan device_id palsu diabaikan.
- Replay retained diabaikan, payload invalid hanya menghasilkan log JSON tanpa payload mentah, reconnect melakukan subscribe ulang, dan SIGTERM berhenti bersih.
- Fixture database dan retained message untuk smoke test sudah dibersihkan. Backend test sudah dihentikan.
- Belum menguji broker berautentikasi atau TLS. Autentikasi PocketBase hanya saat startup; token kedaluwarsa memerlukan restart backend.

## Batasan yang diterima

QoS 1 dapat menghasilkan pesan duplikat. Tidak ada deduplikasi atau jaminan bebas kehilangan data: pesan yang gagal disimpan hanya di-log dan dilewati. Koneksi MQTT boleh reconnect, tetapi penulisan PocketBase yang gagal tidak diulang.
