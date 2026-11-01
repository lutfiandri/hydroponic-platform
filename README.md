# Hydroponic Platform

Hidroponic monitoring system

## PocketBase

Secret lokal ada di `.env` dan tidak masuk Git. Untuk setup baru, copy
`.env.example` ke `.env`, isi password admin minimal 10 karakter dan encryption
key tepat 32 karakter. Gunakan `openssl rand -hex 16` untuk membuat key.
Sesuaikan `POCKETBASE_UID` dan `POCKETBASE_GID` dengan `id -u` dan `id -g`.

```sh
make admin
make migrate
make up
```

Gunakan Makefile supaya folder data dibuat oleh user lokal sebelum Docker berjalan.
Sintaks volume pendek bisa membuat folder milik root jika langsung menjalankan
Compose saat folder belum ada.

- Dashboard: http://localhost:8090/_/
- Health: http://localhost:8090/api/health
- Login memakai kredensial di `.env`.
- Data, public, dan hooks tersimpan di folder `./pocketbase/`.
- JavaScript migrations tersimpan di `./db-migration/` dan hanya dijalankan lewat `make migrate`.
- Startup memakai direktori migration kosong terpisah; auto-generation migration juga dimatikan.
- Folder `pocketbase` harus dibuat sebelum startup dan dimiliki UID/GID di `.env`.
  Kalau telanjur milik root, jalankan `sudo chown -R "$(id -u):$(id -g)" pocketbase`
  setelah UID/GID di `.env` disamakan dengan user lokal.
- Port hanya bisa diakses dari komputer lokal.

```sh
make logs
make down
```

## MQTT

Broker memakai Eclipse Mosquitto 2 dan ikut menyala lewat `make up`.

- Dari host: `mqtt://localhost:1883`.
- Dari container Compose lain: `mqtt://mqtt:1883`.
- Konfigurasi: `mqtt/mosquitto.conf`.
- Data persistent: `mqtt/data/`, bind mount lokal dan diabaikan Git.
- Broker memakai UID/GID yang sama dengan PocketBase; `make up` membuat folder datanya.
- Mode development tanpa autentikasi, port host hanya di localhost.
  ESP32 di LAN belum bisa terhubung; pasang autentikasi sebelum membuka akses LAN/publik.

`make logs` menampilkan log semua service, dan `make down` mematikan keduanya.

## Backend MQTT (Go)

Backend dijalankan dari host, terpisah dari service Compose. Jalankan `make up`
untuk menyalakan PocketBase dan MQTT, lalu lengkapi `.env` sesuai `.env.example`.
Dari root repo, jalankan:

```sh
cd backend
go run .
```

Backend otomatis membaca `.env` di direktori kerja, lalu `../.env` (root repo
jika dijalankan dari `backend`). Prioritas: env proses > `.env` lokal > `.env`
parent. File yang tidak ada diabaikan; file yang rusak atau tidak bisa dibaca
menghentikan startup. Tidak perlu mengekspor env secara manual.
Quote nilai `.env` yang mengandung spasi atau karakter khusus.
Broker default `tcp://localhost:1883`, client ID default
`hydroponic-backend`, dan PocketBase default `http://localhost:8090`.
`MQTT_USERNAME` dan `MQTT_PASSWORD` opsional untuk broker development.
Pakai client ID unik jika menjalankan lebih dari satu backend; beberapa subscriber
pada topic yang sama akan membuat histori duplikat.
Untuk koneksi terenkripsi gunakan `ssl://host:8883` atau `tls://host:8883`
dengan sertifikat tepercaya. Dalam container, gunakan hostname service seperti
`tcp://mqtt:1883` dan `http://pocketbase:8090`, bukan localhost.

Device harus sudah terdaftar di collection `device`. Identitas hardware diambil
dari topic `devices/{device_code}/reading` atau `devices/{device_code}/event`,
bukan dari payload. Payload `device_id` tidak dipakai.

Contoh publish dari root repo setelah broker aktif, ganti `tank-01` dengan
`device_code` yang terdaftar:

```sh
docker compose exec -T mqtt mosquitto_pub -h localhost -q 1 \
  -t devices/tank-01/reading \
  -m '{"recorded_at":"2026-11-01T10:00:00Z","water_level":75,"ppm":0,"water_temperature":26.5,"battery_percentage":90}'

docker compose exec -T mqtt mosquitto_pub -h localhost -q 1 \
  -t devices/tank-01/event \
  -m '{"occurred_at":"2026-11-01T10:01:00Z","event_type":"pump_on","source":"automatic","status":"success","message":"Pompa dinyalakan"}'
```

