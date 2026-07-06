# Ringkasan Audit ISAPI Hikvision - AddPlanScheme

## Payload yang Direkomendasikan

``` json
{
  "broadcastPlanSchemeList": [
    {
      "planSchemeID": "Scheduled B",
      "enabled": true,
      "planSchemeName": "Scheduled B",
      "dailyScheduleInfo": {
        "startTime": "2026-07-06",
        "stopTime": "2026-07-13",
        "dailyScheduleList": [
          {
            "beginTime": "03:21:00+08:00",
            "endTime": "04:39:00+08:00",
            "playNowTime": "",
            "playMode": "order",
            "operation": {
              "audioSource": "customAudio",
              "customAudioID": [1],
              "audioLevel": 5,
              "audioVolume": 100
            }
          }
        ]
      },
      "audioOutID": [1]
    }
  ],
  "terminalInfoList": [
    {
      "terminalID": 1,
      "audioOutID": [1]
    }
  ]
}
```

## Kesalahan yang Ditemukan

-   `dailyscheduleInfo` -\> harus `dailyScheduleInfo`.
-   Object ketiga tidak memiliki `planSchemeName`.
-   Jangan gunakan format waktu `03:21:00 08:00` kecuali memang
    diwajibkan dokumentasi firmware.
-   Hindari format `2026-07-06 08:00` untuk `startTime`/`stopTime`
    kecuali capability perangkat mengharuskannya.

## Format Waktu

Belum dapat dipastikan tanpa endpoint capability.

Kemungkinan format: 1. `03:21:00+08:00` (ISO 8601, paling umum) 2.
`03:21:00` 3. `03:21:00 08:00` (hanya jika dokumentasi firmware
menyatakannya)

Verifikasi melalui:
`GET /ISAPI/VideoIntercom/broadcast/PlanScheme/capabilities?format=json`

## Audit Implementasi Go

### Sudah Baik

-   Menggunakan HTTP Digest Authentication.
-   `resp.Body` selalu dibaca hingga selesai.
-   `resp.Body.Close()` selalu dipanggil.
-   JSON dikirim dengan `application/json`.

### Perlu Diperbaiki

1.  Reuse `http.Client` dan Digest Transport, jangan membuat Digest
    client baru setiap request.
2.  Tambahkan retry hanya untuk timeout, HTTP 500, HTTP 503, atau device
    busy.
3.  Serialkan operasi konfigurasi:
    -   Search
    -   Delete (jika perlu)
    -   Add
    -   Verify
4.  Verifikasi hasil dengan `SearchPlanScheme` setelah `AddPlanScheme`.
5.  Log lengkap:
    -   URL
    -   Payload
    -   HTTP Status
    -   Response Body

## Dugaan Penyebab "Kadang Berhasil, Kadang Gagal"

-   Device masih memproses konfigurasi.
-   Request konfigurasi dikirim terlalu cepat atau paralel.
-   Firmware membutuhkan waktu untuk commit perubahan.
-   Response JSON sebenarnya gagal walaupun HTTP 200.
-   Format payload tidak konsisten.

## Rekomendasi Prioritas

1.  Perbaiki payload.
2.  Reuse Digest client/transport.
3.  Tambahkan retry yang selektif.
4.  Jangan kirim konfigurasi secara paralel.
5.  Verifikasi hasil setelah POST.
6.  Simpan log request dan response lengkap untuk analisis.
