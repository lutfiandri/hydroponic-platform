# HydroMonitor — Desain dashboard frontend

## Keputusan yang disetujui

Bangun ulang frontend menjadi satu dashboard di `/`, mengikuti struktur wireframe pada `/home/lutfiandri/Pictures/Screenshots/Screenshot From 2026-11-01 23-25-33.png`, tetapi menggunakan visual standar shadcn. Data dummy dulu. Layout mobile satu kolom, tanpa navigasi ke halaman lain. User memberi kebebasan merapikan visual selama tetap mengikuti struktur tersebut.

Antislop selalu `during`; preferensi ini sudah ditulis di `AGENTS.md` dan tidak perlu ditanyakan lagi.

## Tujuan dan batas scope

Dashboard menjawab dua pertanyaan: berapa tandon yang perlu perhatian, dan bagaimana pembacaan tiap tandon. Pengguna bisa membaca seluruh informasi langsung tanpa membuka detail.

Termasuk:
- Reset halaman dan style starter TanStack, lalu siapkan shadcn dengan bersih.
- Header, ringkasan status, dan daftar empat tandon dummy.
- Satu tema gelap dan satu layout mobile dengan batas lebar tablet.
- Pembersihan komponen, aset, dan dependency yang hanya dipakai demo starter.

Tidak termasuk:
- Backend, MQTT, autentikasi, pembacaan sensor langsung, atau penyimpanan data.
- Detail tandon, notifikasi, filter, pencarian, pengaturan, grafik, dan theme switcher.
- Layout desktop, sidebar, bottom navigation, atau halaman tambahan.

## Arah visual

Struktur screenshot dipertahankan: identitas aplikasi → ringkasan kondisi → daftar tandon. Kartu berulang masuk akal karena masing-masing mewakili entitas dengan informasi identik, bukan dekorasi.

Gunakan shadcn `new-york`, base color `zinc`, token warna semantik, dan font sans bawaan sistem. Hindari font eksternal, gradient, glass effect, ilustrasi, dan CSS dekoratif. Ketelitian visual datang dari jarak konsisten, alignment, hierarki teks, dan angka yang mudah dibandingkan.

- **ENERGY: rendah.** Monitoring harus tenang, bukan halaman promosi.
- **RHYTHM: sedang dan konsisten.** Ringkasan ringkas, lalu kartu dengan susunan informasi yang sama.
- **MOTION: tidak ada animasi dekoratif.** Tidak perlu animasi masuk, counter, shimmer, atau transisi tema.

Gunakan komponen shadcn melalui CLI resmi dan periksa hasil generate. Jangan meniru Card/Badge dengan markup buatan. Gunakan props/variant bawaan untuk tampilan; `className` pada komponen untuk kebutuhan layout yang diizinkan lint, bukan mengganti visual komponen.

## Layout

- Container `w-full max-w-3xl mx-auto`: maksimal 768px, tetap menyusut mengikuti viewport.
- Padding horizontal `px-4`, vertikal `py-6`; jarak antarbagian memakai skala Tailwind standar.
- Tidak ada breakpoint yang mengubah susunan menjadi layout desktop.
- Satu kolom kartu; empat metrik pada setiap kartu selalu grid dua kolom.
- Tidak menggunakan arbitrary Tailwind values atau menyembunyikan overflow untuk menutupi masalah layout.

### Header

Judul `HydroMonitor`, subjudul `Monitoring seluruh tandon`, serta label `Data demo`. Tidak ada lonceng atau tautan. Label demo menjelaskan bahwa semua pembacaan adalah contoh, bukan kondisi perangkat asli.

### Ringkasan

Satu baris tiga kelompok: jumlah `Tandon`, `Normal`, dan `Warning`. Angka menonjol, label lebih tenang. Gunakan satu bidang ringkasan dengan pemisah yang ringan, bukan tiga kartu dekoratif terpisah.

Seluruh jumlah dihitung dari array dummy yang sama dengan daftar, tidak ditulis ulang secara manual. Fixture awal harus menghasilkan 4 tandon, 3 normal, dan 1 warning.

### Daftar tandon

Heading `Daftar Tandon`; tanpa `Lihat Semua` karena seluruh tandon langsung ditampilkan.

Setiap entitas memakai komposisi Card shadcn:
- Header: nama tandon sebagai heading, tanaman dan fase sebagai deskripsi, Badge status.
- Content: grid 2×2 berisi `Air`, `PPM`, `Suhu`, dan `Baterai`.
- Label kecil, nilai lebih menonjol dengan angka tabular; satuan melekat pada nilainya.
- Ikon lucide boleh mendampingi label metrik secara konsisten; ikon dekoratif disembunyikan dari assistive technology.
- Tidak ada footer `Detail Tandon`, click handler kartu, chevron, atau pointer palsu.

Status `Normal` memakai Badge `secondary`; `Warning` memakai Badge `destructive`. Teks status selalu ditampilkan, sehingga warna bukan satu-satunya penanda. Jangan membuat variant atau warna tambahan hanya demi meniru screenshot.

## Data dan perilaku

