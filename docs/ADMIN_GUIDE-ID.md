# ClassyFM Admin Guide

Panduan praktis admin panel ClassyFM bagi *content editor*. Panduan ini mencakup
tugas-tugas operasional sehari-hari: login, mengelola program dan penyiar (*broadcaster*), memublikasikan berita dan
podcast, mengatur halaman utama (*home page*), serta menjaga tautan situs tetap terbarukan.

> **Catatan untuk superadmin:** Jika suatu akun memiliki role **Superadmin**, akan muncul item tambahan yang tidak dilihat pengguna lain — **Legal Pages** (grup Radio), **Contact** (grup Content), serta grup **System** berisi menu Users dan Activity Log. Fitur-fitur tersebut didokumentasikan secara terpisah di [Superadmin Guide](SUPERADMIN_GUIDE-ID.md). Seluruh panduan di dalam dokumen *ini* juga berlaku bagi superadmin.

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
- [Your profile](#13-your-profile)

---

## 1. Getting started

### Signing in

1. Buka `/admin/login`.
2. Masukkan **Email** dan **Password**, lalu klik **Sign In**.

Jika email atau password salah, pesan kesalahan *"Incorrect email or password."* akan ditampilkan. Akun yang
dinonaktifkan tidak dapat login — hubungi superadmin jika akun dinonaktifkan.

### Forgot your password?

1. Pada halaman login, klik **Forgot password?**.
2. Masukkan alamat email, lalu klik **Send Link**.
3. Jika alamat email tersebut terdaftar, tautan reset akan dikirimkan ke email yang bersangkutan. Buka tautan tersebut, masukkan
   password baru (minimal 8 karakter), lalu konfirmasi.
4. Setelah berhasil direset, halaman akan mengarahkan kembali ke formulir login beserta pesan konfirmasi, dan password baru dapat langsung digunakan.

Demi alasan keamanan, panel selalu menampilkan pesan *"If that address is registered, a reset link has been
sent"* — terlepas dari apakah akun tersebut terdaftar atau tidak. Mereset password akan mengeluarkan sesi aktif
dari seluruh perangkat.

### The screen layout

- **Sidebar (kiri):** Menu navigasi utama. Klik logo di bagian atas untuk kembali ke halaman Dashboard.
  Item menu dikelompokkan sebagai berikut:
  - **Dashboard** (atas)
  - **Radio:** Programs, Broadcasters, About Us
  - **Content:** Hero, Hot Release, Podcast, Newsfeed, Media, Ads
- **Akun (bawah sidebar):** Menampilkan nama akun (klik untuk membuka **Profile**) serta tombol **Log Out**.
- Pada perangkat *mobile*, sidebar tersembunyi di balik tombol hamburger (menu).

### Logging out

Klik **Log Out** pada bagian bawah sidebar. Sesi login akan otomatis berakhir (*logout*) setelah 7 hari.

---

## 2. Things that work the same everywhere

Setiap bagian di admin panel memiliki perilaku dan pola interaksi yang konsisten.

- **Search:** Sebagian besar halaman daftar (*list*) dilengkapi kotak pencarian. Ketik kata kunci lalu
  klik **Search**; klik **Reset** untuk mengosongkan pencarian. Jika tidak ada data yang cocok, pesan
  `No results for "…"` akan ditampilkan.
- **Sorting:** Judul kolom (*header*) tabel yang berupa tautan dapat diklik untuk mengurutkan data. Ikon panah
  menunjukkan kolom yang sedang aktif diurutkan — ▲ menaik (*ascending*), ▼ menurun (*descending*). Klik kembali untuk membalik urutan.
- **Pagination:** Halaman daftar menampilkan 50 baris data per halaman. Jika data melebihi satu halaman,
  penomoran halaman **Page X of Y** beserta tombol **Previous** / **Next** akan muncul.
- **Saving:** Setelah proses simpan (*save*) atau hapus (*delete*) berhasil, halaman akan mengarahkan kembali ke daftar data beserta pesan
  konfirmasi berwarna hijau pada bagian atas (misalnya *"Program saved."*).
- **Errors:** Jika isian formulir tidak valid, formulir akan dimuat ulang dengan pesan kesalahan berwarna merah yang
  menjelaskan bagian yang perlu diperbaiki. Isian yang sudah dimasukkan sebelumnya tetap tersimpan.
- **Deleting:** Tombol hapus (*delete*) akan menampilkan dialog konfirmasi *pop-up* pada browser sebelum data dihapus.
- **Unsaved changes:** Jika formulir edit ditinggalkan dengan perubahan yang belum
  disimpan, browser akan menampilkan peringatan sebelum mengalihkan halaman.
- **Images:** Kolom isian gambar dilengkapi dengan pratinjau (*preview*) langsung. Ukuran gambar otomatis disesuaikan (*resize*) dan
  dikompresi di browser sebelum diunggah (pesan *"Compressing…"* mungkin muncul sejenak). Format gambar yang
  didukung adalah JPG, PNG, WEBP, dan GIF. Setiap kolom gambar mencantumkan rekomendasi dimensi yang disarankan.
  **Pada formulir edit, biarkan kolom gambar kosong jika tidak ingin mengganti gambar yang ada saat ini.**

Di seluruh admin panel, status **Active / Published** ditandai dengan lencana (*badge*) hijau, sedangkan status
**Inactive / Draft / Hidden** ditandai dengan lencana abu-abu.

---

## 3. Dashboard

Dashboard merupakan halaman utama yang menyajikan ringkasan status stasiun radio secara *real-time*. Sebagian informasi pada halaman ini diperbarui secara otomatis.

- **Tindakan cepat (bagian atas):** **New program**, **New hot release**, dan **Refresh feeds**
  (memperbarui berita agregasi secara langsung).
- **Kartu On-air:** Menampilkan hari, tanggal, serta jam operasional stasiun, status *stream*, acara yang sedang mengudara (*on-air*) beserta *progress bar*, dan acara yang akan tayang berikutnya. Panel
  **Now playing** menampilkan lagu yang sedang diputar, jumlah pendengar saat ini beserta puncaknya hari ini, serta tautan **Open the live page**.
- **Ubin statistik (*Stat tiles*):** Kartu ringkasan angka yang dapat diklik untuk menuju ke halaman Programs, Broadcasters, Hot Release, dan Newsfeed.
- **Daftar penanganan (*Needs attention*):** Daftar peringatan yang menandai hal-hal yang memerlukan tindakan — sumber *feed* yang gagal atau dinonaktifkan, berita yang belum dipublikasikan, program aktif tanpa jadwal tayang atau penyiar, slot jadwal yang bertumpang tindih (dua acara terjadwal pada waktu yang sama), slot iklan aktif tanpa banner, dan tautan media sosial yang masih kosong. Setiap item menyediakan tautan langsung ke halaman perbaikan. Jika seluruh konfigurasi sudah benar, pesan *"All clear."* akan ditampilkan.
- **Grafik statistik:** Grafik jumlah pendengar dari waktu ke waktu (*listeners-over-time*) yang dapat disesuaikan (harian, per jam, atau interval 5 menit), grafik *"News arriving"* berisi jumlah item berita masuk per sumber setiap harinya, serta bagan jadwal siaran pekan ini (**This week on air**) di mana garis merah menandai waktu siaran yang sedang berjalan saat ini.
- **Sumber feed (*Feed sources*):** Ringkasan status kesehatan masing-masing sumber berita (Healthy / Failing / Disabled), dilengkapi informasi waktu pengambilan terakhir (*fetch*) dan jumlah item.

---

## 4. Programs

Menu Programs berisi daftar acara siaran yang dijadwalkan. **Sidebar → Radio → Programs.**

### Create or edit a program

1. Klik **Add Program** (atau ikon pensil **Edit** pada baris data program).
2. Isi kolom formulir yang tersedia (lihat rincian di bawah), lalu klik **Save**.

Pada program yang sudah tersimpan, tombol **View public page** dapat diklik untuk membuka halaman publik program tersebut.

**Fields**

| Field | Catatan |
|---|---|
| Title | Wajib. |
| Slug | Wajib, unik. Digunakan pada URL publik `/program/slug`. |
| Description | Teks bebas. |
| Default broadcasters | Pilihan ganda (*multi-select*) berupa daftar kotak centang yang dapat dicari. Digunakan untuk setiap slot jadwal siaran yang tidak menentukan penyiarnya sendiri. |
| Program Image | Disarankan 1600×900 (16:9). |
| Order | Angka lebih kecil tampil lebih dulu. |
| Active | Hilangkan centang untuk menyembunyikan program dari situs publik. |

### Weekly schedule

Pada formulir edit tersedia editor **Weekly Schedule**:

1. Klik **Add slot** untuk menambahkan slot siaran baru.
2. Atur hari (**Day**), serta jam mulai (**Start**) dan selesai (**End**) dengan format HH:MM.
3. Secara opsional, tentukan penyiar (**Broadcasters**) untuk slot tersebut. Biarkan opsi **Program default** terpilih jika ingin menggunakan penyiar bawaan program.
4. Hapus slot menggunakan tombol ikon tempat sampah. Klik **Save** untuk menyimpan seluruh perubahan jadwal.

Setiap slot memerlukan hari yang valid serta jam mulai dan jam selesai yang berbeda.

### Konflik jadwal

Panel memantau slot yang bertumpang tindih waktunya dan menampilkannya agar Anda tidak
tanpa sengaja menjadwalkan dua acara sekaligus:

- **Pada daftar Programs**, sebuah banner peringatan muncul di bagian atas setiap kali ada
  slot yang bertumpang tindih di mana pun pada seluruh jadwal stasiun, dengan merinci
  masing-masing bentrokan — misalnya *"Monday: 'Show A' (08:00–10:00) overlaps 'Show B'
  (09:00–11:00)."* Jika jumlahnya lebih dari sepuluh, sisanya diringkas menjadi satu baris
  *"+N more"*.
- **Pada formulir edit sebuah program**, peringatan serupa hanya menampilkan bentrokan yang
  melibatkan program yang sedang Anda sunting — termasuk program yang dua slotnya sendiri
  saling bertumpang tindih.

Ini adalah **peringatan, bukan penghalang** — Anda tetap dapat menyimpan. Tujuannya membantu
Anda menangkap kesalahan, jadi tinjau peringatan tersebut dan sesuaikan waktunya jika
bentrokan itu memang tidak disengaja.

### Delete a program

Klik tombol **Delete** (ikon tempat sampah), lalu konfirmasi *"Delete this program along with its schedule?"*. Tindakan ini juga akan menghapus seluruh slot jadwal siaran yang terikat pada program tersebut.

---

## 5. Broadcasters

Menu ini mengelola data penyiar dan *host* siaran. **Sidebar → Radio → Broadcasters.** Proses penambahan, pengubahan, dan penghapusan data berfungsi sama seperti pada menu Programs (**Add Broadcaster**, ikon pensil Edit, serta konfirmasi hapus *"Delete this broadcaster?"*).

**Fields**

| Field | Catatan |
|---|---|
| Name | Wajib. |
| Slug | Wajib, unik. Digunakan pada URL publik. |
| Role | mis. "Announcer". |
| Bio | Teks bebas. |
| Birthplace / Date of Birth | Opsional. |
| Instagram / X / Facebook | Tautan sosial opsional. |
| Photo | Disarankan 1000×1250 (potret 4:5). |
| Order | Angka lebih kecil tampil lebih dulu. |
| Active | Hilangkan centang untuk menyembunyikan dari situs publik. |

Formulir edit menampilkan daftar **Appears On** yang berisi program-program di mana penyiar ini ditugaskan. Daftar ini bersifat *read-only* (hanya baca) — penugasan penyiar diubah melalui menu **Programs** (pada kolom *Default broadcasters* atau slot jadwal siaran).

---

## 6. Hero (home page slider)

Hero banner merupakan slider gambar dan berita terbaru yang ditampilkan di bagian atas halaman utama (*home page*). **Sidebar → Content → Hero.**

### Manage slides

Proses tambah, edit, dan hapus slide dilakukan dengan cara yang sama seperti menu lainnya (**Add Slide**, ikon pensil Edit, serta konfirmasi *"Delete this hero slide?"*).

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

Di bagian bawah daftar, buka panel **Hero settings** untuk mengatur kombinasi tampilan antara slide gambar dan berita:

- **Slide order:** Urutan alur slider (Newest first / Random / News first / Image slides first).
- **News items** (0–10) dan **Image slides** (0–10): Jumlah item berita dan slide gambar yang ditampilkan. Atur salah satu angka ke 0 jika hanya ingin menampilkan tipe konten lainnya. Klik **Save hero settings** untuk menyimpan pengaturan.

---

## 7. Hot Release (news articles)

Hot Release merupakan artikel berita editorial resmi yang dibuat oleh stasiun radio. **Sidebar → Content → Hot Release.**

### Create or edit an article

1. Klik **Add Hot Release** (atau Edit).
2. Isi kolom formulir yang tersedia, lalu klik **Save**.

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

**Galeri Middle Images:** Tambahkan gambar pendukung agar tampil di bagian tengah artikel. Urutan gambar dapat disesuaikan dengan menyeret pegangan gambar (*drag handle*) atau menggunakan panah **Move up / Move down**. Centang opsi **Remove** untuk menghapus gambar tertentu, dan gunakan pemilih berkas untuk menambahkan beberapa gambar baru sekaligus.

**Bintang penyorot (Featured) pada daftar:** Setiap baris dilengkapi ikon bintang. Klik ikon bintang untuk menyorot (*featured*) atau membatalkan sorotan artikel secara langsung tanpa perlu membuka formulir edit. Konfirmasi hapus dilakukan dengan *"Delete this Hot Release?"*.

---

## 8. Podcasts

Menu ini mengelola episode podcast yang dihubungkan dari Spotify. **Sidebar → Content → Podcast.**

### Add or edit a podcast

1. Klik **Add Podcast** (atau Edit).
2. Isi kolom formulir yang tersedia, lalu klik **Save**.

**Fields**

| Field | Catatan |
|---|---|
| Title | Wajib. Slug dibuat otomatis dari title. |
| Series | Wajib. Pilih dari dropdown (lihat Podcast Series di bawah). |
| Spotify URL | Wajib. Tautan `open.spotify.com` yang valid. Judul, gambar *thumbnail*, pemutar audio (*player*), dan deskripsi akan ditarik secara otomatis dari tautan ini saat disimpan. |
| Description | Hanya-baca. Ditarik otomatis dari tautan Spotify saat disimpan (dan diperbarui setiap kali Spotify URL diubah atau saat menggunakan **Refresh from Spotify**) — tidak ada kolom untuk diisi. |
| Broadcasters | Multi-select opsional. |
| Publish | Centang untuk memublikasikannya. |

**Title**, **Thumbnail**, dan **Description** semuanya diambil dari Spotify secara otomatis saat menyimpan, jadi tidak perlu diunggah atau diketik.

**Refresh from Spotify:** Pada formulir edit tersedia tombol **Refresh from Spotify**. Klik tombol ini untuk menarik ulang judul, *thumbnail*, dan deskripsi dari tautan Spotify saat itu juga — berguna ketika detail episode berubah di Spotify sementara URL-nya tetap sama (detail tersebut juga diperbarui otomatis setiap kali Anda mengubah Spotify URL).

### Podcast Series

Series digunakan untuk mengelompokkan podcast serta mengontrol tombol filter (*chip filter*) pada halaman publik podcast. Aksesnya dapat dilakukan melalui tombol **Series** pada daftar Podcast.

- **Add Series** / Edit / Delete.
- **Fields:** Name (wajib; slug dibuat otomatis), Sort order, dan **Active** (centang = tampil di situs publik).
- Series yang nonaktif ditandai lencana (*badge*) **Inactive** abu-abu pada daftar dan disembunyikan dari tombol filter (*chip*) podcast publik — podcast yang sudah ada di dalamnya tetap tampil pada daftar lengkap.
- Series yang masih memiliki podcast terikat tidak dapat dihapus. Pindahkan atau ubah series podcast tersebut ke kategori lain terlebih dahulu sebelum menghapus series.

---

## 9. Newsfeed (aggregated news)

Newsfeed menampilkan artikel berita yang diambil secara otomatis dari sumber luar seperti **YouTube**, **KlikPositif**, dan **KataSumbar**. Konten berita ini tidak ditulis di admin panel — panel ini hanya mengatur visibilitas konten pada situs publik. **Sidebar → Content → Newsfeed.**

- **Filter tombol (pil) pada bagian atas:** Memilah berita berdasarkan sumber (All / YouTube / KlikPositif / KataSumbar).
- **Publish / Hide** (ikon mata): Mengatur apakah suatu berita ditampilkan atau disembunyikan dari situs publik.
- **Feature** (ikon bintang): Mengatur apakah berita disorot (*featured*).
- **Refresh Feed:** Mengambil berita terbaru dari sumber luar secara langsung.
- Klik judul berita untuk membuka artikel sumber aslinya di tab baru.

### Feed Sources

Menu ini diakses melalui tombol **Feed Sources** pada halaman Newsfeed. Halaman ini menyajikan status kesehatan dari masing-masing sumber berita (status terakhir, waktu pengambilan data terakhir, serta jumlah item). Pada formulir ini terdapat opsi untuk:

- Menandai atau menghapus centang **Active** untuk mengaktifkan atau menonaktifkan sumber berita.
- Mengatur penyesuaian **Endpoint** (channel ID YouTube atau URL RSS); biarkan kosong jika ingin menggunakan nilai bawaan (*default*).
- Klik **Save** untuk menyimpan pengaturan, atau **Refresh Now** untuk langsung menarik berita terbaru.

Gangguan pada salah satu sumber tidak akan memengaruhi pembaruan dari sumber berita lainnya.

---

## 10. Ads

Menu ini mengelola spanduk iklan (*banner ads*) yang tampil di dua lokasi tetap: slot atas (**top**) dan slot bawah (**bottom**). **Sidebar → Content → Ads.**

### Slot settings

Masing-masing slot dilengkapi dengan panel pengaturan **Ads settings**:

- **Display:** Menentukan mode tampilan, baik *Stacked* (menampilkan seluruh banner secara berurutan) atau *Slideshow* (menampilkan banner secara bergantian).
- **Rotate every (sec):** Durasi pergantian banner (2–60 detik) untuk mode *slideshow*.
- **Placeholder text** dan **Show placeholder when empty:** Pengaturan teks dan opsi tampilan saat slot iklan tidak memiliki banner aktif.
- **Slot enabled:** Mengaktifkan atau menonaktifkan seluruh slot iklan.

Klik **Save ads settings** untuk menyimpan perubahan konfigurasi iklan.

### Banners

1. Klik **Add Banner** (atau tautan "Add banner here" di dalam sebuah slot).
2. Isi kolom formulir yang tersedia, lalu klik **Save**.

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

Pengaturan URL akun media sosial resmi stasiun radio yang ditampilkan sebagai ikon tautan pada bagian *header* dan *footer* situs. **Sidebar → Content → Media.**

Tersedia kolom isian untuk platform **Instagram, Facebook, X, YouTube, Spotify**, dan **TikTok**. Masukkan URL `http(s)` secara lengkap, atau **kosongkan kolom jika ingin menyembunyikan ikon platform tersebut**. Klik **Save** untuk menyimpan. Setiap baris dilengkapi dengan status koneksi (Connected / Not set) serta tombol uji tautan untuk memastikan URL dapat dibuka dengan benar.

---

## 12. About Us

Menu untuk mengelola konten halaman About Us pada situs publik. **Sidebar → Radio → About Us.** Pengaturan dilakukan dalam satu formulir:

- **Banner:** Pilih jenis spanduk utama berupa gambar (**Image**) melalui unggahan file atau video (**Video URL**) menggunakan tautan YouTube. Mengubah pilihan jenis media tidak akan menghapus data pada pilihan lainnya.
- **Text segments:** Terdiri dari tiga bagian informasi (profile, music, audience), di mana setiap bagian memiliki kolom **Title** (Judul) dan **Body** (Isi teks). Gunakan baris kosong untuk memisahkan antarparagraf pada bagian isi teks.

Klik **Save** untuk menyimpan.

---

## 13. Your profile

Menu pengelolaan akun pribadi. **Klik nama akun pada bagian bawah sidebar** (atau akses `/admin/profile`). Halaman ini terbagi menjadi dua formulir terpisah:

- **Account Details:** Memperbarui **Name** (nama) dan **Email** (role akun ditampilkan tetapi tidak dapat diubah di sini). Klik **Save** untuk menyimpan perubahan.
- **Change Password:** Masukkan **New Password** baru (minimal 8 karakter) pada kolom password dan konfirmasi, lalu klik **Change Password**.

