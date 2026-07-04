package repositories

import (
	"ego/database"
	"ego/models"
)

// GetAllSchedules retrieves all broadcast schedules
func GetAllSchedules() ([]models.BroadcastSchedule, error) {
	query := `SELECT id, name, audio_id, device_id, schedule_type, begin_time, end_time, volume, enabled, day_of_week, specific_date, created_at, updated_at 
              FROM broadcast_schedules ORDER BY id DESC`
	rows, err := database.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []models.BroadcastSchedule
	for rows.Next() {
		var s models.BroadcastSchedule
		err := rows.Scan(&s.ID, &s.Name, &s.AudioID, &s.DeviceID, &s.ScheduleType, &s.BeginTime, &s.EndTime, &s.Volume, &s.Enabled, &s.DayOfWeek, &s.SpecificDate, &s.CreatedAt, &s.UpdatedAt)
		if err != nil {
			return nil, err
		}
		schedules = append(schedules, s)
	}
	return schedules, nil
}

// GetScheduleByID retrieves a schedule by ID
func GetScheduleByID(id int) (*models.BroadcastSchedule, error) {
	s := &models.BroadcastSchedule{}
	query := `SELECT id, name, audio_id, device_id, schedule_type, begin_time, end_time, volume, enabled, day_of_week, specific_date, created_at, updated_at 
              FROM broadcast_schedules WHERE id = $1`
	err := database.DB.QueryRow(query, id).Scan(&s.ID, &s.Name, &s.AudioID, &s.DeviceID, &s.ScheduleType, &s.BeginTime, &s.EndTime, &s.Volume, &s.Enabled, &s.DayOfWeek, &s.SpecificDate, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// CreateSchedule inserts a new schedule
func CreateSchedule(s *models.BroadcastSchedule) (int, error) {
	var id int
	query := `INSERT INTO broadcast_schedules (name, audio_id, device_id, schedule_type, begin_time, end_time, volume, enabled, day_of_week, specific_date) 
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id`
	err := database.DB.QueryRow(query, s.Name, s.AudioID, s.DeviceID, s.ScheduleType, s.BeginTime, s.EndTime, s.Volume, s.Enabled, s.DayOfWeek, s.SpecificDate).Scan(&id)
	return id, err
}

// UpdateSchedule updates an existing schedule
func UpdateSchedule(s *models.BroadcastSchedule) error {
	query := `UPDATE broadcast_schedules SET name=$1, audio_id=$2, device_id=$3, schedule_type=$4, begin_time=$5, end_time=$6, volume=$7, enabled=$8, day_of_week=$9, specific_date=$10, updated_at=NOW() WHERE id=$11`
	_, err := database.DB.Exec(query, s.Name, s.AudioID, s.DeviceID, s.ScheduleType, s.BeginTime, s.EndTime, s.Volume, s.Enabled, s.DayOfWeek, s.SpecificDate, s.ID)
	return err
}

// DeleteSchedule deletes a schedule by ID
func DeleteSchedule(id int) error {
	_, err := database.DB.Exec("DELETE FROM broadcast_schedules WHERE id = $1", id)
	return err
}
