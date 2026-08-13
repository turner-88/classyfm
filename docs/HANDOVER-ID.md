# Berita Acara Serah Terima — Website ClassyFM

Dokumen serah terima pekerjaan pengembangan website dan panel admin **Classy 103.4 FM
Padang**, sesuai lingkup yang disepakati pada *Surat Penawaran & Proposal Pengembangan
Website* (`Penawaran_ClassyFM.md`), bagian 2.8 (Hasil yang Diserahkan).

| | |
|---|---|
| **Proyek** | Website & Panel Admin Classy 103.4 FM |
| **Dari** | habiburrahman (Web Developer) |
| **Kepada** | Manajemen Classy 103.4 FM Padang |
| **Tanggal serah terima** | _______________________ |
| **Referensi** | Surat Penawaran & Proposal Pengembangan (disetujui) |

---

## 1. Ringkasan Hasil yang Diserahkan

Sesuai proposal bagian 2.8, tiga hasil pekerjaan berikut diserahkan:

**1. Source code lengkap.** Seluruh kode sumber website dan panel admin — server Go
dengan *rendering* HTML di sisi server, panel admin (CMS), serta seluruh template dan
aset statis yang disertakan langsung di dalam binary (*self-contained* via `go:embed`).
Penyimpanan data menggunakan MySQL.

- Lokasi repositori: _______________________ *(diisi: URL Git atau media penyerahan)*

**2. Website produksi yang aktif dan berjalan.** Aplikasi telah di-*deploy* dan berjalan
di server produksi.

- Situs publik: **https://classyfm.co.id**
- Panel admin: **https://classyfm.co.id/admin**

**3. Dokumen serah terima & panduan penggunaan.** Dokumen ini beserta kumpulan panduan
pendukung (tersedia dalam format Markdown dan PDF di folder `docs/`):

| Dokumen | Isi |
|---|---|
| **Panduan Dasar** (`QUICKSTART-ID`) | Ringkasan langkah operasional harian untuk memulai. |
| **Admin Guide** (`ADMIN_GUIDE-ID`) | Panduan lengkap pengelolaan konten untuk *content editor*. |
| **Superadmin Guide** (`SUPERADMIN_GUIDE-ID`) | Fitur khusus superadmin (Users, Legal, SEO, Activity Log). |
| **API Reference** (`API-ID`) | Dokumentasi API JSON publik untuk aplikasi *mobile*. |

---

## 2. Kredensial Akun Admin

Akun admin awal (role **Superadmin**) telah dibuat untuk pihak Classy 103.4 FM.

| Keterangan | Nilai |
|---|---|
| URL Panel | https://classyfm.co.id/admin/login |
| Email | _______________________ |
| Password | _______________________ |
| Role | Superadmin |

> **Penting — segera ganti password.** Setelah login pertama, ubah password melalui menu
> **Profile** (klik nama akun di bagian bawah sidebar, lalu bagian **Change Password**;
> atau buka `/admin/profile`). Password minimal 8 karakter. Simpan kredensial di tempat
> yang aman dan jangan dibagikan.

Akun admin tambahan untuk anggota tim lain dapat dibuat sendiri melalui menu **Users**
(khusus superadmin) — lihat Superadmin Guide.

---

## 3. Catatan Operator (Akses Darurat)

Terdapat satu jalur login darurat (*break-glass*) bawaan sistem dengan email khusus
`root@local.system`, yang password-nya adalah nilai variabel `SESSION_SECRET` pada server.
Login ini tidak memiliki baris data di tabel pengguna dan tetap berfungsi meski basis data
sedang tidak tersedia. Jalur ini diperuntukkan bagi operator/administrator server untuk
keadaan darurat (mis. memulihkan akses), **bukan untuk penggunaan sehari-hari**, dan bukan
merupakan kredensial yang diserahkan pada bagian 2 di atas. Mengganti `SESSION_SECRET`
akan menonaktifkan token login darurat maupun sesi impersonasi yang aktif.

---

## 4. Ringkasan Infrastruktur & Deployment

- **Aplikasi:** satu binary mandiri (*self-contained*) — template dan aset statis
  disertakan di dalam binary, sehingga deploy sederhana dan minim dependensi.
- **Basis data:** MySQL.
- **Layanan server:** berjalan sebagai *service* `systemd` di server produksi.
- **Backup:** backup data otomatis terjadwal (*systemd timer*).
- **Jaringan:** situs berada di belakang Cloudflare; perubahan CSS/JS mungkin memerlukan
  *purge cache* agar langsung terlihat.
- **Upload gambar:** gambar program/penyiar/berita tersimpan di disk server dan disajikan
  di jalur `/uploads/`.

---

## 5. Cakupan & Batasan

**Termasuk dalam pekerjaan yang diserahkan:**

- Pengembangan situs publik dan panel admin sesuai lingkup proposal.
- Deployment ke server produksi.
- Pengujian fungsional sebelum serah terima.
- Dokumen serah terima ini beserta panduan dasar penggunaan dan kredensial akun admin.

**Tidak termasuk (tanggung jawab pihak Classy 103.4 FM, kecuali disepakati lain):**

- Biaya domain, hosting/server, dan sertifikat SSL.
- Pemeliharaan (*maintenance*) dan dukungan pasca serah terima — dapat diajukan sebagai
  penawaran terpisah bila diperlukan.
- Pengembangan fitur baru di luar lingkup yang telah disepakati.

---

## 6. Persetujuan Serah Terima

Dengan menandatangani dokumen ini, kedua belah pihak menyatakan bahwa hasil pekerjaan
telah diserahkan dan diterima sesuai lingkup yang disepakati.

**Diserahkan oleh (Pengembang):**

Nama: habiburrahman

Jabatan: Web Developer

Tanda tangan / Tanggal: _______________________

<br>

**Diterima oleh (Pihak Classy 103.4 FM):**

Nama: _______________________

Jabatan: _______________________

Tanda tangan / Tanggal: _______________________
