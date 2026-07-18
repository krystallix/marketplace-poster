## Planning: Sistem Scheduler Auto-Posting Marketplace Facebook - Jualan AKI

### 1. Konteks Bisnis
- **Produk**: Aki (accu/battery kendaraan) - Aki Mobil Jogja
- **Lokasi**: Yogyakarta, WIB
- **Referensi tool**: aronk254/Facebook-Marketplace-Auto-Poster

### 2. Kendala Teknis Utama
- Facebook TIDAK punya API resmi untuk Marketplace
- **Solusi**: Browser automation (headless)
- **Risiko**: Facebook bisa ubah struktur DOM/selector kapan saja

### 3. Jadwal Posting (3x/hari, WIB)
| Slot | Jam Target    | Alasan                                          |
|------|---------------|-------------------------------------------------|
| Pagi | 07.00 - 09.00 | Cek HP saat bangun                              |
| Siang| 12.00 - 14.00 | Istirahat kerja, waktu cek cepat                 |
| Sore/Malam | 17.00 - 20.00 | Santai setelah kerja, engagement tinggi |
- **Tambahan**: Random jitter +/- 15-30 menit

### 4. Arsitektur Sistem
- **Scheduler**: Go + robfig/cron/v3
- **Product/Image Picker**: Pilih gambar random
- **Content Generator**: Panggil LLM API untuk parafrase
- **Session Manager**: Simpan cookie/session login FB
- **Poster Engine**: chromedp/playwright-go

### 5. Detail Post
- **Deskripsi Umum**: Harga mulai dari 400-600, tersedia merk dan jenis baru, bergaransi
- **Layanan**: Antar pasang gratis, bisa tukar tambah
- **Lokasi**: Siswanto Aki, Kanggotan 21, Pleret, Bantul
- **Kontak**: WA 081354007400
- **Kategori**: Auto Parts & Accessories

