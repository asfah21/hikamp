# Hikvision ISAPI Integration

## Authentication

All Hikvision communication MUST use HTTP Digest Authentication.

Example: `GET /ISAPI/System/deviceInfo`

Never use Basic Authentication.

Create a reusable Digest Client.

All Hikvision requests must go through: `internal/hikvision/client.go`

---

## Verified Endpoint

### Device Information

- **Method**: GET
- **Endpoint**: `/ISAPI/System/deviceInfo`
- **Purpose**: Read device information
- **Verified working on**: DS-QAE1A80G1-VB, Firmware V1.1.0 build 240416
- **Response**: XML with DeviceInfo fields

---

### Add Broadcast Schedule (AddPlanScheme)

- **Method**: POST
- **Endpoint**: `/ISAPI/VideoIntercom/broadcast/AddPlanScheme?format=json`
- **Content-Type**: application/json
- **Purpose**: Create/replace broadcast schedules
- **⚠️ DESTRUCTIVE**: Replaces ALL existing schedules with the ones in the payload

---

### Modify Broadcast Schedule (ModifyPlanScheme)

- **Method**: POST
- **Endpoint**: `/ISAPI/VideoIntercom/broadcast/ModifyPlanScheme?format=json`
- **Content-Type**: application/json
- **Purpose**: Create or update a single schedule without affecting others (non-destructive)
- **⚠️ Note**: Some firmware versions return 403 "Invalid Operation"

---

### Search Plan Scheme (SearchPlanScheme)

- **Method**: POST (NOT GET!)
- **Endpoint**: `/ISAPI/VideoIntercom/broadcast/SearchPlanScheme?format=json`
- **Content-Type**: application/json
- **Payload**:
```json
{
  "terminalInfoList": [{"terminalID": 1}],
  "planSchemeID": ["1"]
}
```
- **Source**: Web UI JS: `o.a.WSDK_SetDeviceConfig("SearchPlanScheme",null,{type:"POST",data:...})`

---

## Payload Format (from official Hikvision Web UI JS)

### DailySchedule

```json
{
  "broadcastPlanSchemeList": [
    {
      "planSchemeID": "Scheduled B",
      "planSchemeName": "Scheduled B",
      "enabled": true,
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
      "planSchemeName": "Scheduled B",
      "enabled": true,
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

### Key Format Rules (from Web UI JS):

1. **`beginTime` / `endTime`**: `"HH:MM:SS+08:00"` (PLUS separator)
2. **`startTime` / `stopTime`**: `"YYYY-MM-DD+08:00"` (PLUS separator)
3. **`dailyScheduleInfo`**: Capital 'S' (NOT `dailyscheduleInfo`)
4. **`weklyScheduleInfo`**: **TYPO** "wekly" (not "weekly") — this is from the official Web UI!
5. **`planSchemeName`**: Always sent by Web UI (required field)
6. **`playNowTime`**: Used for immediate playback, format `"HH:MM:SS+08:00"`

---

## Audio Information

- **audioSource**: `"customAudio"`
- **audioLevel**: `5`
- **audioVolume**: `0-100`
- **customAudioID**: Array of uploaded audio IDs, e.g. `[1]`
- **audioVolume: -1**: Means use terminal default volume

---

## Terminal Information

```json
"terminalInfoList": [
  {
    "terminalID": 1,
    "audioOutID": [1]
  }
]
```

Support multiple terminals. `audioOutID` can be `[1]` or `[1,2]` for multiple channels.

---

## Client Interface

Every Hikvision communication must use these methods:

- `DeviceInfo()` — GET device info (XML)
- `SearchPlanScheme()` — POST search all schedules
- `CreateSchedule()` — POST AddPlanScheme (destructive)
- `ModifyPlanScheme()` — POST ModifyPlanScheme (non-destructive upsert)
- `DeletePlanScheme()` — POST ModifyPlanScheme with enabled=false
- `SearchAudio()` — GET list audio files
- `SearchAudioByID()` — GET single audio by ID
- `UploadAudio()` — POST multipart upload, returns (int, error)
- `DeleteAudio()` — (not yet implemented)
- `BroadcastNow()` — immediate broadcast (tries ModifyPlanScheme first, falls back to merge+AddPlanScheme)
- `StopBroadcast()` — SearchPlanScheme + disable each via ModifyPlanScheme
- `TestConnection()` — calls DeviceInfo()

---

## Audio Architecture (All from Hikvision)

Audio files are stored **only on the Hikvision device**, not on the local server.

### Flow:
1. **Upload**: User uploads MP3 → directly to Hikvision device → device assigns `customAudioID` → saved to local DB with `hikvision_audio_id`
2. **Sync**: User clicks "Sync from Device" → fetches all audio from Hikvision → upserts into local DB
3. **Broadcast/Schedule**: Uses `hikvision_audio_id` as `customAudioID` in payload

### Database fields:
- `hikvision_audio_id` (INTEGER UNIQUE) — the `customAudioID` from Hikvision device
- `hikvision_path` (VARCHAR) — path on device (e.g., `/emmc/config/Media/file.mp3`)
- No local file storage — `file_path` field removed

### Key rule:
When building broadcast payloads, `customAudioID` must be the **Hikvision device's audio ID**, not the local database ID.

---

## Broadcast Now (Immediate Broadcast)

BroadcastNow uses a dual-strategy approach:

1. **Strategy 1**: Try `ModifyPlanScheme` first (non-destructive, preserves other schedules)
2. **Strategy 2**: If ModifyPlanScheme fails (403 on some firmware), fallback to:
   - `SearchPlanScheme` → merge existing schedules + broadcast_now → `AddPlanScheme`
   - This preserves existing schedules while adding the broadcast_now entry

- `beginTime` = now + 2 seconds (to account for network/processing delay)
- `planSchemeID` = `broadcast_now_<unix_timestamp>`

---

## Upload Audio

- **Method**: POST
- **Endpoint**: `/ISAPI/AccessControl/EventCardLinkageCfg/CustomAudio?format=json`
- **Content-Type**: multipart/form-data
- **Format** (from Web UI):
```
------WebKitFormBoundary...
Content-Disposition: form-data; name="CustomAudioInfo"

