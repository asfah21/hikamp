package repositories

import (
	"ego/database"
	"ego/models"
)

// GetAllLogs retrieves all broadcast logs
func GetAllLogs() ([]models.BroadcastLog, error) {
	query := `SELECT id, time, device_id, device_name, audio_id, audio_name, result, duration, status, error_message, created_at 
              FROM broadcast_logs ORDER BY id DESC`
	rows, err := database.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.BroadcastLog
	for rows.Next() {
		var l models.BroadcastLog
		err := rows.Scan(&l.ID, &l.Time, &l.DeviceID, &l.DeviceName, &l.AudioID, &l.AudioName, &l.Result, &l.Duration, &l.Status, &l.ErrorMessage, &l.CreatedAt)
		if err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, nil
}

// CreateLog inserts a new broadcast log
func CreateLog(l *models.BroadcastLog) (int, error) {
	var id int
	query := `INSERT INTO broadcast_logs (time, device_id, device_name, audio_id, audio_name, result, duration, status, error_message) 
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`
	err := database.DB.QueryRow(query, l.Time, l.DeviceID, l.DeviceName, l.AudioID, l.AudioName, l.Result, l.Duration, l.Status, l.ErrorMessage).Scan(&id)
	return id, err
}

// GetTodayLogs retrieves today's broadcast logs
func GetTodayLogs() ([]models.BroadcastLog, error) {
	query := `SELECT id, time, device_id, device_name, audio_id, audio_name, result, duration, status, error_message, created_at 
              FROM broadcast_logs WHERE DATE(created_at) = CURRENT_DATE ORDER BY id DESC`
	rows, err := database.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.BroadcastLog
	for rows.Next() {
		var l models.BroadcastLog
		err := rows.Scan(&l.ID, &l.Time, &l.DeviceID, &l.DeviceName, &l.AudioID, &l.AudioName, &l.Result, &l.Duration, &l.Status, &l.ErrorMessage, &l.CreatedAt)
		if err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, nil
}

// CleanupOldLogs deletes logs older than specified days
func CleanupOldLogs(days int) error {
	_, err := database.DB.Exec("DELETE FROM broadcast_logs WHERE created_at < NOW() - INTERVAL '1 day' * $1", days)
	return err
}
