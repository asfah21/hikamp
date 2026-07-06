package services

import (
	"ego/internal/hikvision"
	"ego/models"
	"ego/repositories"
	"fmt"
)

// GetAllDevices returns all devices
func GetAllDevices() ([]models.Device, error) {
	return repositories.GetAllDevices()
}

// GetDeviceByID returns a device by ID
func GetDeviceByID(id int) (*models.Device, error) {
	return repositories.GetDeviceByID(id)
}

// CreateDevice creates a new device
func CreateDevice(d *models.Device) (int, error) {
	d.Status = "offline"
	return repositories.CreateDevice(d)
}

// UpdateDevice updates an existing device
func UpdateDevice(d *models.Device) error {
	return repositories.UpdateDevice(d)
}

// DeleteDevice deletes a device
func DeleteDevice(id int) error {
	return repositories.DeleteDevice(id)
}

// TestDeviceConnection tests connection to a device
func TestDeviceConnection(ip string, port int, username, password string) (map[string]string, error) {
	client := hikvision.NewClient(ip, port, username, password)
	info, err := client.DeviceInfo()
	if err != nil {
		return nil, err
	}
	return info, nil
}

// SyncDeviceInfo syncs device information from the actual device
func SyncDeviceInfo(id int) error {
	device, err := repositories.GetDeviceByID(id)
	if err != nil {
		return err
	}

	client := hikvision.NewClient(device.IPAddress, device.Port, device.Username, device.Password)
	info, err := client.DeviceInfo()
	if err != nil {
		repositories.UpdateDeviceStatus(id, "offline", "")
		return err
	}

	firmware := info["firmwareVersion"]
	return repositories.UpdateDeviceStatus(id, "online", firmware)
}

// BroadcastToDevice sends a broadcast command to a Hikvision device.
// audioID is the local audio_files.id — this function looks up the HikvisionAudioID
// to use as customAudioID in the broadcast payload.
func BroadcastToDevice(device *models.Device, audioID int, volume int, durationMinutes int) error {
	// Lookup the audio file to get HikvisionAudioID
	audioFile, err := repositories.GetAudioFileByID(audioID)
	if err != nil {
		return fmt.Errorf("audio file not found (ID: %d): %w", audioID, err)
	}
	if audioFile.HikvisionAudioID == nil || *audioFile.HikvisionAudioID == 0 {
		return fmt.Errorf("audio file '%s' has no Hikvision audio ID. Upload the audio to the device first", audioFile.Name)
	}

	client := hikvision.NewClient(device.IPAddress, device.Port, device.Username, device.Password)

	// Get timezone offset from location settings for proper time formatting
	// Note: getTimezoneOffset returns WITHOUT "+" prefix (Web UI format: "08:00")
	timezoneOffset := "08:00" // default fallback
	location, err := repositories.GetPrayerLocation()
	if err == nil && location.Timezone != "" {
		timezoneOffset = getTimezoneOffset(location.Timezone)
	}

	return client.BroadcastNowWithTimezone(*audioFile.HikvisionAudioID, volume, durationMinutes, timezoneOffset)
}
