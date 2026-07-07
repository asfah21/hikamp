# Ringkasan Audit ISAPI Hikvision - AddPlanScheme

Berdasarkan analisis JS Web UI Hikvision (`12_chunk.ca9cfad0989aa5a9c826.js`).

## Payload yang Benar (dari Web UI)

### DailySchedule
```json
{
  "broadcastPlanSchemeList": [
    {
      "planSchemeID": "Scheduled B",
      "enabled": true,
      "planSchemeName": "Scheduled B",
      "dailyScheduleInfo": {
        "startTime": "2026-07-06+08:00",
        "stopTime": "2026-07-13+08:00",
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

### WeeklySchedule
```json
{
  "broadcastPlanSchemeList": [
    {
      "planSchemeID": "Scheduled B",
      "enabled": true,
      "planSchemeName": "Scheduled B",
      "weklyScheduleInfo": {
        "startTime": "2026-07-06+08:00",
        "stopTime": "2026-07-13+08:00",
        "weeklyScheduleList": [
          {
            "dayOfWeek": 1,
            "scheduleList": [
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

## Format Waktu (Dari Web UI JS)

### beginTime / endTime
- Format: `"HH:MM:SS+08:00"` (PLUS separator)
- Contoh: `"03:21:00+08:00"`
- Sumber: `12_chunk.js` line: `beginTime: "".concat(i.a.changetime(t.from),"+08:00")`

### startTime / stopTime
- Format: `"YYYY-MM-DD+08:00"` (PLUS separator)
- Contoh: `"2026-07-06+08:00"`
- Sumber: `12_chunk.js` line: `B[k[v]].startTime = "".concat(y,"+08:00")`

### playNowTime
- Format: `"HH:MM:SS+08:00"`
- Digunakan untuk "effective immediately" ketika schedule aktif dan waktu sekarang berada dalam range schedule
- Diisi dengan waktu sekarang + 1 menit

## Nama Field (Dari Web UI JS)

### DailySchedule
- `dailyScheduleInfo` — untuk DailySchedule (capital S)
- `dailyscheduleInfo` — juga digunakan di beberapa tempat (lowercase s), konsisten dengan Web UI

### WeeklySchedule
- `weklyScheduleInfo` — **PERHATIAN: typo "wekly" bukan "weekly"!** Ini dari Web UI asli.
- `weeklyScheduleList` — daftar schedule per hari

## Endpoint

### AddPlanScheme (POST)
- URL: `/ISAPI/VideoIntercom/broadcast/AddPlanScheme?format=json`
- Digunakan untuk: **Add, Update, dan Delete** semua schedule sekaligus
- Cara kerja: Kirim SELURUH array `broadcastPlanSchemeList` — device akan replace semua schedule dengan yang dikirim
- Untuk delete: filter array, kirim sisanya
- Untuk update: modify item di array, kirim seluruh array

### SearchPlanScheme (POST)
- URL: `/ISAPI/VideoIntercom/broadcast/SearchPlanScheme?format=json`
- Method: POST (bukan GET!)
- Payload: `{"terminalInfoList":[{"terminalID":1}],"planSchemeID":["1"]}`
- Sumber: `1186.js` line: `o.a.WSDK_SetDeviceConfig("SearchPlanScheme",null,{type:"POST",data:t,...})`

### ModifyPlanScheme
- Tidak digunakan oleh Web UI untuk operasi normal
- Web UI hanya menggunakan AddPlanScheme untuk semua operasi

## Cara Kerja Save (Dari Web UI JS)

1. Ambil semua schedule yang ada dari device via SearchPlanScheme
2. Modifikasi array schedule (add/update/delete item)
3. Kirim seluruh array ke AddPlanScheme
4. Device akan replace semua schedule dengan yang baru

Ini berarti **AddPlanScheme bersifat destructive** — akan menghapus semua schedule yang tidak ada di payload.

## Implikasi untuk Implementasi Go

1. **BroadcastNow**: Harusnya menggunakan `AddPlanScheme` dengan hanya 1 schedule (broadcast_now), tapi ini akan menghapus schedule lain. Solusi: 
   - Sebelum broadcast, simpan schedule yang ada
   - Setelah broadcast selesai, restore schedule yang ada
   - Atau: gunakan `playNowTime` untuk broadcast immediate tanpa menghapus schedule lain

2. **SyncScheduleToDevice**: Harus kirim SEMUA schedule, bukan hanya 1. Tapi karena kita hanya punya schedule lokal, kita perlu:
   - Baca semua schedule dari DB
   - Kirim semua ke device via AddPlanScheme
   - Ini akan replace semua schedule di device

3. **SearchPlanScheme**: Menggunakan POST, bukan GET!

## Catatan Penting

- `planSchemeName` harus ada (Web UI selalu mengirimnya)
- `audioOutID` adalah array, bisa [1] atau [1,2] untuk multiple channel
- `terminalInfoList` harus selalu disertakan
- `playNowTime` digunakan untuk immediate playback
- `audioVolume: -1` berarti menggunakan volume terminal
