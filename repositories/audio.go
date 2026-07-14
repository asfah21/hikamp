package repositories

import (
	"ego/database"
	"ego/models"
)

// GetAllAudioFiles retrieves all audio files
func GetAllAudioFiles() ([]models.AudioFile, error) {
	query := `SELECT id, name, category, duration, duration_str, file_size, hikvision_audio_id, hikvision_path, device_id, created_at, updated_at 
              FROM audio_files ORDER BY hikvision_audio_id ASC NULLS LAST, id ASC`
	rows, err := database.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []models.AudioFile
	for rows.Next() {
		var f models.AudioFile
		err := rows.Scan(&f.ID, &f.Name, &f.Category, &f.Duration, &f.DurationStr, &f.FileSize, &f.HikvisionAudioID, &f.HikvisionPath, &f.DeviceID, &f.CreatedAt, &f.UpdatedAt)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return files, nil
}

// GetAudioFileByID retrieves an audio file by ID
func GetAudioFileByID(id int) (*models.AudioFile, error) {
	f := &models.AudioFile{}
	query := `SELECT id, name, category, duration, duration_str, file_size, hikvision_audio_id, hikvision_path, device_id, created_at, updated_at 
              FROM audio_files WHERE id = $1`
	err := database.DB.QueryRow(query, id).Scan(&f.ID, &f.Name, &f.Category, &f.Duration, &f.DurationStr, &f.FileSize, &f.HikvisionAudioID, &f.HikvisionPath, &f.DeviceID, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return f, nil
}

// CreateAudioFile inserts a new audio file
func CreateAudioFile(f *models.AudioFile) (int, error) {
	var id int
	query := `INSERT INTO audio_files (name, category, duration, duration_str, file_size, hikvision_audio_id, hikvision_path, device_id) 
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`
	err := database.DB.QueryRow(query, f.Name, f.Category, f.Duration, f.DurationStr, f.FileSize, f.HikvisionAudioID, f.HikvisionPath, f.DeviceID).Scan(&id)
	return id, err
}

// UpdateAudioFile updates an audio file
func UpdateAudioFile(f *models.AudioFile) error {
	query := `UPDATE audio_files SET name=$1, category=$2, duration=$3, duration_str=$4, file_size=$5, hikvision_audio_id=$6, hikvision_path=$7, device_id=$8, updated_at=NOW() WHERE id=$9`
	_, err := database.DB.Exec(query, f.Name, f.Category, f.Duration, f.DurationStr, f.FileSize, f.HikvisionAudioID, f.HikvisionPath, f.DeviceID, f.ID)
	return err
}

// DeleteAudioFile deletes an audio file by ID
func DeleteAudioFile(id int) error {
	_, err := database.DB.Exec("DELETE FROM audio_files WHERE id = $1", id)
	return err
}

// SearchAudioFiles searches audio files by name
func SearchAudioFiles(query string) ([]models.AudioFile, error) {
	sql := `SELECT id, name, category, duration, duration_str, file_size, hikvision_audio_id, hikvision_path, device_id, created_at, updated_at 
            FROM audio_files WHERE name ILIKE $1 ORDER BY hikvision_audio_id ASC NULLS LAST, id ASC`
	rows, err := database.DB.Query(sql, "%"+query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []models.AudioFile
	for rows.Next() {
		var f models.AudioFile
		err := rows.Scan(&f.ID, &f.Name, &f.Category, &f.Duration, &f.DurationStr, &f.FileSize, &f.HikvisionAudioID, &f.HikvisionPath, &f.DeviceID, &f.CreatedAt, &f.UpdatedAt)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return files, nil
}

// GetAudioFileByHikvisionID retrieves an audio file by Hikvision audio ID
func GetAudioFileByHikvisionID(hikvisionID int) (*models.AudioFile, error) {
	f := &models.AudioFile{}
	query := `SELECT id, name, category, duration, duration_str, file_size, hikvision_audio_id, hikvision_path, device_id, created_at, updated_at 
              FROM audio_files WHERE hikvision_audio_id = $1`
	err := database.DB.QueryRow(query, hikvisionID).Scan(&f.ID, &f.Name, &f.Category, &f.Duration, &f.DurationStr, &f.FileSize, &f.HikvisionAudioID, &f.HikvisionPath, &f.DeviceID, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return f, nil
}

// GetAudioFilesByDeviceID retrieves all audio files for a specific device
func GetAudioFilesByDeviceID(deviceID int) ([]models.AudioFile, error) {
	query := `SELECT id, name, category, duration, duration_str, file_size, hikvision_audio_id, hikvision_path, device_id, created_at, updated_at 
              FROM audio_files WHERE device_id = $1 ORDER BY hikvision_audio_id ASC NULLS LAST, id ASC`
	rows, err := database.DB.Query(query, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []models.AudioFile
	for rows.Next() {
		var f models.AudioFile
		err := rows.Scan(&f.ID, &f.Name, &f.Category, &f.Duration, &f.DurationStr, &f.FileSize, &f.HikvisionAudioID, &f.HikvisionPath, &f.DeviceID, &f.CreatedAt, &f.UpdatedAt)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return files, nil
}

// UpsertAudioFileByHikvisionID inserts or updates an audio file by Hikvision audio ID

func UpsertAudioFileByHikvisionID(f *models.AudioFile) (int, error) {
	var id int
	query := `INSERT INTO audio_files (name, category, duration, duration_str, file_size, hikvision_audio_id, hikvision_path, device_id) 
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
              ON CONFLICT (hikvision_audio_id) DO UPDATE SET
                name = EXCLUDED.name,
                duration = EXCLUDED.duration,
                duration_str = EXCLUDED.duration_str,
                file_size = EXCLUDED.file_size,
                hikvision_path = EXCLUDED.hikvision_path,
                device_id = EXCLUDED.device_id,
                updated_at = NOW()
              RETURNING id`
	err := database.DB.QueryRow(query, f.Name, f.Category, f.Duration, f.DurationStr, f.FileSize, f.HikvisionAudioID, f.HikvisionPath, f.DeviceID).Scan(&id)
	return id, err
}