Satu fixture TypeScript lokal; tidak ada fetch, timer, random value, fake loading, atau state management tambahan. Field numerik tetap berupa angka. Format tampilan menggunakan locale `id-ID` melalui API platform.

| Tandon | Tanaman | Fase | Status | Air | PPM | Suhu | Baterai |
| --- | --- | --- | --- | --- | --- | --- | --- |
| A | Selada | Vegetatif | Normal | 78% | 1.250 | 25,5°C | 90% |
| B | Pakcoy | Vegetatif | Normal | 65% | 850 | 27,2°C | 72% |
| C | Kangkung | Panen | Warning | 32% | 980 | 26,1°C | 38% |
| D | Selada | Pembibitan | Normal | 84% | 640 | 24,8°C | 96% |

Data A–C mengacu pada nilai yang terlihat dalam screenshot. Detail D adalah contoh tambahan yang sengaja ditentukan untuk melengkapi fixture; bukan klaim membaca bagian screenshot yang terpotong. Status adalah bagian fixture, bukan hasil rule ambang sensor yang belum diminta.

Aliran data: fixture → perhitungan ringkasan → rendering daftar. Jangan membuat service/repository/API abstraction. Pisahkan fixture dari UI, dan pisahkan kartu tandon agar route tetap singkat. Jangan membuat komponen generik untuk tiap label/angka.

Fixture selalu tersedia sehingga tidak ada kebutuhan UI loading/error jaringan. Jika array kosong, tampilkan pesan `Belum ada tandon` dan ringkasan nol; tidak perlu CTA. Jangan menambah state palsu untuk melengkapi demo.

## Reset frontend dan batas aman

Pertahankan React, TanStack Start/Router, Vite, Nitro, TypeScript, Tailwind, utilitas `cn`, dan konfigurasi deployment yang masih dibutuhkan. Ini reset pengalaman frontend, bukan mengganti framework atau menghapus workspace.

- Ganti isi route `/` dengan dashboard.
- Hapus `/about` dan `/demo/table`, lalu regenerate route tree menggunakan tool proyek, bukan mengedit generated file manual.
- Lepas Header, Footer, ThemeToggle, script tema localStorage, dan UI devtools starter dari root.
- Ganti CSS starter secara utuh dengan setup Tailwind/shadcn yang bersih; jangan menumpuk override di atas style lama.
- Tetapkan `lang="id"`, judul `HydroMonitor`, deskripsi monitoring tandon, dan tema gelap konsisten saat server render/hydration.
- Rapikan `components.json` agar komponen UI berada di `frontend/src/components/ui/`.
- Hapus data tabel demo serta aset dan dependency demo setelah memastikan tidak ada import tersisa. Paket tabel/faker/devtools tidak boleh dihapus apabila masih dipakai oleh kebutuhan framework yang dipertahankan.
- Backend, infra, dokumentasi lain, `.agents/`, dan `skills-lock.json` milik user tidak dihapus.
- Tidak ada redirect kompatibilitas ke route lama.

## Aksesibilitas dan verifikasi

- Struktur heading: satu h1, heading daftar h2, nama tandon h3; metrik memakai pasangan label/nilai yang semantik.
- Tidak ada tautan atau tombol tanpa perilaku. Jika tidak ada interaksi, jangan menambah tab stop.
- Kontras teks minimal 4,5:1; teks besar dan elemen nonteks bermakna minimal 3:1. Nilai diperiksa, bukan diasumsikan lolos karena memakai tema gelap.
- Periksa viewport lebar 320, 375, 430, 768, dan 1280px: tanpa horizontal scroll, metrik terbaca, serta container tetap maksimal 768px di layar besar.
- Test kecil untuk jumlah status dari fixture, array kosong, serta format angka Indonesia. Tidak memasang test framework besar untuk logika sederhana; gunakan test runner bawaan Node bila versi lokal mendukung TypeScript.
- Jalankan generate routes, pemeriksaan TypeScript, build, lint proyek, dan wajib `npm --prefix frontend run lint:shadcn`.
- Periksa hasil di browser, console, render awal/hydration, serta pastikan tidak ada navigasi atau elemen interaktif mati. Jangan mengklaim verifikasi browser jika tool/browser tidak tersedia.

## Kondisi lingkungan yang sudah ditemukan

`npm exec --no -- intent list` gagal dengan `npm error could not determine executable to run`. Jangan mengunduh pengganti diam-diam; laporkan keterbatasan ini. Skill shadcn lokal telah dibaca langsung.

Skill graphify di path yang diinstruksikan tidak ditemukan; tidak ada `graphify-out/graph.json`. Struktur frontend diperiksa langsung sebagai fallback, bukan diklaim dianalisis melalui graphify.

## Kriteria selesai

Hanya dashboard baru yang terdaftar sebagai halaman aplikasi; starter tidak tampil lagi. Dashboard sesuai struktur di atas, jelas berlabel demo, menggunakan shadcn standar, dan tetap layout mobile pada semua lebar layar. Seluruh pemeriksaan yang benar-benar dijalankan dilaporkan beserta keterbatasannya.