Sensor boleh di-omit atau null; minimal satu pengukuran harus ada. Angka nol
adalah pengukuran valid, dan backend mengisi seluruh flag `has_*` otomatis.
`water_level`/`battery_percentage` harus 0–100 dan `ppm` minimal 0.
Timestamp wajib RFC3339 dan disimpan dalam UTC. Event memerlukan `event_type`
nonkosong, `source` automatic/manual/system, dan `status` pending/success/failed.
`message` opsional.

Backend subscribe QoS 1 dan subscribe ulang setelah reconnect. Jangan gunakan
retain untuk histori; pesan replay bertanda retained diabaikan. Error pesan
hanya di-log memakai `log/slog` (JSON ke stderr), tanpa payload, token, atau
password. Tidak ada retry penyimpanan, queue, atau deduplikasi: kegagalan simpan
bisa kehilangan data dan pengiriman ulang MQTT bisa menghasilkan duplikat.
Backend tidak mengubah status atau last_seen_at device.
Autentikasi PocketBase dilakukan saat startup; restart backend jika token
kedaluwarsa atau kredensial diubah. Config, login, koneksi awal, atau subscription
yang gagal menghentikan proses dengan exit code nonzero.

```sh
cd backend
go test ./...
go vet ./...
go build -o /tmp/hydroponic-backend .
```

## Mock MQTT

Dari root repo, jalankan tanpa install npm dependency:

```sh
node tools/mock-mqtt.js
```

Script mengirim reading sensor acak dan event `mock_heartbeat` langsung saat
mulai, lalu tiap 10 detik. Ubah `INTERVAL_MS` dan `DEVICE_CODE` di bagian atas
`tools/mock-mqtt.js`; default device adalah `tank-01`. Publish memakai QoS 1
tanpa retain lewat `mosquitto_pub` di container `mqtt`, jadi Docker dan broker
harus aktif (`make up`). Tekan Ctrl+C untuk berhenti.

Jalankan backend Go dan daftarkan `device_code` yang sama di PocketBase kalau
ingin data tersimpan. Script tidak membuat device atau menyalakan backend.
Kegagalan publish di-log, lalu script mencoba data baru di interval berikutnya.

Test script: `node --test tools/mock-mqtt.test.js`.

## Skema database

Migration `db-migration/001_init_hydroponic.js` membuat `zone`, `device`,
`device_reading`, dan `device_event`. Semua akses API dikunci untuk superuser.
Kolom relasi: `device.zone_id`, `device_reading.device_id`, dan `device_event.device_id`.
Kode hardware disimpan di `device.device_code` (unik), terpisah dari record `id`.
Relasi tidak memakai cascade delete supaya histori tidak terhapus tanpa sengaja.

Nama migration memakai nomor urut: `001_...js`, `002_...js`, dan seterusnya.
`make admin` membuat atau memperbarui superuser memakai kredensial `.env`;
jalankan saat setup pertama sebelum migration supaya akun login sudah tersedia.
`make admin` dan `make migrate` menghentikan PocketBase terlebih dahulu dan
membiarkannya berhenti. Setelah migration selesai, jalankan `make up` untuk
menyalakan server. `make up` tidak menjalankan migration aplikasi.

Untuk migration berikutnya:

```sh
make migrate
make up
```

PocketBase tidak menyimpan Number sebagai null. Saat mengirim sensor, sertakan
flag `has_*` yang sesuai, misalnya `"ppm": 0, "has_ppm": true` berarti hasilnya
benar-benar nol. Flag false berarti tidak ada pengukuran dan angka harus diabaikan.
Flag tidak otomatis mengikuti angka; pengirim harus mengisinya. Untuk menghapus
pengukuran, kirim flag false dan angka 0. Semua flag default false.

Frontend harus login sebagai superuser, bukan menyimpan kredensial di source
atau bundle. Token superuser memberi akses penuh ke backend.

Rollback migration menghapus empat koleksi beserta isinya. Backup data sebelum
rollback; jangan gunakan rollback untuk sekadar memperbaiki kredensial login.
