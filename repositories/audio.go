package repositories

import (
	"ego/database"
	"ego/models"
)

// GetAllAudioFiles retrieves all audio files
func GetAllAudioFiles() ([]models.AudioFile, error) {
	query := `SELECT id, name, category, duration, file_size, sample_rate, file_path, device_id, created_at, updated_at 
              FROM audio_files ORDER BY id DESC`
	rows, err := database.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []models.AudioFile
	for rows.Next() {
		var f models.AudioFile
		err := rows.Scan(&f.ID, &f.Name, &f.Category, &f.Duration, &f.FileSize, &f.SampleRate, &f.FilePath, &f.DeviceID, &f.CreatedAt, &f.UpdatedAt)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, nil
}

// GetAudioFileByID retrieves an audio file by ID
func GetAudioFileByID(id int) (*models.AudioFile, error) {
	f := &models.AudioFile{}
	query := `SELECT id, name, category, duration, file_size, sample_rate, file_path, device_id, created_at, updated_at 
              FROM audio_files WHERE id = $1`
	err := database.DB.QueryRow(query, id).Scan(&f.ID, &f.Name, &f.Category, &f.Duration, &f.FileSize, &f.SampleRate, &f.FilePath, &f.DeviceID, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return f, nil
}

// CreateAudioFile inserts a new audio file
func CreateAudioFile(f *models.AudioFile) (int, error) {
	var id int
	query := `INSERT INTO audio_files (name, category, duration, file_size, sample_rate, file_path, device_id) 
              VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`
	err := database.DB.QueryRow(query, f.Name, f.Category, f.Duration, f.FileSize, f.SampleRate, f.FilePath, f.DeviceID).Scan(&id)
	return id, err
}

// UpdateAudioFile updates an audio file
func UpdateAudioFile(f *models.AudioFile) error {
	query := `UPDATE audio_files SET name=$1, category=$2, duration=$3, file_size=$4, sample_rate=$5, file_path=$6, device_id=$7, updated_at=NOW() WHERE id=$8`
	_, err := database.DB.Exec(query, f.Name, f.Category, f.Duration, f.FileSize, f.SampleRate, f.FilePath, f.DeviceID, f.ID)
	return err
}

// DeleteAudioFile deletes an audio file by ID
func DeleteAudioFile(id int) error {
	_, err := database.DB.Exec("DELETE FROM audio_files WHERE id = $1", id)
	return err
}

// SearchAudioFiles searches audio files by name
func SearchAudioFiles(query string) ([]models.AudioFile, error) {
	sql := `SELECT id, name, category, duration, file_size, sample_rate, file_path, device_id, created_at, updated_at 
            FROM audio_files WHERE name ILIKE $1 ORDER BY id DESC`
	rows, err := database.DB.Query(sql, "%"+query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []models.AudioFile
	for rows.Next() {
		var f models.AudioFile
		err := rows.Scan(&f.ID, &f.Name, &f.Category, &f.Duration, &f.FileSize, &f.SampleRate, &f.FilePath, &f.DeviceID, &f.CreatedAt, &f.UpdatedAt)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, nil
}
