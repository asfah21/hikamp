# Hikvision ISAPI Integration

## Authentication

All Hikvision communication MUST use HTTP Digest Authentication.

Example:

GET

/ISAPI/System/deviceInfo

Never use Basic Authentication.

Create a reusable Digest Client.

All Hikvision requests must go through:

internal/hikvision/client.go

---

## Verified Endpoint

### Device Information

Method

GET

Endpoint

/ISAPI/System/deviceInfo

Purpose

Read device information.

Verified working on:

DS-QAE1A80G1-VB
Firmware V1.1.0 build 240416

Expected XML response:

<DeviceInfo>
...
</DeviceInfo>

---

### Add Broadcast Schedule

Method

POST

Endpoint

/ISAPI/VideoIntercom/broadcast/AddPlanScheme?format=json

Content-Type

application/json

Purpose

Create one or more broadcast schedules.

Verified from official Hikvision Web UI Network request.

---

## Payload Example (from official Hikvision Web UI)

**IMPORTANT:** Hikvision Web UI uses NON-standard JSON field naming. Always verify against actual Web UI network traffic.

```json
{
  "broadcastPlanSchemeList": [
    {
      "planSchemeID": "Adzan Dzuhur",
      "enabled": true,

      "dailyscheduleInfo": {
        "startTime": "2026-07-01 08:00",
        "stopTime": "2026-12-31 08:00",

        "dailyScheduleList": [
          {
            "beginTime": "12:02:00 08:00",
            "endTime": "12:05:51 08:00",

            "playMode": "order",

            "operation": {
              "audioSource": "customAudio",
              "customAudioID": [1],
              "audioLevel": 5,
              "audioVolume": 20
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

### Key differences from standard camelCase:
1. **`"dailyscheduleInfo"`** — lowercase 's', NOT `"dailyScheduleInfo"`
2. **No `"planSchemeName"`** — Web UI does NOT send this field
3. **`startTime`/`stopTime`** format: `"YYYY-MM-DD HH:MM"` (with time component)
4. **`beginTime`/`endTime`** format: `"HH:MM:SS HH:MM"` (space separator, NOT `+`)

---

## Weekly Schedule Example

weeklyScheduleInfo

Contains:

startTime

stopTime

weeklyScheduleList

Example:

{
  "dayOfWeek":1,
  "scheduleList":[
    {
      "beginTime":"12:02:00+08:00",
      "endTime":"12:05:51+08:00",

      "playMode":"order",

      "operation":{
        "audioSource":"customAudio",
        "customAudioID":[1],
        "audioLevel":5,
        "audioVolume":20
      }
    }
  ]
}

dayOfWeek

1 = Monday

2 = Tuesday

3 = Wednesday

4 = Thursday

5 = Friday

6 = Saturday

7 = Sunday

---

## Audio Information

audioSource

customAudio

audioLevel

5

audioVolume

0-100

customAudioID

Array of uploaded audio IDs.

Example

[1]

---

## Terminal Information

terminalInfoList

Example

[
  {
    "terminalID":1,
    "audioOutID":[1]
  }
]

Support multiple terminals.

---

## Client Interface

Every Hikvision communication must use these methods.

DeviceInfo()

SearchSchedule()

CreateSchedule()

UpdateSchedule()

DeleteSchedule()

SearchAudio()

UploadAudio()

DeleteAudio()

BroadcastNow()

StopBroadcast()

TestConnection()

---

## Broadcast Now (Immediate Broadcast)

BroadcastNow is implemented using the verified AddPlanScheme endpoint.

It creates a temporary schedule that starts immediately and ends in 5 minutes.

This approach is used because the dedicated real-time broadcast endpoint is not yet verified.

Implementation details:

- Method: POST (via CreateSchedule)
- Endpoint: /ISAPI/VideoIntercom/broadcast/AddPlanScheme?format=json
- Payload: Same as AddPlanScheme but with beginTime = now, endTime = now + 5 minutes
- planSchemeID: Uses timestamp suffix (broadcast_now_<unix>) to avoid conflicts

---

## Unknown Endpoints

The following endpoints are NOT verified.

SearchPlanScheme

ModifyPlanScheme

DeletePlanScheme

UploadAudio

SearchAudio

StopBroadcast

When implementing these methods:

1. First inspect Hikvision Web UI Network requests.

2. Reuse discovered endpoint.

3. Never guess endpoint URLs.

4. Every new endpoint must be documented inside this project.

---

## Development Rule

The browser Developer Tools (F12 > Network) is considered the source of truth for Hikvision-specific API endpoints.

If an endpoint is not documented publicly but is observed in the official Web UI, implement it exactly as captured.

Do not invent ISAPI paths.