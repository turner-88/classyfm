# ClassyFM Superadmin Guide

Panduan ini membahas fitur-fitur yang hanya tersedia untuk akun dengan role **Superadmin**.

Superadmin bisa melakukan semua yang bisa dilakukan content editor — tugas-tugas tersebut
(programs, broadcasters, news, podcasts, ads, chat, dan seterusnya) didokumentasikan di
[Admin Guide](ADMIN_GUIDE-ID.md). Selain itu, superadmin melihat grup **System** di bagian
bawah sidebar dengan dua bagian tambahan: **Users** dan **Activity Log**. Keduanya
dijelaskan di bawah ini.

> **Sekilas tentang role:** ada dua role — **Admin** (content editor) dan **Superadmin**.
> Hanya superadmin yang bisa mengelola akun pengguna dan melihat Activity Log.

## Table of Contents

- [Users](#1-users)
- [Activity Log](#2-activity-log)
- [Dashboard "Recent activity" panel](#3-dashboard-recent-activity-panel)

---

## 1. Users

Mengelola siapa saja yang bisa login ke admin panel. **Sidebar → System → Users.**

Daftar ini bisa dicari dan diurutkan (berdasarkan name dan email) serta menampilkan **Role**
tiap pengguna (badge "Superadmin" berwarna kuning atau "Admin" berwarna abu-abu) dan
**Status** (Active / Inactive). Baris akun sendiri ditandai **Your Account** dan tidak punya
tombol aksi — kelola detail akun sendiri dari
[Profile](ADMIN_GUIDE-ID.md#14-your-profile).

### Create a user

1. Klik **Add User**.
2. Isi field-nya lalu klik **Save**.

**Fields**

| Field | Catatan |
|---|---|
| Name | Wajib. |
| Email | Wajib dan unik. |
| Password | Wajib untuk pengguna baru; minimal 8 karakter. |
| Role | **Admin** atau **Superadmin**. Superadmin bisa mengelola Users dan melihat Activity Log. |
| Active | Hilangkan centang untuk menonaktifkan akun tanpa menghapusnya. |

### Edit a user

Buka pengguna dengan tombol pensil **Edit**, ubah yang perlu, lalu klik **Save**.

- **Password:** biarkan field "New password" **kosong untuk mempertahankan** password saat
  ini. Mengisi password baru akan me-reset-nya **dan mengeluarkan pengguna tersebut dari
  semua session-nya**.
- **Menonaktifkan** pengguna (menghilangkan centang Active) juga mengeluarkannya dari semua
  perangkat dan mencegahnya login.

### Delete a user

Klik **Delete** lalu konfirmasi *"Delete this user?"*. Akun sendiri tidak bisa dihapus dari
sini.

---

## 2. Activity Log

Riwayat perubahan yang dilakukan di admin panel yang bersifat read-only dan hanya bertambah
(append-only) — jejak akuntabilitas siapa melakukan apa. **Sidebar → System → Activity Log.**

Tabel ini bisa dicari (berdasarkan action, entity, detail, atau user) dan diurutkan. Tiap
baris menampilkan:

| Kolom | Arti |
|---|---|
| Time | Kapan aksi terjadi. |
| User | Siapa yang melakukannya (email-nya saat itu). |
| Action | Jenis perubahan — create, update, delete, login, logout, ban / unban, hide / un-hide, password reset, password change, feed refresh, dan seterusnya. |
| Entity | Apa yang terpengaruh, beserta tipe dan `#id`-nya. |
| Detail | Deskripsi singkat yang mudah dibaca. |

Log ini mencatat alamat IP di tiap entri. Ia tidak bisa diedit atau dihapus — ini adalah
catatan permanen.

---

## 3. Dashboard "Recent activity" panel

Di Dashboard, superadmin melihat panel tambahan **Recent activity** (content editor tidak).
Panel ini menampilkan entri Activity Log terbaru dengan label "time ago" dan tautan ke
**Activity log** lengkap — cara cepat untuk melihat sekilas perubahan terkini.
