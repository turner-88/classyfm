# Panduan Dasar Admin ClassyFM

Ringkasan langkah operasional harian untuk mulai mengelola website ClassyFM melalui panel
admin. Untuk penjelasan lengkap tiap fitur, lihat **Admin Guide** (`ADMIN_GUIDE-ID`); untuk
fitur khusus superadmin (Users, Legal Pages, SEO, Activity Log), lihat **Superadmin Guide**
(`SUPERADMIN_GUIDE-ID`).

---

## 1. Login

1. Buka **https://classyfm.co.id/admin/login**.
2. Masukkan **Email** dan **Password**, lalu klik **Sign In**.
3. Lupa password? Klik **Forgot password?**, masukkan email terdaftar, lalu ikuti tautan
   reset yang dikirim ke email.

**Ganti password sendiri:** klik nama akun di bagian bawah sidebar (menu **Profile**),
bagian **Change Password**. **Log Out** juga berada di bagian bawah sidebar.

---

## 2. Tata Letak & Konvensi Umum

- **Sidebar kiri** berisi menu: **Radio** (Programs, Broadcasters, About Us) dan
  **Content** (Hero, Hot Release, Podcast, Event, Newsfeed, Media, Ads). Di *mobile*,
  sidebar berada di balik tombol hamburger.
- Halaman daftar punya **Search**, pengurutan kolom (klik judul kolom), dan **50 baris**
  per halaman.
- Setelah **Save**/**Delete** berhasil, muncul pesan hijau. Hapus selalu menampilkan
  dialog konfirmasi.
- Kolom gambar punya pratinjau; gambar otomatis dikompresi di browser (JPG/PNG/WEBP/GIF).
  **Pada formulir edit, biarkan kolom gambar kosong bila tidak ingin mengganti gambar.**
- Status **Active/Published** = lencana hijau; **Inactive/Draft/Hidden** = lencana abu-abu.

---

## 3. Dashboard

Halaman utama panel: kartu **On-air** (acara yang sedang & akan mengudara, status stream,
lagu yang diputar, jumlah pendengar), kartu statistik, daftar **Needs attention**
(hal-hal yang perlu diperbaiki), grafik pendengar & berita masuk, dan status sumber feed.
Tombol cepat: **New program**, **New hot release**, **Refresh feeds**.

---

## 4. Tugas Harian yang Umum

### Program siaran — **Radio → Programs**
Klik **Add Program** (atau Edit). Isi **Title**, deskripsi, **Default broadcasters**,
gambar, dan centang **Active**. Pada formulir edit, gunakan **Weekly Schedule** → **Add
slot** untuk mengatur hari, jam mulai/selesai, dan penyiar tiap slot. Sistem menampilkan
peringatan bila ada jadwal yang bentrok (peringatan, bukan penghalang).

### Penyiar — **Radio → Broadcasters**
Klik **Add Broadcaster** (atau Edit). Isi **Name**, role, bio, foto (potret 4:5), tautan
sosial, dan **Active**. Penugasan ke program diatur dari menu **Programs**, bukan di sini.

### Berita editorial — **Content → Hot Release**
Klik **Add Hot Release**. Isi **Title**, excerpt, **Content**, gambar sampul, **Publish
Date**, lalu centang **Publish** agar tampil. Ikon bintang pada daftar untuk menyorot
(*featured*) langsung.

### Podcast — **Content → Podcast**
Klik **Add Podcast**. Pilih **Series**, tempel **Spotify URL** (`open.spotify.com`) yang
valid, centang **Publish**. Judul, gambar, pemutar, dan deskripsi ditarik otomatis dari
Spotify saat disimpan — gunakan **Refresh from Spotify** bila detail episode berubah.

### Halaman utama — **Content → Hero**
Kelola slide banner beranda (**Add Slide**): badge, judul, excerpt, gambar (2400×1200),
tautan opsional, **Active**. Panel **Hero settings** mengatur perpaduan slide gambar dan
berita.

### Berita agregasi — **Content → Newsfeed**
Berita otomatis dari YouTube/KlikPositif/KataSumbar. Atur **Publish/Hide** (ikon mata) dan
**Feature** (ikon bintang); **Refresh Feed** menarik berita terbaru. Kelola sumber via
tombol **Feed Sources**.

### Lainnya
- **Content → Event:** acara/promo publik.
- **Content → Ads:** banner iklan slot atas/bawah.
- **Content → Media:** URL media sosial resmi (kosongkan untuk menyembunyikan ikon).
- **Radio → About Us:** konten halaman About.

---

## 5. Selanjutnya

- Pengelolaan konten lengkap dan detail tiap kolom → **Admin Guide** (`ADMIN_GUIDE-ID`).
- Menambah pengguna admin, halaman legal, SEO, dan log aktivitas → **Superadmin Guide**
  (`SUPERADMIN_GUIDE-ID`).