{"CustomAudioInfo":{"customAudioName":"filename.mp3","audioFileFormat":"mp3","audioFileSize":391095}}
------WebKitFormBoundary...
Content-Disposition: form-data; name="audioData"; filename="filename.mp3"
Content-Type: audio/mpeg

<binary data>
------WebKitFormBoundary...--
```

---

## Development Rule

The browser Developer Tools (F12 > Network) is considered the source of truth for Hikvision-specific API endpoints.

If an endpoint is not documented publicly but is observed in the official Web UI, implement it exactly as captured.

Do not invent ISAPI paths.

## Audit Detail (dari JS Web UI)

Berdasarkan analisis JS Web UI Hikvision (`12_chunk.ca9cfad0989aa5a9c826.js`).

### Format Waktu (Dari Web UI JS)

#### beginTime / endTime
- Format: `"HH:MM:SS+08:00"` (PLUS separator)
- Contoh: `"03:21:00+08:00"`
- Sumber: `12_chunk.js` line: `beginTime: "".concat(i.a.changetime(t.from),"+08:00")`

#### startTime / stopTime
- Format: `"YYYY-MM-DD+08:00"` (PLUS separator)
- Contoh: `"2026-07-06+08:00"`
- Sumber: `12_chunk.js` line: `B[k[v]].startTime = "".concat(y,"+08:00")`

#### playNowTime
- Format: `"HH:MM:SS+08:00"`
- Digunakan untuk "effective immediately" ketika schedule aktif dan waktu sekarang berada dalam range schedule
- Diisi dengan waktu sekarang + 1 menit

### Nama Field (Dari Web UI JS)

#### DailySchedule
- `dailyScheduleInfo` — untuk DailySchedule (capital S)
- `dailyscheduleInfo` — juga digunakan di beberapa tempat (lowercase s), konsisten dengan Web UI

#### WeeklySchedule
- `weklyScheduleInfo` — **PERHATIAN: typo "wekly" bukan "weekly"!** Ini dari Web UI asli.
- `weeklyScheduleList` — daftar schedule per hari

### Cara Kerja Save (Dari Web UI JS)

1. Ambil semua schedule yang ada dari device via SearchPlanScheme
2. Modifikasi array schedule (add/update/delete item)
3. Kirim seluruh array ke AddPlanScheme
4. Device akan replace semua schedule dengan yang baru

Ini berarti **AddPlanScheme bersifat destructive** — akan menghapus semua schedule yang tidak ada di payload.

### Catatan Penting

- `planSchemeName` harus ada (Web UI selalu mengirimnya)
- `audioOutID` adalah array, bisa [1] atau [1,2] untuk multiple channel
- `terminalInfoList` harus selalu disertakan
- `playNowTime` digunakan untuk immediate playback
- `audioVolume: -1` berarti menggunakan volume terminal
