# ClassyFM Superadmin Guide

Panduan ini membahas fitur-fitur yang hanya tersedia untuk akun dengan role **Superadmin**.

Superadmin dapat melakukan semua hal yang dapat dilakukan oleh content editor — tugas-tugas tersebut
(programs, broadcasters, news, podcasts, ads, chat, dan seterusnya) didokumentasikan di
[Admin Guide](ADMIN_GUIDE-ID.md). Selain itu, superadmin memiliki akses ke grup **System** di bagian
bawah sidebar dengan dua bagian tambahan: **Users** dan **Activity Log**. Keduanya
dijelaskan di bawah ini.

> **Sekilas tentang role:** Terdapat dua role — **Admin** (content editor) dan **Superadmin**.
> Hanya superadmin yang dapat mengelola akun pengguna dan melihat Activity Log.

## Table of Contents

- [Users](#1-users)
- [Activity Log](#2-activity-log)
- [Dashboard "Recent activity" panel](#3-dashboard-recent-activity-panel)

---

## 1. Users

Mengelola siapa saja yang dapat login ke admin panel. **Sidebar → System → Users.**

Daftar ini dapat dicari dan diurutkan (berdasarkan name dan email) serta menampilkan **Role**
setiap pengguna (badge "Superadmin" berwarna kuning atau "Admin" berwarna abu-abu) dan
**Status** (Active / Inactive). Baris akun sendiri ditandai **Your Account** dan tidak memiliki
tombol aksi. Detail akun sendiri dapat dikelola melalui menu
[Profile](ADMIN_GUIDE-ID.md#14-your-profile).

### Create a user

1. Klik **Add User**.
2. Isi kolom yang tersedia, lalu klik **Save**.

**Fields**

| Field | Catatan |
|---|---|
| Name | Wajib. |
| Email | Wajib dan unik. |
| Password | Wajib untuk pengguna baru; minimal 8 karakter. |
| Role | **Admin** atau **Superadmin**. Superadmin dapat mengelola Users dan melihat Activity Log. |
| Active | Hilangkan centang untuk menonaktifkan akun tanpa menghapusnya. |

### Edit a user

Klik tombol pensil **Edit** pada baris pengguna yang ingin diubah, perbarui informasi yang diperlukan, lalu klik **Save**.

- **Password:** Biarkan kolom "New password" **kosong jika tidak ingin mengubah** password saat
  ini. Mengisi password baru akan mereset password **sekaligus mengeluarkan pengguna tersebut dari
  seluruh sesi aktifnya**.
- **Menonaktifkan** pengguna (menghapus centang pada Active) juga akan mengeluarkannya dari seluruh
  perangkat dan mencabut akses login.

### Delete a user

Klik tombol **Delete**, lalu konfirmasi dengan *"Delete this user?"*. Akun sendiri tidak dapat dihapus
melalui menu ini.

---

## 2. Activity Log

Halaman ini menampilkan riwayat perubahan yang dilakukan pada admin panel. Catatan log ini bersifat
*read-only* dan *append-only* (hanya dapat bertambah), sehingga berfungsi sebagai jejak akuntabilitas aktivitas pengguna. **Sidebar → System → Activity Log.**

Tabel ini dapat dicari (berdasarkan action, entity, detail, atau user) serta diurutkan. Setiap
baris menampilkan:

| Kolom | Arti |
|---|---|
| Time | Kapan aksi terjadi. |
| User | Siapa yang melakukannya (email-nya saat itu). |
| Action | Jenis perubahan — create, update, delete, login, logout, ban / unban, hide / un-hide, password reset, password change, feed refresh, dan seterusnya. |
| Entity | Apa yang terpengaruh, beserta tipe dan `#id`-nya. |
| Detail | Deskripsi singkat yang mudah dibaca. |

Log ini juga mencatat alamat IP pada setiap entri. Data log tidak dapat diedit maupun dihapus
karena bersifat permanen.

---

## 3. Dashboard "Recent activity" panel

Pada halaman Dashboard, superadmin dapat melihat panel tambahan **Recent activity** (panel ini tidak tampil
untuk content editor). Panel ini menampilkan entri Activity Log terbaru dengan label waktu relatif ("time ago")
serta tautan ke halaman **Activity log** lengkap untuk memantau perubahan terkini dengan cepat.

