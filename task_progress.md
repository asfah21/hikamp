# Task Progress

- [x] Analyze the failed payload vs Hikvision Web UI working payload
- [x] Identify differences between our code and Web UI format
- [ ] Fix `dailyScheduleInfo` → `dailyscheduleInfo` (lowercase 's')
- [ ] Fix `startTime`/`stopTime` format to include time component
- [ ] Fix `beginTime`/`endTime` format to use space instead of `+`
- [ ] Remove `planSchemeName` from payload (Web UI doesn't send it)
- [ ] Fix `BroadcastNow` payload to match Web UI format
- [ ] Fix `DeletePlanScheme` payload to match Web UI format
- [ ] Rebuild and test
