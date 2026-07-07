package services

import (
	"math"
	"time"
)

// PrayerTimesResult holds calculated prayer times
type PrayerTimesResult struct {
	Fajr    string
	Dhuhr   string
	Asr     string
	Maghrib string
	Isha    string
}

// CalculatePrayerTimes calculates all prayer times using the praytimes.org algorithm.
// This is a well-tested, open-source algorithm used by millions worldwide.
// Reference: https://praytimes.org/manual
func CalculatePrayerTimes(date time.Time, latitude, longitude float64, timezone string) *PrayerTimesResult {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}

	year, month, day := date.In(loc).Date()

	// Get timezone offset in hours
	_, offsetSec := date.In(loc).Zone()
	tzOffset := float64(offsetSec) / 3600.0

	// Julian Day
	jd := julianDay(year, int(month), day)

	// Sun position
	dec, eqt := sunPosition(jd)

	// Dhuhr (zenith) in local time
	dhuhr := 12.0 + tzOffset - longitude/15.0 - eqt/15.0

	// Fajr: sun angle 20° below horizon (standard for Indonesia/Asia)
	fajr := dhuhr - hourAngle(dec, latitude, -20.0)/15.0

	// Sunrise: sun angle 0.833° below horizon (atmospheric refraction)
	sunrise := dhuhr - hourAngle(dec, latitude, -0.833)/15.0

	// Asr: shadow length = object height (Shafi'i standard)
	asr := dhuhr + asrHourAngle(dec, latitude, 1.0)/15.0

	// Maghrib: sunset (same as sunrise angle)
	maghrib := dhuhr + hourAngle(dec, latitude, -0.833)/15.0

	// Isha: sun angle 18° below horizon (standard for Indonesia/Asia)
	isha := dhuhr + hourAngle(dec, latitude, -18.0)/15.0

	_ = sunrise // not used in output

	return &PrayerTimesResult{
		Fajr:    normalizeTime(fajr),
		Dhuhr:   normalizeTime(dhuhr),
		Asr:     normalizeTime(asr),
		Maghrib: normalizeTime(maghrib),
		Isha:    normalizeTime(isha),
	}
}

// GeneratePrayerTimesForRange generates prayer times for a date range
func GeneratePrayerTimesForRange(latitude, longitude float64, timezone string, startDate, endDate time.Time) []struct {
	Date string
	*PrayerTimesResult
} {
	var results []struct {
		Date string
		*PrayerTimesResult
	}

	current := startDate
	for !current.After(endDate) {
		times := CalculatePrayerTimes(current, latitude, longitude, timezone)
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

// normalizeTime converts decimal hours to HH:MM string, normalized to 0-24
func normalizeTime(decimalHours float64) string {
	h := math.Mod(decimalHours, 24.0)
	if h < 0 {
		h += 24
	}

	hours := int(h)
	minutes := int(math.Round((h - float64(hours)) * 60))

	// Handle rounding overflow
	if minutes >= 60 {
		hours++
		minutes = 0
	}
	if hours >= 24 {
		hours = 0
	}

	return padInt(hours) + ":" + padInt(minutes)
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

	// Obliquity of ecliptic
	epsilon := 23.439291 - t*0.0130042
	epsilonRad := degToRad(epsilon)

	// Right ascension
	ra := math.Atan2(math.Cos(epsilonRad)*math.Sin(lambdaRad), math.Cos(lambdaRad))
	ra = radToDeg(ra)

	// Declination
	declination = math.Asin(math.Sin(epsilonRad) * math.Sin(lambdaRad))
	declination = radToDeg(declination)

	// Equation of time
	l0 := 280.46646 + t*(36000.76983+0.0003032*t)
	l0 = math.Mod(l0, 360.0)
	ra = math.Mod(ra, 360.0)

	equation = l0 - ra
	if equation > 180 {
		equation -= 360
	} else if equation < -180 {
		equation += 360
	}

	return
}

// hourAngle calculates the hour angle for a given solar altitude
func hourAngle(declination, latitude, altitude float64) float64 {
	latRad := degToRad(latitude)
	decRad := degToRad(declination)
	altRad := degToRad(altitude)

	cosHA := (math.Sin(altRad) - math.Sin(latRad)*math.Sin(decRad)) / (math.Cos(latRad) * math.Cos(decRad))

	if cosHA > 1 {
		return 0
	}
	if cosHA < -1 {
		return 180
	}

	return radToDeg(math.Acos(cosHA))
}

// asrHourAngle calculates the hour angle for Asr prayer
// factor: 1 for Shafi'i/Maliki/Hanbali, 2 for Hanafi
func asrHourAngle(declination, latitude float64, factor float64) float64 {
	latRad := degToRad(latitude)
	decRad := degToRad(declination)

	// cot(alt) = factor + tan(lat - dec)
	tanDiff := math.Tan(math.Abs(latRad - decRad))
	cotAlt := factor + tanDiff
	altRad := math.Atan(1.0 / cotAlt)

	return radToDeg(math.Acos(
		(math.Sin(altRad) - math.Sin(latRad)*math.Sin(decRad)) / (math.Cos(latRad) * math.Cos(decRad)),
	))
}

// degToRad converts degrees to radians
func degToRad(deg float64) float64 {
	return deg * math.Pi / 180.0
}

// radToDeg converts radians to degrees
func radToDeg(rad float64) float64 {
	return rad * 180.0 / math.Pi
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
