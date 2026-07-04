package repositories

import (
	"ego/database"
	"ego/models"
)

// GetAllSettings retrieves all settings
func GetAllSettings() ([]models.Setting, error) {
	rows, err := database.DB.Query("SELECT id, key, value, description, updated_at FROM settings ORDER BY id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var settings []models.Setting
	for rows.Next() {
		var s models.Setting
		err := rows.Scan(&s.ID, &s.Key, &s.Value, &s.Description, &s.UpdatedAt)
		if err != nil {
			return nil, err
		}
		settings = append(settings, s)
	}
	return settings, nil
}

// GetSetting retrieves a setting by key
func GetSetting(key string) (*models.Setting, error) {
	s := &models.Setting{}
	err := database.DB.QueryRow("SELECT id, key, value, description, updated_at FROM settings WHERE key = $1", key).Scan(&s.ID, &s.Key, &s.Value, &s.Description, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// SetSetting updates or inserts a setting
func SetSetting(key, value, description string) error {
	var count int
	database.DB.QueryRow("SELECT COUNT(*) FROM settings WHERE key = $1", key).Scan(&count)
	if count > 0 {
		_, err := database.DB.Exec("UPDATE settings SET value=$1, description=$2, updated_at=NOW() WHERE key=$3", value, description, key)
		return err
	}
	_, err := database.DB.Exec("INSERT INTO settings (key, value, description) VALUES ($1, $2, $3)", key, value, description)
	return err
}
