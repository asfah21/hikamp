package repositories

import (
	"ego/database"
	"ego/models"
)

// GetAllSchedules retrieves all broadcast schedules with their entries and devices
func GetAllSchedules() ([]models.BroadcastSchedule, error) {
	query := `SELECT id, name, schedule_type, enabled, day_of_week, specific_date, created_at, updated_at 
              FROM broadcast_schedules ORDER BY id DESC`
	rows, err := database.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []models.BroadcastSchedule
	for rows.Next() {
		var s models.BroadcastSchedule
		err := rows.Scan(&s.ID, &s.Name, &s.ScheduleType, &s.Enabled, &s.DayOfWeek, &s.SpecificDate, &s.CreatedAt, &s.UpdatedAt)
		if err != nil {
			return nil, err
		}
		schedules = append(schedules, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Load entries and devices for each schedule
	for i := range schedules {
		entries, err := GetEntriesBySchedule(schedules[i].ID)
		if err != nil {
			return nil, err
		}
		schedules[i].Entries = entries

		devices, err := GetDevicesBySchedule(schedules[i].ID)
		if err != nil {
			return nil, err
		}
		schedules[i].Devices = devices
	}

	return schedules, nil
}

// GetScheduleByID retrieves a schedule by ID with entries and devices
func GetScheduleByID(id int) (*models.BroadcastSchedule, error) {
	s := &models.BroadcastSchedule{}
	query := `SELECT id, name, schedule_type, enabled, day_of_week, specific_date, created_at, updated_at 
              FROM broadcast_schedules WHERE id = $1`
	err := database.DB.QueryRow(query, id).Scan(&s.ID, &s.Name, &s.ScheduleType, &s.Enabled, &s.DayOfWeek, &s.SpecificDate, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}

	entries, err := GetEntriesBySchedule(s.ID)
	if err != nil {
		return nil, err
	}
	s.Entries = entries

	devices, err := GetDevicesBySchedule(s.ID)
	if err != nil {
		return nil, err
	}
	s.Devices = devices

	return s, nil
}

// GetEntriesBySchedule retrieves all entries for a schedule
func GetEntriesBySchedule(scheduleID int) ([]models.ScheduleEntry, error) {
	query := `SELECT id, schedule_id, audio_id, begin_time, end_time, volume
	          FROM schedule_entries WHERE schedule_id = $1 ORDER BY id`
	rows, err := database.DB.Query(query, scheduleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.ScheduleEntry
	for rows.Next() {
		var e models.ScheduleEntry
		err := rows.Scan(&e.ID, &e.ScheduleID, &e.AudioID, &e.BeginTime, &e.EndTime, &e.Volume)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// GetDevicesBySchedule retrieves all device targets for a schedule
func GetDevicesBySchedule(scheduleID int) ([]models.ScheduleDevice, error) {
	query := `SELECT id, schedule_id, device_id
	          FROM schedule_devices WHERE schedule_id = $1 ORDER BY id`
	rows, err := database.DB.Query(query, scheduleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []models.ScheduleDevice
	for rows.Next() {
		var d models.ScheduleDevice
		err := rows.Scan(&d.ID, &d.ScheduleID, &d.DeviceID)
		if err != nil {
			return nil, err
		}
		devices = append(devices, d)
	}
	return devices, rows.Err()
}

// CreateSchedule inserts a new schedule and its entries + devices
func CreateSchedule(s *models.BroadcastSchedule) (int, error) {
	tx, err := database.DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var id int
	query := `INSERT INTO broadcast_schedules (name, schedule_type, enabled, day_of_week, specific_date) 
              VALUES ($1, $2, $3, $4, $5) RETURNING id`
	err = tx.QueryRow(query, s.Name, s.ScheduleType, s.Enabled, s.DayOfWeek, s.SpecificDate).Scan(&id)
	if err != nil {
		return 0, err
	}

	// Insert entries
	for _, entry := range s.Entries {
		_, err = tx.Exec(`INSERT INTO schedule_entries (schedule_id, audio_id, begin_time, end_time, volume) 
		                  VALUES ($1, $2, $3, $4, $5)`, id, entry.AudioID, entry.BeginTime, entry.EndTime, entry.Volume)
		if err != nil {
			return 0, err
		}
	}

	// Insert devices
	for _, dev := range s.Devices {
		_, err = tx.Exec(`INSERT INTO schedule_devices (schedule_id, device_id) VALUES ($1, $2)`, id, dev.DeviceID)
		if err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return id, nil
}

// UpdateSchedule updates an existing schedule and its entries + devices
func UpdateSchedule(s *models.BroadcastSchedule) error {
	tx, err := database.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `UPDATE broadcast_schedules SET name=$1, schedule_type=$2, enabled=$3, day_of_week=$4, specific_date=$5, updated_at=NOW() WHERE id=$6`
	_, err = tx.Exec(query, s.Name, s.ScheduleType, s.Enabled, s.DayOfWeek, s.SpecificDate, s.ID)
	if err != nil {
		return err
	}

	// Replace entries: delete old, insert new
	_, err = tx.Exec(`DELETE FROM schedule_entries WHERE schedule_id = $1`, s.ID)
	if err != nil {
		return err
	}
	for _, entry := range s.Entries {
		_, err = tx.Exec(`INSERT INTO schedule_entries (schedule_id, audio_id, begin_time, end_time, volume) 
		                  VALUES ($1, $2, $3, $4, $5)`, s.ID, entry.AudioID, entry.BeginTime, entry.EndTime, entry.Volume)
		if err != nil {
			return err
		}
	}

	// Replace devices: delete old, insert new
	_, err = tx.Exec(`DELETE FROM schedule_devices WHERE schedule_id = $1`, s.ID)
	if err != nil {
		return err
	}
	for _, dev := range s.Devices {
		_, err = tx.Exec(`INSERT INTO schedule_devices (schedule_id, device_id) VALUES ($1, $2)`, s.ID, dev.DeviceID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// DeleteSchedule deletes a schedule by ID (cascades to entries and devices)
func DeleteSchedule(id int) error {
	_, err := database.DB.Exec("DELETE FROM broadcast_schedules WHERE id = $1", id)
	return err
}

// DeleteSchedulesByDevice deletes all schedules that reference a specific device
func DeleteSchedulesByDevice(deviceID int) error {
	// First get all schedule IDs that reference this device
	rows, err := database.DB.Query("SELECT schedule_id FROM schedule_devices WHERE device_id = $1", deviceID)
	if err != nil {
		return err
	}
	defer rows.Close()

	var scheduleIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return err
		}
		scheduleIDs = append(scheduleIDs, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// Delete all those schedules (cascade handles entries and devices)
	for _, sid := range scheduleIDs {
		if err := DeleteSchedule(sid); err != nil {
			return err
		}
	}

	return nil
}
