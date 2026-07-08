package services

import (
	"ego/internal/hikvision"
	"ego/models"
	"ego/repositories"
	"fmt"
	"log"
)

// GetAllAudioFiles returns all audio files
func GetAllAudioFiles() ([]models.AudioFile, error) {
	return repositories.GetAllAudioFiles()
}

// GetAudioFileByID returns an audio file by ID
func GetAudioFileByID(id int) (*models.AudioFile, error) {
	return repositories.GetAudioFileByID(id)
}

// CreateAudioFile creates a new audio file record
func CreateAudioFile(f *models.AudioFile) (int, error) {
	return repositories.CreateAudioFile(f)
}

// UpdateAudioFile updates an audio file
func UpdateAudioFile(f *models.AudioFile) error {
	return repositories.UpdateAudioFile(f)
}

// DeleteAudioFile deletes an audio file
func DeleteAudioFile(id int) error {
	return repositories.DeleteAudioFile(id)
}

// SearchAudioFiles searches audio files by name
func SearchAudioFiles(query string) ([]models.AudioFile, error) {
	return repositories.SearchAudioFiles(query)
}

// UpsertAudioFileByHikvisionID inserts or updates an audio file by Hikvision audio ID
func UpsertAudioFileByHikvisionID(f *models.AudioFile) (int, error) {
	return repositories.UpsertAudioFileByHikvisionID(f)
}

// SyncAudioFromDevice performs a full mirror sync from device to local DB.
// Device is the source of truth: deletes local audio not on device, upserts device audio to local.
func SyncAudioFromDevice(deviceID int) (int, error) {
	device, err := GetDeviceByID(deviceID)
	if err != nil {
		return 0, fmt.Errorf("device not found: %w", err)
	}

	client := hikvision.NewClient(device.IPAddress, device.Port, device.Username, device.Password)

	// 1. Fetch all audio from device
	deviceAudioList, err := client.SearchAudio()
	if err != nil {
		return 0, fmt.Errorf("failed to fetch audio from device: %w", err)
	}

	// 2. Get all local audio for this device
	localAudio, err := repositories.GetAudioFilesByDeviceID(deviceID)
	if err != nil {
		return 0, fmt.Errorf("failed to get local audio: %w", err)
	}

	// Build set of device audio IDs for quick lookup
	deviceAudioIDs := make(map[int]bool)
	for _, a := range deviceAudioList {
		deviceAudioIDs[a.CustomAudioID] = true
	}

	// 3. Delete local audio that no longer exists on device
	deleted := 0
	for _, local := range localAudio {
		if local.HikvisionAudioID != nil {
			if !deviceAudioIDs[*local.HikvisionAudioID] {
				if err := repositories.DeleteAudioFile(local.ID); err != nil {
					log.Printf("[AUDIO SYNC FROM] Failed to delete local audio ID %d: %v", local.ID, err)
				} else {
					deleted++
				}
			}
		}
	}

	// 4. Upsert all device audio to local DB
	synced := 0
	for _, audio := range deviceAudioList {
		hikvisionAudioID := audio.CustomAudioID
		hikvisionPath := audio.HikvisionPath
		audioFile := &models.AudioFile{
			Name:             audio.CustomAudioName,
			Category:         "Custom",
			Duration:         audio.Duration,
			DurationStr:      audio.DurationStr,
			FileSize:         int64(audio.AudioFileSize),
			HikvisionAudioID: &hikvisionAudioID,
			HikvisionPath:    &hikvisionPath,
			DeviceID:         &deviceID,
		}
		_, err := repositories.UpsertAudioFileByHikvisionID(audioFile)
		if err != nil {
			log.Printf("[AUDIO SYNC FROM] Failed to upsert audio '%s': %v", audio.CustomAudioName, err)
			continue
		}
		synced++
	}

	log.Printf("[AUDIO SYNC FROM] Device %s: synced %d, deleted %d local orphans", device.Name, synced, deleted)
	return synced, nil
}

// SyncAudioToDevice performs a full mirror sync from local DB to device.
// Local DB is the source of truth: deletes device audio not in local DB, uploads local audio not on device.
func SyncAudioToDevice(deviceID int) (int, error) {
	device, err := GetDeviceByID(deviceID)
	if err != nil {
		return 0, fmt.Errorf("device not found: %w", err)
	}

	client := hikvision.NewClient(device.IPAddress, device.Port, device.Username, device.Password)

	// 1. Fetch all audio from device
	deviceAudioList, err := client.SearchAudio()
	if err != nil {
		return 0, fmt.Errorf("failed to fetch audio from device: %w", err)
	}

	// 2. Get all local audio for this device
	localAudio, err := repositories.GetAudioFilesByDeviceID(deviceID)
	if err != nil {
		return 0, fmt.Errorf("failed to get local audio: %w", err)
	}

	// Build set of local Hikvision audio IDs for quick lookup
	localAudioIDs := make(map[int]bool)
	for _, a := range localAudio {
		if a.HikvisionAudioID != nil {
			localAudioIDs[*a.HikvisionAudioID] = true
		}
	}

	// 3. Delete device audio that no longer exists in local DB
	deleted := 0
	for _, da := range deviceAudioList {
		if !localAudioIDs[da.CustomAudioID] {
			if err := client.DeleteAudio(da.CustomAudioID); err != nil {
				log.Printf("[AUDIO SYNC TO] Failed to delete device audio ID %d: %v", da.CustomAudioID, err)
			} else {
				deleted++
			}
		}
	}

	// 4. Upload local audio that doesn't exist on device
	// Note: We can only upload audio that has a HikvisionAudioID (was previously uploaded)
	// Audio without HikvisionAudioID cannot be uploaded because we don't have the file locally
	for _, local := range localAudio {
		if local.HikvisionAudioID == nil || !localAudioIDs[*local.HikvisionAudioID] {
			// This audio doesn't exist on device, but we can't upload it without the actual file
			// since we don't store files locally. Log a warning.
			log.Printf("[AUDIO SYNC TO] Audio '%s' (ID %d) has no Hikvision ID or not on device — cannot upload (no local file)", local.Name, local.ID)
		}
	}

	log.Printf("[AUDIO SYNC TO] Device %s: deleted %d device orphans, %d local audio(s) already on device", device.Name, deleted, len(localAudio))

	return deleted, nil
}
