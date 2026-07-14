package repositories

import (
	"ego/database"
	"ego/models"
)

// GetAllDevices retrieves all devices
func GetAllDevices() ([]models.Device, error) {
	query := `SELECT id, name, ip_address, port, username, password, location, status, firmware, last_sync, enabled, created_at, updated_at 
              FROM devices ORDER BY id DESC`
	rows, err := database.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []models.Device
	for rows.Next() {
		var d models.Device
		err := rows.Scan(&d.ID, &d.Name, &d.IPAddress, &d.Port, &d.Username, &d.Password, &d.Location, &d.Status, &d.Firmware, &d.LastSync, &d.Enabled, &d.CreatedAt, &d.UpdatedAt)
		if err != nil {
			return nil, err
		}
		devices = append(devices, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return devices, nil
}

// GetDeviceByID retrieves a device by ID
func GetDeviceByID(id int) (*models.Device, error) {
	d := &models.Device{}
	query := `SELECT id, name, ip_address, port, username, password, location, status, firmware, last_sync, enabled, created_at, updated_at 
              FROM devices WHERE id = $1`
	err := database.DB.QueryRow(query, id).Scan(&d.ID, &d.Name, &d.IPAddress, &d.Port, &d.Username, &d.Password, &d.Location, &d.Status, &d.Firmware, &d.LastSync, &d.Enabled, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return d, nil
}

// CreateDevice inserts a new device
func CreateDevice(d *models.Device) (int, error) {
	var id int
	query := `INSERT INTO devices (name, ip_address, port, username, password, location, status, firmware, enabled) 
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`
	err := database.DB.QueryRow(query, d.Name, d.IPAddress, d.Port, d.Username, d.Password, d.Location, d.Status, d.Firmware, d.Enabled).Scan(&id)
	return id, err
}

// UpdateDevice updates an existing device
func UpdateDevice(d *models.Device) error {
	query := `UPDATE devices SET name=$1, ip_address=$2, port=$3, username=$4, password=$5, location=$6, status=$7, firmware=$8, enabled=$9, updated_at=NOW() WHERE id=$10`
	_, err := database.DB.Exec(query, d.Name, d.IPAddress, d.Port, d.Username, d.Password, d.Location, d.Status, d.Firmware, d.Enabled, d.ID)
	return err
}

// DeleteDevice deletes a device by ID
func DeleteDevice(id int) error {
	_, err := database.DB.Exec("DELETE FROM devices WHERE id = $1", id)
	return err
}

// UpdateDeviceStatus updates device status and firmware
func UpdateDeviceStatus(id int, status, firmware string) error {
	query := `UPDATE devices SET status=$1, firmware=$2, last_sync=NOW(), updated_at=NOW() WHERE id=$3`
	_, err := database.DB.Exec(query, status, firmware, id)
	return err
}
