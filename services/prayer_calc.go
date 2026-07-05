package services

import (
	"math"
	"time"
)

// Prayer calculation methods
const (
	MethodMWL     = 3  // Muslim World League (MWL)
	MethodMakkah  = 4  // Umm Al-Qura University, Makkah
	MethodKemenag = 99 // Kemenag RI (Fajr 20°, Isha 18°)
)

// MethodConfig holds calculation parameters for each method
type MethodConfig struct {
	FajrAngle     float64
	IshaAngle     float64
	IshaMethod    string // "angle" or "fixed"
	IshaFixed     string // fixed time for Isha (e.g., "90 min")
	MaghribMethod string // "angle" or "fixed"
	MaghribFixed  string // fixed time for Maghrib
}

var methodConfigs = map[int]MethodConfig{
	MethodMWL:     {FajrAngle: 18, IshaAngle: 17, IshaMethod: "angle"},
	MethodMakkah:  {FajrAngle: 18.5, IshaAngle: 0, IshaMethod: "fixed", IshaFixed: "90 min"},
	MethodKemenag: {FajrAngle: 20, IshaAngle: 18, IshaMethod: "angle"},
}

// PrayerTimesResult holds calculated prayer times
type PrayerTimesResult struct {
	Fajr    string
	Dhuhr   string
	Asr     string
	Maghrib string
	Isha    string
}

// CalculatePrayerTimes calculates all prayer times for a given date and location
func CalculatePrayerTimes(date time.Time, latitude, longitude float64, timezone string, method int) *PrayerTimesResult {
	// Get timezone offset
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}

	// Use the date at the given timezone
	year, month, day := date.In(loc).Date()

	// Calculate Julian Day
	jd := julianDay(year, int(month), day)

	// Calculate sun position
	declination, equation := sunPosition(jd)

	// Calculate Dhuhr (zenith)
	dhuhr := 12.0 + equation - longitude/15.0

	// Get method config
	config := methodConfigs[method]
	if config.FajrAngle == 0 {
		config = methodConfigs[MethodMWL] // default to MWL
	}

	// Calculate prayer times in degrees
	fajrDeg := dhuhr - hourAngle(declination, latitude, config.FajrAngle+90)/15.0
	sunriseDeg := dhuhr - hourAngle(declination, latitude, 90.833)/15.0 // 0.833 for atmospheric refraction
	dhuhrDeg := dhuhr
	asrDeg := dhuhr + hourAngle(declination, latitude, asrAngle(declination, latitude, false))/15.0
	asrHanafiDeg := dhuhr + hourAngle(declination, latitude, asrAngle(declination, latitude, true))/15.0
	maghribDeg := dhuhr + hourAngle(declination, latitude, 90.833)/15.0

	var ishaDeg float64
	if config.IshaMethod == "fixed" {
		// Fixed Isha (e.g., 90 minutes after Maghrib)
		ishaDeg = maghribDeg + 1.5 // 90 minutes = 1.5 hours
	} else {
		ishaDeg = dhuhr + hourAngle(declination, latitude, config.IshaAngle+90)/15.0
	}

	// Convert to time strings
	fajrStr := degreesToTime(fajrDeg)
	sunriseStr := degreesToTime(sunriseDeg)
	dhuhrStr := degreesToTime(dhuhrDeg)
	asrStr := degreesToTime(asrDeg)
	asrHanafiStr := degreesToTime(asrHanafiDeg)
	maghribStr := degreesToTime(maghribDeg)
	ishaStr := degreesToTime(ishaDeg)

	_ = sunriseStr
	_ = asrHanafiStr

	return &PrayerTimesResult{
		Fajr:    fajrStr,
		Dhuhr:   dhuhrStr,
		Asr:     asrStr,
		Maghrib: maghribStr,
		Isha:    ishaStr,
	}
}

// GeneratePrayerTimesForRange generates prayer times for a date range
func GeneratePrayerTimesForRange(latitude, longitude float64, timezone string, method int, startDate, endDate time.Time) []struct {
	Date string
	*PrayerTimesResult
} {
	var results []struct {
		Date string
		*PrayerTimesResult
	}

	current := startDate
	for !current.After(endDate) {
		times := CalculatePrayerTimes(current, latitude, longitude, timezone, method)
		results = append(results, struct {
			Date string
			*PrayerTimesResult
		}{
			Date:              current.Format("2006-01-02"),
			PrayerTimesResult: times,
		})
		current = current.AddDate(0, 0, 1)
	}

	return results
}

