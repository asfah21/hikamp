package services

import (
	"ego/models"
	"ego/repositories"
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
