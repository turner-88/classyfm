-- Legal pages (Privacy Policy, Terms & Conditions) move from hardcoded templates
-- to admin-editable Markdown, mirroring the fixed-key, seed-once, UPDATE-only shape
-- of the about_page_* tables (migration 0017). Body holds Markdown source; the
-- public handler renders it to sanitized HTML at request time.
--
-- Paragraphs are authored one-per-line on purpose: goldmark keeps soft line breaks
-- as "\n", and .prose-article p (whitespace-pre-line) turns those into visible
-- breaks — that is how the multi-line contact block renders without <br>.
CREATE TABLE legal_pages (
  slug       ENUM('privacy','terms') PRIMARY KEY,
  title      VARCHAR(255)  NOT NULL,
  intro      VARCHAR(1000) NOT NULL DEFAULT '',
  body       MEDIUMTEXT    NOT NULL,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO legal_pages (slug, title, intro, body) VALUES
('privacy',
 'Kebijakan Privasi',
 'Bagaimana Classy 103.4 FM mengumpulkan, menyimpan, dan mengelola data serta informasi pribadi Anda.',
 'Kebijakan Privasi ini adalah bentuk komitmen nyata dari kami, Radio CLASSY FM (PT Radio Gema Karang Putih, atau “kami”), untuk menghargai dan melindungi setiap data atau informasi pribadi pengguna website atau aplikasi dan layanan CLASSYFM. Kami mohon Anda untuk membaca, memahami, dan menyetujui bagaimana kami mengumpulkan, menyimpan, dan mengelola data atau informasi yang Anda berikan.

_Kebijakan Privasi ini mencakup hal-hal sebagai berikut:_

## Pengumpulan Data

Kami hanya mengumpulkan data mengenai Anda yang diberikan kepada kami saat menggunakan website atau aplikasi. Data tersebut hanya berupa ID alamat e-mail yang sudah ada di smartphone Anda.

## Penggunaan Data

Atas persetujuan Anda atas Kebijakan Privasi ini, kami akan menggunakan data untuk tujuan sebagai berikut:

1. Mengembangkan dan meningkatkan layanan website atau aplikasi kami.
2. Memberitahu Anda mengenai segala pembaruan atau perubahan pada layanan kami.
3. Untuk melakukan komunikasi mengenai hal-hal yang terkait dengan website atau aplikasi dan/atau untuk menanggapi pertanyaan atau saran yang kami terima dari Anda.

## Penyimpanan dan Penghapusan Data

Kami akan menyimpan data Anda selama akun Anda masih aktif.

## Tautan

Kami dapat menampilkan konten lain yang tertaut ke website atau aplikasi, dan bisa jadi kami tidak dapat mengendalikan atau dimintai pertanggungjawaban atas tindakan dan konten pihak ketiga tersebut. Apabila Anda mengklik iklan atau tautan pihak ketiga, maka dapat dipastikan Anda telah keluar dari website atau aplikasi kami sehingga kebijakan pihak ketiga atas layanan yang Anda klik tidak memiliki keterkaitan dengan website atau aplikasi dan/atau kebijakan kami.

## Perubahan Kebijakan Privasi

Dari waktu ke waktu, kami dapat merevisi Kebijakan Privasi ini untuk mencerminkan perubahan baik dalam aspek hukum, pengumpulan dan praktik penggunaan data, fitur website atau aplikasi kami, atau karena adanya kemajuan dalam teknologi.

Kami menyarankan agar Anda memeriksa halaman Kebijakan Privasi ini dari waktu ke waktu untuk mengetahui perubahan yang terjadi, dan dengan mengakses website atau aplikasi kami maka Anda ditafsirkan telah menyetujui perubahan dalam Kebijakan Privasi ini.

## Persetujuan

Dengan menyetujui Kebijakan Privasi ini, pengguna menyatakan bahwa setiap data pengguna merupakan data yang benar, akurat, terkini, dan sah, serta pengguna memberikan persetujuan kepada kami untuk memperoleh, mengumpulkan, menyimpan, mengelola, dan mempergunakan data tersebut sebagaimana tercantum dalam Kebijakan Privasi.

Anda juga dapat mengubah, menambahkan, dan/atau memperbarui data yang diberikan kepada kami untuk layanan serta fungsi website atau aplikasi.

## Jaminan

Kami menerapkan langkah-langkah teknis dan organisasi yang sesuai untuk melindungi informasi pribadi Anda yang kami kumpulkan, simpan, gunakan, atau olah, atas kemungkinan kerusakan, kehilangan, perubahan, dan pengungkapan atau akses yang tidak disengaja atau melanggar hukum.

Untuk data pengguna yang disusupi karena pelanggaran keamanan, kami akan mengambil langkah-langkah yang wajar untuk menyelidiki situasi tersebut dan memberitahu pengguna yang datanya mungkin telah disusupi, serta mengambil langkah-langkah lainnya berdasarkan hukum dan peraturan yang berlaku.

## Hubungi Kami

**PT Radio Gema Karang Putih**
Gedung Serba Guna
Semen Padang Lt. 2
Indarung - Padang 25237
Sumatera Barat – Indonesia
No Telepon: (0751) 74999, 08116604090
Email: [classyfm@classyfm.co.id](mailto:classyfm@classyfm.co.id)

_Kebijakan Privasi ini dibuat pada 27 Februari 2023._'),
('terms',
 'Syarat dan Ketentuan',
 'Ketentuan penggunaan website, aplikasi, dan layanan Classy 103.4 FM.',
 '## Penjelasan Maksud dan Tujuan

Selamat datang di website atau aplikasi CLASSYFM milik PT Radio Gema Karang Putih (selanjutnya disebut ‘kami’). CLASSYFM merupakan website atau aplikasi radio lokal terestrial yang di-streaming-kan.

Syarat dan ketentuan di bawah ini merupakan ketentuan dalam penggunaan website atau layanan ini.

Harapan kami, Anda membaca Syarat dan Ketentuan ini dengan saksama. Dengan mengakses dan menggunakan website atau layanan ini, berarti Anda telah memahami dan menyetujui untuk terikat dan tunduk dengan semua peraturan yang berlaku pada website ini.

Jika Anda tidak setuju untuk terikat dengan semua peraturan yang berlaku, Anda tidak diharuskan mengakses website atau layanan ini.

## Perubahan Syarat dan Ketentuan

Kami dapat mengubah peraturan umum sewaktu-waktu beserta Syarat dan Ketentuan ini untuk mencerminkan adanya perubahan dalam hukum, pengumpulan dan praktik penggunaan data, fitur CLASSYFM, atau kemajuan dalam teknologi. Oleh karena itu, kami menyarankan agar Anda memeriksa halaman Syarat dan Ketentuan ini dari waktu ke waktu untuk mengetahui perubahan yang terjadi dan menyetujui perubahan dalam Syarat dan Ketentuan ini. Keputusan yang diambil berdasarkan peraturan ini bersifat mutlak dan tidak dapat diganggu gugat.

Dengan terus mengakses atau menggunakan Layanan kami setelah revisi tersebut, maka Anda dianggap setuju untuk terikat oleh ketentuan yang direvisi. Jika Anda tidak menyetujui persyaratan baru, maka Anda dapat menghentikan penggunaan website atau layanan ini.

## Definisi

“CLASSYFM” adalah website atau aplikasi streaming radio Classy terestrial dengan frekuensi 103.4 MHz yang berada di kota Padang, Sumatera Barat, Indonesia.

“Pengelola” adalah PT Radio Gema Karang Putih.

“Layanan” adalah segala bentuk aktivitas yang terjadi di CLASSYFM yang diperuntukkan untuk pengguna.

“Pendengar” adalah setiap orang yang mendengarkan CLASSYFM.

“Musik” adalah konten berupa lagu atau audio yang dimiliki oleh label musik.

“Pengguna” adalah semua orang yang mendaftarkan diri (login) dan yang mengakses CLASSYFM.

Website atau aplikasi ini terdiri dari:

“Streaming” adalah konten siaran radio terestrial yang secara langsung (real time) dimiliki radio Classy FM.

“Schedule” adalah jadwal siaran harian radio Classy FM.

“Chart” adalah tangga lagu mingguan yang menjadi pemuncak di radio Classy FM dan menampilkan video dari YouTube dan lirik yang bersifat free dari penyedia.

“Classier” adalah kumpulan penyiar (announcer) yang siaran di radio Classy FM, dilengkapi dengan data foto dan biodata.

“Podcast” adalah konten berupa rekaman audio dari siaran program terpilih radio Classy FM.

“Connect” adalah ruang untuk pengguna berkomunikasi melalui teks dengan admin Classy dan pengguna lainnya.

“Klikpositif” adalah konten berita dari media online klikpositif.com, yang merupakan bagian dari radio ClassyFM, Corporation.

“Sosial” adalah channel resmi sosial media yang radio Classy FM miliki.

“Profile” adalah pengenalan informasi perusahaan radio Classy FM yang berisikan Visi, Misi, Pendengar, Format siaran, nama badan hukum perusahaan, alamat, dan kontak perusahaan.

“Login with Apple” adalah fasilitas masuk agar dapat menggunakan ruang untuk komunikasi sesama pengguna aplikasi.

“Change Profile” adalah fasilitas untuk membuat nama akun dan mengunggah foto di aplikasi ClassyFM yang dibuat oleh pengunjung saat mendaftar sebagai pengguna.

“Logo Microphone” adalah banner program yang sedang on air saat ini (lingkar merah dan tulisan on air) dan dilanjutkan banner program lainnya.

## Cara Menggunakan Akun dan Layanan

1. Unduh aplikasi pada smartphone Anda.
2. Ketika Anda telah selesai mengunduh aplikasi atau layanan, Anda boleh memilih login dengan akun yang sudah tersedia di smartphone Anda atau skip (lewati).
3. Jika Anda memilih login dengan akun, Anda akan bisa mengakses semua layanan yang ada di aplikasi CLASSYFM.
4. Login dengan akun dapat dilakukan dengan memilih akun yang sudah terpasang di smartphone Anda.
5. Anda dapat mengganti profil, nama, dan foto atau menggunakan foto (avatar) yang sudah disediakan.
6. Dilarang menggunakan foto dan nama yang bersifat vulgar atau cabul atau yang bertentangan dengan hukum positif Indonesia maupun norma-norma yang berlaku secara khusus di semua daerah dan global.
7. Apabila Anda memilih skip (lewati), Anda tidak bisa menggunakan fitur Connect karena Anda akan masuk sebagai Guest (Tamu).

## Penghormatan atas Hak Cipta

Kami menghormati hak kekayaan intelektual orang lain. Merupakan kebijakan kami untuk menanggapi klaim apa pun (“Pelanggaran”) dari siapa pun terhadap konten yang diposting di website atau layanan ini.

Jika Anda adalah pemilik hak cipta, atau diberi wewenang atas nama salah satu pemegang hak cipta, dan meyakini bahwa karya cipta Anda telah disalin dengan cara melakukan pelanggaran hak cipta melalui website atau layanan ini, Anda dapat menyampaikan pemberitahuan secara tertulis kepada kontak yang tertera di website atau layanan ini, dengan menyertakan deskripsi terperinci tentang Pelanggaran yang diduga dilakukan.

Sebaliknya, Anda juga mungkin bertanggung jawab atas kerusakan yang terjadi (termasuk segala biaya yang muncul) akibat kesalahan dalam mengartikan terjadinya pelanggaran terhadap hak cipta yang Anda lakukan.

## Hak Kekayaan Intelektual

Layanan dan konten aslinya, fitur, dan semua fungsi di website atau layanan ini adalah hak milik eksklusif PT Radio Gema Karang Putih.

## Tautan dengan Website atau Aplikasi Lain

Website atau layanan CLASSYFM kami mungkin dapat memiliki tautan dengan website atau layanan pihak ketiga yang tidak dimiliki atau tidak dapat dikontrol penggunaannya oleh PT Radio Gema Karang Putih. Kami juga tidak memiliki kendali dan tidak bertanggung jawab atas konten, kebijakan privasi, atau praktik lainnya dari website atau layanan pihak ketiga mana pun.

Dengan ini Anda menyatakan menyetujui bahwa PT Radio Gema Karang Putih tidak akan bertanggung jawab atau berkewajiban secara langsung atau tidak langsung untuk setiap kerusakan atau kerugian yang disebabkan atau diduga disebabkan karena penggunaan Anda atas konten atau layanan yang tersedia di website tersebut.

Karena itu, kami sangat menyarankan Anda untuk membaca syarat dan ketentuan serta kebijakan privasi dari setiap website atau layanan pihak ketiga yang Anda kunjungi.

## Keamanan dan Pengolahan Informasi

Data akun pengguna website atau layanan ini (jika ada) sepenuhnya terjaga kerahasiaannya oleh PT Radio Gema Karang Putih sepanjang Anda mengakses dan menggunakan layanan ini. Anda setuju memberikan hak kepada PT Radio Gema Karang Putih untuk melakukan analisa dan pengolahan data untuk pengembangan dan peningkatan layanan website dan aplikasi ini.

## Peringatan Risiko

PT Radio Gema Karang Putih beserta anak perusahaan tidak menjamin bahwa a) Layanan akan berfungsi tanpa gangguan, aman, atau tersedia pada waktu atau lokasi tertentu; b) kesalahan atau cacat apa pun akan diperbaiki.

## Cookies

Cookies adalah tempat penyimpanan data kecil yang terdapat pada komputer atau perangkat lain seperti ponsel pintar atau tablet, pada saat Pengguna menelusuri dan/atau mengunjungi website www.classyfm.co.id.

Komputer atau perangkat lain Pengguna akan secara otomatis menerima Cookies pada saat Pengguna menelusuri dan/atau mengunjungi website www.classyfm.co.id serta saat Pengguna menggunakan Layanan ClassyFM (“Penggunaan Cookies”). Namun, Pengguna dapat menentukan pilihan untuk melakukan modifikasi atau memilih untuk menolak penggunaan Cookies melalui preferensi pengaturan web browser Pengguna.

Website www.classyfm.co.id menggunakan Google Analytics (“Fitur”), di mana data yang diperoleh dari penggunaan Fitur tersebut meliputi IP address Pengguna, jenis perangkat Pengguna, dan lain-lain (“Data”). Data tersebut digunakan oleh ClassyFM untuk pengembangan website dan Platform ClassyFM serta pengiklanan. Pengguna dapat memilih untuk tidak terakses oleh Fitur dengan cara mengunduh Google Analytics Opt-Out Add-on pada web browser Pengguna.

ClassyFM dapat memberikan data dan informasi Pengguna yang berasal dari Penggunaan Cookies kepada pihak ketiga, seperti data lokasi, pengidentifikasi iklan, atau alamat email yang digunakan untuk segmentasi periklanan, termasuk namun tidak terbatas pada kebutuhan pemasaran dan periklanan, di mana data dan informasi tersebut tidak dapat diidentifikasi secara pribadi.

Pengguna dapat melakukan kontrol terhadap Penggunaan Cookies melalui preferensi pengaturan web browser Pengguna, yaitu dengan memodifikasi atau memilih untuk menolak Penggunaan Cookies. Namun dengan melakukan pengaturan tersebut, kinerja pelayanan pada saat akses ke website ClassyFM dapat terpengaruh, seperti fungsi dan halaman tertentu pada website www.classyfm.co.id tidak dapat bekerja optimal untuk layanan kepada Pengguna.

## Kepatuhan terhadap Peraturan Pemerintah

Syarat dan Ketentuan ini diatur dan ditafsirkan sesuai dengan hukum positif Indonesia.

Kegagalan kami untuk menegakkan hak atas Syarat dan Ketentuan ini bukan merupakan bentuk pengabaian hak-hak tersebut. Jika salah satu ketentuan ini dianggap tidak sah atau tidak dapat dilaksanakan menurut keputusan pengadilan, maka ketentuan lainnya akan tetap berlaku sepanjang tidak melanggar ketentuan hukum positif Indonesia dan sepanjang tidak dinyatakan tidak sah oleh keputusan pengadilan.

## Hubungi Kami

Apabila Anda memiliki pertanyaan, keluhan, kritik, atau harapan dan saran, silakan menghubungi kami melalui channel berikut ini:

**PT Radio Gema Karang Putih**
Gedung Serba Guna
Semen Padang Lt. 2
Indarung - Padang 25237
Sumatera Barat – Indonesia
No Telepon: (0751) 74999, 08116604090
Email: [classyfm@classyfm.co.id](mailto:classyfm@classyfm.co.id)');
