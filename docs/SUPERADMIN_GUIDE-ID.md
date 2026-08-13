# ClassyFM Superadmin Guide

Panduan ini membahas fitur-fitur yang hanya tersedia untuk akun dengan role **Superadmin**.

Superadmin dapat melakukan semua hal yang dapat dilakukan oleh *content editor* — tugas-tugas tersebut
(program, penyiar, berita, podcast, iklan, dan seterusnya) didokumentasikan di
[Admin Guide](ADMIN_GUIDE-ID.md). Selain itu, superadmin memiliki akses ke menu tambahan yang tidak
dimiliki pengguna lain: **Legal Pages** (di grup **Radio** pada sidebar), **Contact** dan **SEO**
(di grup **Content**), serta grup **System** berisi **Users** dan **Activity Log**. Semuanya
dijelaskan di bawah ini.

> **Sekilas tentang role:** Terdapat dua role — **Admin** (*content editor*) dan **Superadmin**.
> Hanya superadmin yang dapat mengedit halaman legal, detail kontak, dan pengaturan SEO, serta mengelola
> akun pengguna dan melihat Activity Log.

## Daftar Isi

- [Legal Pages](#1-legal-pages)
- [Contact](#2-contact)
- [SEO](#3-seo)
- [Users](#4-users)
- [Activity Log](#5-activity-log)
- [Panel "Recent activity" pada Dashboard](#6-panel-recent-activity-pada-dashboard)

---

## 1. Legal Pages

Mengelola halaman **Privacy Policy** (Kebijakan Privasi) dan **Terms & Conditions** (Syarat dan Ketentuan) yang ditampilkan di situs publik. **Sidebar → Radio → Legal Pages.**

Daftar ini memuat dua halaman tetap — Privacy Policy dan Terms & Conditions. Klik salah satu halaman untuk mengeditnya (tidak tersedia opsi untuk membuat atau menghapus halaman). Setiap formulir edit terdiri dari:

| Kolom Isian | Catatan |
|---|---|
| Title | Wajib. Judul utama pada banner navy halaman. |
| Intro | Opsional. Subjudul di bawah judul; biarkan kosong untuk menyembunyikannya. |
| Body | Isi halaman, ditulis menggunakan editor **Markdown**. |

Editor **Body** dilengkapi toolbar pemformatan. Mode **Markdown** di bagian bawah editor dapat digunakan untuk mengedit kode sumber mentahnya. Konten disimpan dalam format Markdown. Klik **Save** setelah selesai, atau **View public page** untuk membuka halaman publik pada tab baru.

Halaman-halaman ini juga digunakan oleh aplikasi mobile (melalui public API), sehingga pastikan isinya selalu diperbarui (mutakhir).

---

## 2. Contact

Mengatur detail kontak yang digunakan di seluruh situs publik — tombol mengambang **WhatsApp** dan blok **"Get in touch"** pada footer. **Sidebar → Content → Contact.**

Formulir ini terdiri dari satu halaman; masing-masing kolom bersifat opsional, dan mengosongkan salah satu kolom akan menyembunyikan elemen terkait pada situs publik.

| Kolom Isian | Catatan |
|---|---|
| WhatsApp number | Sertakan kode negara (mis. `+62 812 …`). Mengosongkan kolom ini akan menyembunyikan tombol mengambang WhatsApp dan baris WhatsApp pada footer. Jika diisi, harus memuat beberapa digit angka. |
| Prefilled message | Pesan awal yang otomatis terisi saat pengunjung menekan tombol "Message us". Jika dikosongkan, percakapan dimulai tanpa pesan awal. |
| Phone | Ditampilkan sebagai tautan nomor telepon yang dapat ditekan pada footer. Kosongkan untuk menyembunyikannya. |
| Email | Ditampilkan sebagai tautan mailto pada footer. Harus berupa alamat email yang valid (memuat `@`). Kosongkan untuk menyembunyikannya. |

Klik **Save** untuk menerapkan perubahan. Detail ini juga digunakan oleh aplikasi mobile melalui public API.

---

## 3. SEO

Pengaturan default SEO (mesin pencari) dan tampilan pratinjau saat dibagikan ke media sosial untuk seluruh situs. Pengaturan ini diterapkan pada setiap halaman publik yang tidak menyetel nilainya sendiri. **Sidebar → Content → SEO.**

Formulir ini terdiri dari satu halaman; masing-masing kolom bersifat opsional dan berfungsi sebagai nilai bawaan (*fallback*) — judul, deskripsi, atau gambar utama pada masing-masing halaman akan selalu diprioritaskan dibanding nilai default ini.

| Kolom Isian | Catatan |
|---|---|
| Keywords | Daftar kata kunci yang dipisahkan dengan koma, digunakan untuk `meta` keywords situs. |
| Default meta description | Deskripsi yang ditampilkan oleh mesin pencari dan pratinjau tautan untuk halaman yang tidak menyediakan deskripsinya sendiri. |
| Social-share image | Gambar pratinjau (Open Graph) yang tampil saat halaman tanpa gambar khusus dibagikan ke media sosial. Disarankan menggunakan gambar berformat lanskap. Kosongkan kolom ini saat menyimpan jika ingin mempertahankan gambar yang digunakan saat ini. |

Klik **Save** untuk menerapkan.

---

## 4. Users

Mengelola akun pengguna yang memiliki akses login ke admin panel. **Sidebar → System → Users.**

Daftar ini dapat dicari dan diurutkan (berdasarkan nama dan email) serta menampilkan **Role**
setiap pengguna (lencana "Superadmin" berwarna kuning atau "Admin" berwarna abu-abu) dan
**Status** (Active / Inactive). Baris untuk akun pengguna yang sedang login ditandai lencana **Your Account** dan tidak memiliki
tombol aksi. Detail akun pengguna tersebut dapat dikelola melalui menu
[Profile](ADMIN_GUIDE-ID.md#14-profil-pengguna-your-profile).

### Membuat pengguna

1. Klik **Add User**.
2. Isi kolom yang tersedia, lalu klik **Save**.

**Kolom Isian**

| Kolom Isian | Catatan |
|---|---|
| Name | Wajib. Nama lengkap pengguna. |
| Email | Wajib dan unik. Alamat email untuk login. |
| Password | Wajib untuk pengguna baru; minimal 8 karakter. |
| Role | **Admin** atau **Superadmin**. Superadmin dapat mengelola Users dan melihat Activity Log. |
| Active | Hapus centang untuk menonaktifkan akun tanpa menghapusnya. |

### Mengedit pengguna

Klik tombol pensil **Edit** pada baris pengguna yang ingin diubah, perbarui informasi yang diperlukan, lalu klik **Save**.

- **Password:** Biarkan kolom "New password" **kosong jika tidak ingin mengubah** password saat
  ini. Mengisi password baru akan mereset password **sekaligus mengeluarkan pengguna tersebut dari
  seluruh sesi aktifnya**.
- **Menonaktifkan** pengguna (menghapus centang pada Active) juga akan mengeluarkannya dari seluruh
  perangkat dan mencabut akses loginnya.

### Menghapus pengguna

Klik tombol **Delete**, lalu konfirmasi dengan *"Delete this user?"*. Akun sendiri tidak dapat dihapus
melalui menu ini.

---

## 5. Activity Log

Halaman ini menampilkan riwayat aktivitas dan perubahan yang dilakukan di admin panel. Catatan log ini bersifat
*read-only* dan *append-only* (tidak dapat diubah atau dihapus), sehingga berfungsi sebagai audit log dan jejak akuntabilitas aktivitas pengguna. **Sidebar → System → Activity Log.**

Tabel ini dapat dicari (berdasarkan action, entity, detail, atau user) serta diurutkan. Setiap
baris menampilkan:

| Kolom | Arti |
|---|---|
| Time | Waktu terjadinya aksi. |
| User | Pengguna yang melakukan aksi (alamat email saat itu). |
| Action | Jenis perubahan — create, update, delete, login, logout, ban / unban, hide / un-hide, password reset, password change, feed refresh, dan seterusnya. |
| Entity | Objek yang terpengaruh, beserta tipe dan `#id`-nya. |
| Detail | Deskripsi singkat aktivitas yang mudah dibaca. |

Log ini juga mencatat alamat IP pada setiap entri. Data log tidak dapat diedit maupun dihapus
karena bersifat permanen.

---

## 6. Panel "Recent activity" pada Dashboard

Pada halaman Dashboard, superadmin dapat melihat panel tambahan **Recent activity** (panel ini tidak muncul
untuk *content editor*). Panel ini menampilkan entri Activity Log terbaru dengan label waktu relatif (seperti *"5 minutes ago"*)
serta menyediakan tautan langsung ke halaman **Activity Log** lengkap untuk memantau perubahan terkini secara cepat.
