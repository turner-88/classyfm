# ClassyFM Admin Guide

Panduan praktis untuk admin panel ClassyFM bagi content editor. Panduan ini mencakup
tugas-tugas sehari-hari: login, mengelola programs dan broadcasters, memublikasikan news dan
podcast, mengatur home page, memoderasi chat, serta menjaga tautan situs tetap mutakhir.

> **Catatan untuk superadmin:** jika sebuah akun memiliki role **Superadmin**, akan muncul
> pula grup **System** di sidebar (Users, Activity Log). Fitur-fitur tersebut didokumentasikan
> terpisah di [Superadmin Guide](SUPERADMIN_GUIDE-ID.md). Semua yang ada di panduan *ini* juga
> berlaku untuk superadmin.

## Table of Contents

- [Getting started](#1-getting-started)
- [Things that work the same everywhere](#2-things-that-work-the-same-everywhere)
- [Dashboard](#3-dashboard)
- [Programs](#4-programs)
- [Broadcasters](#5-broadcasters)
- [Hero (home page slider)](#6-hero-home-page-slider)
- [Hot Release (news articles)](#7-hot-release-news-articles)
- [Podcasts](#8-podcasts)
- [Newsfeed (aggregated news)](#9-newsfeed-aggregated-news)
- [Ads](#10-ads)
- [Media links](#11-media-links)
- [About Us](#12-about-us)
- [Chat moderation](#13-chat-moderation)
- [Your profile](#14-your-profile)

---

## 1. Getting started

### Signing in

1. Buka `/admin/login`.
2. Masukkan **Email** dan **Password** lalu klik **Sign In**.

Jika email atau password salah, akan muncul "Incorrect email or password." Akun yang
dinonaktifkan tidak bisa login — hubungi superadmin jika akun tampak dinonaktifkan.

### Forgot your password?

1. Di halaman login klik **Forgot password?**
2. Masukkan email lalu klik **Send Link**.
3. Jika alamat tersebut terdaftar, tautan reset akan dikirim ke email. Buka tautan itu, pilih
   password baru (minimal 8 karakter) lalu konfirmasi.
4. Setelah reset, halaman kembali ke login dengan konfirmasi dan password baru bisa dipakai
   untuk masuk.

Demi keamanan, panel selalu menampilkan "If that address is registered, a reset link has been
sent" — terlepas apakah akun tersebut ada atau tidak. Me-reset password akan mengeluarkan sesi
dari semua perangkat.

### The screen layout

- **Sidebar (kiri):** navigasi utama. Logo di bagian atas mengembalikan tampilan ke Dashboard.
  Item menu dikelompokkan:
  - **Dashboard** (atas)
  - **Radio:** Programs, Broadcasters, About Us
  - **Content:** Hero, Hot Release, Podcast, Newsfeed, Media, Ads
  - **Community:** Chat
- **Akun (bawah sidebar):** nama akun (klik untuk membuka **Profile**) dan tombol **Log Out**.
- Di mobile, sidebar disembunyikan di balik tombol hamburger (menu).

### Logging out

Klik **Log Out** di bagian bawah sidebar. Sesi otomatis logout setelah 7 hari.

---

## 2. Things that work the same everywhere

Setelah hal-hal ini dipahami, setiap bagian akan berperilaku konsisten.

- **Search:** sebagian besar halaman list memiliki kotak pencarian. Ketik sebuah kata lalu
  klik **Search**; klik **Reset** untuk menghapusnya. Jika tidak ada yang cocok, muncul
  `No results for "…"`.
- **Sorting:** header tabel yang berupa tautan bisa diklik untuk mengurutkan. Panah
  menunjukkan kolom aktif — ▲ menaik, ▼ menurun. Klik lagi untuk membalik.
- **Pagination:** list menampilkan 50 baris per halaman. Bila ada lebih dari satu halaman,
  muncul **Page X of Y** dengan tombol **Previous** / **Next**.
- **Saving:** setelah save atau delete berhasil, halaman kembali ke list dengan pesan
  konfirmasi hijau di bagian atas (misalnya "Program saved.").
- **Errors:** jika ada yang tidak valid, form dimuat ulang dengan pesan merah yang
  menjelaskan apa yang harus diperbaiki. Isian yang sudah dimasukkan tetap dipertahankan.
- **Deleting:** tombol delete meminta konfirmasi lewat popup browser sebelum ada yang
  dihapus.
- **Unsaved changes:** jika form edit hendak ditinggalkan dengan perubahan yang belum
  disimpan, browser akan memperingatkan sebelum berpindah.
- **Images:** field gambar menampilkan preview langsung. Gambar otomatis di-resize dan
  dikompres di browser sebelum diunggah (sesaat mungkin muncul "Compressing…"). Tipe yang
  diterima adalah JPG, PNG, WEBP, dan GIF, dan tiap field menampilkan dimensi yang disarankan.
  **Pada form edit, biarkan field gambar kosong untuk mempertahankan gambar saat ini** — file
  hanya perlu dipilih jika gambar ingin diganti.

Di seluruh panel, item **Active / Published** ditampilkan dengan badge hijau dan item
**Inactive / Draft / Hidden** dengan badge abu-abu.

---

## 3. Dashboard

Dashboard adalah halaman awal sekaligus ringkasan langsung stasiun. Sebagian isinya
menyegarkan diri secara otomatis.

- **Quick actions (atas):** **New program**, **New hot release**, dan **Refresh feeds**
  (mengambil aggregated news terbaru saat itu juga).
- **On-air card:** menampilkan hari, tanggal, dan jam stasiun, apakah stream sedang aktif,
  apa yang sedang on air (dengan progress bar), serta apa yang tayang berikutnya. Panel
  **Now playing** menampilkan lagu yang sedang diputar, berapa banyak orang yang mendengarkan
  saat ini dan puncak hari ini, serta tautan **Open the live page**.
- **Stat tiles:** kartu yang bisa diklik untuk Programs, Broadcasters, Hot Release, dan
  newsfeed — masing-masing langsung menautkan ke bagiannya.
- **Needs attention:** daftar triase yang menandai hal-hal yang perlu diperbaiki — feed
  source yang gagal atau dinonaktifkan, news yang belum dipublikasikan dan menunggu review,
  program aktif tanpa jam tayang atau tanpa broadcaster, slot ads yang aktif tanpa banner,
  dan tautan sosial yang kosong. Tiap item menautkan ke halaman tempat masalahnya bisa
  diperbaiki. Bila tidak ada yang perlu dikerjakan, tertulis "All clear."
- **Charts:** grafik listeners-over-time (beralih antara tampilan Daily / Hourly / 5-minute),
  grafik "News arriving" berisi jumlah item per hari per source, dan peta jadwal **This week
  on air** (blok menautkan ke jadwal program; garis merah menandai "now").
- **Feed sources:** ringkasan kesehatan tiap source (Healthy / Failing / Disabled) dengan
  waktu fetch terakhir dan jumlah item.

---

## 4. Programs

Programs adalah acara-acara pada jadwal. **Sidebar → Radio → Programs.**

### Create or edit a program

1. Klik **Add Program** (atau ikon pensil **Edit** pada suatu baris).
2. Isi field-nya (lihat di bawah) lalu klik **Save**.

Pada program yang sudah ada, tombol **View public page** membuka halaman live-nya.

**Fields**

| Field | Catatan |
|---|---|
| Title | Wajib. |
| Slug | Wajib, unik. Digunakan pada URL publik `/program/slug`. |
| Description | Teks bebas. |
| Default broadcasters | Multi-select (daftar checkbox yang bisa dicari). Digunakan untuk setiap schedule slot yang tidak menetapkan host-nya sendiri. |
| Program Image | Disarankan 1600×900 (16:9). |
| Order | Angka lebih kecil tampil lebih dulu. |
| Active | Hilangkan centang untuk menyembunyikan program dari situs publik. |

### Weekly schedule

Pada form edit terdapat editor **Weekly Schedule**:

1. Klik **Add slot** untuk tiap slot siaran.
2. Tetapkan **Day**, waktu **Start**, dan **End** (HH:MM).
3. Opsional, tetapkan **Broadcasters** untuk slot tersebut. Biarkan sebagai **Program
   default** untuk memakai default broadcasters program.
4. Hapus slot dengan tombol tempat sampahnya. Klik **Save** untuk menyimpan seluruh jadwal.

Tiap slot butuh day yang valid dan waktu start yang berbeda dari waktu end-nya.

### Delete a program

Klik tombol **Delete** (tempat sampah) lalu konfirmasi *"Delete this program along with its
schedule?"* — ini juga menghapus schedule slot milik program tersebut.

---

## 5. Broadcasters

Para penyiar dan host on-air. **Sidebar → Radio → Broadcasters.** Create, edit, dan delete
bekerja persis seperti Programs (**Add Broadcaster**, ikon pensil Edit, Delete → *"Delete
this broadcaster?"*).

**Fields**

| Field | Catatan |
|---|---|
| Name | Wajib. |
| Slug | Wajib, unik. Digunakan pada URL publik. |
| Role | mis. "Announcer". |
| Bio | Teks bebas. |
| Birthplace / Date of Birth | Opsional. |
| Instagram / Twitter / Facebook | Tautan sosial opsional. |
| Photo | Disarankan 1000×1250 (potret 4:5). |
| Order | Angka lebih kecil tampil lebih dulu. |
| Active | Hilangkan centang untuk menyembunyikan dari situs publik. |

Form edit menampilkan daftar **Appears On** berisi programs tempat broadcaster ini
ditugaskan. Di sini bersifat read-only — penugasan diubah dari sisi **Program** (field
*Default broadcasters* program atau broadcasters pada suatu schedule slot).

---

## 6. Hero (home page slider)

Hero berputar di bagian atas home page yang memadukan slide gambar dengan news terbaru.
**Sidebar → Content → Hero.**

### Manage slides

Create/edit/delete slide seperti bagian lainnya (**Add Slide**, Edit, Delete → *"Delete this
hero slide?"*).

**Slide fields**

| Field | Catatan |
|---|---|
| Badge | Label kecil yang tampil pada slide. |
| Title | Judul slide. |
| Excerpt | Teks pendukung singkat. |
| Image | Wajib. Disarankan 2400×1200 (full-bleed). |
| Link | Opsional. URL `http(s)` lengkap atau path `/relative`. |
| Open the link in a new tab | Opsional. |
| Order | Angka lebih kecil tampil lebih dulu. |
| Active | Hilangkan centang untuk menyembunyikan slide. |

### Hero settings

Di bawah list, buka panel **Hero settings** untuk mengatur bagaimana slide dan news
dipadukan:

- **Slide order:** Newest first / Random / News first / Image slides first.
- **News items** (0–10) dan **Image slides** (0–10): berapa banyak masing-masing yang
  disertakan. Tetapkan salah satu ke 0 untuk hanya menampilkan tipe lainnya. Klik **Save hero
  settings**.

---

## 7. Hot Release (news articles)

Hot Release adalah news editorial milik stasiun sendiri. **Sidebar → Content → Hot Release.**

### Create or edit an article

1. Klik **Add Hot Release** (atau Edit).
2. Isi field-nya lalu klik **Save**.

**Fields**

| Field | Catatan |
|---|---|
| Title | Wajib. |
| Slug | Wajib, unik. Digunakan pada URL `/news/…`. |
| Excerpt | Ringkasan singkat. |
| Content | Teks biasa — tag HTML apa pun otomatis dibuang saat disimpan. |
| Hot Release Image | Gambar sampul. Disarankan 1600×900. |
| Middle Images | Galeri di dalam artikel (lihat di bawah). |
| Publish Date | Wajib (tanggal + waktu). |
| Publish | Centang untuk memublikasikannya. |
| Featured | Centang untuk menyorotinya. |

**Middle Images gallery:** tambahkan beberapa gambar agar tampil di dalam artikel. Urutkan
ulang gambar yang ada dengan menyeret pegangannya atau memakai panah **Move up / Move down**,
centang **Remove** untuk menghapus salah satunya, dan gunakan pemilih multi-file untuk
menambahkan gambar baru di akhir.

**Feature toggle dari list:** tiap baris memiliki bintang — klik untuk menjadikan artikel
featured/tidak featured tanpa membukanya. Delete mengonfirmasi *"Delete this Hot Release?"*.

---

## 8. Podcasts

Episode podcast yang ditarik dari Spotify. **Sidebar → Content → Podcast.**

### Add or edit a podcast

1. Klik **Add Podcast** (atau Edit).
2. Isi field-nya lalu klik **Save**.

**Fields**

| Field | Catatan |
|---|---|
| Title | Wajib. Slug dibuat otomatis dari title. |
| Series | Wajib. Pilih dari dropdown (lihat Podcast Series di bawah). |
| Spotify URL | Wajib. Tautan `open.spotify.com` yang valid — thumbnail dan player ditarik darinya saat disimpan. |
| Description | Wajib. Teks biasa (HTML dibuang). |
| Broadcasters | Multi-select opsional. |
| Publish | Centang untuk memublikasikannya. |

**Thumbnail** diambil dari Spotify secara otomatis saat menyimpan, jadi tidak perlu diunggah.

### Podcast Series

Series mengelompokkan podcast dan menggerakkan chip filter di halaman podcast publik.
Aksesnya lewat tombol **Series** pada list Podcast.

- **Add Series** / Edit / Delete.
- **Fields:** Name (wajib; slug dibuat otomatis) dan Sort order.
- Series yang masih punya podcast yang ditugaskan padanya tidak bisa dihapus — pindahkan dulu
  podcast-podcast tersebut ke series lain.

---

## 9. Newsfeed (aggregated news)

Newsfeed menampilkan artikel yang ditarik otomatis dari **YouTube**, **KlikPositif**, dan
**KataSumbar**. Konten ini tidak dibuat atau diedit di sini — yang diatur hanyalah apa yang
tampil secara publik. **Sidebar → Content → Newsfeed.**

- **Filter** dengan pil di bagian atas: All / YouTube / KlikPositif / KataSumbar.
- **Publish / Hide** (ikon mata): mengatur apakah suatu item tampil di situs publik.
- **Feature** (ikon bintang): mengatur apakah suatu item disorot.
- **Refresh Feed:** ambil item terbaru saat itu juga.
- Judul menautkan keluar ke artikel aslinya.

### Feed Sources

Akses ini dari tombol **Feed Sources** pada halaman Newsfeed. Ia menampilkan kesehatan tiap
source (status terakhir, waktu fetch terakhir, jumlah item). Dalam satu form tersedia opsi
untuk:

- Centang/hilangkan centang **Active** untuk mengaktifkan atau menonaktifkan sebuah source.
- Menetapkan override **Endpoint** (channel ID atau RSS URL); biarkan kosong untuk memakai
  default bawaan.
- Klik **Save** untuk menyimpan perubahan, atau **Refresh Now** untuk mengambil segera.

Satu source yang rusak tidak pernah memblokir yang lain.

---

## 10. Ads

Banner ads yang tampil di dua tempat tetap: slot **top** dan slot **bottom**.
**Sidebar → Content → Ads.**

### Slot settings

Tiap slot memiliki panel **Ads settings**:

- **Display:** Stacked (semua banner sekaligus) atau Slideshow (bergantian).
- **Rotate every (sec):** 2–60, dipakai pada mode slideshow.
- **Placeholder text** dan **Show placeholder when empty:** apa yang ditampilkan saat slot
  tidak punya banner aktif.
- **Slot enabled:** mengaktifkan atau menonaktifkan seluruh slot.

Klik **Save ads settings** untuk menyimpan perubahan slot.

### Banners

1. Klik **Add Banner** (atau tautan "Add banner here" di dalam sebuah slot).
2. Isi field-nya lalu klik **Save**.

**Fields**

| Field | Catatan |
|---|---|
| Title | Label internal. |
| Alt text | Deskripsi untuk aksesibilitas. |
| Link | URL opsional — terbuka di tab baru. |
| Image | Wajib. Disarankan 1200×150 (leaderboard); di-letterbox, bukan di-crop. |
| Targeting | Centang **All pages**, atau hilangkan centangnya lalu pilih halaman tertentu. |
| Placement | Slot mana (top / bottom). |
| Order | Angka lebih kecil tampil lebih dulu. |
| Active | Hilangkan centang untuk menyembunyikan banner. |

Delete mengonfirmasi *"Delete this banner?"*.

---

## 11. Media links

URL akun resmi stasiun untuk tiap platform, ditampilkan sebagai ikon di header dan footer
situs. **Sidebar → Content → Media.**

Ada satu field masing-masing untuk **Instagram, Facebook, X, YouTube, Spotify**, dan
**TikTok**. Masukkan URL `http(s)` lengkap, atau biarkan sebuah field **kosong untuk
menyembunyikan ikon tersebut**. Klik **Save**. Tiap baris menampilkan badge status (Connected
/ Not set) dan tombol buka-tautan untuk menguji URL yang disimpan.

---

## 12. About Us

Konten halaman About Us publik. **Sidebar → Radio → About Us.** Ini adalah satu form:

- **Banner:** pilih **Image** (unggah) atau **Video URL** (tautan YouTube). Beralih di antara
  keduanya tetap mempertahankan apa pun yang tersimpan untuk yang lain.
- **Text segments:** tiga bagian (profile, music, audience), masing-masing dengan **Title**
  dan **Body**. Pisahkan paragraf pada body dengan satu baris kosong.

Klik **Save**.

---

## 13. Chat moderation

Memoderasi chatroom publik "Connect". **Sidebar → Community → Chat.** Ada dua tab di bagian
atas: **Messages** dan **Users**.

### Messages tab

- **Search** berdasarkan teks pesan atau penulis.
- **Live toggle:** aktifkan untuk otomatis menambahkan pesan baru begitu tiba; saat mati,
  gunakan tombol **Refresh**.
- **Hide / Un-hide** (ikon mata): soft-delete sebuah pesan (pesan yang disembunyikan tampil
  dicoret) atau memulihkannya.
- **Ban / Unban author** (ikon ban): mem-ban orang yang mengirim pesan; ban berlaku pada aksi
  mereka berikutnya.

### Users tab

Menampilkan pengguna chat yang login dengan Google (name, email, badge **Admin** bila
berlaku, dan status ban). Gunakan tombol **Ban / Unban** per baris.

### Public chat kill switch

Di bagian bawah kedua tab, toggle **Enabled** mengaktifkan atau menonaktifkan posting publik
di seluruh situs. Saat mati, pengunjung masih bisa membaca chat tetapi tidak bisa memposting.
Klik **Save**.

---

## 14. Your profile

Mengelola akun sendiri. **Klik nama akun di bagian bawah sidebar** (atau buka
`/admin/profile`). Ada dua form terpisah:

- **Account Details:** perbarui **Name** dan **Email** (role ditampilkan tetapi tidak bisa
  diedit di sini). Klik **Save**.
- **Change Password:** masukkan **New Password** (minimal 8 karakter) lalu konfirmasi, lalu
  klik **Change Password**.
