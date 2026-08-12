# ClassyFM Public JSON API

API JSON publik untuk aplikasi *mobile* ClassyFM. Seluruh *endpoint* berada di bawah jalur **`/api/v1`**.

- **Base URL:** `https://classyfm.co.id/api/v1`
- **Format:** JSON (`Content-Type: application/json; charset=utf-8`)
- **Sifat:** Sepenuhnya *read-only* dan **tanpa *cookie*** — setiap *endpoint* merupakan `GET` publik, tanpa autentikasi dan tanpa *request body*.

---

## Table of Contents

- [Conventions](#conventions)
  - [Response envelopes](#response-envelopes)
  - [Errors](#errors)
  - [Caching](#caching)
  - [Pagination & filtering](#pagination--filtering)
  - [URLs & images](#urls--images)
  - [Degraded mode](#degraded-mode)
- [Index of Endpoints](#endpoints)
- [App bootstrap endpoints](#app-bootstrap-1)
- [Content endpoints](#content-endpoints)
- [Live endpoints](#live-endpoints)
  - [Playing the live stream in an app](#playing-the-live-stream-in-an-app)
  - [When the backend API is unreachable](#when-the-backend-api-is-unreachable)
- [Advertising endpoints](#advertising)
- [Legal pages](#legal-pages)
- [Object reference](#object-reference)

---

## Conventions

### Response envelopes

| Shape | Digunakan oleh | Body |
|-------|---------|------|
| **List** | endpoint roster/list | `{ "data": [ ... ] }` |
| **Paginated** | `GET /news?source=…`, <br>`GET /podcasts` | `{ "data": [ ... ], "meta": { "page": 1, "total_pages": 3, "total": 27 } }` |
| **Object** | endpoint detail & agregat | objek langsung (tanpa pembungkus) |

### Errors

Error mengembalikan `{ "error": "message" }` dengan status HTTP yang relevan:

| Status | Arti |
|--------|---------|
| `400` | Path id atau parameter query tidak valid |
| `404` | Resource tidak ditemukan, atau fitur belum dikonfigurasi |
| `500` | Error server / database |

### Caching

Setiap respons membawa header `Cache-Control`:

- `public, max-age=60` — Konten yang jarang berubah (programs, news, podcasts, about, ads, home, config).
- `no-store` — Data siaran langsung (*live*) (now-playing, status jadwal, TikTok live).

### Pagination & filtering

Pagination hanya berlaku pada `GET /news?source=…` dan `GET /podcasts`:

- `page` — Berbasis 1 dengan nilai bawaan (*default*) `1` (nilai `< 1` akan dianggap sebagai `1`).
- Jumlah item per halaman bersifat tetap, yaitu **12 item**.
- `meta.total_pages` bernilai minimal `1` meskipun tidak ada data.

### URLs & images

Seluruh kolom gambar dan tautan dikembalikan dalam bentuk **URL absolut** (diarahkan ke domain utama situs, `https://classyfm.co.id`). URL yang sudah berupa alamat absolut — seperti *thumbnail* berita agregasi — akan diteruskan tanpa perubahan.

### Degraded mode

Jika basis data (*database*) tidak dapat diakses, *endpoint* jenis **daftar (*list*)** akan menangani kondisi kesalahan secara halus (*degrade gracefully*) ke
`{ "data": [] }` dan *endpoint* jenis **detail** mengembalikan `404` — kedua jenis *endpoint* tersebut tidak pernah mengembalikan kode kesalahan `500` akibat masalah koneksi basis data.

---

## Endpoints

### App bootstrap endpoints

Dua *endpoint* agregasi yang dipanggil oleh aplikasi saat pertama kali dijalankan.

| Method | Endpoint | Auth | Deskripsi |
|--------|----------|------|-------------|
| `GET` | [`/api/v1/config`](#get-apiv1config) | open | Bootstrap aplikasi: identity, stream, social links |
| `GET` | [`/api/v1/home`](#get-apiv1home) | open | Feed layar home agregat dalam satu request |

### Content endpoints 

| Method | Endpoint | Auth | Deskripsi |
|--------|----------|------|-------------|
| `GET` | [`/api/v1/programs`](#get-apiv1programs) | open | Roster program aktif, masing-masing ditandai on-air |
| `GET` | [`/api/v1/programs/{slug}`](#get-apiv1programsslug) | open | Satu program dengan weekly schedule & broadcasters |
| `GET` | [`/api/v1/broadcasters`](#get-apiv1broadcasters) | open | Roster broadcaster aktif |
| `GET` | [`/api/v1/broadcasters/{slug}`](#get-apiv1broadcastersslug) | open | Satu broadcaster beserta program yang dibawakannya |
| `GET` | [`/api/v1/news`](#get-apiv1news) | open | Preview news yang dikelompokkan, atau satu source dengan pagination |
| `GET` | [`/api/v1/news/{slug}`](#get-apiv1newsslug) | open | Satu artikel `hot_release` dengan galeri & terkait |
| `GET` | [`/api/v1/podcasts`](#get-apiv1podcasts) | open | Podcast yang dipublikasikan, dengan pagination |
| `GET` | [`/api/v1/podcasts/{slug}`](#get-apiv1podcastsslug) | open | Satu podcast dengan series & broadcasters |
| `GET` | [`/api/v1/podcast-series`](#get-apiv1podcast-series) | open | Daftar podcast series yang aktif |
| `GET` | [`/api/v1/about`](#get-apiv1about) | open | Banner halaman about, segmen & preview broadcaster |

### Live endpoints

| Method | Endpoint | Auth | Deskripsi |
|--------|----------|------|-------------|
| `GET` | [`/api/v1/now-playing`](#get-apiv1now-playing) | open | Metadata now-playing stream & program on-air |
| `GET` | [`/api/v1/schedule/today`](#get-apiv1scheduletoday) | open | Schedule hari ini dengan status on-air/progress live |
| `GET` | [`/api/v1/schedule/current`](#get-apiv1schedulecurrent) | open | Program yang sedang on-air |
| `GET` | [`/api/v1/tiktok/live`](#get-apiv1tiktoklive) | open | Status TikTok live |

### Advertising endpoints

| Method | Endpoint | Auth | Deskripsi |
|--------|----------|------|-------------|
| `GET` | [`/api/v1/ads`](#get-apiv1ads) | open | Banner ads untuk suatu halaman, berdasarkan placement slot |

### Legal pages

| Method | Endpoint | Auth | Deskripsi |
|--------|----------|------|-------------|
| `GET` | [`/api/v1/legal`](#get-apiv1legal) | open | Daftar halaman legal (slug, title, updated) |
| `GET` | [`/api/v1/legal/{slug}`](#get-apiv1legalslug) | open | Satu halaman legal: body Markdown + HTML hasil render |

> **Catatan:** Fitur Connect chat tidak lagi dilayani oleh API ini — kini fitur tersebut
> berjalan langsung di atas Firebase Realtime Database (berbagi dengan aplikasi *mobile*),
> sehingga *endpoint* `/api/v1/connect/*` sudah tidak tersedia.

---

## App bootstrap endpoints

Kedua *endpoint* menggunakan metode `GET` dan di-*cache* dengan opsi `public, max-age=60`.

### `GET /api/v1/config`

Bootstrap aplikasi: identity stasiun radio, URL stream, tautan sosial, dan detail kontak.

```json
{
  "station": { "name": "Classy 103.4 FM", "slogan": "…" },
  "stream_url": "https://c4.siar.us:10340/stream.mp3",
  "social": { "instagram": "https://…", "youtube": "https://…" },
  "contact": {
    "whatsapp_number": "0812…",
    "whatsapp_message": "Halo Classy FM…",
    "whatsapp_url": "https://wa.me/62812…?text=Halo%20Classy%20FM",
    "phone": "0751…",
    "phone_href": "tel:+62751…",
    "email": "info@classyfm.co.id"
  }
}
```

Properti `social` memuat nama platform sesuai dengan konfigurasi pada admin panel.
Properti `contact` adalah blok kontak yang dikelola admin (lihat [`contact`](#contact));
nilainya `null` bila backend tidak memiliki database, dan tiap field-nya dihilangkan bila
kosong. `whatsapp_url` (deep link `wa.me` siap pakai) dan `phone_href` (`tel:`) dibangun
di sisi server — gunakan keduanya alih-alih menyusun sendiri. Blok `contact` **tidak**
dibatasi oleh feature flag apa pun: ia mengembalikan apa pun yang disimpan admin, dan
aplikasi yang menentukan cara menampilkannya (misalnya tombol WhatsApp).

### `GET /api/v1/home`

Feed layar home agregat dalam satu request.

```json
{
  "hero":     [ /* heroSlide objects */ ],
  "on_air":   { /* scheduleRow, or null */ },
  "programs": [ /* program objects, up to 6 */ ],
  "news":     [ /* newsGroup objects, up to 4 items each */ ]
}
```

Lihat [`heroSlide`](#heroslide).

---

## Content endpoints

Semua adalah `GET` dan di-cache `public, max-age=60`.

### `GET /api/v1/programs`

Daftar program siaran aktif, dilengkapi dengan penanda status siaran langsung (*on-air*).

```json
{ "data": [
  {
    "title": "Morning Show",
    "slug": "morning-show",
    "description": "…",
    "image_url": "https://classyfm.co.id/uploads/abc.jpg",
    "url": "https://classyfm.co.id/program/morning-show",
    "on_air": true
  }
] }
```

Lihat [`program`](#program) untuk field objeknya.

### `GET /api/v1/programs/{slug}`

Menampilkan detail satu program siaran beserta seluruh jadwal mingguan (*weekly schedule*) dan penyiarnya. Mengembalikan `404` jika *slug* tidak ditemukan atau program tidak aktif.

```json
{
  "title": "Morning Show",
  "slug": "morning-show",
  "description": "…",
  "image_url": "https://classyfm.co.id/uploads/abc.jpg",
  "url": "https://classyfm.co.id/program/morning-show",
  "on_air": true,
  "schedule": [
    { "from_day": 1, "to_day": 5, "start_time": "06:00", "end_time": "09:00", "host": "Anda, Yeni" }
  ],
  "broadcasters": [ /* broadcaster objects */ ]
}
```

`schedule[]` adalah [`scheduleGroup`](#schedulegroup); `broadcasters[]` adalah objek
[`broadcaster`](#broadcaster).

### `GET /api/v1/broadcasters`

Daftar penyiar aktif. Nilai `on_air` akan bernilai `true` apabila penyiar tersebut sedang membawakan program yang sedang mengudara.

```json
{ "data": [ /* broadcaster objects */ ] }
```

### `GET /api/v1/broadcasters/{slug}`

Menampilkan detail satu penyiar beserta daftar program yang dibawakannya. Mengembalikan `404` jika penyiar tidak ditemukan.

```json
{
  "name": "Anda",
  "slug": "anda",
  "role": "Announcer",
  "photo_url": "https://classyfm.co.id/uploads/anda.jpg",
  "bio": "…",
  "birth_place": "Padang",
  "birth_date": "…",
  "instagram": "…",
  "twitter": "…",
  "facebook": "…",
  "url": "https://classyfm.co.id/broadcasters/anda",
  "on_air": false,
  "programs": [ /* program objects */ ]
}
```

### `GET /api/v1/news`

Tersedia dalam dua mode penggunaan:

**Tanpa parameter `source`** (atau menggunakan nama sumber yang tidak valid) — Menampilkan pratinjau berita yang dikelompokkan per sumber (maksimal 6 item per kelompok sumber):

```json
{ "data": [
  { "source": "klikpositif", "label": "KlikPositif", "items": [ /* news items */ ] },
  { "source": "katasumbar",  "label": "KataSumbar",  "items": [ … ] },
  { "source": "hot_release", "label": "Hot Release", "items": [ … ] },
  { "source": "youtube",     "label": "YouTube",     "items": [ … ] }
] }
```

**Dengan parameter `source` yang valid** — Menampilkan daftar berita dari sumber tersebut lengkap dengan halaman (*pagination*):

| Query param | Catatan |
|-------------|-------|
| `source` | Salah satu dari `youtube`, `klikpositif`, `katasumbar`, `hot_release`. <br>Nilai tidak valid kembali ke mode grouped. |
| `page` | Berbasis 1, ukuran halaman 12. |

```json
{ "data": [ /* news items */ ], "meta": { "page": 1, "total_pages": 4, "total": 42 } }
```

Setiap item berbentuk objek [`newsItem`](#newsitem). Catatan: Berita tipe `hot_release` mengarahkan tautan ke
`/news/{slug}` pada situs, sedangkan berita dari sumber lain (*aggregated news*) mempertahankan URL eksternal aslinya.

### `GET /api/v1/news/{slug}`

Menampilkan detail artikel berita `hot_release`, lengkap dengan pemisah paragraf, galeri gambar di tengah artikel,
dan daftar artikel terkait. Mengembalikan `404` jika artikel tidak ditemukan (hanya artikel `hot_release` yang memiliki halaman detail di situs).

```json
{
  "source": "hot_release",
  "source_label": "Hot Release",
  "title": "…",
  "slug": "…",
  "excerpt": "…",
  "image_url": "https://…",
  "url": "https://classyfm.co.id/news/…",
  "published_at": "2026-08-09T10:00:00Z",
  "is_featured": true,
  "paragraphs": ["…", "…"],
  "gallery_index": 3,
  "middle_images": ["https://…", "https://…"],
  "related": [ /* news items */ ]
}
```

`gallery_index` menunjukkan posisi indeks paragraf tempat galeri gambar `middle_images` disisipkan.

### `GET /api/v1/podcasts`

Daftar podcast terpublikasi yang diurutkan dari yang terbaru, dilengkapi dengan halaman (*pagination*). Hanya episode dari serial yang **aktif** yang ditampilkan — menandai sebuah serial nonaktif di admin panel akan menghapus episodenya dari `data` maupun `meta.total`.

| Query param | Catatan |
|-------------|-------|
| `series` | Parameter penyaring *slug* serial podcast (opsional). Jika *slug* tidak dikenal/tidak valid, parameter akan diabaikan (menampilkan seluruh podcast); jika *slug* merujuk ke serial yang *nonaktif*, hasilnya kosong. |
| `page` | Berbasis 1, ukuran halaman 12. |

```json
{ "data": [ /* podcast objects */ ], "meta": { "page": 1, "total_pages": 2, "total": 18 } }
```

### `GET /api/v1/podcasts/{slug}`

Menampilkan detail satu podcast beserta nama serial dan penyiarnya. Mengembalikan `404` jika podcast tidak ditemukan — termasuk podcast terpublikasi yang serialnya telah ditandai nonaktif.

```json
{
  "title": "…",
  "slug": "…",
  "description": "…",
  "spotify_url": "https://open.spotify.com/…",
  "thumb_url": "https://…",
  "series_name": "…",
  "url": "https://classyfm.co.id/podcast/…",
  "broadcasters": [ /* broadcaster objects */ ]
}
```

### `GET /api/v1/podcast-series`

Daftar serial podcast yang **aktif** (digunakan sebagai opsi penyaring pada parameter `?series=`). Serial yang ditandai nonaktif di admin panel tidak disertakan.

```json
{ "data": [ { "name": "…", "slug": "…" } ] }
```

### `GET /api/v1/about`

Menampilkan konten halaman About Us: spanduk (*banner*), segmen teks informasi, dan pratinjau penyiar (maksimal 8 penyiar).

```json
{
  "banner": {
    "media_type": "video",
    "image_url": "https://…",
    "video_url": "https://youtu.be/…",
    "embed_url": "https://www.youtube.com/embed/…"
  },
  "segments": [ { "segment": "profile", "title": "…", "body": "…" } ],
  "broadcasters": [ /* broadcaster objects */ ]
}
```

`banner.embed_url` hanya tersedia jika `media_type` bernilai `"video"` dan URL video merupakan
tautan YouTube yang dapat diproses. Nilai `segment` berupa salah satu dari `profile`, `music`,
atau `audience`.

---

## Live endpoints

Semua adalah `GET`, di-cache `no-store` — status stream live dan progress schedule.

### `GET /api/v1/now-playing`

Metadata lagu yang sedang diputar (*now-playing*) pada *stream*, beserta informasi program siaran yang sedang mengudara saat *stream* aktif.

```json
{
  "artist": "…",
  "song": "…",
  "has_song": true,
  "cover_url": "https://…",
  "live": true,
  "program": { /* scheduleRow, present only when live and a program is on air */ }
}
```

Lihat detail objek [`scheduleRow`](#schedulerow). (Jumlah pendengar Shoutcast tidak ditampilkan pada *endpoint* publik ini dan hanya dapat diakses oleh admin).

### `GET /api/v1/schedule/today`

Jadwal acara siaran lengkap hari ini beserta status siaran langsung dan persentase durasi yang telah berjalan.

```json
{ "data": [ /* scheduleRow objects */ ] }
```

### `GET /api/v1/schedule/current`

Menampilkan informasi program yang sedang mengudara dalam bentuk objek [`scheduleRow`](#schedulerow), atau
mengembalikan `{ "on_air": false }` jika tidak ada program yang sedang tayang.

### `GET /api/v1/tiktok/live`

```json
{ "live": true, "title": "…" }
```

### Playing the live stream in an app

Aliran audio (*audio stream*) merupakan **MP3 Shoutcast eksternal** yang diakses secara langsung. *Stream* tersebut di-host pada server Shoutcast terpisah dan **bukan** pada `classyfm.co.id`, sehingga API ini tidak melakukan *proxy* atau pengalihan (*redirect*) audio. Aplikasi pemutar audio pada *mobile* dapat memutarnya **secara langsung**:

1. **Inisialisasi URL Stream:** Ambil nilai `stream_url` dari *endpoint* [`/api/v1/config`](#get-apiv1config) saat aplikasi pertama kali dijalankan, lalu berikan ke pemutar audio bawaan (*native player*) perangkat. *Stream* MP3 Shoutcast ini dapat diakses publik tanpa autentikasi, *header* khusus, maupun *proxy*. Sangat disarankan untuk mengambil URL dari konfigurasi API dibandingkan melakukan *hardcode*, agar pemindahan server Shoutcast di masa mendatang dapat dilakukan dari sisi server tanpa perlu memperbarui aplikasi.
2. **Pembaruan Metadata Now-Playing:** Lakukan pemanggilan berkala (*polling*) ke *endpoint* [`/api/v1/now-playing`](#get-apiv1now-playing) melalui API ini, dan **bukan** langsung ke server Shoutcast. Pemanggilan ini digunakan untuk memperbarui tampilan antarmuka *now-playing*, serta notifikasi dan *lock-screen metadata*. API server akan mengambil dan menyimpan sementara (*cache*) metadata lagu serta status *live* dari server Shoutcast. Hal ini menghindarkan aplikasi dari kendala CORS dan mencegah lonjakan beban pada server Shoutcast — **hindari mengambil data (*scraping*) langsung dari server Shoutcast**. Respons *endpoint* ini memiliki *header* `no-store`, namun server hanya memperbarui metadata dari sumber hulu (*upstream*) setiap ~12 detik. Oleh karena itu, **pemanggilan berulang yang lebih cepat dari ~15 detik tidak akan memberikan perubahan data** — gunakan interval pemanggilan sekitar 15 detik selama audio diputar, dan hentikan pemanggilan saat pemutaran dihentikan atau aplikasi berjalan di latar belakang tanpa audio.
   - Tampilkan informasi `artist` + `song` jika `has_song` bernilai `true`. Jika bernilai `false` (misalnya saat identitas stasiun diputar atau metadata kosong), gunakan nama stasiun radio sebagai alternatif (*fallback*).
   - Gunakan `cover_url` untuk gambar album/sampul (*artwork*). Namun, karena kolom ini bersifat *best-effort* dan dapat bernilai kosong (`""`), sediakan gambar bawaan (*placeholder*) atau gambar program siaran yang sedang berjalan sebagai alternatif.
3. **Penanganan Status Siaran (Live vs. Off-Air):** Sesuaikan tampilan antarmuka antara kondisi "on-air" dan "off-air" berdasarkan penanda (*flag*) `live`. Ketika `live` bernilai `true`, objek opsional `program` (berupa [`scheduleRow`](#schedulerow)) akan menyediakan informasi acara siaran, penyiar, serta persentase durasi berjalan (`progress` 0–100). Informasi yang sama juga dapat diperoleh melalui *endpoint* [`/api/v1/schedule/current`](#get-apiv1schedulecurrent), sedangkan *endpoint* [`/api/v1/schedule/today`](#get-apiv1scheduletoday) dapat digunakan untuk menampilkan daftar acara berikutnya (*up next*).

### When the backend API is unreachable

Karena aliran audio menggunakan **MP3 Shoutcast langsung** tanpa melalui *proxy* API, gangguan pada server *backend* (seperti kendala jaringan, *timeout*, atau kode status `5xx`) **tidak akan memutus pemutaran audio** — gangguan tersebut hanya berdampak pada metadata di sekitar pemutar audio. Biarkan pemutaran audio tetap berjalan dan cukup sesuaikan tampilan metadata *now-playing*:

1. **Simpan Konfigurasi secara Lokal (*Persist Config*):** Simpan respons dari *endpoint* [`/api/v1/config`](#get-apiv1config) yang berhasil diterima (minimal nilai `stream_url`, beserta nama/slogan stasiun dan tautan media sosial) ke dalam penyimpanan lokal perangkat. Saat aplikasi dijalankan, mulai pemutaran audio menggunakan `stream_url` yang tersimpan di memori lokal meskipun permintaan *config* terbaru gagal diambil. **Jangan melakukan *hardcode* pada URL stream** — aplikasi tidak boleh membuat perkiraan URL secara mandiri. Pada kondisi *cold start* pertama kali (aplikasi baru diinstal, belum ada data yang tersimpan, dan API tidak terjangkau), tampilan status "Stream tidak tersedia / Coba lagi" dapat dimunculkan, lalu lakukan pengambilan *config* kembali setelah koneksi pulih.
2. **Pertahankan Pemutaran Audio saat Metadata Gagal Ditarik:** Kegagalan pemanggilan *endpoint* [`/api/v1/now-playing`](#get-apiv1now-playing) akibat *timeout* atau respon `5xx` hanya memengaruhi pembaruan metadata dan bukan indikasi pemutusan siaran audio — jangan menghentikan atau mereset pemutar audio karena masalah tersebut. Pertahankan informasi `artist`/`song` serta status `live` terakhir yang berhasil diterima, atau tampilkan nama stasiun dan gambar *placeholder* bawaan sebagai alternatif.
3. **Penerapan *Exponential Backoff* dan Pemulihan Otomatis:** Jika terjadi kegagalan pemanggilan berulang kali, perpanjang interval pemanggilan (misalnya menggunakan metode *exponential backoff* hingga maksimal ~60 detik) agar tidak membebani jaringan. Ketika pemanggilan berikutnya berhasil, kembalikan interval ke rentang normal ~15 detik, perbarui tampilan antarmuka *now-playing*, dan perbarui simpanan *config* lokal. Seluruh proses ini berjalan otomatis tanpa memerlukan tindakan pengguna atau *restart* aplikasi.

Jumlah listener live sengaja tidak tersedia untuk aplikasi.

---

## Advertising endpoints

`GET`, di-cache `public, max-age=60`.

### `GET /api/v1/ads`

Banner ads untuk suatu halaman, dikelompokkan ke placement slot `top` dan `bottom`.

| Query param | Catatan |
|-------------|-------|
| `page` | Parameter halaman target. Nilai tidak valid mengembalikan `400`. Jika dihilangkan, mengembalikan hanya banner yang ditargetkan ke setiap halaman. |

Nilai `page` yang valid: `home`, `about`, `program`, `program_detail`, `live`, `news`,
`news_detail`, `broadcasters`, `broadcaster_detail`.

```json
{
  "top": {
    "slideshow": false,
    "rotate_ms": 6000,
    "placeholder": false,
    "placeholder_text": "",
    "banners": [
      {
        "image_url": "https://classyfm.co.id/uploads/ad.jpg",
        "link_url": "https://sponsor.example",
        "alt": "…",
        "title": "…"
      }
    ]
  },
  "bottom": { /* same shape */ }
}
```

`banners` pada setiap slot berbentuk larik (*array*) dan akan bernilai kosong jika slot tidak memiliki banner aktif. Nilai `slideshow` memberi petunjuk pada klien untuk memutar pergantian banner setiap `rotate_ms` milidetik alih-alih menampilkan seluruh banner secara berurutan (*stacked*). Nilai `placeholder` (beserta `placeholder_text`) menandakan bahwa slot kosong harus tetap mempertahankan ukurannya dan tidak menyusut.

---

## Legal pages

Halaman **Privacy Policy** (Kebijakan Privasi) dan **Terms & Conditions** (Syarat dan
Ketentuan) adalah konten yang dikelola admin dan disajikan sebagai JSON, sehingga aplikasi
dapat me-*render*-nya secara *native* (atau di WebView dari `body_html`) pada layar
Pengaturan/Legal. Keduanya `GET` dan di-*cache* `public, max-age=60`. Halaman disajikan
dalam bahasa Indonesia dan isinya dapat berubah sewaktu-waktu, jadi ambil ulang alih-alih
menyimpan salinan lokal.

### `GET /api/v1/legal`

Daftar halaman legal, cukup untuk membangun menu tanpa mengirim seluruh body.

```json
{ "data": [
  { "slug": "privacy", "title": "Kebijakan Privasi", "updated_at": "2026-08-10T09:00:00Z" },
  { "slug": "terms",   "title": "Syarat dan Ketentuan", "updated_at": "2026-08-10T09:00:00Z" }
] }
```

### `GET /api/v1/legal/{slug}`

Satu halaman legal secara lengkap. `{slug}` adalah salah satu dari `privacy`, `terms`;
nilai lain (atau halaman yang tidak ada) mengembalikan `404`.

```json
{
  "slug": "privacy",
  "title": "Kebijakan Privasi",
  "intro": "…",
  "body": "## Sumber Markdown …",
  "body_html": "<h2>…</h2>",
  "updated_at": "2026-08-10T09:00:00Z"
}
```

`body` adalah sumber Markdown; `body_html` adalah konten yang sama, di-*render* menjadi
HTML tersanitasi di sisi server — pakai mana pun yang sesuai UI Anda. Lihat
[`legalPage`](#legalpage).

> Halaman ini juga dapat diakses sebagai HTML di `https://classyfm.co.id/privacy-policy`
> dan `https://classyfm.co.id/terms-and-conditions` (rute milik situs web).

---

## Object reference

Field yang ditandai *(optional)* dihilangkan dari JSON saat kosong.

### program

| Field | Type | Notes |
|-------|------|-------|
| `title` | string | |
| `slug` | string | |
| `description` | string | *(optional)* |
| `image_url` | string | *(optional)* URL absolut |
| `url` | string | URL program di situs |
| `on_air` | bool | sedang tayang |

### scheduleGroup

| Field | Type | Notes |
|-------|------|-------|
| `from_day` | int | 0 = Minggu … 6 = Sabtu |
| `to_day` | int | akhir rentang hari (penomoran sama) |
| `start_time` | string | `HH:MM` |
| `end_time` | string | `HH:MM` |
| `host` | string | *(optional)* set broadcaster efektif, mis. `"Anda, Yeni"` |

### broadcaster

| Field | Type | Notes |
|-------|------|-------|
| `name` | string | |
| `slug` | string | |
| `role` | string | *(optional)* |
| `photo_url` | string | *(optional)* URL absolut |
| `bio` | string | *(optional)* |
| `birth_place` | string | *(optional)* |
| `birth_date` | string | *(optional)* |
| `instagram` | string | *(optional)* |
| `twitter` | string | *(optional)* |
| `facebook` | string | *(optional)* |
| `url` | string | URL broadcaster di situs |
| `on_air` | bool | sedang membawakan program yang tayang |

### newsItem

| Field | Type | Notes |
|-------|------|-------|
| `source` | string | `youtube` \| `klikpositif` \| `katasumbar` \| `hot_release` |
| `source_label` | string | nama source yang mudah dibaca |
| `title` | string | |
| `slug` | string | *(optional)* ada untuk `hot_release` |
| `excerpt` | string | *(optional)* |
| `image_url` | string | *(optional)* |
| `url` | string | di situs untuk `hot_release`, eksternal untuk lainnya |
| `published_at` | string | timestamp RFC 3339 |
| `is_featured` | bool | *(optional)* |

### newsGroup

| Field | Type | Notes |
|-------|------|-------|
| `source` | string | kode/identitas sumber (*source*) |
| `label` | string | label tampilan |
| `items` | newsItem[] | |

### podcast

| Field | Type | Notes |
|-------|------|-------|
| `title` | string | |
| `slug` | string | |
| `description` | string | *(optional)* |
| `spotify_url` | string | *(optional)* |
| `thumb_url` | string | *(optional)* URL absolut |
| `series_name` | string | *(optional)* |
| `broadcaster` | string | *(optional)* hanya tampilan list |
| `url` | string | URL podcast di situs |

### scheduleRow

| Field | Type | Notes |
|-------|------|-------|
| `title` | string | judul program |
| `slug` | string | *(optional)* |
| `host` | string | *(optional)* |
| `image` | string | *(optional)* URL absolut |
| `start_time` | string | `HH:MM` |
| `end_time` | string | `HH:MM` |
| `on_air` | bool | |
| `progress` | int | 0–100, % berjalannya slot |
| `ended` | bool | slot sudah selesai hari ini |

### heroSlide

| Field | Type | Notes |
|-------|------|-------|
| `href` | string | *(optional)* target tautan |
| `target_blank` | bool | *(optional)* buka di tab baru |
| `image_url` | string | *(optional)* |
| `badge_label` | string | *(optional)* |
| `title` | string | |
| `excerpt` | string | *(optional)* |
| `date` | string | *(optional)* timestamp RFC 3339 |

### contact

Dikembalikan inline di dalam `contact` pada [`/config`](#get-apiv1config). Field opsional
dihilangkan bila kosong; seluruh objek bernilai `null` bila backend tidak memiliki
database.

| Field | Type | Notes |
|-------|------|-------|
| `whatsapp_number` | string | *(optional)* bentuk tampilan, sesuai input admin |
| `whatsapp_message` | string | *(optional)* pesan prefilled |
| `whatsapp_url` | string | *(optional)* deep link `wa.me` siap pakai; dihilangkan bila tanpa nomor |
| `phone` | string | *(optional)* bentuk tampilan |
| `phone_href` | string | *(optional)* `tel:+<digit>`; dihilangkan bila tanpa nomor |
| `email` | string | *(optional)* |

### legalPage

| Field | Type | Notes |
|-------|------|-------|
| `slug` | string | `privacy` \| `terms` |
| `title` | string | |
| `intro` | string | *(optional)* |
| `body` | string | sumber Markdown |
| `body_html` | string | HTML tersanitasi hasil render dari `body` |
| `updated_at` | string | timestamp RFC 3339 |
