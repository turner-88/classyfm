# ClassyFM Admin Guide

Panduan praktis admin panel ClassyFM bagi *content editor*. Panduan ini mencakup
tugas-tugas operasional sehari-hari: login, mengelola program dan penyiar (*broadcaster*), memublikasikan berita dan
podcast, mengatur halaman utama (*home page*), serta menjaga konten dan tautan situs agar selalu diperbarui (mutakhir).

> **Catatan untuk superadmin:** Jika suatu akun memiliki role **Superadmin**, akan muncul menu tambahan yang tidak dilihat pengguna lain — **Legal Pages** (grup Radio), **Contact** dan **SEO** (grup Content), serta grup **System** berisi menu Users dan Activity Log. Fitur-fitur tersebut didokumentasikan secara terpisah dalam [Superadmin Guide](SUPERADMIN_GUIDE-ID.md). Seluruh panduan di dalam dokumen *ini* juga berlaku bagi superadmin.

## Daftar Isi

- [Memulai](#1-memulai)
- [Perilaku dan konvensi umum](#2-perilaku-dan-konvensi-umum)
- [Dashboard](#3-dashboard)
- [Programs](#4-programs)
- [Broadcasters](#5-broadcasters)
- [Hero (slider halaman utama)](#6-hero-slider-halaman-utama)
- [Hot Release (artikel berita)](#7-hot-release-artikel-berita)
- [Podcasts](#8-podcasts)
- [Event](#9-event)
- [Newsfeed (agregasi berita)](#10-newsfeed-agregasi-berita)
- [Ads](#11-ads)
- [Tautan media sosial (Media links)](#12-tautan-media-sosial-media-links)
- [About Us](#13-about-us)
- [Profil pengguna (Your profile)](#14-profil-pengguna-your-profile)

---

## 1. Memulai

### Login (Masuk)

1. Buka `/admin/login`.
2. Masukkan **Email** dan **Password**, lalu klik **Sign In**.

Jika email atau password salah, pesan kesalahan *"Incorrect email or password."* akan ditampilkan. Akun yang
dinonaktifkan tidak dapat login — hubungi superadmin jika akun dinonaktifkan.

### Lupa password?

1. Pada halaman login, klik **Forgot password?**.
2. Masukkan alamat email yang terdaftar, lalu klik **Send Link**.
3. Jika alamat email tersebut terdaftar, tautan reset akan dikirimkan ke email tersebut. Buka tautan tersebut, masukkan
   password baru (minimal 8 karakter), lalu konfirmasi.
4. Setelah berhasil direset, halaman akan mengarahkan kembali ke formulir login beserta pesan konfirmasi, dan password baru dapat langsung digunakan.

Demi alasan keamanan, panel selalu menampilkan pesan *"If that address is registered, a reset link has been
sent"* — terlepas dari apakah akun tersebut terdaftar atau tidak. Mereset password akan mengeluarkan sesi aktif
dari seluruh perangkat.

### Tata Letak Tampilan

- **Sidebar (kiri):** Menu navigasi utama. Klik logo di bagian atas untuk kembali ke halaman Dashboard.
  Item menu dikelompokkan sebagai berikut:
  - **Dashboard** (atas)
  - **Radio:** Programs, Broadcasters, About Us
  - **Content:** Hero, Hot Release, Podcast, Event, Newsfeed, Media, Ads
- **Akun (bawah sidebar):** Menampilkan nama akun (klik untuk membuka **Profile**) serta tombol **Log Out**.
- Pada perangkat *mobile*, sidebar tersembunyi di balik tombol hamburger (menu).

### Log Out (Keluar)

Klik **Log Out** pada bagian bawah sidebar. Sesi login akan otomatis berakhir (*logout*) setelah 7 hari.

---

## 2. Perilaku dan Konvensi Umum

Setiap bagian di admin panel memiliki perilaku dan pola interaksi yang konsisten.

- **Search:** Sebagian besar halaman daftar (*list*) dilengkapi kotak pencarian. Ketik kata kunci lalu
  klik **Search**; klik **Reset** untuk mengosongkan pencarian. Jika tidak ada data yang cocok, pesan
  `No results for "…"` akan ditampilkan.
- **Sorting:** Judul kolom (*header*) tabel yang berupa tautan dapat diklik untuk mengurutkan data. Ikon panah
  menunjukkan kolom yang sedang aktif diurutkan — ▲ menaik (*ascending*), ▼ menurun (*descending*). Klik kembali untuk membalik urutan.
- **Pagination:** Halaman daftar menampilkan 50 baris data per halaman. Jika data melebihi satu halaman,
  penomoran halaman **Page X of Y** beserta tombol **Previous** / **Next** akan muncul.
- **Saving:** Setelah proses menyimpan (*save*) atau menghapus (*delete*) berhasil, halaman akan mengarahkan kembali ke daftar data beserta pesan
  konfirmasi berwarna hijau pada bagian atas (misalnya *"Program saved."*).
- **Errors:** Jika isian formulir tidak valid, formulir akan dimuat ulang dengan pesan kesalahan berwarna merah yang
  menjelaskan bagian yang perlu diperbaiki. Isian yang sudah dimasukkan sebelumnya tetap dipertahankan.
- **Deleting:** Tombol hapus (*delete*) akan menampilkan dialog konfirmasi *pop-up* pada browser sebelum data dihapus.
- **Unsaved changes:** Jika formulir edit ditinggalkan dalam keadaan memiliki perubahan yang belum
  disimpan, browser akan menampilkan peringatan konfirmasi sebelum mengalihkan halaman.
- **Images:** Kolom isian gambar dilengkapi dengan pratinjau (*preview*) langsung. Ukuran gambar otomatis disesuaikan (*resize*) dan
  dikompresi di browser sebelum diunggah (pesan *"Compressing…"* mungkin muncul sejenak). Format gambar yang
  didukung adalah JPG, PNG, WEBP, dan GIF. Setiap kolom gambar mencantumkan rekomendasi dimensi gambar.
  **Pada formulir edit, biarkan kolom gambar kosong jika tidak ingin mengganti gambar yang ada saat ini.**

Di seluruh admin panel, status **Active / Published** ditandai dengan lencana (*badge*) hijau, sedangkan status
**Inactive / Draft / Hidden** ditandai dengan lencana abu-abu.

---

## 3. Dashboard

Dashboard merupakan halaman utama yang menyajikan ringkasan status stasiun radio secara *real-time*. Sebagian informasi pada halaman ini diperbarui secara otomatis.

- **Tombol tindakan cepat (bagian atas):** **New program**, **New hot release**, dan **Refresh feeds**
  (memperbarui berita agregasi secara langsung).
- **Kartu On-air:** Menampilkan hari, tanggal, serta jam operasional stasiun, status *stream*, acara yang sedang mengudara (*on-air*) beserta *progress bar*, dan acara yang akan tayang berikutnya. Jika dua atau lebih program memiliki slot jadwal yang bentrok, kartu ini menampilkan acara dengan slot waktu terpendek (lihat *Bentrokan/Konflik jadwal* pada bagian Programs). Panel
  **Now playing** menampilkan lagu yang sedang diputar, jumlah pendengar saat ini beserta puncaknya hari ini, serta tautan **Open the live page**.
- **Kartu statistik (*Stat tiles*):** Kartu ringkasan angka yang dapat diklik untuk menuju ke halaman Programs, Broadcasters, Hot Release, dan Newsfeed.
- **Daftar penanganan (*Needs attention*):** Daftar peringatan yang menandai hal-hal yang memerlukan tindakan — sumber *feed* yang gagal atau dinonaktifkan, berita yang belum dipublikasikan, program aktif tanpa jadwal tayang atau penyiar, slot jadwal yang bentrok (dua acara terjadwal pada waktu yang sama), slot iklan aktif tanpa banner, dan tautan media sosial yang masih kosong. Setiap item menyediakan tautan langsung ke halaman perbaikan. Jika seluruh konfigurasi sudah benar, pesan *"All clear."* akan ditampilkan.
- **Grafik statistik:** Grafik riwayat jumlah pendengar (*listeners over time*) yang dapat disesuaikan (harian, per jam, atau interval 5 menit), grafik *"News arriving"* berisi jumlah item berita masuk per sumber setiap harinya, serta bagan jadwal siaran pekan ini (**This week on air**) di mana garis merah menandai waktu siaran yang sedang berjalan saat ini.
- **Sumber feed (*Feed sources*):** Ringkasan status kesehatan masing-masing sumber berita (Healthy / Failing / Disabled), dilengkapi informasi waktu pengambilan terakhir (*fetch*) dan jumlah item.

---

## 4. Programs

Menu Programs berisi daftar acara siaran yang dijadwalkan. **Sidebar → Radio → Programs.**

### Membuat atau mengedit program

1. Klik **Add Program** (atau ikon pensil **Edit** pada baris data program).
2. Isi kolom formulir yang tersedia (lihat rincian di bawah), lalu klik **Save**.

Pada program yang sudah tersimpan, tombol **View public page** dapat diklik untuk membuka halaman publik program tersebut.

**Kolom Isian**

| Kolom Isian | Catatan |
|---|---|
| Title | Wajib. Judul program siaran. Slug URL (`/program/…`) dibuat otomatis dari judul dan hanya berubah bila judul diubah. |
| Description | Teks bebas. |
| Default broadcasters | Pilihan ganda (*multi-select*) berupa daftar kotak centang yang dapat dicari. Digunakan untuk setiap slot jadwal siaran yang tidak menentukan penyiarnya sendiri. |
| Program Image | Disarankan 1600×900 (16:9). |
| Order | Angka lebih kecil tampil lebih dulu. |
| Active | Hilangkan centang untuk menyembunyikan program dari situs publik. |

### Jadwal mingguan (*Weekly schedule*)

Pada formulir edit tersedia editor **Weekly Schedule**:

1. Klik **Add slot** untuk menambahkan slot siaran baru.
2. Atur hari (**Day**), serta jam mulai (**Start**) dan selesai (**End**) dengan format HH:MM.
3. Secara opsional, tentukan penyiar (**Broadcasters**) untuk slot tersebut. Biarkan opsi **Program default** terpilih jika ingin menggunakan penyiar bawaan program.
4. Hapus slot menggunakan tombol ikon tempat sampah. Klik **Save** untuk menyimpan seluruh perubahan jadwal.

Setiap slot memerlukan hari yang valid serta jam mulai dan jam selesai yang berbeda.

### Bentrokan/Konflik jadwal

Sistem memantau slot jadwal yang bertumpang tindih (bentrok) dan menampilkannya agar pengguna tidak
secara tidak sengaja menjadwalkan dua acara sekaligus:

- **Pada daftar Programs**, sebuah banner peringatan muncul di bagian atas setiap kali ada
  slot yang bentrok di mana pun pada seluruh jadwal stasiun, dengan merinci
  masing-masing bentrokan — misalnya *"Monday: 'Show A' (08:00–10:00) overlaps 'Show B'
  (09:00–11:00)."* Jika jumlahnya lebih dari sepuluh, sisanya diringkas menjadi satu baris
  *"+N more"*.
- **Pada formulir edit sebuah program**, peringatan serupa hanya menampilkan bentrokan yang
  melibatkan program yang sedang disunting — termasuk jika ada dua slot jadwal pada program itu sendiri
  yang saling bentrok.

Ini adalah **peringatan, bukan penghalang** — perubahan tetap dapat disimpan. Tujuannya adalah membantu
menemukan kesalahan, sehingga peringatan tersebut dapat ditinjau dan disesuaikan waktunya jika
bentrokan itu memang tidak disengaja.

### Program yang tayang saat jadwal bentrok

Jika dua program terjadwal pada waktu yang sama, kartu **on-air** publik — pada halaman
**live** (`/live`), **beranda**, dan **Dashboard** admin — menampilkan program dengan
durasi (*timespan*) **terpendek**. Acara yang lebih sempit dan spesifik lebih diutamakan
daripada blok yang lebih lebar (misalnya program 07:00–10:00 ditampilkan alih-alih blok
sepanjang hari). Jika dua slot yang bentrok memiliki durasi yang sama, program yang mulai
lebih awal yang ditampilkan.

### Menghapus program

Klik tombol **Delete** (ikon tempat sampah), lalu konfirmasi *"Delete this program along with its schedule?"*. Tindakan ini juga akan menghapus seluruh slot jadwal siaran yang terikat pada program tersebut.

---

## 5. Broadcasters

Menu ini mengelola data penyiar dan *host* siaran. **Sidebar → Radio → Broadcasters.** Proses penambahan, pengubahan, dan penghapusan data berfungsi sama seperti pada menu Programs (**Add Broadcaster**, ikon pensil Edit, serta konfirmasi hapus *"Delete this broadcaster?"*).

**Kolom Isian**

| Kolom Isian | Catatan |
|---|---|
| Name | Wajib. Nama penyiar. Slug URL (`/broadcasters/…`) dibuat otomatis dari nama dan hanya berubah bila nama diubah. |
| Role | mis. "Announcer". |
| Bio | Teks bebas. |
| Birthplace / Date of Birth | Opsional. Tempat & tanggal lahir. |
| Instagram / X / Facebook | Tautan sosial opsional. |
| Photo | Disarankan 1000×1250 (potret 4:5). |
| Order | Angka lebih kecil tampil lebih dulu. |
| Active | Hilangkan centang untuk menyembunyikan dari situs publik. |

Formulir edit menampilkan daftar **Appears On** yang berisi program-program di mana penyiar ini ditugaskan. Daftar ini bersifat *read-only* (hanya-baca) — penugasan penyiar diubah melalui menu **Programs** (pada kolom *Default broadcasters* atau slot jadwal siaran).

---

## 6. Hero (slider halaman utama)

Hero banner merupakan slider gambar dan berita terbaru yang ditampilkan di bagian atas halaman utama (*home page*). **Sidebar → Content → Hero.**

### Mengelola slide

Proses tambah, edit, dan hapus slide dilakukan dengan cara yang sama seperti menu lainnya (**Add Slide**, ikon pensil Edit, serta konfirmasi *"Delete this hero slide?"*).

**Kolom isian slide**

| Kolom Isian | Catatan |
|---|---|
| Badge | Label kecil yang tampil pada slide. |
| Title | Judul slide. |
| Excerpt | Teks pendukung singkat. |
| Image | Wajib. Disarankan 2400×1200 (full-bleed). |
| Link | Opsional. URL `http(s)` lengkap atau path `/relative`. |
| Open the link in a new tab | Opsional. Centang untuk membuka tautan di tab baru. |
| Order | Angka lebih kecil tampil lebih dulu. |
| Active | Hilangkan centang untuk menyembunyikan slide. |

### Pengaturan Hero

Di bagian bawah daftar, buka panel **Hero settings** untuk mengatur kombinasi tampilan antara slide gambar dan berita:

- **Slide order:** Urutan alur slider (Newest first / Random / News first / Image slides first).
- **News items** (0–10) dan **Image slides** (0–10): Jumlah item berita dan slide gambar yang ditampilkan. Atur salah satu angka ke 0 jika hanya ingin menampilkan tipe konten lainnya. Klik **Save hero settings** untuk menyimpan pengaturan.

---

## 7. Hot Release (artikel berita)

Hot Release merupakan artikel berita editorial resmi yang dibuat oleh stasiun radio. **Sidebar → Content → Hot Release.**

### Membuat atau mengedit artikel

1. Klik **Add Hot Release** (atau Edit).
2. Isi kolom formulir yang tersedia, lalu klik **Save**.

**Kolom Isian**

| Kolom Isian | Catatan |
|---|---|
| Title | Wajib. Judul artikel. Slug URL (`/news/…`) dibuat otomatis dari judul dan hanya berubah bila judul diubah. |
| Excerpt | Ringkasan singkat. |
| Content | Teks biasa — tag HTML apa pun otomatis dibuang saat disimpan. |
| Hot Release Image | Gambar sampul. Disarankan 1600×900. |
| Middle Images | Galeri di dalam artikel (lihat di bawah). |
| Publish Date | Wajib (tanggal + waktu). |
| Publish | Centang untuk memublikasikannya. |
| Featured | Centang untuk menjadikannya kartu utama berukuran besar pada bagian Hot Release di halaman beranda dan halaman `/news`. Hanya satu Hot Release yang dapat disorot pada satu waktu — menyorot yang ini akan membatalkan sorotan pada yang lain. |

**Galeri Middle Images:** Tambahkan gambar pendukung agar tampil di bagian tengah artikel. Urutan gambar dapat disesuaikan dengan menyeret pegangan gambar (*drag handle*) atau menggunakan panah **Move up / Move down**. Centang opsi **Remove** untuk menghapus gambar tertentu, dan gunakan pemilih berkas untuk menambahkan beberapa gambar baru sekaligus.

**Bintang penyorot (Featured) pada daftar:** Setiap baris dilengkapi ikon bintang. Klik ikon bintang untuk menyorot (*featured*) atau membatalkan sorotan artikel secara langsung tanpa perlu membuka formulir edit. Konfirmasi hapus dilakukan dengan *"Delete this Hot Release?"*.

---

## 8. Podcasts

Menu ini mengelola episode podcast yang dihubungkan dari Spotify. **Sidebar → Content → Podcast.**

### Menambahkan atau mengedit podcast

1. Klik **Add Podcast** (atau Edit).
2. Isi kolom formulir yang tersedia, lalu klik **Save**.

**Kolom Isian**

| Kolom Isian | Catatan |
|---|---|
| Series | Wajib. Pilih dari dropdown (lihat Podcast Series di bawah). |
| Spotify URL | Wajib. Tautan `open.spotify.com` yang valid. Judul, gambar *thumbnail*, pemutar audio (*player*), dan deskripsi akan ditarik secara otomatis dari tautan ini saat disimpan. |
| Title | Hanya-baca (*read-only*). Ditarik otomatis dari tautan Spotify saat disimpan (dan diperbarui setiap kali Spotify URL diubah atau saat menggunakan **Refresh from Spotify**) — tidak ada kolom untuk diisi manual. Slug URL dibuat otomatis darinya. |
| Description | Hanya-baca (*read-only*). Ditarik otomatis dari tautan Spotify saat disimpan (dan diperbarui setiap kali Spotify URL diubah atau saat menggunakan **Refresh from Spotify**) — tidak ada kolom untuk diisi manual. |
| Broadcasters | Multi-select opsional. |
| Publish | Centang untuk memublikasikannya. |

**Title**, **Thumbnail**, dan **Description** semuanya diambil dari Spotify secara otomatis saat menyimpan, jadi tidak perlu diunggah atau diketik.

**Refresh from Spotify:** Pada formulir edit tersedia tombol **Refresh from Spotify**. Klik tombol ini untuk menarik ulang judul, *thumbnail*, dan deskripsi dari tautan Spotify saat itu juga — berguna ketika detail episode berubah di Spotify sementara URL-nya tetap sama (detail tersebut juga diperbarui otomatis setiap kali Spotify URL diubah).

### Seri Podcast (*Podcast Series*)

Series digunakan untuk mengelompokkan podcast serta mengontrol tombol filter (*chip filter*) pada halaman publik podcast. Aksesnya dapat dilakukan melalui tombol **Series** pada daftar Podcast.

- **Add Series** / Edit / Delete.
- **Kolom Isian:** Name (wajib; slug dibuat otomatis), Sort order, dan **Active** (centang = tampil di situs publik).
- Series yang nonaktif ditandai lencana (*badge*) **Inactive** abu-abu pada daftar dan disembunyikan dari tombol filter (*chip*) podcast publik — podcast yang sudah ada di dalamnya tetap tampil pada daftar lengkap.
- Series yang masih memiliki podcast terikat tidak dapat dihapus. Pindahkan atau ubah series podcast tersebut ke kategori lain terlebih dahulu sebelum menghapus series.

---

## 9. Event

Acara (*event*) dan promo yang ditampilkan di situs publik. **Sidebar → Content → Event.**

Menu **Event** pada situs publik baru muncul setelah minimal ada satu event yang dipublikasikan — sebelum itu bagian ini tidak terlihat oleh pengunjung, sehingga item dapat disiapkan sebagai draf terlebih dahulu.

### Membuat atau mengedit event

1. Klik **Add Event** (atau ikon pensil **Edit** pada suatu baris).
2. Isi kolom formulir yang tersedia lalu klik **Save**.

Pada event yang sudah tersimpan, tersedia tombol **View public page** untuk membuka halaman publiknya.

**Kolom Isian**

| Kolom Isian | Keterangan |
|---|---|
| Title | Wajib. Slug URL dibuat otomatis dari judul dan tetap stabil pada pengeditan berikutnya (hanya berubah bila judul diubah). |
| Category | **Event** atau **Promo**. |
| Description | Wajib. Teks biasa — seluruh tag HTML dihapus otomatis saat menyimpan. Pisahkan paragraf dengan satu baris kosong. |
| Event date & time | Opsional — kosongkan untuk promo berkelanjutan tanpa tanggal tetap. |
| Location | Opsional (mis. "Plaza Andalas, Padang"). |
| Call-to-action link | Opsional. URL `http(s)` lengkap — memunculkan tombol "Learn more" pada halaman detail (formulir pendaftaran, situs mitra, WhatsApp, …). |
| Banner | Gambar. Orientasi lanskap (16:10) paling ideal. Pada form edit, kosongkan untuk mempertahankan banner saat ini. |
| Publish | Centang agar event tampil publik. |

### Menghapus event

Klik tombol **Delete** (tempat sampah) dan konfirmasi *"Delete this event?"*.

---

## 10. Newsfeed (agregasi berita)

Newsfeed menampilkan artikel berita yang diambil secara otomatis dari sumber luar seperti **YouTube**, **KlikPositif**, dan **KataSumbar**. Konten berita ini tidak ditulis di admin panel — panel ini hanya mengatur visibilitas konten pada situs publik. **Sidebar → Content → Newsfeed.**

- **Filter tombol (pil/chip) pada bagian atas:** Memilah berita berdasarkan sumber (All / YouTube / KlikPositif / KataSumbar).
- **Publish / Hide** (ikon mata): Mengatur apakah suatu berita ditampilkan atau disembunyikan dari situs publik.
- **Feature** (ikon bintang): Menjadikan berita sebagai kartu utama berukuran besar di bagian atas baris sumbernya pada halaman beranda dan halaman `/news`. Hanya satu berita per sumber yang dapat disorot pada satu waktu — menyorot satu berita akan membatalkan sorotan pada berita lain di sumber yang sama.
- **Refresh Feed:** Mengambil berita terbaru dari sumber luar secara langsung.
- Klik judul berita untuk membuka artikel sumber aslinya di tab baru.

### Sumber Berita (*Feed Sources*)

Menu ini diakses melalui tombol **Feed Sources** pada halaman Newsfeed. Halaman ini menyajikan status kesehatan dari masing-masing sumber berita (status terakhir, waktu pengambilan data terakhir, serta jumlah item). Pada formulir ini terdapat opsi untuk:

- Menandai atau menghapus centang **Active** untuk mengaktifkan atau menonaktifkan sumber berita.
- Mengatur penyesuaian **Endpoint** (channel ID YouTube atau URL RSS); biarkan kosong jika ingin menggunakan nilai bawaan (*default*).
- Klik **Save** untuk menyimpan pengaturan, atau **Refresh Now** untuk langsung menarik berita terbaru.

Gangguan pada salah satu sumber tidak akan memengaruhi pembaruan dari sumber berita lainnya.

---

## 11. Ads

Menu ini mengelola spanduk iklan (*banner ads*) yang tampil di dua lokasi tetap: slot atas (**top**) dan slot bawah (**bottom**). **Sidebar → Content → Ads.**

### Pengaturan Slot

Masing-masing slot dilengkapi dengan panel pengaturan **Ads settings**:

- **Display:** Menentukan mode tampilan, baik *Stacked* (menampilkan seluruh banner secara berurutan) atau *Slideshow* (menampilkan banner secara bergantian).
- **Rotate every (sec):** Durasi pergantian banner (2–60 detik) untuk mode *slideshow*.
- **Placeholder text** dan **Show placeholder when empty:** Pengaturan teks dan opsi tampilan saat slot iklan tidak memiliki banner aktif.
- **Slot enabled:** Mengaktifkan atau menonaktifkan seluruh slot iklan.

Klik **Save ads settings** untuk menyimpan perubahan konfigurasi iklan.

### Mengelola Banner

1. Klik **Add Banner** (atau tautan "Add banner here" di dalam sebuah slot).
2. Isi kolom formulir yang tersedia, lalu klik **Save**.

**Kolom Isian**

| Kolom Isian | Catatan |
|---|---|
| Title | Label internal. |
| Alt text | Deskripsi untuk aksesibilitas. |
| Link | URL opsional — terbuka di tab baru. |
| Image | Wajib. Disarankan 1200×150 (leaderboard); di-letterbox, bukan di-crop. |
| Targeting | Centang **All pages**, atau hilangkan centangnya lalu pilih halaman tertentu. |
| Placement | Slot mana (top / bottom). |
| Order | Angka lebih kecil tampil lebih dulu. |
| Active | Hilangkan centang untuk menyembunyikan banner. |

Tombol **Delete** akan menampilkan konfirmasi *"Delete this banner?"*.

---

## 12. Tautan media sosial (*Media links*)

Pengaturan URL akun media sosial resmi stasiun radio yang ditampilkan sebagai ikon tautan pada bagian *header* dan *footer* situs. **Sidebar → Content → Media.**

Tersedia kolom isian untuk platform **Instagram, Facebook, X, YouTube, Spotify**, dan **TikTok**. Masukkan URL `http(s)` secara lengkap, atau **kosongkan kolom jika ingin menyembunyikan ikon platform tersebut**. Klik **Save** untuk menyimpan. Setiap baris dilengkapi dengan status koneksi (Connected / Not set) serta tombol uji tautan untuk memastikan URL dapat dibuka dengan benar.

---

## 13. About Us

Menu untuk mengelola konten halaman About Us pada situs publik. **Sidebar → Radio → About Us.** Pengaturan dilakukan dalam satu formulir:

- **Banner:** Pilih jenis spanduk utama berupa gambar (**Image**) melalui unggahan file atau video (**Video URL**) menggunakan tautan YouTube. Mengubah pilihan jenis media tidak akan menghapus data pada pilihan lainnya.
- **Text segments:** Terdiri dari tiga bagian informasi (profile, music, audience), di mana setiap bagian memiliki kolom **Title** (Judul) dan **Body** (Isi teks). Gunakan baris kosong untuk memisahkan antarparagraf pada bagian isi teks.

Klik **Save** untuk menyimpan.

---

## 14. Profil pengguna (*Your profile*)

Menu pengelolaan akun pribadi. **Klik nama akun pada bagian bawah sidebar** (atau akses `/admin/profile`). Halaman ini terbagi menjadi dua formulir terpisah:

- **Account Details:** Memperbarui **Name** (nama) dan **Email** (role akun ditampilkan tetapi tidak dapat diubah di sini). Klik **Save** untuk menyimpan perubahan.
- **Change Password:** Masukkan password baru (minimal 8 karakter) pada kolom **New Password** dan konfirmasi, lalu klik **Change Password**.
