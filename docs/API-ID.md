# ClassyFM Public JSON API

API JSON publik yang berorientasi baca (read-oriented) untuk aplikasi mobile ClassyFM. Semua
endpoint dipasang di bawah **`/api/v1`**.

- **Base URL:** `https://classyfm.co.id/api/v1`
- **Format:** JSON (`Content-Type: application/json; charset=utf-8`)
- **Orientasi:** read-only, **tanpa cookie (cookie-free)** — satu-satunya operasi tulis
  adalah endpoint Connect chat, yang diautentikasi dengan Bearer token, bukan session cookie.

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
- [Connect chat endpoints](#connect-chat-api)
  - [Signing in from a mobile app](#signing-in-from-a-mobile-app)
- [Authentication](#authentication)
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
| `400` | Request body buruk atau kosong, path id tidak valid |
| `401` | Butuh autentikasi / Google atau Bearer token tidak valid |
| `403` | Di-ban dari chat |
| `404` | Resource tidak ditemukan, atau fitur belum dikonfigurasi |
| `429` | Rate limit terlampaui |
| `500` | Error server / database |
| `503` | Chat sementara tidak tersedia |

Dua endpoint yang di-rate-limit (`POST /api/v1/connect/session` sebesar 30/menit dan
`POST /api/v1/connect/messages` sebesar 20/menit) dibatasi **per IP klien** dalam
**jendela tetap 60 detik**. Request yang melebihi batas mengembalikan `429` dengan header
`Retry-After: 60` dan body JSON
`{ "error": "too many requests, try again later" }`.

### Caching

Setiap respons membawa header `Cache-Control`:

- `public, max-age=60` — konten yang berubah lambat (programs, news, podcasts, about,
  ads, home, config).
- `no-store` — data live atau per-pengguna (now-playing, status schedule, TikTok live, semua
  endpoint chat).

<br><br>

### Pagination & filtering

Pagination hanya berlaku pada `GET /news?source=…` dan `GET /podcasts`:

- `page` — berbasis 1, default `1` (nilai `< 1` diperlakukan sebagai `1`).
- Ukuran halaman tetap **12**.
- `meta.total_pages` minimal `1` bahkan saat kosong.

### URLs & images

Semua field gambar dan tautan dikembalikan sebagai **URL absolut** (di-resolve terhadap
origin situs, `https://classyfm.co.id`). URL yang sudah absolut — mis. thumbnail aggregated
news — diteruskan tanpa perubahan.

### Degraded mode

Jika database tidak tersedia, endpoint **list** menurun dengan anggun (degrade gracefully) ke
`{ "data": [] }` dan endpoint **detail** mengembalikan `404` — keduanya tidak pernah 500
karena DB tidak ada.

---

## Endpoints

### App bootstrap endpoints

Dua endpoint agregat yang pertama kali dihubungi aplikasi saat peluncuran.

| Method | Endpoint | Auth | Deskripsi |
|--------|----------|------|-------------|
| `GET` | [`/api/v1/config`](#get-apiv1config) | open | Bootstrap aplikasi: identity, stream, social, flag chat |
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
| `GET` | [`/api/v1/podcast-series`](#get-apiv1podcast-series) | open | Daftar podcast series |
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

### Connect chat endpoints

| Method | Endpoint | Auth | Deskripsi |
|--------|----------|------|-------------|
| `GET` | [`/api/v1/connect/messages`](#get-apiv1connectmessages--open) | open | Poll pesan chat |
| `POST` | [`/api/v1/connect/session`](#post-apiv1connectsession--open-rate-limited-30min) | open (30/menit) | Tukar Google ID token dengan Connect token |
| `GET` | [`/api/v1/connect/me`](#get-apiv1connectme--bearer-required) | Bearer | Identity chat di balik token |
| `POST` | [`/api/v1/connect/messages`](#post-apiv1connectmessages--bearer-required-rate-limited-20min) | Bearer (20/menit) | Kirim pesan chat |

---

## App bootstrap endpoints

Keduanya `GET`, di-cache `public, max-age=60` — endpoint yang dipanggil aplikasi saat
peluncuran.

### `GET /api/v1/config`

Bootstrap aplikasi: identity stasiun, URL stream, tautan sosial, dan apakah sign-in chat
tersedia.

```json
{
  "station": { "name": "Classy 103.4 FM", "slogan": "…" },
  "stream_url": "https://c4.siar.us:10340/stream.mp3",
  "social": { "instagram": "https://…", "youtube": "https://…" },
  "connect": { "enabled": true }
}
```

`connect.enabled` bernilai `true` hanya saat Google sign-in dikonfigurasi. Kunci `social`
adalah nama platform sebagaimana dikonfigurasi di admin panel.

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

Roster program aktif, masing-masing ditandai apakah sedang on air saat ini.

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

Satu program dengan seluruh weekly schedule dan broadcasters-nya. `404` jika slug tidak
dikenali atau program tidak aktif.

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

Roster broadcaster aktif. `on_air` bernilai `true` jika ada program yang dibawakan
broadcaster tersebut sedang tayang.

```json
{ "data": [ /* broadcaster objects */ ] }
```

### `GET /api/v1/broadcasters/{slug}`

Satu broadcaster beserta program yang dibawakannya. `404` jika tidak dikenali.

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

Dua mode:

**Tanpa `source`** (atau dengan source yang tidak dikenali) — preview yang dikelompokkan,
hingga 6 item per grup source:

```json
{ "data": [
  { "source": "klikpositif", "label": "KlikPositif", "items": [ /* news items */ ] },
  { "source": "katasumbar",  "label": "KataSumbar",  "items": [ … ] },
  { "source": "hot_release", "label": "Hot Release", "items": [ … ] },
  { "source": "youtube",     "label": "YouTube",     "items": [ … ] }
] }
```

**Dengan `source` yang valid** — item dari source tersebut, dengan pagination:

| Query param | Catatan |
|-------------|-------|
| `source` | Salah satu dari `youtube`, `klikpositif`, `katasumbar`, `hot_release`. <br>Nilai tidak valid kembali ke mode grouped. |
| `page` | Berbasis 1, ukuran halaman 12. |

```json
{ "data": [ /* news items */ ], "meta": { "page": 1, "total_pages": 4, "total": 42 } }
```

Item adalah objek [`newsItem`](#newsitem). Catatan: item `hot_release` menautkan ke
`/news/{slug}` di situs; source (aggregated) lain mempertahankan `url` eksternalnya.

### `GET /api/v1/news/{slug}`

Satu artikel `hot_release`, dengan paragraf yang dipisah, galeri gambar di tengah artikel,
dan artikel terkait. `404` jika tidak dikenali (hanya artikel `hot_release` yang punya detail
di situs).

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

`gallery_index` adalah indeks paragraf yang setelahnya galeri `middle_images` sebaiknya
disisipkan.

### `GET /api/v1/podcasts`

Podcast yang dipublikasikan, terbaru dulu, dengan pagination.

| Query param | Catatan |
|-------------|-------|
| `series` | Filter slug series opsional. Slug tidak valid diabaikan (mengembalikan semua). |
| `page` | Berbasis 1, ukuran halaman 12. |

```json
{ "data": [ /* podcast objects */ ], "meta": { "page": 1, "total_pages": 2, "total": 18 } }
```

### `GET /api/v1/podcasts/{slug}`

Satu podcast dengan nama series dan broadcasters-nya. `404` jika tidak dikenali.

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

Daftar podcast series (untuk pemilih filter yang memberi umpan ke `?series=`).

```json
{ "data": [ { "name": "…", "slug": "…" } ] }
```

### `GET /api/v1/about`

Konten halaman about: banner, segmen teks, dan preview broadcaster (maks 8).

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

`banner.embed_url` hanya ada saat `media_type` bernilai `"video"` dan URL video adalah
tautan YouTube yang bisa di-resolve. `segment` adalah salah satu dari `profile`, `music`,
`audience`.

---

## Live endpoints

Semua adalah `GET`, di-cache `no-store` — status stream live dan progress schedule.

### `GET /api/v1/now-playing`

Metadata now-playing stream, ditambah program on-air saat stream sedang live.

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

Lihat [`scheduleRow`](#schedulerow). (Jumlah listener Shoutcast sengaja tidak diekspos di
sini — hanya untuk admin.)

### `GET /api/v1/schedule/today`

Schedule lengkap hari ini dengan status on-air/progress live.

```json
{ "data": [ /* scheduleRow objects */ ] }
```

### `GET /api/v1/schedule/current`

Program yang sedang on-air sebagai satu objek [`scheduleRow`](#schedulerow), atau
`{ "on_air": false }` saat tidak ada yang tayang.

### `GET /api/v1/tiktok/live`

```json
{ "live": true, "title": "…" }
```

### Playing the live stream in an app

Audio stream adalah **MP3 Shoutcast eksternal langsung** — ia berada di host Shoutcast,
**bukan** di `classyfm.co.id`, dan API ini tidak pernah mem-proxy atau me-redirect audio.
Aplikasi memutarnya **secara langsung**:

1. **Bootstrap URL-nya.** Baca `stream_url` dari [`/api/v1/config`](#get-apiv1config)
   sekali saat startup dan serahkan ke pemutar audio native perangkat. Ini adalah MP3
   streaming (Shoutcast) yang polos dan tanpa autentikasi — tanpa header, tanpa token, tanpa
   proxy. Lebih baik membacanya dari `config` daripada meng-hardcode-nya, agar host Shoutcast
   bisa dipindahkan dari sisi server tanpa merilis aplikasi.
2. **Gerakkan now-playing dari loop poll — via API ini, jangan langsung ke Shoutcast.** Poll
   [`/api/v1/now-playing`](#get-apiv1now-playing) pada timer untuk memperbarui UI now-playing
   (dan metadata lock-screen / notifikasi apa pun). API mengambil dan meng-cache metadata
   track serta status `live` dari server Shoutcast, sehingga aplikasi menghindari
   CORS dan tidak membebani box Shoutcast — **jangan scrape endpoint Shoutcast sendiri.**
   Responsnya `no-store` tetapi server menyegarkan metadata upstream-nya hanya tiap ~12 detik,
   jadi **polling lebih cepat dari ~15 detik tidak menghasilkan apa-apa** — pilih interval
   sekitar 15 detik selama pemutar aktif, dan hentikan polling saat ia berhenti atau di
   background tanpa audio.
   - Tampilkan `artist` + `song` saat `has_song` bernilai `true`; saat `false` tidak ada
     metadata track (station ID / tanpa judul) — gunakan nama stasiun sebagai fallback.
   - Gunakan `cover_url` untuk artwork, tetapi ini best-effort dan bisa berupa `""` — gunakan
     placeholder bawaan atau gambar program on-air sebagai fallback.
3. **Tangani live vs. off-air.** Alihkan UI antara "on air" dan "off air" berdasarkan flag
   `live`. Saat `live` bernilai `true`, objek `program` opsional (sebuah
   [`scheduleRow`](#schedulerow)) memberi acara saat ini, host, dan `progress` (0–100);
   [`/api/v1/schedule/current`](#get-apiv1schedulecurrent) mengembalikan hal yang sama secara
   mandiri, dan [`/api/v1/schedule/today`](#get-apiv1scheduletoday) mendukung daftar "up
   next".

### When the backend API is unreachable

Karena audio adalah **MP3 Shoutcast langsung** dan API ini tidak pernah mem-proxy-nya,
downtime backend (network error, timeout, `5xx`) **tidak** mengganggu pemutaran — hanya
*metadata* di sekitarnya. Jaga audio tetap berjalan dan turunkan hanya chrome now-playing:

1. **Persist `config`.** Cache respons [`/api/v1/config`](#get-apiv1config) sukses terakhir
   (minimal `stream_url`, ditambah nama/slogan stasiun dan tautan sosial) di penyimpanan
   lokal. Saat peluncuran, mulai pemutaran dari `stream_url` yang di-cache bahkan saat
   `config` tidak bisa di-fetch ulang. **Tidak ada** stream URL bawaan/hardcoded — aplikasi
   tidak pernah mengarangnya sendiri. Pada cold start sesungguhnya (peluncuran pertama kali,
   tidak ada yang di-cache, dan API tidak terjangkau) tidak ada URL untuk diputar: tampilkan
   status "stream unavailable / retry" dan fetch `config` lagi begitu konektivitas kembali.
2. **Jaga audio tetap hidup saat `now-playing` gagal.** Poll
   [`/api/v1/now-playing`](#get-apiv1now-playing) yang gagal/timeout/`5xx` adalah celah
   metadata, bukan kegagalan stream — jangan pernah menghentikan atau me-reset pemutar
   karenanya. Pertahankan `artist`/`song` dan status `live` yang terakhir diketahui, atau
   gunakan nama stasiun + artwork placeholder bawaan sebagai fallback.
3. **Back off, lalu pulih.** Pada kegagalan poll berulang, lebarkan interval (mis.
   exponential backoff hingga ~60 detik) alih-alih membebani terus. Pada poll sukses
   berikutnya, kembali ke irama ~15 detik normal, segarkan UI now-playing, dan cache ulang
   `config`. Tidak perlu aksi pengguna atau restart aplikasi.

Jumlah listener live sengaja tidak tersedia untuk aplikasi.

---

## Advertising endpoints

`GET`, di-cache `public, max-age=60`.

### `GET /api/v1/ads`

Banner ads untuk suatu halaman, dikelompokkan ke placement slot `top` dan `bottom`.

| Query param | Catatan |
|-------------|-------|
| `page` | Kunci halaman target. Kunci tidak valid mengembalikan `400`. Jika dihilangkan, mengembalikan hanya banner yang ditargetkan ke setiap halaman. |

Kunci `page` yang valid: `home`, `about`, `program`, `program_detail`, `live`, `news`,
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

`banners` tiap slot adalah array (kosong saat slot tidak punya banner). `slideshow`
memberi tahu klien untuk merotasi banner tiap `rotate_ms` alih-alih menumpuknya;
`placeholder` (dengan `placeholder_text`) menyatakan bahwa slot kosong sebaiknya menahan
ruangnya alih-alih menciut.

---

## Connect chat endpoints

Chatroom Connect melalui JSON, dipasang di bawah `/api/v1/connect`. **Baca bersifat open**;
**tulis membutuhkan Bearer token** yang diperoleh dengan menukar Google ID token (lihat
[Authentication](#authentication)). Semua respons chat bersifat `no-store`.

### `GET /api/v1/connect/messages` — open

Poll pesan chat.

| Query param | Catatan |
|-------------|-------|
| `since` | Message id opsional. Dengannya, mengembalikan hingga 200 pesan yang lebih baru dari id tersebut (delta poll). Tanpanya, mengembalikan 50 pesan terbaru. |

```json
{ "messages": [ /* chatMessage objects */ ] }
```

### `POST /api/v1/connect/session` — open (rate-limited 30/min)

Tukar Google ID token native dengan Connect session token. Aplikasi memperoleh ID token
melalui Google sign-in milik platform (sehingga Google OAuth tidak pernah berjalan di
WebView), dengan meneruskan Google client id situs sebagai `serverClientId`-nya.

**Request:**

```json
{ "id_token": "<google-id-token>" }
```

**Response:**

```json
{
  "token": "<connect-session-token>",
  "user": { "id": 42, "name": "…", "avatar": "https://…", "is_admin": false }
}
```

`token` diverifikasi di sisi server terhadap endpoint `tokeninfo` Google (memeriksa audience
dan issuer), lalu pengguna chat di-upsert dan badge admin dihitung ulang. Kirim `token`
sebagai `Authorization: Bearer <token>` pada panggilan terautentikasi berikutnya.

**Errors:** `404` jika login chat tidak dikonfigurasi, `400` jika `id_token` hilang,
`401 invalid Google token`, `500` saat gagal. Request body dibatasi 16 KiB.

### `GET /api/v1/connect/me` — Bearer required

Mengembalikan identity chat di balik token.

```json
{ "user": { "id": 42, "name": "…", "avatar": "https://…", "is_admin": false } }
```

### `POST /api/v1/connect/messages` — Bearer required (rate-limited 20/min)

Kirim pesan chat. Body di-sanitize, di-trim, dan dibatasi **1000 karakter**.

**Request:**

```json
{ "body": "Hello!" }
```

**Response** — pesan yang tersimpan, agar aplikasi bisa menampilkannya secara optimistic:

```json
{ "message": { "id": 101, "user_id": 42, "name": "…", "avatar": "…", "is_admin": false, "body": "Hello!", "time": "14:03" } }
```

**Errors:** `401` jika belum terautentikasi, `403 your account is blocked from chat` jika
di-ban, `400` jika body kosong/tidak valid, `503` jika chat tidak tersedia. Body dibatasi 16 KiB.

### Signing in from a mobile app

Langkah-langkah di sisi aplikasi. Mekanisme token di baliknya — verifikasi, TTL, penyimpanan,
re-auth — ada di [Authentication](#authentication); ini hanya urutan operasinya:

1. **Cek ketersediaan.** Hanya tawarkan sign-in saat
   [`/api/v1/config`](#get-apiv1config) melaporkan `connect.enabled: true`; saat bernilai
   `false`, Google sign-in tidak dikonfigurasi di sisi server dan `/connect/session`
   mengembalikan `404`. (Baca — polling pesan — tidak butuh sign-in; wajibkan hanya sebelum
   memposting.)
2. **Sign in di perangkat.** Jalankan Google sign-in native platform (jangan pernah WebView)
   dan terima Google **ID token**, dengan meneruskan Google client id ClassyFM sebagai
   `serverClientId` — lihat [Authentication](#authentication) untuk nilai tersebut dan
   alasannya.
3. **Tukar ID token.** `POST` `{ "id_token": "…" }` ke sini dan simpan Bearer `token` yang
   dikembalikan sesuai [Authentication](#authentication). Objek `user` sudah cukup untuk
   menampilkan identity yang sudah sign-in secara langsung.
4. **Gunakan token.** Kirim `Authorization: Bearer <token>` pada
   [`/connect/me`](#get-apiv1connectme--bearer-required) dan
   [`POST /connect/messages`](#post-apiv1connectmessages--bearer-required-rate-limited-20min).

---

## Authentication

Tulis chat menggunakan **Bearer token yang di-sign dengan HMAC** dan bersifat self-contained —
tanpa baris session di sisi server, tanpa cookie.

1. **Memperoleh token.** Aplikasi native sign-in dengan Google di perangkat dan menerima
   Google **ID token**. Ia mem-`POST` token itu ke `/api/v1/connect/session`, yang
   memverifikasinya terhadap endpoint `tokeninfo` Google (memastikan audience sama dengan
   Google client id situs yang dikonfigurasi dan issuer-nya adalah Google), meng-upsert
   pengguna chat, dan mengembalikan **Connect session token**. Aplikasi harus meneruskan
   nilai Google client-id yang sama ini sebagai `serverClientId`-nya saat sign-in — peroleh
   dari tim ClassyFM.

2. **Menggunakan token.** Kirim pada endpoint terautentikasi:

   ```
   Authorization: Bearer <connect-session-token>
   ```

3. **Properti token.**
   - Payload hanya membawa chat user id dan expiry, di-sign dengan `SESSION_SECRET`.
   - **TTL: 30 hari.**
   - Pada tiap request, identity (name, avatar, status admin/ban) dibaca ulang dari database,
     sehingga ban dan perubahan role berlaku seketika tanpa menerbitkan ulang token.
   - Penegakan bersifat per-endpoint: `401` saat token hilang/tidak valid.

4. **Siklus hidup aplikasi.**
   - **Penyimpanan & expiry.** Simpan token di secure store platform (bukan preferences
     biasa). Ia valid selama 30 hari; tidak ada endpoint refresh — jalankan ulang pertukaran
     Google sign-in untuk mendapatkan yang baru.
   - **Re-auth pada `401`.** Panggilan terautentikasi mana pun bisa mengembalikan `401`
     setelah token expired atau menjadi tidak valid — hapus token yang tersimpan dan minta
     sign-in lagi.
   - **`403` bersifat final.** `403 your account is blocked from chat` berarti akun di-ban;
     jangan coba lagi atau menerbitkan ulang — identity yang sama akan terus ditolak.

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
| `source` | string | kunci source |
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

### connectUser

| Field | Type | Notes |
|-------|------|-------|
| `id` | int | chat user id |
| `name` | string | |
| `avatar` | string | *(optional)* |
| `is_admin` | bool | |

### chatMessage

| Field | Type | Notes |
|-------|------|-------|
| `id` | int | message id (dipakai sebagai cursor `since`) |
| `user_id` | int | chat user id penulis |
| `name` | string | nama penulis |
| `avatar` | string | URL avatar penulis |
| `is_admin` | bool | penulis adalah moderator |
| `body` | string | teks pesan yang sudah di-sanitize |
| `time` | string | `HH:MM`, timezone stasiun |
