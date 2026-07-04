package repositories

import (
	"ego/database"
	"ego/models"
)

// GetPrayerLocation retrieves the prayer location
func GetPrayerLocation() (*models.PrayerLocation, error) {
	p := &models.PrayerLocation{}
	query := `SELECT id, latitude, longitude, timezone, method, created_at, updated_at FROM prayer_locations LIMIT 1`
	err := database.DB.QueryRow(query).Scan(&p.ID, &p.Latitude, &p.Longitude, &p.Timezone, &p.Method, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// SavePrayerLocation saves or updates the prayer location
func SavePrayerLocation(p *models.PrayerLocation) error {
	// Check if exists
	var count int
	database.DB.QueryRow("SELECT COUNT(*) FROM prayer_locations").Scan(&count)
	if count > 0 {
		_, err := database.DB.Exec(`UPDATE prayer_locations SET latitude=$1, longitude=$2, timezone=$3, method=$4, updated_at=NOW() WHERE id=(SELECT id FROM prayer_locations LIMIT 1)`, p.Latitude, p.Longitude, p.Timezone, p.Method)
		return err
	}
	_, err := database.DB.Exec(`INSERT INTO prayer_locations (latitude, longitude, timezone, method) VALUES ($1, $2, $3, $4)`, p.Latitude, p.Longitude, p.Timezone, p.Method)
	return err
}

// GetPrayerTimes retrieves prayer times for a date range
func GetPrayerTimes(startDate, endDate string) ([]models.PrayerTime, error) {
	query := `SELECT id, date, fajr, dhuhr, asr, maghrib, isha, location_id, created_at 
              FROM prayer_times WHERE date >= $1 AND date <= $2 ORDER BY date ASC`
	rows, err := database.DB.Query(query, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var times []models.PrayerTime
	for rows.Next() {
		var t models.PrayerTime
		err := rows.Scan(&t.ID, &t.Date, &t.Fajr, &t.Dhuhr, &t.Asr, &t.Maghrib, &t.Isha, &t.LocationID, &t.CreatedAt)
		if err != nil {
			return nil, err
		}
		times = append(times, t)
	}
	return times, nil
}

// SavePrayerTime inserts a prayer time
func SavePrayerTime(t *models.PrayerTime) error {
	_, err := database.DB.Exec(`INSERT INTO prayer_times (date, fajr, dhuhr, asr, maghrib, isha, location_id) 
		VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT (date, location_id) DO UPDATE 
		SET fajr=$2, dhuhr=$3, asr=$4, maghrib=$5, isha=$6`,
		t.Date, t.Fajr, t.Dhuhr, t.Asr, t.Maghrib, t.Isha, t.LocationID)
	return err
}