// julianDay calculates the Julian Day number
func julianDay(year, month, day int) float64 {
	if month <= 2 {
		year--
		month += 12
	}

	a := float64(year) / 100.0
	b := 2.0 - a + float64(int(a/4.0))

	return float64(int(365.25*float64(year+4716))) + float64(int(30.6001*float64(month+1))) + float64(day) + b - 1524.5
}

// sunPosition calculates the sun's declination and equation of time
func sunPosition(jd float64) (declination, equation float64) {
	t := (jd - 2451545.0) / 36525.0

	// Mean anomaly
	m := 357.52911 + t*(35999.05029-0.0001537*t)
	m = degToRad(m)

	// Sun's equation of center
	c := (1.914602-t*(0.004817+0.000014*t))*math.Sin(m) + (0.019993-0.000101*t)*math.Sin(2*m) + 0.000289*math.Sin(3*m)

	// Sun's true longitude
	lambda := 280.46646 + t*(36000.76983+0.0003032*t)
	lambda = math.Mod(lambda, 360.0)
	lambda += c
	lambdaRad := degToRad(lambda)

	// Sun's right ascension
	epsilon := 23.439291 - t*0.0130042
	epsilonRad := degToRad(epsilon)

	ra := math.Atan2(math.Cos(epsilonRad)*math.Sin(lambdaRad), math.Cos(lambdaRad))
	ra = radToDeg(ra)

	// Declination
	declination = math.Asin(math.Sin(epsilonRad) * math.Sin(lambdaRad))
	declination = radToDeg(declination)

	// Equation of time
	// Mean longitude
	l0 := 280.46646 + t*(36000.76983+0.0003032*t)
	l0 = math.Mod(l0, 360.0)

	// Normalize RA
	ra = math.Mod(ra, 360.0)

	equation = l0 - ra
	if equation > 180 {
		equation -= 360
	} else if equation < -180 {
		equation += 360
	}

	return
}

// hourAngle calculates the hour angle for a given altitude
func hourAngle(declination, latitude, altitude float64) float64 {
	latRad := degToRad(latitude)
	decRad := degToRad(declination)
	altRad := degToRad(altitude)

	cosHA := (math.Sin(altRad) - math.Sin(latRad)*math.Sin(decRad)) / (math.Cos(latRad) * math.Cos(decRad))

	if cosHA > 1 {
		return 0 // Sun never rises
	}
	if cosHA < -1 {
		return 180 // Sun never sets
	}

	return radToDeg(math.Acos(cosHA))
}

// asrAngle calculates the shadow angle for Asr prayer
func asrAngle(declination, latitude float64, hanafi bool) float64 {
	latRad := degToRad(latitude)
	decRad := degToRad(declination)

	// Shadow length factor: 1 for Shafi'i/Maliki/Hanbali, 2 for Hanafi
	factor := 1.0
	if hanafi {
		factor = 2.0
	}

	// Calculate the altitude of the sun for Asr
	// Using: cot(alt) = factor + tan(lat - dec)
	tanDiff := math.Tan(math.Abs(latRad - decRad))
	cotAlt := factor + tanDiff

	altRad := math.Atan(1.0 / cotAlt)
	return radToDeg(altRad)
}

// degreesToTime converts decimal hours to time string (HH:MM)
func degreesToTime(decimalHours float64) string {
	// Normalize to 0-24
	hours := math.Mod(decimalHours, 24.0)
	if hours < 0 {
		hours += 24
	}

	h := int(hours)
	m := int(math.Round((hours - float64(h)) * 60))

	// Handle rounding overflow
	if m >= 60 {
		h++
		m = 0
	}
	if h >= 24 {
		h = 0
	}

	return formatTime(h, m)
}

// formatTime formats hour and minute to HH:MM string
func formatTime(h, m int) string {
	return padInt(h) + ":" + padInt(m)
}

// padInt pads an integer to 2 digits
func padInt(n int) string {
	if n < 10 {
		return "0" + intToString(n)
	}
	return intToString(n)
}

// intToString converts int to string
func intToString(n int) string {
	if n == 0 {
		return "0"
	}

	negative := false
	if n < 0 {
		negative = true
		n = -n
	}

	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}

	if negative {
		return "-" + string(digits)
	}
	return string(digits)
}

// degToRad converts degrees to radians
func degToRad(deg float64) float64 {
	return deg * math.Pi / 180.0
}

// radToDeg converts radians to degrees
func radToDeg(rad float64) float64 {
	return rad * 180.0 / math.Pi
}
