# Surat Penawaran & Proposal Pengembangan Website
### Classy 103.4 FM Padang

---

**Dari:** habiburrahman (Web Developer)
**Kontak:** [email] · [nomor telepon]

**Kepada:** Manajemen Classy 103.4 FM Padang
**Tanggal:** 22 Juli 2026

**Perihal:** Penawaran Jasa Pengembangan Website & Panel Admin Classy 103.4 FM

---

## 1. Surat Penawaran

Dengan hormat,

Bersama surat ini, kami mengajukan penawaran jasa pengembangan website resmi untuk
**Classy 103.4 FM Padang**, mencakup situs publik (beranda, jadwal program, streaming
radio, berita, profil penyiar) beserta panel admin untuk pengelolaan konten secara
mandiri oleh tim internal.

Total investasi yang diajukan untuk keseluruhan pekerjaan ini adalah:

> **Rp 5.000.000,-** (Lima Juta Rupiah)

Nilai ini kami tawarkan sebagai bentuk kerja sama awal dan komitmen jangka panjang
dengan Classy 103.4 FM, bukan sekadar estimasi harga pasar. Rincian lingkup pekerjaan,
teknologi, jadwal, dan rincian biaya dijabarkan pada proposal di bagian berikut.

Kami berharap penawaran ini dapat menjadi awal kerja sama yang baik. Kami terbuka
untuk diskusi lebih lanjut mengenai detail teknis maupun penyesuaian ruang lingkup.

Hormat kami,

**habiburrahman**
Web Developer

---

## 2. Proposal Pengembangan

### 2.1 Gambaran Umum Proyek

ClassyFM adalah website dan panel admin untuk stasiun radio Classy 103.4 FM Padang,
dibangun menggunakan Go dengan rendering HTML di sisi server dan penyimpanan data
MySQL. Aplikasi dirancang sebagai satu binary mandiri (*self-contained*) — seluruh
template dan aset statis disertakan langsung di dalam binary — sehingga proses deploy
sederhana dan minim dependensi di server.

Website terdiri dari dua bagian utama:

- **Situs publik**, untuk audiens umum: beranda, jadwal program siaran, pemutar radio
  live streaming, berita/hot release, dan profil penyiar.
- **Panel admin**, sistem CMS ber-autentikasi sesi untuk tim internal mengelola
  program, penyiar, berita, sumber feed berita, dan pengguna admin.

### 2.2 Ruang Lingkup Pekerjaan

**A. Situs Publik**
- Halaman beranda
- Jadwal & informasi program siaran
- Pemutar radio live streaming
- Halaman berita / hot release
- Halaman profil penyiar (broadcasters)

**B. Panel Admin (CMS)**
- Autentikasi & manajemen sesi login admin
- Manajemen program siaran (tambah/ubah/hapus)
- Manajemen profil penyiar
- Manajemen berita & hot release
- Manajemen sumber feed berita otomatis
- Manajemen pengguna admin & hak akses
- Log audit aktivitas admin
- Reset password admin

**C. Agregasi Berita Otomatis**
- Pengambilan berita otomatis dari kanal YouTube (RSS) dan sumber berita berbasis
  WordPress secara berkala
- Pemantauan status tiap sumber feed, sehingga satu sumber yang bermasalah tidak
  mengganggu sumber lainnya
- Tombol "Refresh" manual di panel admin untuk menarik berita terbaru kapan saja

**D. Infrastruktur & Deployment**
- Penyimpanan data menggunakan MySQL
- Deploy sebagai satu binary mandiri di server (systemd service)
- Backup data otomatis terjadwal
- Pengelolaan upload gambar (program, penyiar, berita)

### 2.3 Teknologi yang Digunakan

| Komponen | Teknologi |
|---|---|
| Backend | Go |
| Database | MySQL |
| Query layer | sqlc (SQL bertipe aman) |
| Migrasi database | golang-migrate |
| Routing | chi router |
| Styling | Tailwind CSS |
| Deployment | systemd, binary mandiri (self-contained) |

### 2.4 Jadwal Pengerjaan

Estimasi waktu pengerjaan: **2–4 minggu**, dengan tahapan sebagai berikut:

| Tahap | Aktivitas | Estimasi Waktu |
|---|---|---|
| 1 | Setup proyek, database, dan backend inti | Minggu ke-1 |
| 2 | Pengembangan panel admin (CMS) | Minggu ke-1–2 |
| 3 | Pengembangan situs publik & integrasi feed berita | Minggu ke-2–3 |
| 4 | Deployment, pengujian (QA), dan serah terima | Minggu ke-3–4 |

### 2.5 Rincian Biaya

| Komponen | Biaya (Rp) |
|---|---:|
| Pengembangan situs publik & panel admin | 3.500.000 |
| Setup deployment & infrastruktur server | 1.000.000 |
| Pengujian (QA) & serah terima | 500.000 |
| **Total** | **5.000.000** |

*Catatan: biaya di atas belum termasuk biaya domain, hosting/server, dan sertifikat
SSL — item ini menjadi tanggung jawab pihak Classy 103.4 FM kecuali disepakati lain.*

### 2.6 Syarat Pembayaran

- **50%** (Rp 2.500.000) sebagai uang muka (DP) saat penawaran disetujui
- **50%** (Rp 2.500.000) saat website selesai dan diserahterimakan

*Skema pembayaran ini dapat disesuaikan sesuai kesepakatan kedua belah pihak.*

### 2.7 Yang Termasuk & Tidak Termasuk

**Termasuk dalam penawaran ini:**
- Pengembangan situs publik dan panel admin sesuai ruang lingkup di atas
- Deployment ke server produksi
- Pengujian fungsional sebelum serah terima
- Dokumen serah terima singkat (akun admin, panduan dasar penggunaan)

**Tidak termasuk dalam penawaran ini:**
- Biaya domain, hosting/server, dan SSL
- Pemeliharaan (maintenance) dan dukungan pasca serah terima — dapat diajukan
  sebagai penawaran terpisah bila diperlukan
- Pengembangan fitur baru di luar ruang lingkup yang disebutkan

### 2.8 Hasil yang Diserahkan (Deliverables)

- Source code lengkap website & panel admin
- Website yang telah aktif dan berjalan di server produksi
- Dokumen serah terima singkat, termasuk kredensial akun admin

---

Demikian penawaran dan proposal ini kami sampaikan. Kami siap berdiskusi lebih lanjut
mengenai detail teknis, jadwal, maupun penyesuaian anggaran sesuai kebutuhan Classy
103.4 FM.

**Disetujui oleh (pihak Classy 103.4 FM):**

Nama: _______________________
Jabatan: _______________________
Tanda tangan / Tanggal: _______________________
